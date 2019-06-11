package service

import (
	"sort"
	"sync"
	"time"

	"github.com/imdevlab/g/utils"
	"github.com/vmihailenco/msgpack"
	"go.uber.org/zap"

	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/collector/service/plugin"
	"github.com/bsed/trace/pkg/alert"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/bsed/trace/pkg/stats"
	"github.com/bsed/trace/pkg/util"
)

// 服务统计数据只实时计算1分钟的点，不做任何滑动窗口
// 通过nats或者其他mq将1分钟的数据发送给聚合计算服务，在聚合服务上做告警策略

// App 单个服务信息
type App struct {
	mutex            *sync.RWMutex
	appType          int32                     // 服务类型
	taskID           int64                     // 定时任务ID
	name             string                    // 服务名称
	order            OrderlyKeys               // 排序打点
	agents           map[string]*util.Agent    // agent集合
	apis             map[string]struct{}       // 接口信息
	stopC            chan bool                 // 停止通道
	tickerC          chan bool                 // 定时任务通道
	apiTickerC       chan bool                 // 定时任务通道
	spanC            chan *trace.TSpan         // span类型通道
	spanChunkC       chan *trace.TSpanChunk    // span chunk类型通道
	apiC             chan *alert.Data          // api 统计信息
	statC            chan *pinpoint.TAgentStat // jvm状态类型通道
	httpCodes        map[int32]struct{}        // http标准code
	statsCache       map[int64]*plugin.Stats   // 计算点集合
	apiCache         map[int64]*stats.App      // api聚合数据
	policyUpdateDate int64                     // 策略更新时间
	checkTime        int64                     // 检查时间
	defaultCode      map[int32]struct{}        // 默认code， 不会被策略覆盖
}

func newApp(name string) *App {
	app := &App{
		mutex:       &sync.RWMutex{},
		name:        name,
		agents:      make(map[string]*util.Agent),
		stopC:       make(chan bool, 1),
		tickerC:     make(chan bool, 10),
		apiTickerC:  make(chan bool, 10),
		spanC:       make(chan *trace.TSpan, 200),
		spanChunkC:  make(chan *trace.TSpanChunk, 200),
		apiC:        make(chan *alert.Data, 200),
		statC:       make(chan *pinpoint.TAgentStat, 200),
		apis:        make(map[string]struct{}),
		statsCache:  make(map[int64]*plugin.Stats),
		httpCodes:   make(map[int32]struct{}),
		defaultCode: make(map[int32]struct{}),
		apiCache:    make(map[int64]*stats.App),
	}

	for _, code := range misc.Conf.Stats.DefaultCode {
		app.httpCodes[code] = struct{}{}
		app.defaultCode[code] = struct{}{}
	}

	return app
}

// clearCode 只清除默认以外的code
func (a *App) clearCode() {
	a.mutex.Lock()
	for code := range a.httpCodes {
		// 只能删除非默认code
		if _, ok := a.defaultCode[code]; !ok {
			delete(a.httpCodes, code)
		}
	}
	a.mutex.Unlock()
}

// getCode 查看code是否存在
func (a *App) getCode(code int32) (struct{}, bool) {
	a.mutex.RLock()
	value, ok := a.httpCodes[code]
	a.mutex.RUnlock()
	return value, ok
}

// clearCode 只清除200以外的code
func (a *App) addCode(code int32) {
	a.mutex.Lock()
	a.httpCodes[code] = struct{}{}
	a.mutex.Unlock()
}

// online agent上线
func (a *App) online(agentid string) error {
	a.mutex.RLock()
	agent, ok := a.agents[agentid]
	a.mutex.RUnlock()
	if !ok {
		return nil
	}
	if !agent.IsLive {
		if err := gCollector.storage.UpdateAgentState(a.name, agentid, true); err != nil {
			logger.Warn("update agent state Store", zap.String("error", err.Error()))
			return err
		}
		agent.IsLive = true
	}
	return nil
}

// storeAgent 保存agent
func (a *App) storeAgent(agentID string, isLive bool) {
	a.mutex.RLock()
	agent, ok := a.agents[agentID]
	a.mutex.RUnlock()
	if !ok {
		agent = util.NewAgent()
		a.mutex.Lock()
		a.agents[agentID] = agent
		a.mutex.Unlock()
	}
	agent.IsLive = isLive
	return
}

