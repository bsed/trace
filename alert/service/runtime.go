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

func (a *App) runtimeCounter() {
	for agentID, cpuRuntimes := range a.runtimeCache.cpuload {
		a.cpuloadStats(agentID, cpuRuntimes)
	}

	for agentID, jvmRuntimes := range a.runtimeCache.jvmHeap {
		a.jvmHeapStats(agentID, jvmRuntimes)
	}
}

// exStats 内部异常计算
func (a *App) cpuloadStats(agentID string, cpuloadMap map[int64]*CpuloadPolymerize) {
	alert, ok := a.Alerts[constant.ALERT_APM_CPU_USED_RATIO]
	if !ok {
		delete(a.runtimeCache.cpuload, agentID)
		return
	}
	// 清空之前节点
	a.orderly = a.orderly[:0]
	// 赋值
	for key := range cpuloadMap {
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
		polymerize := newCpuloadPolymerize()
		for index := 0; index < alert.Duration; index++ {
			pointIndex := int64(index*60) + firstIndex
			tmpPolymerize, ok := cpuloadMap[pointIndex]
			if ok {
				polymerize.SystemCpuload += tmpPolymerize.SystemCpuload
				polymerize.JVMCpuload += tmpPolymerize.JVMCpuload
				polymerize.Count += tmpPolymerize.Count
				// 这里只删除一个点就可以做成滑动窗口了,如果是数据延迟很多的情况那么全部删除计算几点
				if index == 0 || lostData == true {
					delete(cpuloadMap, pointIndex)
				}
			}
		}

		if polymerize.Count != 0 {
			polymerize.Value = (float64(polymerize.SystemCpuload) / float64(alert.Duration)) * 100
			isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
			// log.Println("cpu计算信息", polymerize.Value, polymerize.SystemCpuload, alert.Duration, isAlarm)
			// 告警信息入库
			id := gAlert.getAlertID()
			msg := &control.AlarmMsg{
				AppName:        a.name,
				AgentID:        agentID,
				Type:           constant.ALERT_APM_CPU_USED_RATIO,
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

}

// jvmHeapStats 内部异常计算
func (a *App) jvmHeapStats(agentID string, jvmHeapMap map[int64]*JVMHeapPolymerize) {
	alert, ok := a.Alerts[constant.ALERT_APM_MEM_USED_RATION]
	if !ok {
		delete(a.runtimeCache.jvmHeap, agentID)
		return
	}
	// 清空之前节点
	a.orderly = a.orderly[:0]
	// 赋值
	for key := range jvmHeapMap {
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
		polymerize := newJVMHeapPolymerize()
		for index := 0; index < alert.Duration; index++ {
			pointIndex := int64(index*60) + firstIndex
			tmpPolymerize, ok := jvmHeapMap[pointIndex]
			if ok {
				polymerize.JVMHeap += tmpPolymerize.JVMHeap
				polymerize.Count += tmpPolymerize.Count
				// 这里只删除一个点就可以做成滑动窗口了,如果是数据延迟很多的情况那么全部删除计算几点
				if index == 0 || lostData == true {
					delete(jvmHeapMap, pointIndex)
				}
			}
		}
		if polymerize.Count != 0 {
			polymerize.Value = (float64(polymerize.JVMHeap) / float64(polymerize.Count)) / (1024 * 1024)
			isAlarm = compare(polymerize.Value, alert.Value, alert.Compare)
			id := gAlert.getAlertID()
			msg := &control.AlarmMsg{
				AppName:        a.name,
				AgentID:        agentID,
				Type:           constant.ALERT_APM_MEM_USED_RATION,
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

}

// RuntimeAnalyze runtime分析
type RuntimeAnalyze struct {
	cpuload map[string]map[int64]*CpuloadPolymerize
	jvmHeap map[string]map[int64]*JVMHeapPolymerize
}

func newRuntimeAnalyze() *RuntimeAnalyze {
	return &RuntimeAnalyze{
		cpuload: make(map[string]map[int64]*CpuloadPolymerize),
		jvmHeap: make(map[string]map[int64]*JVMHeapPolymerize),
	}
}

// CpuloadPolymerize ...
type CpuloadPolymerize struct {
	JVMCpuload    float64 `msg:"jc"` // jvm cpuload
	SystemCpuload float64 `msg:"sc"` // system cpuload
	Value         float64 `msg:"value"`
	Count         int     // 计数器，多少个包
}

func newCpuloadPolymerize() *CpuloadPolymerize {
	return &CpuloadPolymerize{}
}

// JVMHeapPolymerize ...
type JVMHeapPolymerize struct {
	JVMHeap int64   `msg:"jh"` // jvm heap
	Value   float64 `msg:"value"`
	Count   int     // 计数器，多少个包
}

func newJVMHeapPolymerize() *JVMHeapPolymerize {
	return &JVMHeapPolymerize{}
}

func (a *App) runtimeAlarmStore(alert *AlertInfo, alertValue float64, agentID, hostName string) error {
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
		"",
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

// 检查是有有runtime的计算策略,没有直接丢弃包文，减少计算量
func (a *App) runtimeFilter() bool {
	if _, ok := a.Alerts[constant.ALERT_APM_CPU_USED_RATIO]; ok {
		return true
	}
	if _, ok := a.Alerts[constant.ALERT_APM_MEM_USED_RATION]; ok {
		return true
	}
	return false
}

// RuntimeCache runtime数据计算
func (a *App) RuntimeCache(runtimes *alert.Runtimes, dataTime int64) {
	for agentID, runtime := range runtimes.Runtimes {
		// cpu load聚合
		cpuloadMap, ok := a.runtimeCache.cpuload[agentID]
		if !ok {
			cpuloadMap = make(map[int64]*CpuloadPolymerize)
			a.runtimeCache.cpuload[agentID] = cpuloadMap
		}
		cpuPolymerize, ok := cpuloadMap[dataTime]
		if !ok {
			cpuPolymerize = newCpuloadPolymerize()
			cpuloadMap[dataTime] = cpuPolymerize
		}
		cpuPolymerize.JVMCpuload = runtime.JVMCpuload
		cpuPolymerize.SystemCpuload = runtime.SystemCpuload
		cpuPolymerize.Count = runtime.Count

		// jvmheap 聚合
		jvmheapMap, ok := a.runtimeCache.jvmHeap[agentID]
		if !ok {
			jvmheapMap = make(map[int64]*JVMHeapPolymerize)
			a.runtimeCache.jvmHeap[agentID] = jvmheapMap
		}
		jvmPolymerize, ok := jvmheapMap[dataTime]
		if !ok {
			jvmPolymerize = newJVMHeapPolymerize()
			jvmheapMap[dataTime] = jvmPolymerize
		}
		jvmPolymerize.JVMHeap = runtime.JVMHeap
		jvmPolymerize.Count = runtime.Count
	}
}
