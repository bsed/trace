package app

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type Trace struct {
	ID         string `json:"id"`
	API        string `json:"api"`
	Elapsed    int    `json:"y"`
	AgentID    string `json:"agent_id"`
	InputDate  int64  `json:"x"`
	Error      int    `json:"error"`
	RemoteAddr string `json:"remote_addr"`
}

type Traces []*Trace

func (a Traces) Len() int { // 重写 Len() 方法
	return len(a)
}
func (a Traces) Swap(i, j int) { // 重写 Swap() 方法
	a[i], a[j] = a[j], a[i]
}
func (a Traces) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
	return a[j].Elapsed < a[i].Elapsed
}

type ChartTraces struct {
	Suc   bool    `json:"is_suc"`
	Xaxis []int64 `json:"timeXticks"`
	Title string  `json:"subTitle"`

	Series []*TraceSeries `json:"series"`
}

type TraceSeries struct {
	Name  string   `json:"name"`
	Color string   `json:"color"`
	Data  []*Trace `json:"data"`
}

// traceSeries":[{"name":"success","color":"rgb(18, 147, 154,.5)","data":[{"x":1545200556716,"y":7,"traceId":"yunbaoParkApp3^1545036617750^4217","agentId":"agencyBookKeep3","startTime":"1545200556716","url":"/agencyBookKeep/financialstatementscs/getOneAccountingDataByParams","traceIp":"127.0.0.1"},
func QueryTraces(c echo.Context) error {
	appName := c.FormValue("app_name")
	api := c.FormValue("api")
	min, _ := strconv.Atoi(c.FormValue("min_elapsed"))
	max, _ := strconv.Atoi(c.FormValue("max_elapsed"))
	limit, err := strconv.Atoi(c.FormValue("limit"))

	searchError, _ := strconv.ParseBool(c.FormValue("search_error"))
	searchTraceID := c.FormValue("search_trace_id")
	// raddr := c.FormValue("remote_addr")
	if err != nil {
		limit = 50
	}

	start, end, _ := misc.StartEndDate(c)

	traceMap := make(map[string]*Trace)

	var q *gocql.Query
	if searchTraceID != "" {
		q = misc.TraceCql.Query("SELECT app_name,trace_id,api,duration,agent_id,input_date,error,remote_addr FROM traces WHERE trace_id=?", searchTraceID)
		iter := q.Iter()
		var elapsed, isError int
		var inputDate int64
		var tid, agentID, remoteAddr, appname string

		for iter.Scan(&appname, &tid, &api, &elapsed, &agentID, &inputDate, &isError, &remoteAddr) {
			if appName == appname {
				traceMap[tid] = &Trace{tid, api, elapsed, agentID, inputDate, isError, remoteAddr}
			}
		}

		if err := iter.Close(); err != nil {
			g.L.Warn("close iter error:", zap.Error(err), zap.String("query", q.String()))
			return err
		}

	} else {
		// 根据查询条件，拼装查询语句和参数
		// 默认条件1
		qs := "SELECT trace_id,api,duration,agent_id,input_date,error,remote_addr FROM traces_index WHERE app_name=?"
		args := []interface{}{appName}

		// api条件
		if api != "" {
			qs = qs + " and api=?"
			args = append(args, api)
		}

		// 默认条件2
		qs = qs + " and input_date > ? and input_date < ?"
		args = append(args, start.Unix()*1000, end.Unix()*1000)

		// // 查询指定的remote_addr
		// if raddr != "" {
		// 	qs = qs + " and remote_addr=?"
		// 	args = append(args, raddr)
		// }

		needFiltering := false
		if min > 0 {
			qs = qs + " and duration >= ?"
			args = append(args, min)
			needFiltering = true
		}
		// 最大查询时间条件
		if max > 0 {
			qs = qs + " and duration <= ?"
			args = append(args, max)
			needFiltering = true
		}

		// 仅查询错误条件
		if searchError {
			// 只搜索错误
			qs = qs + " and error=1"
			needFiltering = true
		}

		if needFiltering {
			qs = qs + " ALLOW FILTERING"
		}

		q = misc.TraceCql.Query(qs, args...).Consistency(gocql.One)

		st := time.Now()
		iter := q.Iter()
		var elapsed, isError int
		var inputDate int64
		var tid, agentID, remoteAddr string

		for iter.Scan(&tid, &api, &elapsed, &agentID, &inputDate, &isError, &remoteAddr) {
			traceMap[tid] = &Trace{tid, api, elapsed, agentID, inputDate, isError, remoteAddr}
		}

		fmt.Printf("链路总数: %d, 查询耗时:%d \n", len(traceMap), time.Now().Sub(st).Nanoseconds()/1e6)
		if err := iter.Close(); err != nil {
			g.L.Warn("close iter error:", zap.Error(err), zap.String("query", q.String()))
			return err
		}
	}

	traces := make(Traces, 0, len(traceMap))
	for _, t := range traceMap {
		traces = append(traces, t)
	}

	sort.Sort(traces)

	// 取出耗时最高的limit数量的trace
	if limit < len(traces) {
		traces = traces[:limit]
	}

	ct := &ChartTraces{}
	if len(traces) == 0 {
		ct.Suc = false
	} else {
		ct.Suc = true
		ct.Xaxis = []int64{start.Unix() / 1e6, end.Unix() / 1e6}

		var sucTraces Traces
		var errTraces Traces

		for _, t := range traces {
			if t.Error == 0 {
				sucTraces = append(sucTraces, t)
			} else {
				errTraces = append(errTraces, t)
			}
		}

		ct.Title = fmt.Sprintf("success: %d, error: %d", len(sucTraces), len(errTraces))
		sucData := &TraceSeries{
			Name:  "success",
			Color: "rgb(18, 147, 154,.5)",
			Data:  sucTraces,
		}

		errData := &TraceSeries{
			Name:  "error",
			Color: "rgba(223, 83, 83, .5)",
			Data:  errTraces,
		}

		ct.Series = []*TraceSeries{sucData, errData}
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ct,
	})
}

