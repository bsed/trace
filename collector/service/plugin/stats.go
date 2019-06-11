package plugin

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/bsed/trace/pkg/stats"
	"go.uber.org/zap"
)

var logger *zap.Logger

// Stats 计算模块
type Stats struct {
	httpCodes map[int32]struct{} // http标准code
	API       *stats.ApiStore    // api统计
	SQL       *stats.SQLS        // sql语句计算统计
	Method    *stats.Methods     // 接口计算统计
	Exception *stats.Exceptions  // 异常计算统计
	APIMap    *stats.ApiMap      // 接口拓扑图
	SrvMap    *stats.SrvMap      // 应用拓扑图
	Runtime   *stats.Runtimes    // runtime计算
	getNbyIP  func(string) (string, bool)
	getNbyApi func(string) (string, bool)
}

// NewStats ....
func NewStats(httpCodes map[int32]struct{}, mutex *sync.RWMutex, l *zap.Logger, f func(string) (string, bool), f2 func(string) (string, bool)) *Stats {
	logger = l
	stats := &Stats{
		httpCodes: make(map[int32]struct{}),
		API:       stats.NewApiStore(),
		SQL:       stats.NewSQLS(),
		Method:    stats.NewMethods(),
		Exception: stats.NewExceptions(),
		APIMap:    stats.NewApiMap(),
		SrvMap:    stats.NewSrvMap(),
		Runtime:   stats.NewRuntimes(),
		getNbyIP:  f,
		getNbyApi: f2,
	}
	// 添加策略
	mutex.RLock()
	for code := range httpCodes {
		stats.httpCodes[code] = struct{}{}
	}
	mutex.RUnlock()
	return stats
}

// 如果该应用为父节点，那么要统计自己的api，目标为自己，请求者为unknow
func (s *Stats) selfApiCounter(span *trace.TSpan) {
	// 查找应用
	app, ok := s.API.Apps[span.GetApplicationName()]
	if !ok {
		app = stats.NewApp()
		s.API.Apps[span.GetApplicationName()] = app
	}

	url, ok := app.Urls[span.GetRPC()]
	if !ok {
		url = stats.NewUrl()
		app.Urls[span.GetRPC()] = url
	}

	url.Duration += span.GetElapsed()
	url.AccessCount++
	if span.GetElapsed() > url.MaxDuration {
		url.MaxDuration = span.GetElapsed()
	}

	if url.MinDuration == 0 || url.MinDuration > span.GetElapsed() {
		url.MinDuration = span.GetElapsed()
	}

	cacheCode := false
	var code int32

	// Annotations
	for _, annotation := range span.GetAnnotations() {
		if annotation.GetKey() == constant.HTTP_STATUS_CODE {
			cacheCode = true
			code = annotation.GetValue().GetIntValue()
			break
		}
		if annotation.GetKey() == constant.DUBBO_STATUS_ANNOTATION_KEY {
			cacheCode = true
			code = annotation.GetValue().GetIntValue()
			break
		}
	}

	if cacheCode {
		// 如果是dubbo请求，那么查看结果返回是否为OK，不为OK那么认为是失败的
		if span.ServiceType == constant.DUBBO_PROVIDER {
			if code != constant.DUBBO_RESULT_STATUS_OK {
				url.AccessErrCount++
			}
		} else {
			// 检查code是否正确, 如果在策略里面没找到发现code，那么说明这次http请求失败
			if _, ok := s.httpCodes[code]; !ok {
				url.AccessErrCount++
			}
		}
	}
	// 耗时小于满意时间满意次数加1
	if span.GetElapsed() < misc.Conf.Stats.SatisfactionTime {
		url.SatisfactionCount++
		// 耗时小于可容忍时间，可容忍次数加一， 其他都为沮丧次数
	} else if span.GetElapsed() < misc.Conf.Stats.TolerateTime {
		url.TolerateCount++
	}
}

// urlTarger url统计
func (s *Stats) urlTarget(event *trace.TSpanEvent) {
	target, url, cacheCode, httpCode, isOK := s.analysishttpEvent(event)
	if isOK {
		s.urlCounter(target, url, cacheCode, httpCode, event)
	}
}