// stats 计算模块
func (a *App) stats() {
	// defer func() {
	// 	if err := recover(); err != nil {
	// 		logger.Error("app stats", zap.Any("msg", err), zap.String("name", a.name))
	// 		return
	// 	}
	// }()

	for {
		select {
		// 二次聚合之后的api信息入库
		case _, ok := <-a.apiTickerC:
			if ok {
				if err := a.apiStatsStore(); err != nil {
					logger.Warn("api stats & store error", zap.String("error", err.Error()))
				}
			}
		// 信息统计入库
		case _, ok := <-a.tickerC:
			if ok {
				// 链路统计信息入库
				if err := a.statsStore(); err != nil {
					logger.Warn("stats store error", zap.String("error", err.Error()))
				}
			}
			break
		// span处理
		case span, ok := <-a.spanC:
			if ok {
				if err := a.statsSpan(span); err != nil {
					logger.Warn("stats span error", zap.String("error", err.Error()))
				}
			}
			break
		// spanChunk处理
		case spanChunk, ok := <-a.spanChunkC:
			if ok {
				if err := a.statsSpanChunk(spanChunk); err != nil {
					logger.Warn("stats span error", zap.String("error", err.Error()))
				}
			}
		// 接收到其他collecotor发送来的api信息，进行二次聚合
		case packet, ok := <-a.apiC:
			if ok {
				if err := a.statsApi(packet); err != nil {
					logger.Warn("stats api error", zap.String("error", err.Error()))
				}
			}
			break
		// agent stat数据统计
		case agentStat, ok := <-a.statC:
			if ok {
				if err := a.statsAgentStat(agentStat); err != nil {
					logger.Warn("stats agent stat error", zap.String("error", err.Error()))
				}
			}
			break
		case <-a.stopC:
			return
		}
	}
}

// statsApi 二次聚合模块
func (a *App) statsApi(packet *alert.Data) error {
	newApp := stats.NewApp()
	if err := msgpack.Unmarshal(packet.Payload, newApp); err != nil {
		logger.Warn("msgpack unmarshal", zap.String("error", err.Error()))
		return err
	}
	// 查找Api相关缓存，不存在新申请
	cacheApp, ok := a.apiCache[packet.Time]
	if !ok {
		cacheApp = stats.NewApp()
		a.apiCache[packet.Time] = cacheApp
	}

	// 计算http信息
	for urlStr, tmpUrl := range newApp.Urls {
		url, ok := cacheApp.Urls[urlStr]
		if !ok {
			url = stats.NewUrl()
			cacheApp.Urls[urlStr] = url
		}
		url.Duration += tmpUrl.Duration
		if url.MinDuration > tmpUrl.MinDuration {
			url.MinDuration = tmpUrl.MinDuration
		}
		if url.MaxDuration < tmpUrl.MaxDuration {
			url.MaxDuration = tmpUrl.MaxDuration
		}

		url.AccessCount += tmpUrl.AccessCount
		url.AccessErrCount += tmpUrl.AccessErrCount
		url.SatisfactionCount += tmpUrl.SatisfactionCount
		url.TolerateCount += tmpUrl.TolerateCount
	}

	// 计算Dubbo信息
	for dubboStr, tmpDubbo := range newApp.Dubbos {
		dubbo, ok := cacheApp.Dubbos[dubboStr]
		if !ok {
			dubbo = stats.NewDubbo()
			cacheApp.Dubbos[dubboStr] = dubbo
		}
		dubbo.Duration += tmpDubbo.Duration
		if dubbo.MinDuration > tmpDubbo.MinDuration {
			dubbo.MinDuration = tmpDubbo.MinDuration
		}
		if dubbo.MaxDuration < tmpDubbo.MaxDuration {
			dubbo.MaxDuration = tmpDubbo.MaxDuration
		}

		dubbo.AccessCount += tmpDubbo.AccessCount
		dubbo.AccessErrCount += tmpDubbo.AccessErrCount
		dubbo.SatisfactionCount += tmpDubbo.SatisfactionCount
		dubbo.TolerateCount += tmpDubbo.TolerateCount
	}

	return nil
}

