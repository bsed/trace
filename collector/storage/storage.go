package storage

import (
	"encoding/json"
	"math/rand"
	"time"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/collector/misc"
	"github.com/bsed/trace/pkg/constant"

	"github.com/bsed/trace/pkg/network"
	"github.com/bsed/trace/pkg/pinpoint/thrift/pinpoint"
	"github.com/bsed/trace/pkg/pinpoint/thrift/trace"
	"github.com/bsed/trace/pkg/sql"
	"github.com/bsed/trace/pkg/stats"

	"github.com/sunface/talent"
	"go.uber.org/zap"
)

// Storage 存储
type Storage struct {
	staticCql      *gocql.Session
	traceCql       *gocql.Session
	spanChans      []chan *trace.TSpan
	spanChunkChans []chan *trace.TSpanChunk
	logger         *zap.Logger
	// spanChan       chan *trace.TSpan
	// spanChunkChan  chan *trace.TSpanChunk
}

// NewStorage 新建存储
func NewStorage(logger *zap.Logger) *Storage {
	return &Storage{
		spanChans:      make([]chan *trace.TSpan, misc.Conf.Storage.GoruntineNum),
		spanChunkChans: make([]chan *trace.TSpanChunk, misc.Conf.Storage.GoruntineNum),
		logger:         logger,
		// spanChunkChans []chan *trace.TSpanChunk
		// metricsChan:   make(chan *util.MetricData, misc.Conf.Storage.MetricCacheLen+500),
	}
}

// init 初始化存储
func (s *Storage) init() error {
	rand.Seed(time.Now().UnixNano())
	if err := s.initTraceCql(); err != nil {
		s.logger.Warn("init trace cql error", zap.String("error", err.Error()))
		return err
	}

	if err := s.initStaticCql(); err != nil {
		s.logger.Warn("init static cql error", zap.String("error", err.Error()))
		return err
	}
	return nil
}

func (s *Storage) initTraceCql() error {
	// connect to the cluster
	cluster := gocql.NewCluster(misc.Conf.Storage.Cluster...)
	cluster.Keyspace = misc.Conf.Storage.TraceKeyspace
	cluster.Consistency = gocql.Quorum
	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cluster.NumConns = misc.Conf.Storage.NumConns
	cluster.ReconnectInterval = 1 * time.Second
	session, err := cluster.CreateSession()
	if err != nil {
		s.logger.Warn("create session", zap.String("error", err.Error()))
		return err
	}

	s.traceCql = session
	return nil
}

func (s *Storage) initStaticCql() error {
	// connect to the cluster
	cluster := gocql.NewCluster(misc.Conf.Storage.Cluster...)
	cluster.Keyspace = misc.Conf.Storage.StaticKeyspace
	cluster.Consistency = gocql.Quorum
	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cluster.NumConns = misc.Conf.Storage.NumConns
	cluster.ReconnectInterval = 1 * time.Second
	session, err := cluster.CreateSession()
	if err != nil {
		s.logger.Warn("create session", zap.String("error", err.Error()))
		return err
	}
	s.staticCql = session
	return nil
}

// Start ...
func (s *Storage) Start() error {
	if err := s.init(); err != nil {
		s.logger.Warn("storage init", zap.String("error", err.Error()))
		return err
	}

	for index := 0; index < misc.Conf.Storage.GoruntineNum; index++ {
		spanChan := make(chan *trace.TSpan, misc.Conf.Storage.SpanCacheLen+500)
		spanChunkChan := make(chan *trace.TSpanChunk, misc.Conf.Storage.SpanChunkCacheLen+500)
		s.spanChans[index] = spanChan
		s.spanChunkChans[index] = spanChunkChan
		go s.spanStore(spanChan)
		go s.spanChunkStore(spanChunkChan)
	}

	// go s.systemStore()
	return nil
}

// SpanStore span存储
func (s *Storage) SpanStore(span *trace.TSpan) {
	index := rand.Intn(misc.Conf.Storage.GoruntineNum)
	s.spanChans[index] <- span
}