func (s *Stats) urlCounter(target string, urlStr string, findCode bool, code int32, event *trace.TSpanEvent) {
	// 通过IP获取app name
	ip, err := getip(target)
	if err == nil {
		appName, ok := s.getNbyIP(ip) //gCollector.apps.getNameByIP(ip)
		if ok {
			target = appName
		}
	} else {
		// 如果不是IP尝试切一下-，如果还是找不到那么就使用destinationID
		target = cutVip(target)
	}

	// 查找应用
	app, ok := s.API.Apps[target]
	if !ok {
		app = stats.NewApp()
		s.API.Apps[target] = app
	}

	url, ok := app.Urls[urlStr]
	if !ok {
		url = stats.NewUrl()
		app.Urls[urlStr] = url
	}

	url.Duration += event.GetEndElapsed()
	url.AccessCount++
	if event.GetEndElapsed() > url.MaxDuration {
		url.MaxDuration = event.GetEndElapsed()
	}

	if url.MinDuration == 0 || url.MinDuration > event.GetEndElapsed() {
		url.MinDuration = event.GetEndElapsed()
	}
	// 检查code是否正确, 如果在策略里面没找到发现code，那么说明这次http请求失败
	if findCode {
		if _, ok := s.httpCodes[code]; !ok {
			// url.AccessCodeErrCount++
			url.AccessErrCount++
		}
	} else {
		// 没找到code的情况下应该用抛出异常
		if event.GetExceptionInfo() != nil {
			url.AccessErrCount++
		}
	}

	// 耗时小于满意时间满意次数加1
	if event.GetEndElapsed() < misc.Conf.Stats.SatisfactionTime {
		url.SatisfactionCount++
		// 耗时小于可容忍时间，可容忍次数加一， 其他都为沮丧次数
	} else if event.GetEndElapsed() < misc.Conf.Stats.TolerateTime {
		url.TolerateCount++
	}

}

// SpanCounter 计算
func (s *Stats) SpanCounter(span *trace.TSpan) error {
	isCount := true
	// 计算API信息
	for _, event := range span.GetSpanEventList() {
		s.eventCounter(span.GetApplicationName(), span.GetRPC(), event, isCount)
		isCount = false
	}

	// 如果该应用为父节点，那么要统计自己的api，目标为自己，请求者为unknow
	if span.GetParentSpanId() == -1 {
		s.selfApiCounter(span)
	}

	// 计算API被哪些服务调用
	{
		s.apiMapCounter(span)
	}

	// 计算服务拓扑图
	{
		s.parentMapCounter(span)
	}

	return nil
}

// RuntimeCounter runtime counter 计算
func (s *Stats) RuntimeCounter(agentStat *pinpoint.TAgentStat) error {
	runtime, ok := s.Runtime.Runtimes[agentStat.GetAgentId()]
	if !ok {
		runtime = stats.NewRuntime()
		s.Runtime.Runtimes[agentStat.GetAgentId()] = runtime
	}

	runtime.Count++
	runtime.SystemCpuload += agentStat.CpuLoad.GetSystemCpuLoad()
	runtime.JVMCpuload += agentStat.CpuLoad.GetJvmCpuLoad()
	runtime.JVMHeap += agentStat.Gc.GetJvmMemoryHeapUsed()
	return nil
}

// SpanChunkCounter counter 计算
func (s *Stats) SpanChunkCounter(spanChunk *trace.TSpanChunk) error {
	for _, event := range spanChunk.GetSpanEventList() {
		s.eventCounter(spanChunk.GetApplicationName(), "", event, false)
	}
	return nil
}

// sqlCount 计算sql
func (s *Stats) sqlCount(event *trace.TSpanEvent) {
	var sqlID int32
	cacheSQLID := false
	for _, annotation := range event.GetAnnotations() {
		if annotation.GetKey() == constant.SQL_ID {
			sqlID = annotation.Value.GetIntStringStringValue().GetIntValue()
			cacheSQLID = true
			break
		}
	}
	// 没有sqlID 直接返回
	if !cacheSQLID {
		logger.Warn("unfind sql id")
		return
	}

	sql, ok := s.SQL.Get(sqlID)
	if !ok {
		sql = stats.NewSQL()
		s.SQL.Store(sqlID, sql)
	}

	sql.Duration += event.GetEndElapsed()
	sql.Count++

	if event.GetEndElapsed() > sql.MaxDuration {
		sql.MaxDuration = event.GetEndElapsed()
	}

	if sql.MinDuration == 0 || sql.MinDuration > event.GetEndElapsed() {
		sql.MinDuration = event.GetEndElapsed()
	}

	// 是否有异常抛出
	if event.GetExceptionInfo() != nil {
		sql.ErrCount++
	}
}