func QueryTrace(c echo.Context) error {
	tid := c.FormValue("trace_id")

	// 加载trace的所有span
	spans := make(traceSpans, 0)
	err := spans.load(tid)
	if err != nil {
		return err
	}

	spans.sort()

	// 将span和events组合成链路tree
	tree := make(TraceTree, 0)
	for _, span := range spans {
		// 防止span重复处理,因为在event中会递归处理next span
		if span.dealed {
			continue
		}
		tree.addSpan(span, &spans)
		span.dealed = true
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   tree,
	})
}

/*-----------------------------数据结构和方法定义------------------------------------*/
//
// trace span
//
type traceSpan struct {
	id          int64 // span id
	pid         int64 // the parent span id
	appName     string
	agentID     string
	serviceType int
	startTime   int64 // ms timestamp
	events      []*SpanEvent
	duration    int
	api         string // 接口url
	methodID    int
	remoteAddr  string
	annotations []*TempTag

	dealed bool
}

type traceSpans []*traceSpan

func (o traceSpans) Len() int {
	return len(o)
}

func (o traceSpans) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o traceSpans) Less(i, j int) bool {
	return o[i].startTime < o[j].startTime
}

func (spans *traceSpans) load(tid string) error {
	q := misc.TraceCql.Query(`SELECT span_id,parent_id,app_name,agent_id,input_date,duration,api,service_type,
	end_point,remote_addr,error,event_list,method_id,annotations,exception_info from traces where trace_id=?`, tid)
	iter := q.Iter()

	var spanID, pid, inputDate int64
	var elapsed, serviceType, isErr, methodID int
	var appName, agentID, api, endPoint, remoteAddr, events, annotations, exception string

	// parse span
	for iter.Scan(&spanID, &pid, &appName, &agentID, &inputDate, &elapsed, &api, &serviceType,
		&endPoint, &remoteAddr, &isErr, &events, &methodID, &annotations, &exception) {
		// 首先把span本身转为segment
		var tags []*TempTag
		json.Unmarshal([]byte(annotations), &tags)

		span := &traceSpan{
			appName:     appName,
			agentID:     agentID,
			duration:    elapsed,
			api:         api,
			serviceType: serviceType,
			startTime:   inputDate,
			// startTime:   misc.Timestamp2TimeString(inputDate),
			id:          spanID,
			pid:         pid,
			remoteAddr:  remoteAddr,
			annotations: tags,
			methodID:    methodID,
		}

		// 解析span的events，并根据sequence进行排序(从小到大)
		var spanEvents SpanEvents
		json.Unmarshal([]byte(events), &spanEvents)
		// 加载span chunk
		q1 := misc.TraceCql.Query(`SELECT event_list from traces_chunk where trace_id=? and span_id=?`, tid, span.id)
		iter1 := q1.Iter()
		var eventsChunkS string
		for iter1.Scan(&eventsChunkS) {
			var eventsChunk SpanEvents
			json.Unmarshal([]byte(eventsChunkS), &eventsChunk)

			spanEvents = append(spanEvents, eventsChunk...)
		}

		if err := iter1.Close(); err != nil {
			g.L.Warn("close iter error:", zap.Error(err))
			// return err
		}
		sort.Sort(spanEvents)
		span.events = spanEvents

		*spans = append(*spans, span)
	}
	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
		return err
	}

	return nil
}

