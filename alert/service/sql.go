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

// SQLAnalyze sql分析
type SQLAnalyze struct {
	sqlAlert map[int32]*Polymerizes
}

func newSQLAnalyze() *SQLAnalyze {
	return &SQLAnalyze{
		sqlAlert: make(map[int32]*Polymerizes),
	}
}

// SQLCache sql计算分析
func (a *App) SQLCache(sqls *alert.SQLs, dataTime int64) {
	for _, sql := range sqls.SQLs {
		// 普通监控
		a.sqlcache(constant.ALERT_APM_SQL_ERROR_RATIO, sql, dataTime)
	}
}

// sqlAnalyze ...
func (a *App) sqlcache(alertType int, sql *alert.SQL, dataTime int64) {
	// 查找通用策略类型
	alert, ok := a.Alerts[alertType]
	if ok {
		polymerizes, ok := a.sqlCache.sqlAlert[sql.ID]
		if !ok {
			polymerizes = newPolymerizes()
			a.sqlCache.sqlAlert[sql.ID] = polymerizes
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
		polymerize.Count = sql.Count
		polymerize.ErrCount = sql.Errcount
		polymerize.Duration = sql.Duration
		logger.Debug("sql analyze", zap.Int("alert.Type", alert.Type), zap.String("appName", a.name), zap.Int32("sqlID", sql.ID), zap.Any("polymerize", polymerize))
	}
}

func (a *App) sqlAlarmStore(alert *AlertInfo, alertValue float64, agentID, hostName string, sqlID int32) error {
	var InsertAPIAlertHistory string = `INSERT INTO alert_history (const_id, id, app_name, 
		type, api,  alert, alert_value, channel, users, input_date) VALUES (?,?,?,?,?,?,?,?,?,?);`
	alertName, _ := constant.AlertDesc(alert.Type)
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
		time.Now().UnixNano(),
		a.name,
		1,
		fmt.Sprintf("%d", sqlID),
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

// sqlStats sql通用计算
func (a *App) sqlStats() {
	for sqlID, polymerizes := range a.sqlCache.sqlAlert {
		for alertType, polymerizeMap := range polymerizes.Polymerizes {
			a.sqlCounter(sqlID, alertType, polymerizeMap, polymerizes)
		}
	}
}

// sqlCounter sql计算是否需要告警
func (a *App) sqlCounter(sqlID int32, alertType int, polymerizeMap map[int64]*Polymerize, polymerizes *Polymerizes) {
	// 查找SQL符合alertType的策略
	alert, ok := a.Alerts[alertType]
	if !ok {
		// 找不到策略直接删除
		delete(polymerizes.Polymerizes, alertType)
		return
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
		// sql错误率
		case constant.ALERT_APM_SQL_ERROR_RATIO:
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
			SQL:            fmt.Sprintf("%d", sqlID),
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
