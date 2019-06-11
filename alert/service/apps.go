package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bsed/trace/alert/misc"
	"github.com/bsed/trace/pkg/alert"

	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/sql"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

// Apps apps
type Apps struct {
	sync.RWMutex
	Apps          map[string]*App
	Cache         map[string]struct{} // app 缓存
	DefaultAlerts []*AlertInfo        // 默认alerts
}

func newApps() *Apps {
	return &Apps{
		Apps:  make(map[string]*App),
		Cache: make(map[string]struct{}), // 缓存app以及该app的策略更新时间
	}
}

func (a *Apps) start() error {
	// 加载所有app
	if err := a.cacheApps(); err != nil {
		logger.Warn("load apps error", zap.String("error", err.Error()))
		return err
	}

	// 自动加载默认策略
	if err := a.loadDefaultPolicy(); err != nil {
		logger.Warn("load policy error", zap.String("error", err.Error()))
		return err
	}

	// 自动扫描策略
	if err := a.loadPolicy(); err != nil {
		logger.Warn("load policy error", zap.String("error", err.Error()))
		return err
	}

	// 设置默认策略
	if err := a.setDefaultPolicy(); err != nil {
		logger.Warn("load policy error", zap.String("error", err.Error()))
	}

	go func() {
		for {
			time.Sleep(time.Duration(misc.Conf.App.LoadInterval) * time.Second)
			// 加载app
			if err := a.cacheApps(); err != nil {
				logger.Warn("load apps error", zap.String("error", err.Error()))
			}

			// 加载策略
			if err := a.loadPolicy(); err != nil {
				logger.Warn("load policy error", zap.String("error", err.Error()))
			}
		}
	}()

	return nil
}

func (a *Apps) loadPolicy() error {
	cql := gAlert.GetStaticCql()
	if cql == nil {
		return fmt.Errorf("unfind cql")
	}

	query := cql.Query(sql.LoadPolicys).Iter()
	defer func() {
		if err := query.Close(); err != nil {
			logger.Warn("close iter error:", zap.Error(err))
		}
	}()
	var name, owner, apiAlertsStr, channel, group, policyID string
	var users []string
	var updateDate int64

	checkTime := time.Now().Unix()
	for query.Scan(&name, &owner, &apiAlertsStr, &channel, &group, &policyID, &updateDate, &users) {
		a.RLock()
		app, ok := a.Apps[name]
		a.RUnlock()
		// 如果已经存在策略并且updatedate不相当，那么删除历史
		if ok {
			if app.policy.UpdateDate == updateDate {
				app.policy.checkTime = checkTime
				continue
			}
			// log.Println("删除更新策略", name)
			// 定时任务移除
			gAlert.tickers.RemoveTask(app.taskID)
			// 策略被更新，需要删除
			a.remove(name)
		}
		var tmpapiAlerts []*util.ApiAlert
		if err := json.Unmarshal([]byte(apiAlertsStr), &tmpapiAlerts); err != nil {
			logger.Warn("json Unmarshal", zap.String("error", err.Error()))
			continue
		}
		app = newApp()
		app.name = name
		app.policy.AppName = name
		app.policy.Owner = owner
		app.policy.Channel = channel
		app.policy.Group = group
		app.policy.ID = policyID
		app.policy.UpdateDate = updateDate
		app.policy.Users = users
		app.policy.checkTime = checkTime
		// 自定义模版
		app.policyType = constant.POLICY_Type_CUSTOM

		// 根据alertid加具体载策略,如果policyID为null那么代表该模版的策略被删除，所以不用统计
		if len(policyID) == 0 {
			continue
		}

		alertsQuery := cql.Query(sql.LoadAlert, policyID)
		var tmpAlerts []*util.Alert
		if err := alertsQuery.Scan(&tmpAlerts); err != nil {
			logger.Warn("load alert scan error", zap.String("error", err.Error()), zap.String("sql", sql.LoadAlert))
			continue
		}
		if len(tmpAlerts) == 0 {
			continue
		}

		for _, tmpAlert := range tmpAlerts {
			alert := newAlertInfo()
			alert.Compare = tmpAlert.Compare
			alert.Duration = tmpAlert.Duration
			alert.Value = tmpAlert.Value
			alert.Keys = strings.Split(tmpAlert.Keys, ",")
			alertType, ok := constant.AlertType(tmpAlert.Name)
			alert.Unit = tmpAlert.Unit
			if !ok {
				logger.Warn("alertType unfind error", zap.String("name", tmpAlert.Name))
				continue
			}
			alert.Type = alertType
			app.Alerts[alertType] = alert
		}
		// 加载特殊监控
		app.loadAPIAlerts(tmpapiAlerts)
		// app start
		if err := app.start(); err != nil {
			logger.Warn("app start", zap.String("error", err.Error()), zap.String("appName", name))
			continue
		}
		taskID := gAlert.tickers.NewID()
		app.taskID = taskID
		gAlert.tickers.AddTask(taskID, app.tChan)
		// 保存策略
		a.add(app)
	}

	// 对比模版checktime，发现checktime不相等，那么代表该模版已经被删除
	a.checkVersion(checkTime)
	return nil
}