// span排序
// 1. 找到没有父节点的节点，放在最前面，然后按照时间排序
// 2. 有父节点的节点，放在父节点后面

func (spans *traceSpans) sort() {
	nspans := make(traceSpans, 0)
	tmpSpans := make(traceSpans, 0)
	bucket := make(map[int64]int)
	for i, span := range *spans {
		bucket[span.id] = i
	}

	// 找到起始span
	// span之间可能存在以下几种关系
	// 通过父子串联
	// 若不存在父子关系(离散服务的span，特殊情况)，则通过时间排序(不同服务器时间不同，因此不够可靠)？
	for _, span := range *spans {
		_, ok := bucket[span.pid]
		if !ok { // 无父节点
			nspans = append(nspans, span)
		} else { // 有父节点
			tmpSpans = append(tmpSpans, span)
		}
	}

	// 对父节点按照时间排序
	sort.Sort(nspans)
	// 把子节点添加到span列表中
	nspans = append(nspans, tmpSpans...)

	*spans = nspans
}

//
// 为了形成全链路，我们需要把trace的span和event组成一个tree结构,span和event对应的是tree node
// Tags、Exceptions都将转换为node进行展示
type TraceTreeNode struct {
	ID          string `json:"id"` // seg的id,p-0/p-0-1类的形式，通过这种形式形成层级tree
	Sequence    int    `json:"seq"`
	SpanID      string `json:"span_id"`
	Type        string `json:"type"`       // 1: span 2. event 3. tag
	Depth       int    `json:"depth"`      //node在完整的链路树上所处的层级，绝对层级 Tree Depth
	SpanDepth   int    `json:"span_depth"` //node对应的event在对应span中的层级,相对层级 span depth
	AppName     string `json:"app_name"`
	MethodID    int    `json:"method_id"`
	Method      string `json:"method"`
	Duration    int    `json:"duration"` // 耗时，-1 代表不显示耗时信息
	Params      string `json:"params"`
	ServiceType string `json:"service_type"`
	AgentID     string `json:"agent_id"`
	Class       string `json:"class"`
	StartTime   string `json:"start_time"`
	Icon        string `json:"icon"`     // 有些节点会显示特殊的icon，作为样式
	IsError     bool   `json:"is_error"` // 是否是错误/异常，

	DID    string `json:"did"` // 用于debug,event的destination_id
	NID    string `json:"nid"` // 用于debug, event的next_span_id
	spanID int64
	tags   []*TraceTag // Seg标签
}

// 初始化node
func (node *TraceTreeNode) init(spanID int64, appName, agentID, params string, service, methodID int) {
	node.SpanID = strconv.FormatInt(spanID, 10)
	node.spanID = spanID
	node.AppName = appName
	node.Params = params
	node.AgentID = agentID

	node.ServiceType = constant.ServiceType[service]

	// 通过method id 查询method
	method := misc.GetMethodByID(appName, methodID)
	node.Class, node.Method = misc.SplitMethod(method)
	node.MethodID = methodID
}

func (node *TraceTreeNode) setDepth(spanDepth int, treeDepth int) {
	node.SpanDepth = spanDepth
	node.Depth = treeDepth
}

// 若nid 为 p-0-1，则当前node的ID为p-0-2
func (node *TraceTreeNode) setNeighborID(s string) {
	sep := strings.LastIndex(s, "-")
	s2 := s[sep+1:]
	i, _ := strconv.Atoi(s2)
	i = i + 1
	node.ID = s[:sep+1] + strconv.Itoa(i)
}