// apiMapCounter 接口被哪些服务调用计算
func (s *Stats) apiMapCounter(span *trace.TSpan) {
	apiStr := span.GetRPC()
	if len(apiStr) <= 0 {
		return
	}

	api, ok := s.APIMap.Apis[apiStr] //.APIS[]
	if !ok {
		api = stats.NewApi()
		s.APIMap.Apis[apiStr] = api
	}

	var parentName string
	var parentType int16
	// spanID 为-1的情况该服务就是父节点，查不到被谁调用，这里可以考虑能不能抓到请求者到IP
	if span.ParentSpanId == -1 {
		parentName = "UNKNOWN"
		parentType = constant.SERVERTYPE_UNKNOWN
	} else {
		parentName = span.GetParentApplicationName()
		parentType = span.GetParentApplicationType()
	}

	parents, ok := api.Parents[parentName]
	if !ok {
		parents = stats.NewParent()
		parents.Type = parentType
		api.Parents[parentName] = parents
	}

	parents.AccessCount++
	parents.AccessDuration += span.Elapsed
	if span.GetErr() != 0 {
		parents.ExceptionCount++
	}
}

// parentMapCounter 计算服务拓扑图
func (s *Stats) parentMapCounter(span *trace.TSpan) {
	// spanID 为-1的情况该服务就是父节点，请求者应该是没接入监控
	if span.ParentSpanId == -1 {
		s.SrvMap.UnknowParent.AccessCount++
		s.SrvMap.UnknowParent.AccessDuration += span.GetElapsed()
		return
	}
}

// isDubbo 检查event类型是否为rpc接口
func isDubbo(eventType int16) bool {
	// dubbo从请求方统计，其他类型可以直接抛弃
	if eventType == constant.DUBBO_CONSUMER {
		return true
	}
	return false
}

// isHttp 检查event类型是否为http接口
func isHttp(eventType int16) bool {
	if eventType == constant.HTTP_CLIENT_3 ||
		eventType == constant.HTTP_CLIENT_3_INTERNAL ||
		eventType == constant.HTTP_CLIENT_4 ||
		eventType == constant.HTTP_CLIENT_4_INTERNAL {
		return true
	}
	return false
}

// isDB 检查event类型是否为数据库操作
func isDB(eventType int16) bool {
	if eventType == constant.MYSQL_EXECUTE_QUERY ||
		eventType == constant.ORACLE_EXECUTE_QUERY ||
		eventType == constant.MARIADB_EXECUTE_QUERY {
		return true
	}
	return false
}

func getip(destinationID string) (string, error) {
	strs := strings.Split(destinationID, ":")
	if len(strs) != 2 {
		return "", fmt.Errorf("unknow addr")
	}
	if len(strs[0]) == 0 {
		return "", fmt.Errorf("error ip")
	}
	return strs[0], nil
}

func gethost(destinationID string) (string, error) {
	strs := strings.Split(destinationID, ":")
	if len(strs) != 2 {
		return "", fmt.Errorf("unknow addr")
	}
	if len(strs[0]) == 0 {
		return "", fmt.Errorf("error ip")
	}
	return strs[0], nil
}

// dubboCounter dubbo计算
func (s *Stats) dubboCounter(event *trace.TSpanEvent) {
	dubboAPI, code, ok := s.analysisDubboEvent(event)
	// 通过api获取app name
	appName, ok := s.getNbyApi(dubboAPI) //gCollector.apps.getNameByIP(ip)
	if !ok {
		return
	}
	app, ok := s.API.Apps[appName]
	if !ok {
		app = stats.NewApp()
		s.API.Apps[appName] = app
	}
	dubbo, ok := app.Dubbos[dubboAPI]
	if !ok {
		dubbo = stats.NewDubbo()
		app.Dubbos[dubboAPI] = dubbo
	}
	dubbo.AccessCount++
	dubbo.Duration += event.GetEndElapsed()

	if dubbo.MinDuration == 0 || dubbo.MinDuration > event.GetEndElapsed() {
		dubbo.MinDuration = event.GetEndElapsed()
	}

	if event.GetEndElapsed() > dubbo.MaxDuration {
		dubbo.MaxDuration = event.GetEndElapsed()
	}
	// 不是DUBBO_RESULT_STATUS_OK的状态都是错误
	if code != constant.DUBBO_RESULT_STATUS_OK {
		dubbo.AccessErrCount++
	}

}

