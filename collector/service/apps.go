package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocql/gocql"
	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/bsed/trace/pkg/sql"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

// Apps 所有app服务信息
type Apps struct {
	sync.RWMutex
	apps  map[string]*App // app集合
	ips   map[string]string
	hosts map[string]string
	dubbo *DubboAPIMap // dubbo api映射记录
}

func (a *Apps) start() error {
	if err := a.loadAppsSrv(); err != nil {
		return err
	}

	if err := a.loadApiCodeSrv(); err != nil {
		return err
	}

	if err := a.loadAppNameDubboSrv(); err != nil {
		return err
	}
	return nil
}

// loadApiCode 加载api code
func (a *Apps) loadApiCode(cql *gocql.Session) error {
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
		app, ok := a.apps[name]
		a.RUnlock()
		// 如果app不存在直接返回即可
		if !ok {
			continue
		}
		// 检查策略是否不更新
		if app.policyUpdateDate == updateDate {
			app.checkTime = checkTime
			continue
		}

		// 策略被更新，需要删除
		app.clearCode()
		var tmpapiAlerts []*util.ApiAlert
		if err := json.Unmarshal([]byte(apiAlertsStr), &tmpapiAlerts); err != nil {
			logger.Warn("json Unmarshal", zap.String("error", err.Error()))
			continue
		}
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
			alertType, ok := constant.AlertType(tmpAlert.Name)
			if !ok {
				logger.Warn("alertType unfind error", zap.String("name", tmpAlert.Name))
				continue
			}

			if alertType != constant.ALERT_APM_API_ERROR_RATIO && alertType != constant.ALERT_APM_API_ERROR_COUNT {
				continue
			}
			codesStr := strings.Split(tmpAlert.Keys, ",")
			for _, codeStr := range codesStr {
				code, err := strconv.Atoi(codeStr)
				if err == nil {
					app.addCode(int32(code))
				}
			}
		}
	}

	return nil
}

// loadApiCode 加载api code
func (a *Apps) loadApiCodeSrv() error {
	cql := gCollector.storage.GetStaticCql()
	if err := a.loadApiCode(cql); err != nil {
		logger.Warn("load api code", zap.String("error", err.Error()))
		return err
	}

	go func() {
		for {
			time.Sleep(time.Duration(misc.Conf.Apps.LoadInterval) * time.Second)
			cql := gCollector.storage.GetStaticCql()
			if err := a.loadApiCode(cql); err != nil {
				logger.Warn("load api code", zap.String("error", err.Error()))
			}
		}
	}()
	return nil
}

// loadAppsStart 加载app
func (a *Apps) loadAppsSrv() error {
	cql := gCollector.storage.GetStaticCql()
	if err := a.loadApps(cql); err != nil {
		logger.Warn("loadApps", zap.String("error", err.Error()))
		return err
	}

	go func() {
		for {
			time.Sleep(time.Duration(misc.Conf.Apps.LoadInterval) * time.Second)
			cql := gCollector.storage.GetStaticCql()
			if err := a.loadApps(cql); err != nil {
				logger.Warn("loadApps", zap.String("error", err.Error()))
			}
		}
	}()
	return nil
}

func (a *Apps) loadApps(cql *gocql.Session) error {
	if cql == nil {
		return fmt.Errorf("get cql failed")
	}
	appsIter := cql.Query(sql.LoadApps).Consistency(gocql.One).Iter()
	defer func() {
		if err := appsIter.Close(); err != nil {
			logger.Warn("close apps iter error:", zap.Error(err))
		}
	}()

	var appName string
	for appsIter.Scan(&appName) {
		var appType int32
		var agentID, ip string
		var startTime int64
		var isLive bool
		var hostName string

		// 不管有没有agent， 都先存一下app
		a.storeApp(appName)

		agentsIter := cql.Query(sql.LoadAgents, appName).Consistency(gocql.One).Iter()
		for agentsIter.Scan(&appType, &agentID, &startTime, &ip, &isLive, &hostName) {
			a.storeAgent(appName, agentID, appType, startTime, isLive, hostName, ip)
			a.storeIPandHost(appName, ip, hostName)
		}

		if err := agentsIter.Close(); err != nil {
			logger.Warn("close apps iter error:", zap.Error(err))
		}
	}

	return nil
}

// loadAppsStart 加载app
func (a *Apps) loadAppNameDubboSrv() error {
	cql := gCollector.storage.GetStaticCql()
	if err := a.loadAppNameDubbo(cql); err != nil {
		logger.Warn("load app name by dubbo type", zap.String("error", err.Error()))
		return err
	}

	go func() {
		for {
			time.Sleep(time.Duration(misc.Conf.Apps.LoadInterval) * time.Second)
			cql := gCollector.storage.GetStaticCql()
			if err := a.loadAppNameDubbo(cql); err != nil {
				logger.Warn("load app name by dubbo type", zap.String("error", err.Error()))
			}
		}
	}()
	return nil
}

