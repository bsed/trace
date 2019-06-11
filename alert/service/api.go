package service

import (
	"fmt"
	"sort"
	"time"

	"github.com/bsed/trace/alert/control"
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/util"
	"go.uber.org/zap"
)

// APIAnalyze api 分析
type APIAnalyze struct {
	universalAlert map[string]*Polymerizes // 普通监控
	specialAlert   map[string]*Polymerizes // 特殊监控
}

// apiStats api通用计算
func (a *App) apiStats() {
	for apiStr, polymerizes := range a.apiCache.specialAlert {
		for alertType, polymerizeMap := range polymerizes.Polymerizes {
			a.apiCounter(apiStr, alertType, polymerizeMap, polymerizes, true)
		}
	}

	for apiStr, polymerizes := range a.apiCache.universalAlert {
		for alertType, polymerizeMap := range polymerizes.Polymerizes {
			a.apiCounter(apiStr, alertType, polymerizeMap, polymerizes, false)
		}
	}
}
func newAPIAnalyze() *APIAnalyze {
	return &APIAnalyze{
		universalAlert: make(map[string]*Polymerizes), // 普通监控
		specialAlert:   make(map[string]*Polymerizes), // 特殊监控
	}
}

func (a *App) apiAlarmStore(alert *AlertInfo, alertValue float64, agentID, hostName string, api string, id int64) error {
	var InsertAPIAlertHistory string = `INSERT INTO alert_history (const_id, id, app_name, 
		type, api,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`
	alertName, _ := constant.AlertDesc(alert.Type)
	// logger.Info("Api告警信息", zap.String("appName", a.name), zap.String("api", api), zap.String("agentID", agentID), zap.String("hostName", hostName),
	// 	zap.String("策略名", alertName), zap.Float64("策略阀值", alert.Value), zap.String("策略单位", alert.Unit), zap.Int("对比类型", alert.Compare),
	// 	zap.Float64("告警数据", alarmValue), zap.String("告警时间", time.Now().String()))
	cql := gAlert.GettraceCql()
	if cql == nil {
		logger.Warn("get cql failed")
		return fmt.Errorf("get cql failed")
	}

	tmpAlert := &util.Alert{
		Name:     alertName,
		Compare:  alert.Compare,
		Unit:     alert.Unit,
		Duration: alert.Duration,
		Value:    alert.Value,
	}

	query := cql.Query(InsertAPIAlertHistory,
		1,
		id,
		a.name,
		1,
		api,
		tmpAlert,
		alertValue,
		a.policy.Channel,
		a.policy.Users,
		time.Now().Unix(),
	)

	if err := query.Exec(); err != nil {
		logger.Warn("alarm store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

// apiCounter api计算是否需要告警
func (a *App) apiCounter(apiStr string, alertType int, polymerizeMap map[int64]*Polymerize, polymerizes *Polymerizes, isSpecial bool) {
	var alert *AlertInfo
	var ok bool
	// 查找策略,普通api监控
	if !isSpecial {
		alert, ok = a.Alerts[alertType]
		if !ok {
			// 找不到策略直接删除
			delete(polymerizes.Polymerizes, alertType)
			return
		}
	} else {
		// 指定api监控
		// 查找API所有策略
		alerts, ok := a.SpecialAlert.API[apiStr]
		if !ok {
			// 找不到API直接删除
			delete(a.apiCache.specialAlert, apiStr)
			return
		}
		// 查找API符合alertType的策略
		alert, ok = alerts[alertType]
		if !ok {
			// 找不到策略直接删除
			delete(polymerizes.Polymerizes, alertType)
			return
		}
	}

	// 清空之前节点
	a.orderly = a.orderly[:0]
	// 赋值
	for key := range polymerizeMap {
		a.orderly = append(a.orderly, key)
	}

	// 排序，在告警服务中数据点非常少，所以排序性能问题不用过多考虑
	sort.Sort(a.orderly)
	// 如果没有计算节点直接返回
	if a.orderly.Len() <= 0 {
		return
	}
	firstIndex := a.orderly[0] // 第一个点
	statsFlg := false
	for index := len(a.orderly) - 1; index >= 0; index-- {
		if a.orderly[index] >= firstIndex+int64((alert.Duration-1)*60) {
			statsFlg = true
			break
		}
	}

	lostData := false
	// 数据没来的情况直接删除所以节点，不需要滑动了
	if !statsFlg {
		// 取当前时间,节点不够，可能是数据还没到，或者就没有这个数据，所以取当前时间对比一下延迟2分钟
		now := time.Now()
		// 取整点分钟的秒
		roundMin := now.Unix() - int64(now.Second())
		// 延迟2分钟没数据，那么表示可以计算了
		if roundMin >= firstIndex+int64(alert.Duration*60)+60 {
			statsFlg = true
			lostData = true
		}
	}
	// 通过上面的条件判断是否需要进行聚合计算
	if statsFlg {
		var isAlarm bool
		polymerize := newPolymerize()
		for index := 0; index < alert.Duration; index++ {
			pointIndex := int64(index*60) + firstIndex
			tmpPolymerize, ok := polymerizeMap[pointIndex]
			if ok {
				polymerize.Count += tmpPolymerize.Count
				polymerize.ErrCount += tmpPolymerize.ErrCount
				polymerize.Duration += tmpPolymerize.Duration
				polymerize.Value += tmpPolymerize.Value
				// 这里只删除一个点就可以做成滑动窗口了,如果是数据延迟很多的情况那么全部删除计算几点
				if index == 0 || lostData == true {
					delete(polymerizeMap, pointIndex)
				}
			}
		}
		// 通过不同告警类型来计算
		switch alertType {
		// 接口平均耗时
		case constant.ALERT_APM_API_DURATION:
			if polymerize.Count != 0 {
				polymerize.Value = float64(polymerize.Duration) / float64(polymerize.Count)
				isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
			}
			break
		// 接口访问次数
		case constant.ALERT_APM_API_COUNT:
			polymerize.Value = float64(polymerize.Count)
			isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
		// 接口错误次数
		case constant.ALERT_APM_API_ERROR_COUNT:
			polymerize.Value = float64(polymerize.ErrCount)
			isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
			break
		// 接口错误率
		case constant.ALERT_APM_API_ERROR_RATIO:
			if polymerize.Count != 0 {
				polymerize.Value = (float64(polymerize.ErrCount) / float64(polymerize.Count)) * 100
				isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
			}
			break
		}

		id := gAlert.getAlertID()

		msg := &control.AlarmMsg{
			AppName:        a.name,
			Type:           alertType,
			API:            apiStr,
			ThresholdValue: alert.Value,
			AlertValue:     polymerize.Value,
			Channel:        a.policy.Channel,
			Users:          a.policy.Users,
			Time:           time.Now().Unix(),
			IsRecovery:     isAlarm,
			Unit:           alert.Unit,
			ID:             id,
		}
		if err := gAlert.control.AlertPush(msg); err != nil {
			logger.Warn("alert push error", zap.String("error", err.Error()))
		}
	}
}

func (a *App) apiStorePolymerize(api *alert.API, alertType int, dataTime int64) {
	// 查找该api是否是特殊计算的api,如果存在那么查找api的几个指标是否已经过滤过了
	specialAlerts, ok := a.SpecialAlert.API[api.Desc]
	if ok {
		if _, ok := specialAlerts[alertType]; ok {
			return
		}
	}
	// 查找通用策略类型
	alert, ok := a.Alerts[alertType]
	if ok {
		polymerizes, ok := a.apiCache.universalAlert[api.Desc]
		if !ok {
			polymerizes = newPolymerizes()
			a.apiCache.universalAlert[api.Desc] = polymerizes
		}

		polymerizeMap, ok := polymerizes.Polymerizes[alert.Type]
		if !ok {
			polymerizeMap = make(map[int64]*Polymerize)
			polymerizes.Polymerizes[alert.Type] = polymerizeMap
		}

		// 时间戳为key保存计算数据，用来聚合
		polymerize, ok := polymerizeMap[dataTime]
		if !ok {
			polymerize = newPolymerize()
			polymerizeMap[dataTime] = polymerize
		}

		// 保存数据
		polymerize.Count = api.AccessCount
		polymerize.ErrCount = api.AccessErrCount
		polymerize.Duration = api.Duration
		logger.Debug("api universal analyze", zap.Int("alert.Type", alert.Type), zap.String("appName", a.name), zap.String("api", api.Desc), zap.Any("polymerize", polymerize))
	}
}

// apiSpecialAnalyze api特殊监控计算
func (a *App) apiSpecialCache(api *alert.API, alert *AlertInfo, dataTime int64) {
	// 先查找这个api是否有已经缓存了特殊监控
	polymerizes, ok := a.apiCache.specialAlert[api.Desc]
	if !ok {
		polymerizes = newPolymerizes()
		a.apiCache.specialAlert[api.Desc] = polymerizes
	}

	// 查找策略是否已经缓存
	polymerizeMap, ok := polymerizes.Polymerizes[alert.Type]
	if !ok {
		polymerizeMap = make(map[int64]*Polymerize) //newPolymerize()
		polymerizes.Polymerizes[alert.Type] = polymerizeMap
	}
	// 时间戳为key保存计算数据，用来聚合
	polymerize, ok := polymerizeMap[dataTime]
	if !ok {
		polymerize = newPolymerize()
		polymerizeMap[dataTime] = polymerize
	}

	// 保存数据
	polymerize.Count = api.AccessCount
	polymerize.ErrCount = api.AccessErrCount
	polymerize.Duration = api.Duration
}

// apiUniversalCache api通用计算
func (a *App) apiUniversalCache(api *alert.API, dataTime int64) {
	a.apiStorePolymerize(api, constant.ALERT_APM_API_COUNT, dataTime)
	a.apiStorePolymerize(api, constant.ALERT_APM_API_DURATION, dataTime)
	a.apiStorePolymerize(api, constant.ALERT_APM_API_ERROR_COUNT, dataTime)
	a.apiStorePolymerize(api, constant.ALERT_APM_API_ERROR_RATIO, dataTime)
}

func (a *App) loadAPIAlerts(tmpapiAlerts []*util.ApiAlert) {
	for _, tmpAPIAlert := range tmpapiAlerts {
		for _, tmpalert := range tmpAPIAlert.Alerts {
			alertType, ok := constant.AlertType(tmpalert.Key)
			if !ok {
				logger.Warn("alertType unfind error", zap.String("name", tmpalert.Key))
				continue
			}
			// 到通用alert列表中去找同类型的alert,然后复用该alert里面的值，找不到直接返回
			universalAlert, ok := a.Alerts[alertType]
			if !ok {
				logger.Warn("alert unfind error", zap.String("name", tmpalert.Key), zap.Int("alertType", alertType), zap.String("appName", a.name))
				continue
			}
			// 创建特殊alert，然后赋值（复用universalAlert中的值）
			specialAlerts, ok := a.SpecialAlert.API[tmpAPIAlert.Api]
			if !ok {
				// 如果map不存在，那么申请
				specialAlerts = make(map[int]*AlertInfo)
				a.SpecialAlert.API[tmpAPIAlert.Api] = specialAlerts
			}
			specialAlert := newAlertInfo()
			specialAlert.Type = universalAlert.Type
			specialAlert.Compare = universalAlert.Compare
			specialAlert.Duration = universalAlert.Duration
			specialAlert.Unit = universalAlert.Unit
			specialAlert.Value = tmpalert.Value
			// 保存
			specialAlerts[universalAlert.Type] = specialAlert
		}
	}
}

// 检查是有有api的计算策略,没有直接丢弃包文，减少计算量
func (a *App) apiFilter() bool {
	if _, ok := a.Alerts[constant.ALERT_APM_API_ERROR_RATIO]; ok {
		return true
	}
	if _, ok := a.Alerts[constant.ALERT_APM_API_ERROR_COUNT]; ok {
		return true
	}
	if _, ok := a.Alerts[constant.ALERT_APM_API_DURATION]; ok {
		return true
	}
	if _, ok := a.Alerts[constant.ALERT_APM_API_COUNT]; ok {
		return true
	}
	return false
}

// APICache api计算分析
func (a *App) APICache(apis *alert.APIs, dataTime int64) {
	for _, api := range apis.APIS {
		// api信息不正确直接丢弃
		if len(api.Desc) <= 0 {
			continue
		}
		// 特殊api策略数据统计
		specialAlerts, ok := a.SpecialAlert.API[api.Desc]
		if ok {
			for _, specialAlert := range specialAlerts {
				a.apiSpecialCache(api, specialAlert, dataTime)
			}
		}
		// 普通监控
		a.apiUniversalCache(api, dataTime)
	}
}