// stats 计算模块
func (a *App) statsSpan(span *trace.TSpan) error {
	// api缓存并入库
	if !a.apiIsExist(span.GetRPC()) {
		if err := gCollector.storage.StoreAPI(span); err != nil {
			logger.Warn("store api", zap.String("error", err.Error()))
			return err
		}
		a.storeAPI(span.GetRPC())
		// 保存dubbo api
		if span.GetServiceType() == constant.DUBBO_PROVIDER {
			gCollector.apps.dubbo.Add(span.GetRPC(), span.GetApplicationName())
		}
	}

	// 计算当前span时间范围点
	t, err := utils.MSToTime(span.StartTime)
	if err != nil {
		logger.Warn("ms to time", zap.Int64("time", span.StartTime), zap.String("error", err.Error()))
		return err
	}

	// 获取时间戳并将其精确到分钟
	spanTime := t.Unix() - int64(t.Second())

	// 查找时间点，不存在新申请, span统计的范围是分钟，所以这里直接用优化过后的spanTime
	stats, ok := a.statsCache[spanTime]
	if !ok {
		stats = plugin.NewStats(a.httpCodes, a.mutex, logger, getNameByIP, getNameByDubboAPI)
		a.statsCache[spanTime] = stats
	}
	stats.SpanCounter(span)
	return nil
}

// statsAgentStat jvm等数据计算
func (a *App) statsAgentStat(agentStat *pinpoint.TAgentStat) error {

	// 计算当前agent stat时间范围点
	t, err := utils.MSToTime(agentStat.GetTimestamp())
	if err != nil {
		logger.Warn("ms to time", zap.Int64("time", agentStat.GetTimestamp()), zap.String("error", err.Error()))
		return err
	}

	// 获取时间戳并将其精确到分钟
	agentStatTime := t.Unix() - int64(t.Second())

	// 查找时间点，不存在新申请
	stats, ok := a.statsCache[agentStatTime]
	if !ok {
		stats = plugin.NewStats(a.httpCodes, a.mutex, logger, getNameByIP, getNameByDubboAPI)
		a.statsCache[agentStatTime] = stats
	}

	stats.RuntimeCounter(agentStat)
	return nil
}

// statsSpanChunk 计算模块
func (a *App) statsSpanChunk(spanChunk *trace.TSpanChunk) error {
	// 计算当前spanChunk时间范围点
	// SpanChunk 里面没有span start time信息，只能用当前时间来做
	t := time.Now()
	// 获取时间戳
	spanChunkTime := t.Unix() - int64(t.Second())

	// 查找时间点，不存在新申请
	stats, ok := a.statsCache[spanChunkTime]
	if !ok {
		stats = plugin.NewStats(a.httpCodes, a.mutex, logger, getNameByIP, getNameByDubboAPI)
		a.statsCache[spanChunkTime] = stats
	}

	stats.SpanChunkCounter(spanChunk)

	return nil
}

func (a *App) start() {
	defer func() {
		if err := recover(); err != nil {
			logger.Error("app start", zap.Any("msg", err))
			return
		}
	}()

	// 获取任务ID
	a.taskID = gCollector.ticker.NewID()
	logger.Info("app start", zap.String("name", a.name), zap.Int64("taskID", a.taskID))

	// 统计信息定时服务
	gCollector.ticker.AddTask(a.taskID, a.tickerC)

	// api二次聚合计算定时服务
	gCollector.apiTicker.AddTask(a.taskID, a.apiTickerC)

	// 启动计算模块
	go a.stats()
}

// close 退出
func (a *App) close() {
	// 获取任务ID
	logger.Info("app close", zap.String("name", a.name), zap.Int64("taskID", a.taskID))

	gCollector.ticker.RemoveTask(a.taskID)
	gCollector.apiTicker.RemoveTask(a.taskID)

	close(a.tickerC)
	close(a.apiTickerC)
	close(a.stopC)
	close(a.spanC)
	close(a.spanChunkC)
	close(a.statC)
}

// apiIsExist 检查api是否缓存
func (a *App) apiIsExist(api string) bool {
	a.mutex.RLock()
	_, isExist := a.apis[api]
	a.mutex.RUnlock()
	return isExist
}

