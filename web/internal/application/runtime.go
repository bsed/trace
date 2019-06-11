package app

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/pkg/stats"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

func QueryAgents(c echo.Context) error {
	appName := c.FormValue("app_name")
	agents, err := queryAgents(appName)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   agents,
	})
}

type jvmMetric struct {
	cpu            float64
	scpu           float64
	heap           int64
	heapMax        int64
	permgen        float64
	fullgcDuration int64
	fullgcCount    int64
	date           int64
}

type jvmMetrics []jvmMetric

func (a jvmMetrics) Len() int { // 重写 Len() 方法
	return len(a)
}
func (a jvmMetrics) Swap(i, j int) { // 重写 Swap() 方法
	a[i], a[j] = a[j], a[i]
}
func (a jvmMetrics) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
	return a[i].date < a[j].date
}

type RuntimeResult struct {
	Timeline           []string  `json:"timeline"`
	JvmCpuList         []float64 `json:"jvm_cpu_list"`
	SysCpuList         []float64 `json:"sys_cpu_list"`
	JvmHeapList        []int64   `json:"jvm_heap_list"`
	HeapMaxList        []int64   `json:"heap_max_list"`
	FullgcCountList    []int64   `json:"fullgc_count_list"`
	FullgcDurationList []int64   `json:"fullgc_duration_list"`
}

func RuntimeDashboard(c echo.Context) error {
	appName := c.FormValue("app_name")
	agentID := c.FormValue("agent_id")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}
	data, err := dashData(appName, agentID, start.Unix(), end.Unix())
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   data,
	})
}

func RuntimeDashByUnixTime(c echo.Context) error {
	appName := c.FormValue("app_name")
	agentID := c.FormValue("agent_id")
	start, _ := strconv.ParseFloat(c.FormValue("start"), 64)
	end, _ := strconv.ParseFloat(c.FormValue("end"), 64)
	if start == 0 || end == 0 {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	data, err := dashData(appName, agentID, int64(start), int64(end))
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   data,
	})
}

func dashData(appName, agentID string, start, end int64) (*RuntimeResult, error) {
	q := misc.TraceCql.Query(`SELECT input_date,metrics  FROM agent_runtime WHERE app_name = ?  and agent_id = ? and input_date > ? and input_date < ? `, appName, agentID, start, end)
	iter := q.Iter()

	var ms jvmMetrics
	var metrics string
	var inputDate int64
	for iter.Scan(&inputDate, &metrics) {
		m := &stats.JVMInfo{}
		json.Unmarshal([]byte(metrics), &m)

		ms = append(ms, jvmMetric{m.CPULoad.Jvm, m.CPULoad.System, m.GC.HeapUsed, m.GC.HeapMax, m.GC.JvmPoolPermGenUsed, m.GC.JvmGcOldTime, m.GC.GcOldCount, inputDate})
	}

	sort.Sort(ms)

	var timeline []string
	var jvmCPUList []float64
	var sysCPUList []float64
	var jvmHeapList []int64
	var heapMaxList []int64
	var fullgcCountList []int64
	var fullgcDurationList []int64
	// var jvmPermgenList []float64
	for _, m := range ms {
		timeline = append(timeline, misc.TimeToChartString1(time.Unix(m.date, 0)))
		jvmCPUList = append(jvmCPUList, utils.DecimalPrecision(m.cpu*100))  // 百分比
		sysCPUList = append(sysCPUList, utils.DecimalPrecision(m.scpu*100)) // 百分比
		jvmHeapList = append(jvmHeapList, m.heap/(1024*1024))               // 字节 - > MB
		heapMaxList = append(heapMaxList, m.heapMax/(1024*1024))            // 字节 - > MB
		fullgcCountList = append(fullgcCountList, m.fullgcCount)
		fullgcDurationList = append(fullgcDurationList, m.fullgcDuration)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return nil, err
	}

	return &RuntimeResult{timeline, jvmCPUList, sysCPUList, jvmHeapList, heapMaxList, fullgcCountList, fullgcDurationList}, nil
}
