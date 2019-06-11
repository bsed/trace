package app

/* 应用Dashboard */
import (
	"math"
	"net/http"
	"sort"
	"time"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type DashResult struct {
	Suc         bool      `json:"suc"` //是否有数据
	Timeline    []string  `json:"timeline"`
	CountList   []int     `json:"count_list"`
	ElapsedList []float64 `json:"elapsed_list"`
	ApdexList   []float64 `json:"apdex_list"`
	ErrorList   []float64 `json:"error_list"`
	ExList      []int     `json:"ex_list"`
}

func Dashboard(c echo.Context) error {
	appName := c.FormValue("app_name")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	timeline, timeBucks, suc := dashboard(appName, start, end)

	// 把结果数据按照时间点顺序存放
	//请求次数列表
	countList := make([]int, 0)
	//耗时列表
	elapsedList := make([]float64, 0)
	//apdex列表
	apdexList := make([]float64, 0)
	//错误率列表
	errorList := make([]float64, 0)
	//异常率列表
	exList := make([]int, 0)
	for _, ts := range timeline {
		app := timeBucks[ts]
		if math.IsNaN(app.AverageElapsed) {
			app.AverageElapsed = 0
		}
		if math.IsNaN(app.Apdex) {
			app.Apdex = 1
		}
		if math.IsNaN(app.ErrorPercent) {
			app.ErrorPercent = 0
		}

		countList = append(countList, app.Count)
		elapsedList = append(elapsedList, app.AverageElapsed)
		apdexList = append(apdexList, app.Apdex)
		errorList = append(errorList, app.ErrorPercent)
		exList = append(exList, app.ExPercent)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data: DashResult{
			Suc:         suc,
			Timeline:    timeline,
			CountList:   countList,
			ElapsedList: elapsedList,
			ApdexList:   apdexList,
			ErrorList:   errorList,
			ExList:      exList,
		},
	})
}

func dashboard(appName string, start, end time.Time) ([]string, map[string]*Stat, bool) {
	timeline := make([]string, 0)
	timeBucks := make(map[string]*Stat)
	suc := true
	// 小于等于60分钟的，一分钟一个点
	// 其他情况按照时间做聚合

	// 查询时间间隔要转换为30的倍数，然后按照倍数聚合相应的点，最终形成30个图绘制点
	//计算间隔
	intv := int(end.Sub(start).Minutes())

	// 180分钟之内不做数据聚合，保持原始数据
	if intv <= 180 {
		q := misc.TraceCql.Query(`SELECT duration,count,err_count,satisfaction,tolerate,input_date FROM api_stats WHERE app_name = ? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
		iter := q.Iter()

		if iter.NumRows() == 0 {
			suc = false
			return timeline, timeBucks, suc
		}

		// apps := make(map[string]*AppStat)
		var count int
		var tElapsed, errCount, satisfaction, tolerate int
		var inputDate int64
		for iter.Scan(&tElapsed, &count, &errCount, &satisfaction, &tolerate, &inputDate) {
			ts := misc.TimeToChartString2(time.Unix(inputDate, 0))
			timeline = append(timeline, ts)

			timeBucks[ts] = &Stat{
				Count:          count,
				ErrorPercent:   utils.DecimalPrecision(100 * (float64(errCount) / float64(count))),
				AverageElapsed: utils.DecimalPrecision(float64(tElapsed) / float64(count)),
				Apdex:          utils.DecimalPrecision((float64(satisfaction) + float64(tolerate)/2) / float64(count)),
			}

		}

		if err := iter.Close(); err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		}

		// 获取JVM异常率
		q1 := misc.TraceCql.Query(`SELECT count,input_date  FROM exception_stats WHERE app_name=? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
		iter1 := q1.Iter()

		var count1 int
		var inputDate1 int64
		for iter1.Scan(&count1, &inputDate1) {
			ts := misc.TimeToChartString2(time.Unix(inputDate1, 0))
			stat, ok := timeBucks[ts]
			if ok {
				stat.exCount += count1
			}
		}

		if err := iter1.Close(); err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
		}

		// 对timeline进行排序
		sort.Strings(timeline)

		for _, app := range timeBucks {
			if app.Count != 0 {
				app.ExPercent = 100 * app.exCount / app.Count
			}
		}
	} else {
		var step int

		// 把start-end分为30个桶
		if intv%30 != 0 {
			start = end.Add(-(time.Duration(intv/30+1)*30*time.Minute - time.Minute))
		} else {
			start = start.Add(time.Minute)
		}

		current := start
		if end.Sub(start).Minutes() <= 60 {
			step = 1
		} else {
			step = int(end.Sub(start).Minutes())/30 + 1
		}

		for {
			if current.Unix() > end.Unix() {
				break
			}
			cs := misc.TimeToChartString2(current)
			timeline = append(timeline, cs)
			timeBucks[cs] = &Stat{}
			current = current.Add(time.Duration(step) * time.Minute)
		}

		// 读取相应数据，按照时间填到对应的桶中
		q := misc.TraceCql.Query(`SELECT duration,count,err_count,satisfaction,tolerate,input_date FROM api_stats WHERE app_name = ? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
		iter := q.Iter()

		if iter.NumRows() == 0 {
			suc = false
			return timeline, timeBucks, suc
		}

		// apps := make(map[string]*AppStat)
		var count int
		var tElapsed, errCount, satisfaction, tolerate int
		var inputDate int64
		for iter.Scan(&tElapsed, &count, &errCount, &satisfaction, &tolerate, &inputDate) {
			t := time.Unix(inputDate, 0)
			// 计算该时间落在哪个时间桶里
			i := int(t.Sub(start).Minutes()) / step
			t1 := start.Add(time.Minute * time.Duration(i*step))

			ts := misc.TimeToChartString2(t1)
			app := timeBucks[ts]
			app.Count += count
			app.totalElapsed += float64(tElapsed)
			app.errCount += float64(errCount)
			app.satisfaction += float64(satisfaction)
			app.tolerate += float64(tolerate)
		}

		if err := iter.Close(); err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		}

		// 读取JVM异常数据，按照时间填到对应的桶中
		q1 := misc.TraceCql.Query(`SELECT count,input_date  FROM exception_stats WHERE app_name=? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
		iter1 := q1.Iter()

		// apps := make(map[string]*AppStat)
		var count1 int
		var inputDate1 int64
		for iter1.Scan(&count1, &inputDate1) {
			t := time.Unix(inputDate1, 0)
			// 计算该时间落在哪个时间桶里
			i := int(t.Sub(start).Minutes()) / step
			t1 := start.Add(time.Minute * time.Duration(i*step))

			ts := misc.TimeToChartString2(t1)
			app, ok := timeBucks[ts]
			if ok {
				app.exCount += count1
			}
		}

		if err := iter.Close(); err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		}

		// 对每个桶里的数据进行计算
		for _, app := range timeBucks {
			if app.Count != 0 {
				app.ExPercent = 100 * app.exCount / app.Count
			}
			app.ErrorPercent = utils.DecimalPrecision(100 * app.errCount / float64(app.Count))
			app.AverageElapsed = utils.DecimalPrecision(app.totalElapsed / float64(app.Count))
			app.Apdex = utils.DecimalPrecision((app.satisfaction + app.tolerate/2) / float64(app.Count))
			app.Count = app.Count / step
		}
	}

	return timeline, timeBucks, suc
}