// SpanChunkStore spanChunk存储
func (s *Storage) SpanChunkStore(span *trace.TSpanChunk) {
	index := rand.Intn(misc.Conf.Storage.GoruntineNum)
	s.spanChunkChans[index] <- span
}

// Close ...
func (s *Storage) Close() error {
	return nil
}

// AgentStore agent信息存储
func (s *Storage) AgentStore(agentInfo *network.AgentInfo, islive bool) error {
	query := s.staticCql.Query(
		sql.InsertAgent,
		agentInfo.AppName,
		agentInfo.AgentID,
		agentInfo.ServiceType,
		agentInfo.HostName,
		agentInfo.IP4S,
		agentInfo.StartTimestamp,
		agentInfo.EndTimestamp,
		agentInfo.IsContainer,
		agentInfo.OperatingEnv,
		misc.Conf.Collector.Addr,
		islive,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("agent store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// UpdateAgentState agent在线状态更新
func (s *Storage) UpdateAgentState(appname string, agentid string, islive bool) error {
	var entTime int64
	if !islive {
		entTime = time.Now().Unix() * 1000
	}
	query := s.staticCql.Query(
		sql.UpdateAgentState,
		islive,
		entTime,
		appname,
		agentid,
	).Consistency(gocql.One)

	if err := query.Exec(); err != nil {
		s.logger.Warn("update agent state error", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// GetCql 获取cql
func (s *Storage) GetStaticCql() *gocql.Session {
	return s.staticCql
}

// GetCql 获取cql
func (s *Storage) GetTraceCql() *gocql.Session {
	return s.traceCql
}

// AppNameStore 存储Appname
func (s *Storage) AppNameStore(name string) error {
	query := s.staticCql.Query(
		sql.InsertApp,
		name,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("insert app name error", zap.String("SQL", query.String()), zap.String("error", err.Error()), zap.String("appName", name))
		return err
	}
	return nil
}

// AgentInfoStore ...
func (s *Storage) AgentInfoStore(appName, agentID string, startTime int64, agentInfo []byte) error {
	query := s.staticCql.Query(
		sql.InsertAgentInfo,
		appName,
		agentID,
		startTime,
		string(agentInfo),
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("agent info store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

// AppMethodStore ...
func (s *Storage) AppMethodStore(appName string, apiInfo *trace.TApiMetaData) error {
	query := s.staticCql.Query(
		sql.InsertMethod,
		appName,
		apiInfo.ApiId,
		apiInfo.ApiInfo,
		apiInfo.GetLine(),
		apiInfo.GetType(),
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("api store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

// AppSQLStore sql语句存储，sql语句需要base64转码，防止sql注入
func (s *Storage) AppSQLStore(appName string, sqlInfo *trace.TSqlMetaData) error {
	newSQL := g.B64.EncodeToString(talent.String2Bytes(sqlInfo.Sql))
	query := s.staticCql.Query(
		sql.InsertSQL,
		appName,
		sqlInfo.SqlId,
		newSQL,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("sql store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// AppStringStore ...
func (s *Storage) AppStringStore(appName string, strInfo *trace.TStringMetaData) error {
	query := s.staticCql.Query(
		sql.InsertString,
		appName,
		strInfo.StringId,
		strInfo.StringValue,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("string store", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

// spanStore ...
func (s *Storage) spanStore(spanChan chan *trace.TSpan) {
	ticker := time.NewTicker(time.Duration(misc.Conf.Storage.SpanStoreInterval) * time.Millisecond)
	var spansQueue []*trace.TSpan
	for {
		select {
		case span, ok := <-spanChan:
			if ok {
				spansQueue = append(spansQueue, span)
				if len(spansQueue) >= misc.Conf.Storage.SpanCacheLen {
					// 插入
					for _, span := range spansQueue {
						if err := s.WriteSpan(span); err != nil {
							s.logger.Warn("write span", zap.String("error", err.Error()))
							continue
						}
					}
					// 清空缓存
					spansQueue = spansQueue[:0]
				}
			}
			break
		case <-ticker.C:
			if len(spansQueue) > 0 {
				// 插入
				for _, span := range spansQueue {
					if err := s.WriteSpan(span); err != nil {
						s.logger.Warn("write span", zap.String("error", err.Error()))
						continue
					}
				}
				// 清空缓存
				spansQueue = spansQueue[:0]
			}
			break
		}
	}
}

// // spanStore ...
// func (s *Storage) spanStore2() {
// 	ticker := time.NewTicker(time.Duration(misc.Conf.Storage.SpanStoreInterval) * time.Millisecond)
// 	var spansQueue []*trace.TSpan
// 	for {
// 		select {
// 		case span, ok := <-s.spanChan:
// 			if ok {
// 				spansQueue = append(spansQueue, span)
// 				if len(spansQueue) >= misc.Conf.Storage.SpanCacheLen {
// 					// 插入
// 					for _, qSpan := range spansQueue {
// 						if err := s.WriteSpan(qSpan); err != nil {
// 							s.logger.Warn("write span", zap.String("error", err.Error()))
// 							continue
// 						}
// 					}
// 					// 清空缓存
// 					spansQueue = spansQueue[:0]
// 				}
// 			}
// 			break
// 		case <-ticker.C:
// 			if len(spansQueue) > 0 {
// 				// 插入
// 				for _, span := range spansQueue {
// 					if err := s.WriteSpan(span); err != nil {
// 						s.logger.Warn("write span", zap.String("error", err.Error()))
// 						continue
// 					}
// 				}
// 				// 清空缓存
// 				spansQueue = spansQueue[:0]
// 			}
// 			break
// 		}
// 	}
// }

// // spanChunkStore ...
// func (s *Storage) spanChunkStore2() {
// 	ticker := time.NewTicker(time.Duration(misc.Conf.Storage.SpanStoreInterval) * time.Millisecond)
// 	var spansChunkQueue []*trace.TSpanChunk
// 	for {
// 		select {
// 		case spanChunk, ok := <-s.spanChunkChan:
// 			if ok {
// 				spansChunkQueue = append(spansChunkQueue, spanChunk)
// 				if len(spansChunkQueue) >= misc.Conf.Storage.SpanChunkCacheLen {
// 					// 插入
// 					if err := s.writeSpanChunk(spansChunkQueue); err != nil {
// 						s.logger.Warn("write spanChunk", zap.String("error", err.Error()))
// 						continue
// 					}
// 					// 清空缓存
// 					spansChunkQueue = spansChunkQueue[:0]
// 				}
// 			}
// 			break
// 		case <-ticker.C:
// 			if len(spansChunkQueue) > 0 {
// 				// 插入
// 				if err := s.writeSpanChunk(spansChunkQueue); err != nil {
// 					s.logger.Warn("write spanChunk", zap.String("error", err.Error()))
// 					continue
// 				}

// 				// 清空缓存
// 				spansChunkQueue = spansChunkQueue[:0]
// 			}
// 			break
// 		}
// 	}
// }

// spanChunkStore ...
func (s *Storage) spanChunkStore(spanChunkChan chan *trace.TSpanChunk) {
	ticker := time.NewTicker(time.Duration(misc.Conf.Storage.SpanStoreInterval) * time.Millisecond)
	var spansChunkQueue []*trace.TSpanChunk
	for {
		select {
		case spanChunk, ok := <-spanChunkChan:
			if ok {
				spansChunkQueue = append(spansChunkQueue, spanChunk)
				if len(spansChunkQueue) >= misc.Conf.Storage.SpanChunkCacheLen {
					// 插入
					for _, qSapnChunk := range spansChunkQueue {
						if err := s.writeSpanChunk(qSapnChunk); err != nil {
							s.logger.Warn("write spanChunk", zap.String("error", err.Error()))
							continue
						}
					}
					// 清空缓存
					spansChunkQueue = spansChunkQueue[:0]
				}
			}
			break
		case <-ticker.C:
			if len(spansChunkQueue) > 0 {
				// 插入
				for _, sapnChunk := range spansChunkQueue {
					if err := s.writeSpanChunk(sapnChunk); err != nil {
						s.logger.Warn("write spanChunk", zap.String("error", err.Error()))
						continue
					}
				}
				// 清空缓存
				spansChunkQueue = spansChunkQueue[:0]
			}
			break
		}
	}
}

// WriteSpan2 ...
func (s *Storage) WriteSpan2(spans []*trace.TSpan) error {
	if err := s.writeSpan2(spans); err != nil {
		s.logger.Warn("write spans", zap.String("error", err.Error()))
		return err
	}

	if err := s.traceIndex2(spans); err != nil {
		s.logger.Warn("write span index", zap.String("error", err.Error()))
		return err
	}
	return nil
}

// traceIndex ...
func (s *Storage) traceIndex2(spans []*trace.TSpan) error {
	batchInsert := s.traceCql.NewBatch(gocql.UnloggedBatch)
	// 通过event来判断是否存在异常
	for _, span := range spans {
		isErr := span.GetErr()
		if isErr == 0 {
			for _, event := range span.GetSpanEventList() {
				if event.GetExceptionInfo() != nil {
					isErr = 1
					break
				}
			}
		}

		batchInsert.Query(
			sql.InsertTraceIndex,
			span.GetApplicationName(),
			span.GetAgentId(),
			span.GetTransactionId(),
			span.GetSpanId(),
			span.GetRPC(),
			span.GetRemoteAddr(),
			span.GetStartTime(),
			span.GetElapsed(),
			isErr,
		)
	}
	if err := s.traceCql.ExecuteBatch(batchInsert); err != nil {
		s.logger.Warn("insert trace index error", zap.String("error", err.Error()), zap.String("SQL", sql.InsertRuntimeStat))
		return err
	}
	return nil
}

// writeSpan ...
func (s *Storage) writeSpan2(spans []*trace.TSpan) error {
	batchInsert := s.traceCql.NewBatch(gocql.UnloggedBatch)
	for _, span := range spans {
		// @TODO 转码优化
		annotations, _ := json.Marshal(span.GetAnnotations())
		spanEvenlist, _ := json.Marshal(span.GetSpanEventList())
		exceptioninfo, _ := json.Marshal(span.GetExceptionInfo())

		// 通过event来判断是否存在异常
		isErr := span.GetErr()
		if isErr == 0 {
			for _, event := range span.GetSpanEventList() {
				if event.GetExceptionInfo() != nil {
					isErr = 1
					break
				}
			}
		}

		batchInsert.Query(sql.InsertSpan,
			span.GetTransactionId(),
			span.GetSpanId(),
			span.GetApplicationName(),
			span.GetAgentId(),
			span.GetElapsed(),
			span.GetRPC(),
			span.GetServiceType(),
			span.GetEndPoint(),
			span.GetRemoteAddr(),
			annotations,
			spanEvenlist,
			span.GetParentSpanId(),
			span.GetApiId(),
			exceptioninfo,
			isErr,
			span.GetStartTime(),
		) //.Consistency(gocql.One)
	}
	if err := s.traceCql.ExecuteBatch(batchInsert); err != nil {
		s.logger.Warn("agent stat batch", zap.String("error", err.Error()), zap.String("SQL", sql.InsertRuntimeStat))
		return err
	}
	return nil
}

// WriteSpan ...
func (s *Storage) WriteSpan(span *trace.TSpan) error {
	if err := s.writeSpan(span); err != nil {
		s.logger.Warn("write span", zap.String("error", err.Error()))
		return err
	}
	if err := s.traceIndex(span); err != nil {
		s.logger.Warn("appTraceIndex error", zap.String("error", err.Error()))
		return err
	}
	// if err := s.writeIndexes(span); err != nil {
	// 	s.logger.Warn("write span index", zap.String("error", err.Error()))
	// 	return err
	// }
	return nil
}

// writeSpan ...
func (s *Storage) writeSpan(span *trace.TSpan) error {
	// @TODO 转码优化
	annotations, _ := json.Marshal(span.GetAnnotations())
	spanEvenlist, _ := json.Marshal(span.GetSpanEventList())
	exceptioninfo, _ := json.Marshal(span.GetExceptionInfo())

	// 通过event来判断是否存在异常
	isErr := span.GetErr()
	if isErr == 0 {
		for _, event := range span.GetSpanEventList() {
			if event.GetExceptionInfo() != nil {
				isErr = 1
				break
			}
		}
	}
	query := s.traceCql.Query(
		sql.InsertSpan,
		span.GetTransactionId(),
		span.GetSpanId(),
		span.GetApplicationName(),
		span.GetAgentId(),
		span.GetElapsed(),
		span.GetRPC(),
		span.GetServiceType(),
		span.GetEndPoint(),
		span.GetRemoteAddr(),
		annotations,
		spanEvenlist,
		span.GetParentSpanId(),
		span.GetApiId(),
		exceptioninfo,
		isErr,
		span.GetStartTime(),
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("write span", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// // writeSpanChunk ...
// func (s *Storage) writeSpanChunk(spanChunks []*trace.TSpanChunk) error {
// 	batchInsert := s.traceCql.NewBatch(gocql.UnloggedBatch)
// 	for _, spanChunk := range spanChunks {
// 		spanEvenlist, _ := json.Marshal(spanChunk.GetSpanEventList())
// 		batchInsert.Query(
// 			sql.InsertSpanChunk,
// 			spanChunk.GetTransactionId(),
// 			spanChunk.GetSpanId(),
// 			time.Now().UnixNano(),
// 			spanEvenlist,
// 		)
// 	}

// 	if err := s.traceCql.ExecuteBatch(batchInsert); err != nil {
// 		s.logger.Warn("agent stat batch", zap.String("error", err.Error()), zap.String("SQL", sql.InsertRuntimeStat))
// 		return err
// 	}
// 	return nil
// }

// writeSpanChunk ...
func (s *Storage) writeSpanChunk(spanChunk *trace.TSpanChunk) error {

	spanEvenlist, _ := json.Marshal(spanChunk.GetSpanEventList())

	query := s.traceCql.Query(
		sql.InsertSpanChunk,
		spanChunk.GetTransactionId(),
		spanChunk.GetSpanId(),
		time.Now().UnixNano(),
		spanEvenlist,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("write spanChunk", zap.String("error", err.Error()), zap.String("SQL", query.String()))
		return err
	}

	return nil
}

// writeIndexes ...
func (s *Storage) writeIndexes(span *trace.TSpan) error {
	if err := s.traceIndex(span); err != nil {
		s.logger.Warn("appTraceIndex error", zap.String("error", err.Error()))
		return err
	}
	return nil
}

// traceIndex ...
func (s *Storage) traceIndex(span *trace.TSpan) error {
	// 通过event来判断是否存在异常
	isErr := span.GetErr()
	if isErr == 0 {
		for _, event := range span.GetSpanEventList() {
			if event.GetExceptionInfo() != nil {
				isErr = 1
				break
			}
		}
	}

	query := s.traceCql.Query(
		sql.InsertTraceIndex,
		span.GetApplicationName(),
		span.GetAgentId(),
		span.GetTransactionId(),
		span.GetSpanId(),
		span.GetRPC(),
		span.GetRemoteAddr(),
		span.GetStartTime(),
		span.GetElapsed(),
		isErr,
	).Consistency(gocql.One)

	if err := query.Exec(); err != nil {
		s.logger.Warn("inster trace index error", zap.String("error", err.Error()), zap.String("SQL", query.String()))
		return err
	}

	return nil
}

// WriteAgentStatBatch ....
func (s *Storage) WriteAgentStatBatch(appName, agentID string, agentStatBatch *pinpoint.TAgentStatBatch, infoB []byte) error {
	batchInsert := s.traceCql.NewBatch(gocql.UnloggedBatch)

	for _, agentStat := range agentStatBatch.AgentStats {
		jvmInfo := stats.NewJVMInfo()
		jvmInfo.CPULoad.Jvm = agentStat.CpuLoad.GetJvmCpuLoad()
		jvmInfo.CPULoad.System = agentStat.CpuLoad.GetSystemCpuLoad()
		jvmInfo.GC.Type = agentStat.Gc.GetType()
		jvmInfo.GC.HeapUsed = agentStat.Gc.GetJvmMemoryHeapUsed()
		jvmInfo.GC.HeapMax = agentStat.Gc.GetJvmMemoryHeapMax()
		jvmInfo.GC.NonHeapUsed = agentStat.Gc.GetJvmMemoryNonHeapUsed()
		jvmInfo.GC.NonHeapMax = agentStat.Gc.GetJvmMemoryHeapMax()
		jvmInfo.GC.GcOldCount = agentStat.Gc.GetJvmGcOldCount()
		jvmInfo.GC.JvmGcOldTime = agentStat.Gc.GetJvmGcOldTime()
		jvmInfo.GC.JvmGcNewCount = agentStat.Gc.GetJvmGcDetailed().GetJvmGcNewCount()
		jvmInfo.GC.JvmGcNewTime = agentStat.Gc.GetJvmGcDetailed().GetJvmGcNewTime()
		jvmInfo.GC.JvmPoolCodeCacheUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolCodeCacheUsed()
		jvmInfo.GC.JvmPoolNewGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolNewGenUsed()
		jvmInfo.GC.JvmPoolOldGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolOldGenUsed()
		jvmInfo.GC.JvmPoolSurvivorSpaceUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolSurvivorSpaceUsed()
		jvmInfo.GC.JvmPoolPermGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolPermGenUsed()
		jvmInfo.GC.JvmPoolMetaspaceUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolMetaspaceUsed()

		body, err := json.Marshal(jvmInfo)
		if err != nil {
			s.logger.Warn("json marshal", zap.String("error", err.Error()))
			continue
		}

		t, err := utils.MSToTime(agentStat.GetTimestamp())
		if err != nil {
			s.logger.Warn("ms to time", zap.Int64("time", agentStat.GetTimestamp()), zap.String("error", err.Error()))
			continue
		}

		batchInsert.Query(
			sql.InsertRuntimeStat,
			appName,
			agentID,
			t.Unix(),
			body,
			1)
	}
	if err := s.traceCql.ExecuteBatch(batchInsert); err != nil {
		s.logger.Warn("agent stat batch", zap.String("error", err.Error()), zap.String("SQL", sql.InsertRuntimeStat),
			zap.String("appName", appName), zap.String("agentID", agentID), zap.Any("value", agentStatBatch))
		return err
	}

	return nil
}

// WriteAgentStat  ...
func (s *Storage) WriteAgentStat(appName, agentID string, agentStat *pinpoint.TAgentStat, infoB []byte) error {
	jvmInfo := stats.NewJVMInfo()
	jvmInfo.CPULoad.Jvm = agentStat.CpuLoad.GetJvmCpuLoad()
	jvmInfo.CPULoad.System = agentStat.CpuLoad.GetSystemCpuLoad()
	jvmInfo.GC.Type = agentStat.Gc.GetType()
	jvmInfo.GC.HeapUsed = agentStat.Gc.GetJvmMemoryHeapUsed()
	jvmInfo.GC.HeapMax = agentStat.Gc.GetJvmMemoryHeapMax()
	jvmInfo.GC.NonHeapUsed = agentStat.Gc.GetJvmMemoryNonHeapUsed()
	jvmInfo.GC.NonHeapMax = agentStat.Gc.GetJvmMemoryHeapMax()
	jvmInfo.GC.GcOldCount = agentStat.Gc.GetJvmGcOldCount()
	jvmInfo.GC.JvmGcOldTime = agentStat.Gc.GetJvmGcOldTime()
	jvmInfo.GC.JvmGcNewCount = agentStat.Gc.GetJvmGcDetailed().GetJvmGcNewCount()
	jvmInfo.GC.JvmGcNewTime = agentStat.Gc.GetJvmGcDetailed().GetJvmGcNewTime()
	jvmInfo.GC.JvmPoolCodeCacheUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolCodeCacheUsed()
	jvmInfo.GC.JvmPoolNewGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolNewGenUsed()
	jvmInfo.GC.JvmPoolOldGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolOldGenUsed()
	jvmInfo.GC.JvmPoolSurvivorSpaceUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolSurvivorSpaceUsed()
	jvmInfo.GC.JvmPoolPermGenUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolPermGenUsed()
	jvmInfo.GC.JvmPoolMetaspaceUsed = agentStat.Gc.GetJvmGcDetailed().GetJvmPoolMetaspaceUsed()

	body, err := json.Marshal(jvmInfo)
	if err != nil {
		s.logger.Warn("json marshal", zap.String("error", err.Error()))
		return err
	}

	t, err := utils.MSToTime(agentStat.GetTimestamp())
	if err != nil {
		s.logger.Warn("ms to time", zap.Int64("time", agentStat.GetTimestamp()), zap.String("error", err.Error()))
		return err
	}

	query := s.staticCql.Query(
		sql.InsertRuntimeStat,
		appName,
		agentID,
		t.Unix(),
		body,
		1,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("inster agentstat", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}

	return nil
}

// StoreAPI 存储API信息
func (s *Storage) StoreAPI(span *trace.TSpan) error {
	query := s.staticCql.Query(
		sql.InsertAPIs,
		span.GetApplicationName(),
		span.GetRPC(),
		span.GetServiceType(),
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("store api", zap.String("SQL", query.String()), zap.String("error", err.Error()))
		return err
	}
	return nil
}

// InsertAPIStats ...
func (s *Storage) InsertAPIStats(appName string, inputDate int64, urlStr string, url *stats.Url) error {
	query := s.traceCql.Query(sql.InsertAPIStats,
		appName,
		url.AccessCount,
		url.AccessErrCount,
		url.Duration,
		url.MaxDuration,
		url.MinDuration,
		url.SatisfactionCount,
		url.TolerateCount,
		urlStr,
		inputDate).Consistency(gocql.One)

	if err := query.Exec(); err != nil {
		s.logger.Warn("inster api stats error", zap.String("error", err.Error()), zap.String("sql", query.String()))
		return err
	}

	return nil
}

// // InsertDubboAPIStats ...
// func (s *Storage) InsertDubboAPIStats(appName string, inputDate int64, dubboStr string, dubbo *stats.Dubbo) error {
// 	query := s.traceCql.Query(sql.InsertAPIStats,
// 		appName,
// 		dubbo.AccessCount,
// 		dubbo.AccessErrCount,
// 		dubbo.Duration,
// 		dubbo.MaxDuration,
// 		dubbo.MinDuration,
// 		dubbo.SatisfactionCount,
// 		dubbo.TolerateCount,
// 		dubboStr,
// 		inputDate).Consistency(gocql.One)

// 	if err := query.Exec(); err != nil {
// 		s.logger.Warn("inster dubbo api stats error", zap.String("error", err.Error()), zap.String("sql", query.String()))
// 		return err
// 	}

// 	return nil
// }

// InsertDubboStats ...
func (s *Storage) InsertDubboStats(appName string, inputDate int64, dubboApi string, dubbo *stats.Dubbo) error {
	query := s.traceCql.Query(sql.InsertAPIStats,
		appName,
		dubbo.AccessCount,
		dubbo.AccessErrCount,
		dubbo.Duration,
		dubbo.MaxDuration,
		dubbo.MinDuration,
		dubbo.SatisfactionCount,
		dubbo.TolerateCount,
		dubboApi,
		inputDate).Consistency(gocql.One)

	if err := query.Exec(); err != nil {
		s.logger.Warn("inster dubbo api stats error", zap.String("error", err.Error()), zap.String("sql", query.String()))
		return err
	}

	return nil
}

// InsertMethodStats 接口计算数据存储
func (s *Storage) InsertMethodStats(appName string, inputTime int64, apiStr string, methodID int32, methodInfo *stats.Method) error {
	query := s.traceCql.Query(sql.InsertMethodStats,
		appName,
		apiStr,
		inputTime,
		methodID,
		methodInfo.Type,
		methodInfo.Duration,
		methodInfo.MaxDuration,
		methodInfo.MinDuration,
		methodInfo.Count,
		methodInfo.ErrCount,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("insert method error", zap.String("error", err.Error()), zap.String("SQL", query.String()))
		return err
	}
	return nil
}

// InsertExceptionStats ...
func (s *Storage) InsertExceptionStats(appName string, inputTime int64, methodID int32, exceptions map[int32]*stats.Exception) error {
	for classID, exinfo := range exceptions {
		query := s.traceCql.Query(sql.InsertExceptionStats,
			appName,
			methodID,
			classID,
			inputTime,
			exinfo.Duration,
			exinfo.MaxDuration,
			exinfo.MinDuration,
			exinfo.Count,
			exinfo.Type,
		).Consistency(gocql.One)
		if err := query.Exec(); err != nil {
			s.logger.Warn("insert exception error", zap.String("error", err.Error()), zap.String("SQL", query.String()))
			return err
		}
	}
	return nil
}

// // InsertParentMap ...
// func (s *Storage) InsertParentMap(appName string, appType int32, inputTime int64, parentName string, parent *stats.SrvParent) error {
// 	query := s.traceCql.Query(sql.InsertParentMap,
// 		parentName,
// 		parent.Type,
// 		appName,
// 		appType,
// 		0,
// 		0,
// 		0,
// 		parent.TargetCount,
// 		parent.TargetErrCount,
// 		inputTime,
// 	)
// 	if err := query.Exec(); err != nil {
// 		s.logger.Warn("insert parent map error", zap.String("error", err.Error()), zap.String("sql", query.String()))
// 		return err
// 	}
// 	return nil
// }

// InsertTargetMap ...
func (s *Storage) InsertTargetMap(appName string,
	appType int32, inputDate int64,
	targetType int32, targetName string,
	target *stats.Target) error {

	query := s.traceCql.Query(sql.InsertTargetMap,
		appName,
		appType,
		targetName,
		targetType,
		target.AccessCount,
		target.AccessErrCount,
		target.AccessDuration, // access_err_count
		inputDate,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("insert child map error", zap.String("error", err.Error()), zap.String("sql", query.String()))
		return err
	}

	return nil
}

// InsertUnknowParentMap ...
func (s *Storage) InsertUnknowParentMap(targetName string, targetType int32, inputDate int64, unknowParent *stats.UnknowParent) error {
	query := s.traceCql.Query(sql.InsertUnknowParentMap,
		"UNKNOWN",
		constant.UNKNOWN,
		targetName,
		targetType,
		unknowParent.AccessCount,
		0,
		unknowParent.AccessDuration,
		inputDate,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("insert unknow parent map error", zap.String("error", err.Error()), zap.String("sql", query.String()))
		return err
	}

	return nil
}

// InsertAPIMapStats Api被调用统计信息
func (s *Storage) InsertAPIMapStats(appName string, appType int32, inputTime int64, apiStr string, parentname string, parentInfo *stats.Parent) error {
	query := s.traceCql.Query(sql.InsertAPIMapStats,
		parentname,
		parentInfo.Type,
		appName,
		appType,
		parentInfo.AccessCount,
		parentInfo.AccessErrCount,
		parentInfo.AccessDuration,
		apiStr,
		inputTime,
	).Consistency(gocql.One)
	if err := query.Exec(); err != nil {
		s.logger.Warn("insert api map error", zap.String("error", err.Error()), zap.String("sql", query.String()))
		return err
	}

	return nil
}

// InsertSQLStats ...
func (s *Storage) InsertSQLStats(appName string, inputTime int64, sqlID int32, sqlInfo *stats.SQL) error {
	query := s.traceCql.Query(sql.InsertSQLStats,
		appName,
		sqlID,
		inputTime,
		sqlInfo.Duration,
		sqlInfo.MaxDuration,
		sqlInfo.MinDuration,
		sqlInfo.Count,
		sqlInfo.ErrCount,
	).Consistency(gocql.One)

	if err := query.Exec(); err != nil {
		s.logger.Warn("sql stats insert error", zap.String("error", err.Error()), zap.String("SQL", query.String()))
		return err
	}
	return nil
}