// storeAPI 缓存api
func (a *App) storeAPI(api string) {
	a.mutex.Lock()
	a.apis[api] = struct{}{}
	a.mutex.Unlock()
}

func (a *App) recvSpan(appName, agentID string, span *trace.TSpan) error {
	a.spanC <- span
	return nil
}

func (a *App) recvSpanChunk(appName, agentID string, spanChunk *trace.TSpanChunk) error {
	a.spanChunkC <- spanChunk
	return nil
}

func (a *App) recvApi(packet *alert.Data) error {
	a.apiC <- packet
	return nil
}

func (a *App) recvAgentStat(appName, agentID string, agentStat *pinpoint.TAgentStat) error {
	a.statC <- agentStat
	return nil
}

// statsStore 链路统计信息入库
func (a *App) statsStore() error {
	// 清空之前节点
	a.order = a.order[:0]

	// 赋值
	for key := range a.statsCache {
		a.order = append(a.order, key)
	}

	// 排序
	sort.Sort(a.order)

	// 如果没有计算节点直接返回
	if a.order.Len() <= 0 {
		return nil
	}

	inputDate := a.order[0]
	now := time.Now().Unix()

	if now < inputDate+misc.Conf.Stats.DeferTime {
		return nil
	}

	// 接口入库
	for methodID, method := range a.statsCache[inputDate].Method.Methods {
		gCollector.storage.InsertMethodStats(a.name, inputDate, a.statsCache[inputDate].Method.ApiStr, methodID, method)
	}

	sqls := alert.NewSQLs()

	// sql入库
	for sqlID, sql := range a.statsCache[inputDate].SQL.SQLS {
		gCollector.storage.InsertSQLStats(a.name, inputDate, sqlID, sql)
		alertSql := alert.NewSQL()
		alertSql.Count = sql.Count
		alertSql.Duration = sql.Duration
		alertSql.Errcount = sql.ErrCount
		alertSql.ID = sqlID
		sqls.SQLs[sqlID] = alertSql
	}

	// 有sql数据才发送
	if len(sqls.SQLs) > 0 {
		data := alert.NewData()
		data.AppName = a.name
		data.Type = constant.ALERT_TYPE_SQL
		data.Time = inputDate
		payload, err := msgpack.Marshal(sqls)
		if err != nil {
			logger.Warn("msgpack", zap.String("error", err.Error()))
		} else {
			data.Payload = payload
			// 推送
			gCollector.publish(data)
		}
	}

	rs := alert.NewRuntimes()
	for agentID, runtime := range a.statsCache[inputDate].Runtime.Runtimes {
		r := alert.NewRuntime()
		r.JVMCpuload = runtime.JVMCpuload
		r.SystemCpuload = runtime.SystemCpuload
		r.JVMHeap = runtime.JVMHeap
		r.Count = runtime.Count
		rs.Runtimes[agentID] = r
	}

	if len(rs.Runtimes) > 0 {
		data := alert.NewData()
		data.AppName = a.name
		data.Type = constant.ALERT_TYPE_RUNTIME
		data.Time = inputDate
		payload, err := msgpack.Marshal(rs)
		if err != nil {
			logger.Warn("msgpack", zap.String("error", err.Error()))
		} else {
			data.Payload = payload
			// 推送
			gCollector.publish(data)
		}
	}

	// 异常入库
	for methodID, exceptions := range a.statsCache[inputDate].Exception.ExMethods {
		gCollector.storage.InsertExceptionStats(a.name, inputDate, methodID, exceptions.Exceptions)
	}

	// 异常数大于0才需要上报
	if a.statsCache[inputDate].Exception.ErrCount > 0 {
		exception := alert.NewException()
		exception.Count = a.statsCache[inputDate].Exception.Count
		exception.ErrCount = a.statsCache[inputDate].Exception.ErrCount

		data := alert.NewData()
		data.AppName = a.name
		data.Type = constant.ALERT_TYPE_EXCEPTION
		data.Time = inputDate
		payload, err := msgpack.Marshal(exception)
		if err != nil {
			logger.Warn("msgpack", zap.String("error", err.Error()))
		} else {
			data.Payload = payload
			// 推送
			gCollector.publish(data)
		}
	}

	// 插入被访问者
	for targetType, targets := range a.statsCache[inputDate].SrvMap.Targets {
		for targetName, target := range targets {
			gCollector.storage.InsertTargetMap(a.name, a.appType, inputDate, int32(targetType), targetName, target)
		}
	}

	unknowParent := a.statsCache[inputDate].SrvMap.UnknowParent
	// 只有被调用才可以入库
	if unknowParent.AccessCount > 0 {
		gCollector.storage.InsertUnknowParentMap(a.name, a.appType, inputDate, unknowParent)
	}

	// api被调用情况
	for apiStr, apiInfo := range a.statsCache[inputDate].APIMap.Apis {
		for parentName, parentInfo := range apiInfo.Parents {
			gCollector.storage.InsertAPIMapStats(a.name, a.appType, inputDate, apiStr, parentName, parentInfo)
		}
	}

	// 二次聚合发送
	for appName, app := range a.statsCache[inputDate].API.Apps {
		packet := alert.NewData()
		packet.AppName = appName
		packet.Type = constant.ALERT_TYPE_API
		packet.Time = inputDate
		payload, err := msgpack.Marshal(app)
		if err != nil {
			logger.Warn("msgpack", zap.String("error", err.Error()))
		} else {
			packet.Payload = payload
			// 推送
			topic, err := gCollector.getCollecotorTopic(appName)
			if err != nil {
				logger.Warn("get topic failed", zap.String("appName", appName), zap.String("error", err.Error()))
				continue
			}

			data, err := msgpack.Marshal(packet)
			if err != nil {
				logger.Warn("msgpack", zap.String("error", err.Error()))
				break
			}

			if err := gCollector.mq.Publish(topic, data); err != nil {
				logger.Warn("publish", zap.Error(err))
			}
		}
	}

	// 上报打点信息并删除该时间点信息
	delete(a.statsCache, inputDate)
	return nil
}