// 若传入父id为p-0-1，则设置id为p-0-1-0
func (node *TraceTreeNode) setChildID(s string) {
	node.ID = s + "-0"
}

// 解析annotations，转换为span tag
func (node *TraceTreeNode) setTags(tags []*TempTag) {
	for _, tag := range tags {
		if (tag.Key == constant.STRING_ID) || (tag.Key <= constant.CACHE_ARGS0 && tag.Key >= constant.CACHE_ARGSN) {
			// 添加method_id : method的tag
			methodID := int(tag.Value.IntValue)
			method := misc.GetMethodByID(node.AppName, methodID)
			stag := &TraceTag{constant.AnnotationKeys[tag.Key], method}
			node.tags = append(node.tags, stag)
		} else if tag.Key == constant.SQL_ID {
			// {"key":20,"value":{"intStringStringValue":{"intValue":1,"stringValue1":"0","stringValue2":"testC, testC, 2019-04-15 08:43:03.713, null, null, testCNickName, testC_64b3def7-1a76-4ed7-bf21-67f5afc440fc, E10ADC3949BA59ABBE56E057F20F883E, null, 0"}}}
			sqlID := int(tag.Value.IntStringStringValue.IntValue)
			// 添加sqlID: sql的tag
			stag1 := &TraceTag{constant.AnnotationKeys[tag.Key], misc.GetSqlByID(node.AppName, sqlID)}
			// 添加sql bind value的tag
			stag2 := &TraceTag{constant.AnnotationKeys[constant.SQL_BINDVALUE], tag.Value.IntStringStringValue.StringValue2}
			node.tags = append(node.tags, stag1, stag2)
		} else {
			var stag *TraceTag
			switch tag.Key {
			case constant.HTTP_STATUS_CODE, constant.JSON_LIB_ANNOTATION_KEY_JSON_LENGTH, constant.GSON_ANNOTATION_KEY_JSON_LENGTH, constant.JACKSON_ANNOTATION_KEY_LENGTH_VALUE, constant.DUBBO_STATUS_ANNOTATION_KEY:
				stag = &TraceTag{constant.AnnotationKeys[tag.Key], strconv.Itoa(int(tag.Value.IntValue))}
			case constant.REDIS_IO:
				v := fmt.Sprintf("write=%d,read=%d", tag.Value.IntBooleanIntBooleanValue.IntValue1, tag.Value.IntBooleanIntBooleanValue.IntValue2)
				stag = &TraceTag{constant.AnnotationKeys[tag.Key], v}
			case constant.RETURN_DATA:
				v := strconv.FormatBool(tag.Value.BoolValue)
				stag = &TraceTag{constant.AnnotationKeys[tag.Key], v}
			case constant.HTTP_IO:
				if tag.Value.StringValue != "" {
					stag = &TraceTag{constant.AnnotationKeys[tag.Key], tag.Value.StringValue}
				} else {
					v := fmt.Sprintf("write=%d,read=%d", tag.Value.IntBooleanIntBooleanValue.IntValue1, tag.Value.IntBooleanIntBooleanValue.IntValue2)
					stag = &TraceTag{constant.AnnotationKeys[tag.Key], v}
				}
			default:
				if constant.AnnotationKeys[tag.Key] == "" {
					g.L.Info("invalid tag key", zap.Int("key", tag.Key))
					// continue
				}

				v := tag.Value.StringValue
				if v == "" {
					g.L.Info("invalid tag value", zap.Int("key", tag.Key), zap.Any("val", tag.Value))
					stag = &TraceTag{constant.AnnotationKeys[tag.Key], fmt.Sprintf("%v", v)}
					// continue
				} else {
					stag = &TraceTag{constant.AnnotationKeys[tag.Key], v}
				}
			}

			node.tags = append(node.tags, stag)
		}
	}
}

type TraceTree []*TraceTreeNode