func (a *Apps) loadAppNameDubbo(cql *gocql.Session) error {
	if cql == nil {
		return fmt.Errorf("get cql failed")
	}
	apisIter := cql.Query(sql.LoadDubboApis).Consistency(gocql.One).Iter()
	defer func() {
		if err := apisIter.Close(); err != nil {
			logger.Warn("close apps iter error:", zap.Error(err))
		}
	}()

	var appName, api string
	var apiType int32
	for apisIter.Scan(&appName, &api, &apiType) {
		if int16(apiType) == constant.DUBBO_PROVIDER {
			gCollector.apps.dubbo.Add(api, appName)
		}
	}
	return nil
}

// isExist app是否存在
func (a *Apps) storeIPandHost(appName, ip, host string) bool {
	a.Lock()
	a.ips[ip] = appName
	a.hosts[ip] = appName
	a.Unlock()
	return true
}

func (a *Apps) getNameByIP(ip string) (string, bool) {
	a.RLock()
	name, ok := a.ips[ip]
	a.RUnlock()
	return name, ok
}

func (a *Apps) getNameByHost(host string) (string, bool) {
	a.RLock()
	name, ok := a.hosts[host]
	a.RUnlock()
	return name, ok
}

func (a *Apps) storeApp(appName string) {
	a.RLock()
	app, ok := a.apps[appName]
	a.RUnlock()
	if !ok {
		app = newApp(appName)
		a.Lock()
		a.apps[appName] = app
		a.Unlock()
		app.start()
	}
}

func (a *Apps) storeAgent(appName, agentID string, appType int32, startTime int64, isLive bool, hostName, ip string) {
	a.RLock()
	app, ok := a.apps[appName]
	a.RUnlock()
	if !ok {
		app = newApp(appName)
		a.Lock()
		a.apps[appName] = app
		a.Unlock()
		app.start()
	}
	app.appType = appType
	app.storeAgent(agentID, isLive)
}

func newApps() *Apps {
	return &Apps{
		apps:  make(map[string]*App),
		ips:   make(map[string]string),
		hosts: make(map[string]string),
		dubbo: NewDubboAPIMap(),
	}
}

func (a *Apps) getApp(appName string) (*App, bool) {
	a.RLock()
	app, ok := a.apps[appName]
	a.RUnlock()
	return app, ok
}

// routerSapn 路由span
func (a *Apps) routerStatBatch(appName, agentID string, stats *pinpoint.TAgentStatBatch) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}

	// 接收 stat
	for _, stat := range stats.AgentStats {
		if err := app.recvAgentStat(appName, agentID, stat); err != nil {
			logger.Warn("recv agent stat", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
			return err
		}
	}

	return nil
}

// routerSapn 路由span
func (a *Apps) routerStat(appName, agentID string, stat *pinpoint.TAgentStat) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}

	// 接收 stat
	if err := app.recvAgentStat(appName, agentID, stat); err != nil {
		logger.Warn("recv agent stat", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// routerSapn 路由span
func (a *Apps) routerSapn(appName, agentID string, span *trace.TSpan) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}

	// 接收span
	if err := app.recvSpan(appName, agentID, span); err != nil {
		logger.Warn("recv span", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// routerSapnChunk 路由sapnChunk
func (a *Apps) routersapnChunk(appName, agentID string, spanChunk *trace.TSpanChunk) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}

	// 接收spanChunk
	if err := app.recvSpanChunk(appName, agentID, spanChunk); err != nil {
		logger.Warn("recv spanChunk", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// routerAgentStat 路由agentStat
func (a *Apps) routerAgentStat(appName, agentID string, agentStat *pinpoint.TAgentStat) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}

	// 接收agent Stat
	if err := app.recvAgentStat(appName, agentID, agentStat); err != nil {
		logger.Warn("recv spanChunk", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// routerAgentStatBatch 路由agentStatBatch
func (a *Apps) routerAgentStatBatch(appName, agentID string, agentStatBatch *pinpoint.TAgentStatBatch) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	for _, agentStat := range agentStatBatch.AgentStats {
		// 接收agent Stat
		if err := app.recvAgentStat(appName, agentID, agentStat); err != nil {
			logger.Warn("recv spanChunk", zap.String("appName", appName), zap.String("agentID", agentID), zap.String("error", err.Error()))
			return err
		}
	}
	return nil
}

func (a *Apps) routerApi(packet *alert.Data) error {
	app, ok := a.getApp(packet.AppName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", packet.AppName)
	}

	if err := app.recvApi(packet); err != nil {
		logger.Warn("recv api", zap.String("appName", packet.AppName), zap.String("error", err.Error()))
		return err
	}
	return nil
}

func (a *Apps) online(appName, agentID string) error {
	app, ok := a.getApp(appName)
	if !ok {
		return fmt.Errorf("unfind app, app name is %s", appName)
	}
	return app.online(agentID)
}