// eventCounter event数据统计 计算api信息， sql、method、异常等
func (s *Stats) eventCounter(appName, apiStr string, event *trace.TSpanEvent, isCount bool) {
	if isDubbo(event.GetServiceType()) {
		// dubbo
		s.dubboCounter(event)
	} else if isDB(event.GetServiceType()) {
		// 数据库统计
		s.sqlCount(event)
	} else if isHttp(event.GetServiceType()) {
		// http统计
		s.urlTarget(event)
	}

	// method统计
	s.methodCount(apiStr, event)

	// exception
	s.exceptionCount(event, isCount)

	// app后续服务拓扑图计算
	s.targetMapCounter(event)
}

// exceptionCount 异常统计
func (s *Stats) exceptionCount(event *trace.TSpanEvent, isCount bool) {
	if isCount {
		// span总数
		s.Exception.Count++
	}

	// 参看是否存在异常，不存在直接返回
	exInfo := event.GetExceptionInfo()
	if exInfo == nil {
		return
	}

	method, ok := s.Exception.Get(event.GetApiId())
	if !ok {
		method = stats.NewExMethod()
		s.Exception.Store(event.GetApiId(), method)
	}

	ex, ok := method.Get(exInfo.GetIntValue())
	if !ok {
		ex = stats.NewException()
		method.Store(exInfo.GetIntValue(), ex)
	}

	ex.Duration += event.GetEndElapsed()
	ex.Type = int(event.GetServiceType())
	ex.Count++

	// 错误总数
	s.Exception.ErrCount++

	if event.GetEndElapsed() > ex.MaxDuration {
		ex.MaxDuration = event.GetEndElapsed()
	}

	if ex.MinDuration == 0 || ex.MinDuration > event.GetEndElapsed() {
		ex.MinDuration = event.GetEndElapsed()
	}
}

func (s *Stats) methodCount(apiStr string, event *trace.TSpanEvent) {
	// 保存api到methods
	if len(apiStr) > 0 {
		if len(s.Method.ApiStr) == 0 {
			s.Method.ApiStr = apiStr
		}
	}
	method, ok := s.Method.Get(event.GetApiId())
	if !ok {
		method = stats.NewMethod(event.GetServiceType())
		s.Method.Store(event.GetApiId(), method)
	}

	method.Duration += event.GetEndElapsed()
	method.Count++

	if event.GetEndElapsed() > method.MaxDuration {
		method.MaxDuration = event.GetEndElapsed()
	}

	if method.MinDuration == 0 || method.MinDuration > event.GetEndElapsed() {
		method.MinDuration = event.GetEndElapsed()
	}

	// 是否有异常抛出
	if event.GetExceptionInfo() != nil {
		method.ErrCount++
	}
}

func cutVip(destination string) string {
	var name string
	names := strings.Split(destination, "-")
	if len(names) == 1 {
		name = destination
	} else if len(names) == 3 {
		name = names[1]
	} else if len(names) == 4 {
		name = names[1] + names[2]
	}
	return name
}