func (tree *TraceTree) addSpan(span *traceSpan, spans *traceSpans) {
	n := &TraceTreeNode{}
	n.init(span.id, span.appName, span.agentID, span.api, span.serviceType, span.methodID)
	n.setTags(span.annotations)
	n.Duration = span.duration
	// span本身一定是http/dubbo/rpc服务的入口，因此要做特殊标示
	n.Icon = "hand"
	n.Type = "span"
	//remote addr -> tag
	n.tags = append(n.tags, &TraceTag{"remote_addr", span.remoteAddr})

	//@test
	n.Sequence = 0

	//set node id
	if len(*tree) == 0 {
		n.setDepth(0, 0)
		// 第一个span也是第一个node
		n.ID = "p-0"
	} else {
		// 找到父节点
		// 若上一个节点是父节点
		lastn := (*tree)[len(*tree)-1]
		if lastn.spanID == span.pid {
			// 若上一个节点的depth是-1，那该span是它的兄弟节点
			// 否则，该span是它的子节点
			if lastn.SpanDepth == -1 {
				n.setDepth(0, lastn.Depth)
				n.setNeighborID(lastn.ID)
			} else {
				n.setDepth(0, lastn.Depth+1)
				n.setChildID(lastn.ID)
			}
		} else {
			// 找到上一个根span，该span是他的兄弟span
			for i := len(*tree) - 1; i >= 0; i-- {
				lastn1 := (*tree)[i]
				if lastn1.Depth == 0 {
					n.setDepth(0, 0)
					n.setNeighborID(lastn1.ID)
					break
				}
			}
		}
	}

	*tree = append(*tree, n)

	// tags -> node
	for _, tag := range n.tags {
		en := &TraceTreeNode{
			Type: "tag",
		}
		en.spanID = span.id
		en.setDepth(-1, n.Depth+1)
		en.setChildID(n.ID)

		en.Params = tag.Value
		// 获取exception id
		en.Method = tag.Key

		en.Duration = -1
		en.Icon = "info"
		*tree = append(*tree, en)
	}

	// 处理span的event
	for _, event := range span.events {
		tree.addEvent(event, span, spans)
	}
}

func (tree *TraceTree) addEvent(event *SpanEvent, span *traceSpan, spans *traceSpans) {
	n := &TraceTreeNode{}
	n.init(span.id, span.appName, span.agentID, event.DestinationID, event.ServiceType, event.MethodID)
	n.DID = event.DestinationID
	n.NID = strconv.FormatInt(event.NextSpanID, 10)
	n.setTags(event.Annotations)
	n.Duration = event.EndElapsed
	n.Type = "event"

	//@test
	n.Sequence = event.Sequence

	// 若当前event的span depth为-1，则该event为叶子node
	//     我们要找到上一个不是叶子的节点，然后把该event作为该节点的最新的叶子node
	// 若当前event的span depth不为-1
	//     我们要找到depth-1的节点，然后该节点是当前event的父节点
	if event.Depth == -1 {
		var lastn *TraceTreeNode
		// 找到第一个类型不为tag的node
		for i := len(*tree) - 1; i >= 0; i-- {
			if (*tree)[i].Type != "tag" {
				lastn = (*tree)[i]
				break
			}
		}
		if lastn.SpanDepth == -1 {
			// 上一个节点也是叶子节点
			// 因此该event是上一个节点的邻节点
			n.setDepth(event.Depth, lastn.Depth)
			// lastn的邻节点，因此id + 1: 例如p-0-1 -> p-0-2
			n.setNeighborID(lastn.ID)
		} else {
			// 当前的event是上一个节点的子节点
			n.setDepth(event.Depth, lastn.Depth+1)
			n.setChildID(lastn.ID)
		}
	} else {
		// 寻找该event的兄弟节点或者父节点
		for i := len(*tree) - 1; i >= 0; i-- {
			// 先寻找兄弟节点：span id相同depth相同
			if ((*tree)[i].SpanDepth == event.Depth) && ((*tree)[i].spanID == span.id) {
				n.setDepth(event.Depth, (*tree)[i].Depth)
				n.setNeighborID((*tree)[i].ID)
				break
			}
			// 再寻找父节点
			//@bug,若depth不连续，此处就寻找不到父节点，因此需要继续向上寻找
			if ((*tree)[i].SpanDepth == event.Depth-1) && ((*tree)[i].spanID == span.id) {
				n.setDepth(event.Depth, (*tree)[i].Depth+1)
				n.setChildID((*tree)[i].ID)
				break
			}
		}
	}

	*tree = append(*tree, n)

	// 将exception转为当前event node的叶子node
	// 叶子节点的span depth = -1
	if event.ExceptionInfo != nil {
		en := &TraceTreeNode{}
		en.setDepth(-1, n.Depth+1)
		en.setChildID(n.ID)
		en.spanID = span.id
		en.Params = event.ExceptionInfo.StringValue
		// 获取exception id
		en.Method = misc.GetExceptionByID(n.AppName, int(event.ExceptionInfo.IntValue))
		en.IsError = true
		en.Duration = -1
		en.Icon = "bug"
		en.Type = "tag"
		*tree = append(*tree, en)
	}

	// tags -> node
	for _, tag := range n.tags {
		en := &TraceTreeNode{}
		en.setDepth(-1, n.Depth+1)
		en.setChildID(n.ID)
		en.Type = "tag"
		en.Params = tag.Value
		// 获取exception id
		en.Method = tag.Key
		en.spanID = span.id
		en.Duration = -1
		en.Icon = "info"
		*tree = append(*tree, en)
	}

	// 若next span id 存在，需要接着上一个event，来排放next span
	if event.NextSpanID != -1 {
		for _, span := range *spans {
			if span.id == event.NextSpanID {
				// 添加span
				tree.addSpan(span, spans)

				span.dealed = true
			}
		}
	}
}

