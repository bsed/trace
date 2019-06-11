package service

import (
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/vmihailenco/msgpack"

	"go.uber.org/zap"
)

// App app
type App struct {
	name         string             // app name
	policyType   int                // 策略模版类型 1:默认策略模版， 2:自定义策略模版
	orderly      Orderly            // 排序工具
	policy       *Policy            // Policy
	SpecialAlert *SpecialAlert      // 特殊监控
	Alerts       map[int]*AlertInfo // 策略模版，通用策略
	tChan        chan bool          // 任务channel
	taskID       int64              // 任务ID
	stopC        chan bool          // stop chan
	apisC        chan *alert.Data   // api 数据通道
	sqlsC        chan *alert.Data   // sql 数据通道
	exsC         chan *alert.Data   // ex 数据通道
	runtimeC     chan *alert.Data   // runtime 数据通道
	apiCache     *APIAnalyze        // api聚合
	sqlCache     *SQLAnalyze        // sql聚合
	exCache      *EXAnalyze         // 异常聚合
	runtimeCache *RuntimeAnalyze    // runtime聚合
}

func newApp() *App {
	return &App{
		policy:       newPolicy(),
		SpecialAlert: newSpecialAlert(),
		Alerts:       make(map[int]*AlertInfo),
		tChan:        make(chan bool, 50),
		stopC:        make(chan bool, 1),
		apisC:        make(chan *alert.Data, 50),
		sqlsC:        make(chan *alert.Data, 50),
		exsC:         make(chan *alert.Data, 50),
		runtimeC:     make(chan *alert.Data, 50),
		apiCache:     newAPIAnalyze(),
		sqlCache:     newSQLAnalyze(),
		exCache:      newEXAnalyze(),
		runtimeCache: newRuntimeAnalyze(),
	}
}

func (a *App) start() error {
	go a.analyzeSrv()
	return nil
}

func (a *App) analyzeSrv() {
	for {
		select {
		case <-a.stopC:
			return
		case _, ok := <-a.tChan:
			if ok {
				// startTime := time.Now()
				a.apiStats()
				a.sqlStats()
				a.exStats()
				a.runtimeCounter()
				// logger.Debug("定时任务", zap.String("appName", a.name), zap.Float64("耗时", time.Now().Sub(startTime).Seconds()))
			}
			break
		case data, ok := <-a.apisC:
			if ok {
				// 如果没有api策略直接丢弃数据
				if !a.apiFilter() {
					break
				}
				apis := alert.NewAPIs()
				if err := msgpack.Unmarshal(data.Payload, apis); err != nil {
					logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
					break
				}
				a.APICache(apis, data.Time)
			}
			break
		case data, ok := <-a.sqlsC:
			if ok {
				// 如果没有api策略直接丢弃数据
				if !a.sqlFilter() {
					break
				}
				sqls := alert.NewSQLs()
				if err := msgpack.Unmarshal(data.Payload, sqls); err != nil {
					logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
					break
				}
				a.SQLCache(sqls, data.Time)
			}
			break
		case data, ok := <-a.exsC:
			if ok {
				// 如果没有内部异常策略直接丢弃数据
				if !a.exFilter() {
					break
				}
				exception := alert.NewException()
				if err := msgpack.Unmarshal(data.Payload, exception); err != nil {
					logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
					break
				}
				a.EXCache(exception, data.Time)
			}
			break
		case data, ok := <-a.runtimeC:
			if ok {
				// 如果没有内部异常策略直接丢弃数据
				if !a.runtimeFilter() {
					break
				}
				runtimes := alert.NewRuntimes()
				if err := msgpack.Unmarshal(data.Payload, runtimes); err != nil {
					logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
					break
				}

				a.RuntimeCache(runtimes, data.Time)
			}
			break
		}
	}
}

func (a *App) close() error {
	close(a.tChan)
	close(a.stopC)
	return nil
}

func (a *App) exsRecv(alertData *alert.Data) {
	a.exsC <- alertData
}

func (a *App) runtimeRecv(alertData *alert.Data) {
	a.runtimeC <- alertData
}

func (a *App) sqlsRecv(alertData *alert.Data) {
	a.sqlsC <- alertData
}

func (a *App) apisRecv(alertData *alert.Data) {
	a.apisC <- alertData
}

// 检查是有有异常的计算策略,没有直接丢弃包文，减少计算量
func (a *App) exFilter() bool {
	if _, ok := a.Alerts[constant.ALERT_APM_EXCEPTION_RATIO]; ok {
		return true
	}
	return false
}

// 检查是有有sql的计算策略,没有直接丢弃包文，减少计算量
func (a *App) sqlFilter() bool {
	if _, ok := a.Alerts[constant.ALERT_APM_SQL_ERROR_RATIO]; ok {
		return true
	}
	return false
}

func newPolymerizes() *Polymerizes {
	return &Polymerizes{
		Polymerizes: make(map[int]map[int64]*Polymerize),
	}
}

// Polymerizes ...
type Polymerizes struct {
	Polymerizes map[int]map[int64]*Polymerize // key为alertType, value为该指标时间戳集合的打点数据
}

// Polymerize 监控聚合数据
type Polymerize struct {
	Count    int     `msg:"count"`
	ErrCount int     `msg:"errcount"`
	Duration int32   `msg:"duration"`
	Value    float64 `msg:"value"`
}

func newPolymerize() *Polymerize {
	return &Polymerize{}
}