// targetMapCounter 计算target(child)拓扑图
func (s *Stats) targetMapCounter(event *trace.TSpanEvent) {
	var destinationID, dubboAPI string
	cacheCode := false
	var code int32
	analysisOk := false

	// http 统计
	if event.ServiceType == constant.HTTP_CLIENT_3 ||
		event.ServiceType == constant.HTTP_CLIENT_3_INTERNAL ||
		event.ServiceType == constant.HTTP_CLIENT_4 ||
		event.ServiceType == constant.HTTP_CLIENT_4_INTERNAL {

		// target, cacheCode, httpCode, true
		destinationID, _, cacheCode, code, analysisOk = s.analysishttpEvent(event)
		if analysisOk == false {
			return
		}
		// dubbo
	} else if event.ServiceType == constant.DUBBO_CONSUMER {
		dubboAPI, code, analysisOk = s.analysisDubboEvent(event)
		if !analysisOk {
			return
		}
		// sql
	} else if event.ServiceType == constant.MYSQL_EXECUTE_QUERY ||
		event.ServiceType == constant.REDIS ||
		event.ServiceType == constant.ORACLE_EXECUTE_QUERY ||
		event.ServiceType == constant.MARIADB_EXECUTE_QUERY {
		// 数据库
		destinationID = event.GetDestinationId()
		if len(destinationID) <= 0 {
			return
		}
	} else {
		return
	}

	targets, ok := s.SrvMap.Targets[event.ServiceType]
	if !ok {
		targets = make(map[string]*stats.Target)
		s.SrvMap.Targets[event.ServiceType] = targets
	}

	// http && dubbo做特殊处理
	if event.ServiceType == constant.HTTP_CLIENT_3 ||
		event.ServiceType == constant.HTTP_CLIENT_3_INTERNAL ||
		event.ServiceType == constant.HTTP_CLIENT_4 ||
		event.ServiceType == constant.HTTP_CLIENT_4_INTERNAL {
		ip, err := getip(destinationID)
		if err == nil {
			appName, ok := s.getNbyIP(ip) //gCollector.apps.getNameByIP(ip)
			if ok {
				destinationID = appName
			}
		} else {
			// 如果不是IP尝试切一下-，如果还是找不到那么就使用destinationID
			destinationID = cutVip(destinationID)
		}
	} else if event.ServiceType == constant.DUBBO_CONSUMER {
		appName, ok := s.getNbyApi(dubboAPI)
		if ok {
			destinationID = appName
		}
	}

	target, ok := targets[destinationID]
	if !ok {
		target = stats.NewTarget()
		targets[destinationID] = target
	}

	// http && dubbo code统计
	if event.ServiceType == constant.HTTP_CLIENT_3 ||
		event.ServiceType == constant.HTTP_CLIENT_3_INTERNAL ||
		event.ServiceType == constant.HTTP_CLIENT_4 ||
		event.ServiceType == constant.HTTP_CLIENT_4_INTERNAL {
		if cacheCode {
			if _, ok := s.httpCodes[code]; !ok {
				target.AccessErrCount++
			}
		} else if event.GetExceptionInfo() != nil {
			target.AccessErrCount++
		}
	} else if event.ServiceType == constant.DUBBO_CONSUMER {
		if code != constant.DUBBO_RESULT_STATUS_OK {
			target.AccessErrCount++
		}
	}
	target.AccessCount++
	target.AccessDuration += event.EndElapsed
}

// analysisDubboEvent dubbo数据解析
func (s *Stats) analysisDubboEvent(event *trace.TSpanEvent) (string, int32, bool) {
	var dubboAPI string
	cacheAPI := false
	var code int32
	cacheCode := false
	for _, annotation := range event.GetAnnotations() {
		if annotation.GetKey() == constant.DUBBO_RPC {
			dubboAPI = annotation.GetValue().GetStringValue()
			cacheAPI = true
		}
		if annotation.GetKey() == constant.DUBBO_STATUS_ANNOTATION_KEY {
			code = annotation.GetValue().GetIntValue()
			cacheCode = true
		}
		if cacheCode && cacheAPI {
			break
		}
	}

	if cacheCode && cacheAPI {
		return dubboAPI, code, true
	}
	return "", 0, false
}