func (a *Apps) checkVersion(checkTime int64) {
	a.Lock()
	for name, app := range a.Apps {
		// 默认模版没有更新时间，直接跳过
		if app.policyType == constant.POLICY_Type_DEFAULT {
			continue
		}
		if app.policy.checkTime != checkTime {
			// 定时任务移除
			gAlert.tickers.RemoveTask(app.taskID)
			// 策略被更新，需要删除
			app.close()
			delete(a.Apps, name)
		}
	}
	a.Unlock()
}

func (a *Apps) remove(name string) {
	a.RLock()
	app, ok := a.Apps[name]
	a.RUnlock()
	if !ok {
		return
	}

	app.close()

	a.Lock()
	delete(a.Apps, name)
	a.Unlock()
}

func (a *Apps) add(app *App) {
	a.Lock()
	a.Apps[app.policy.AppName] = app
	a.Unlock()
}

func (a *Apps) nolockAdd(app *App) {
	a.Apps[app.policy.AppName] = app
}

// apiRouter api 路由
func (a *Apps) apiRouter(appName string, alertData *alert.Data) error {
	a.RLock()
	app, ok := a.Apps[appName]
	a.RUnlock()
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	app.apisRecv(alertData)
	return nil
}

// sqlRouter sql 路由
func (a *Apps) sqlRouter(appName string, alertData *alert.Data) error {
	a.RLock()
	app, ok := a.Apps[appName]
	a.RUnlock()
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	app.sqlsRecv(alertData)
	return nil
}

// exRouter 异常路由
func (a *Apps) exRouter(appName string, alertData *alert.Data) error {
	a.RLock()
	app, ok := a.Apps[appName]
	a.RUnlock()
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	app.exsRecv(alertData)
	return nil
}

// runtimeRoute 异常路由
func (a *Apps) runtimeRoute(appName string, alertData *alert.Data) error {
	a.RLock()
	app, ok := a.Apps[appName]
	a.RUnlock()
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	app.runtimeRecv(alertData)
	return nil
}

// cacheApps 缓存所有应用
func (a *Apps) cacheApps() error {
	cql := gAlert.GetStaticCql()
	if cql == nil {
		return fmt.Errorf("unfind cql")
	}

	query := cql.Query(sql.LoadApps).Iter()
	defer func() {
		if err := query.Close(); err != nil {
			logger.Warn("loadApps close iter error:", zap.Error(err))
		}
	}()
	var name string
	for query.Scan(&name) {
		a.Lock()
		if _, ok := a.Cache[name]; !ok {
			a.Cache[name] = struct{}{}
		}
		// log.Println("app name is", name)
		a.Unlock()
	}
	return nil
}

// loadDefaultPolicy 加载默认策略
func (a *Apps) loadDefaultPolicy() error {
	cql := gAlert.GetStaticCql()
	if cql == nil {
		return fmt.Errorf("unfind cql")
	}
	alertsQuery := cql.Query(sql.LoaddefaultPolicy)
	var tmpAlerts []*util.Alert
	if err := alertsQuery.Scan(&tmpAlerts); err != nil {
		logger.Warn("load alert scan error", zap.String("error", err.Error()), zap.String("sql", sql.LoadAlert))
		return err
	}
	if len(tmpAlerts) == 0 {
		return nil
	}

	for _, tmpAlert := range tmpAlerts {
		alert := newAlertInfo()
		alert.Compare = tmpAlert.Compare
		alert.Duration = tmpAlert.Duration
		alert.Value = tmpAlert.Value
		alert.Keys = strings.Split(tmpAlert.Keys, ",")
		alertType, ok := constant.AlertType(tmpAlert.Name)
		alert.Unit = tmpAlert.Unit
		if !ok {
			logger.Warn("alertType unfind error", zap.String("name", tmpAlert.Name))
			continue
		}
		alert.Type = alertType
		a.DefaultAlerts = append(a.DefaultAlerts, alert)
	}
	return nil
}

func (a *Apps) setDefaultPolicy() error {
	a.Lock()
	for appName := range a.Cache {
		_, ok := a.Apps[appName]
		if !ok {
			// 添加策略
			app := newApp()
			app.name = appName
			app.policy.AppName = appName
			app.policy.Owner = "13269"
			app.policy.Channel = "email"
			app.policy.UpdateDate = time.Now().Unix()
			app.policy.Users = []string{"13269"}
			app.policyType = constant.POLICY_Type_DEFAULT

			for _, alert := range a.DefaultAlerts {
				app.Alerts[alert.Type] = alert
			}

			// 默认策略加载
			taskID := gAlert.tickers.NewID()
			app.taskID = taskID
			gAlert.tickers.AddTask(taskID, app.tChan)
			// log.Println("设置默认策略", appName, a.DefaultAlerts)
			// 保存策略
			a.nolockAdd(app)
		}
	}
	a.Unlock()
	return nil
}