// apiStatsStore api信息二次聚合并入库
func (a *App) apiStatsStore() error {
	// 清空之前节点
	a.order = a.order[:0]

	// 赋值
	for key := range a.apiCache {
		a.order = append(a.order, key)
	}

	// 排序
	sort.Sort(a.order)

	// 如果没有计算节点直接返回
	if a.order.Len() <= 0 {
		return nil
	}

	inputDate := a.order[0]
	now := time.Now().Unix()

	// 延迟ApiStatsInterval
	if now < inputDate+misc.Conf.Apps.ApiStatsInterval+60 {
		return nil
	}

	apis := alert.NewAPIs()
	// 遍历入库
	for urlStr, url := range a.apiCache[inputDate].Urls {
		gCollector.storage.InsertAPIStats(a.name, inputDate, urlStr, url)
		apiAlert := &alert.API{
			Desc:           urlStr,
			AccessCount:    url.AccessCount,
			AccessErrCount: url.AccessErrCount,
			Duration:       url.Duration,
		}
		apis.APIS[urlStr] = apiAlert
	}

	// 遍历入库
	for dubboAPI, dubbo := range a.apiCache[inputDate].Dubbos {
		gCollector.storage.InsertDubboStats(a.name, inputDate, dubboAPI, dubbo)
		apiAlert := &alert.API{
			Desc:           dubboAPI,
			AccessCount:    dubbo.AccessCount,
			AccessErrCount: dubbo.AccessErrCount,
			Duration:       dubbo.Duration,
		}
		apis.APIS[dubboAPI] = apiAlert
	}

	// 发送给alert服务
	// 有api数据发送给mq
	if len(apis.APIS) > 0 {
		data := alert.NewData()
		data.AppName = a.name
		data.Type = constant.ALERT_TYPE_API
		data.Time = inputDate
		payload, err := msgpack.Marshal(apis)
		if err != nil {
			logger.Warn("msgpack", zap.String("error", err.Error()))
		} else {
			data.Payload = payload
			// 推送
			gCollector.publish(data)
		}
	}
	// 上报打点信息并删除该时间点信息
	delete(a.apiCache, inputDate)
	return nil
}