// analysishttpEvent 解析httpevent，获取目标地址或者域名已经url和code
func (s *Stats) analysishttpEvent(event *trace.TSpanEvent) (string, string, bool, int32, bool) {
	cacheCode := false
	cacheURL := false
	cacheTarget := false
	var target, url string
	var httpCode int32
	if event.GetServiceType() == constant.HTTP_CLIENT_4 || event.GetServiceType() == constant.HTTP_CLIENT_4_INTERNAL {
		for _, annotation := range event.GetAnnotations() {
			if annotation.GetKey() == constant.HTTP_INTERNAL_DISPLAY {
				cacheTarget = true
				target = annotation.GetValue().GetStringValue()
			}
			if annotation.GetKey() == constant.HTTP_URL {
				cacheURL = true
				url = annotation.GetValue().GetStringValue()
			}
			if annotation.GetKey() == constant.HTTP_STATUS_CODE {
				cacheCode = true
				httpCode = annotation.GetValue().GetIntValue()
			}
		}
	} else if event.GetServiceType() == constant.HTTP_CLIENT_3 || event.GetServiceType() == constant.HTTP_CLIENT_3_INTERNAL {
		for _, annotation := range event.GetAnnotations() {
			if annotation.GetKey() == constant.HTTP_INTERNAL_DISPLAY {
				cacheTarget = true
				target = annotation.GetValue().GetStringValue()
			}
			if annotation.GetKey() == constant.HTTP_URL {
				cacheURL = true
				url = annotation.GetValue().GetStringValue()
			}
			if annotation.GetKey() == constant.HTTP_STATUS_CODE {
				cacheCode = true
				httpCode = annotation.GetValue().GetIntValue()
			}
			if annotation.GetKey() == constant.RETURN_DATA {
				cacheCode = true
				returnData := annotation.GetValue().GetBoolValue()
				if returnData {
					return "", "", false, 0, false
				}
			}
		}
	}
	// 只有同时发现url和target才进入计算
	if cacheURL && cacheTarget {
		return target, url, cacheCode, httpCode, true
	}
	return "", "", false, 0, false
}

// analysishttpEvent 解析httpevent，获取目标地址或者域名已经url和code
func (s *Stats) analysishttpEvent333(event *trace.TSpanEvent) (string, string, bool, int32, bool) {

	cacheCode := false
	cacheURL := false
	cacheTarget := false
	var target, url string
	var httpCode int32

	for _, annotation := range event.GetAnnotations() {
		if annotation.GetKey() == constant.HTTP_INTERNAL_DISPLAY {
			cacheTarget = true
			target = annotation.GetValue().GetStringValue()
		}
		if annotation.GetKey() == constant.HTTP_URL {
			cacheURL = true
			url = annotation.GetValue().GetStringValue()
		}
		if annotation.GetKey() == constant.HTTP_STATUS_CODE {
			cacheCode = true
			httpCode = annotation.GetValue().GetIntValue()
		}
	}
	// 只有同时发现url和target才进入计算
	if cacheURL && cacheTarget {
		return target, url, cacheCode, httpCode, true
	}
	return "", "", false, 0, false
}

// // analysishttpEvent 解析httpevent，获取目标地址或者域名已经url和code
// func (s *Stats) analysishttpEvent(event *trace.TSpanEvent) (string, string, bool, int32, bool) {
// 	cacheTarget := false
// 	cacheRetunData := false
// 	cacheURL := false
// 	cacheCode := false
// 	var target, url string
// 	var httpCode int32
// 	// var retunData string
// 	for _, annotation := range event.GetAnnotations() {
// 		if annotation.GetKey() == constant.HTTP_INTERNAL_DISPLAY {
// 			cacheTarget = true
// 			target = annotation.GetValue().GetStringValue()
// 		}
// 		if annotation.GetKey() == constant.HTTP_URL {
// 			cacheURL = true
// 			url = annotation.GetValue().GetStringValue()
// 		}
// 		if annotation.GetKey() == constant.HTTP_STATUS_CODE {
// 			cacheCode = true
// 			httpCode = annotation.GetValue().GetIntValue()
// 		}
// 		if annotation.GetKey() == constant.RETURN_DATA {
// 			cacheRetunData = true
// 			// retunData = annotation.GetValue().GetStringValue()
// 			// continue
// 		}
// 	}

// 	if cacheTarget && !cacheRetunData {
// 		if event.GetExceptionInfo() != nil {
// 			// log.Println("Analysishttp 1  target =", target, ", code =", httpCode, ", cacheCode =", cacheCode)
// 			return target, "/", cacheCode, httpCode, true
// 		}
// 	} else if cacheURL {
// 		// log.Println("Analysishttp 2 target =", event.GetDestinationId(), ", code =", httpCode, ", cacheCode =", cacheCode)
// 		return event.GetDestinationId(), url, cacheCode, httpCode, true
// 	}
// 	return "", "", false, 0, false
// }

// //go:binary-only-package

// package plugin