// 标签，原数据为annotations，统一转换为tag
type TraceTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// 原数据annotations使用的格式
type TempTag struct {
	Key   int       `json:"key"`
	Value *TagValue `json:"value"`
}

type TagValue struct {
	StringValue                   string                         `json:"stringValue,omitempty"`
	BoolValue                     bool                           `json:"boolValue,omitempty"`
	IntValue                      int32                          `json:"intValue,omitempty"`
	LongValue                     int64                          `json:"longValue,omitempty"`
	ShortValue                    int16                          `json:"shortValue,omitempty"`
	DoubleValue                   float64                        `json:"doubleValue,omitempty"`
	BinaryValue                   []byte                         `json:"binaryValue,omitempty"`
	ByteValue                     int8                           `json:"byteValue,omitempty"`
	IntStringValue                *IntStringValue                `json:"intStringValue,omitempty"`
	IntStringStringValue          *IntStringStringValue          `json:"intStringStringValue,omitempty"`
	LongIntIntByteByteStringValue *LongIntIntByteByteStringValue `json:"longIntIntByteByteStringValue,omitempty"`
	IntBooleanIntBooleanValue     *IntBooleanIntBooleanValue     `json:"intBooleanIntBooleanValue,omitempty"`
}

type IntStringValue struct {
	IntValue    int32  `json:"intValue"`
	StringValue string `json:"stringValue,omitempty"`
}

type IntStringStringValue struct {
	IntValue     int32  `json:"intValue"`
	StringValue1 string `json:"stringValue1,omitempty"`
	StringValue2 string `json:"stringValue2,omitempty"`
}

type LongIntIntByteByteStringValue struct {
	LongValue   int64  `json:"longValue"`
	IntValue1   int32  `json:"intValue1"`
	IntValue2   int32  `json:"intValue2,omitempty"`
	ByteValue1  int8   `json:"byteValue1,omitempty"`
	ByteValue2  int8   `json:"byteValue2,omitempty"`
	StringValue string `json:"stringValue,omitempty"`
}

type IntBooleanIntBooleanValue struct {
	IntValue1  int32 `json:"intValue1"`
	BoolValue1 bool  `json:"boolValue1"`
	IntValue2  int32 `json:"intValue2"`
	BoolValue2 bool  `json:"boolValue2"`
}

type SpanEvent struct {
	Sequence      int             `json:"sequence"`
	StartElapsed  int             `json:"startElapsed"`
	EndElapsed    int             `json:"endElapsed"`
	ServiceType   int             `json:"serviceType"`
	EndPoint      string          `json:"endPoint"`
	Annotations   []*TempTag      `json:"annotations"`
	Depth         int             `json:"depth"`
	NextSpanID    int64           `json:"nextSpanId"`
	DestinationID string          `json:"destinationId"`
	MethodID      int             `json:"apiId"`
	ExceptionInfo *IntStringValue `json:"exceptionInfo"`
}
type SpanEvents []*SpanEvent

func (o SpanEvents) Len() int {
	return len(o)
}

func (o SpanEvents) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o SpanEvents) Less(i, j int) bool {
	return o[i].Sequence < o[j].Sequence
}
