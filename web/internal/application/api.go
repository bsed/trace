package app

/* 接口统计 */
import (
	"math"
	"net/http"
	"time"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type ApiStat struct {
	API            string  `json:"api"`
	MaxElapsed     int     `json:"max_elapsed"`
	MinElapsed     int     `json:"min_elapsed"`
	Count          int     `json:"count"`
	AverageElapsed float64 `json:"average_elapsed"`
	ErrorCount     int     `json:"error_count"`
}

// type ApiStats []*ApiStat

// func (a ApiStats) Len() int { // 重写 Len() 方法
// 	return len(a)
// }
// func (a ApiStats) Swap(i, j int) { // 重写 Swap() 方法
// 	a[i], a[j] = a[j], a[i]
// }
// func (a ApiStats) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
// 	return a[j].AverageElapsed < a[i].AverageElapsed
// }

// 单个应用下，所有接口的统计信息
func ApiStats(c echo.Context) error {
	appName := c.FormValue("app_name")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	// 读取相应数据，按照时间填到对应的桶中
	q := `SELECT api,max_duration,min_duration,duration, count,err_count FROM api_stats WHERE app_name = ? and input_date > ? and input_date < ? `
	iter := misc.TraceCql.Query(q, appName, start.Unix(), end.Unix()).Iter()

	// apps := make(map[string]*AppStat)
	var maxElapsed, minElapsed, count, errCount, elapsed int
	var api string
	ass := make(map[string]*ApiStat)
	for iter.Scan(&api, &maxElapsed, &minElapsed, &elapsed, &count, &errCount) {
		as, ok := ass[api]
		if !ok {
			ass[api] = &ApiStat{api, maxElapsed, minElapsed, count, utils.DecimalPrecision(float64(elapsed / count)), errCount}
		} else {
			// 取最大值
			if maxElapsed > as.MaxElapsed {
				as.MaxElapsed = maxElapsed
			}
			// 取最小值
			if minElapsed < as.MinElapsed {
				as.MinElapsed = minElapsed
			}

			as.Count += count
			as.ErrorCount += errCount
			// 平均 = 过去的平均 * 过去总次数  + 最新的平均 * 最新的次数/ (过去总次数 + 最新次数)
			as.AverageElapsed = utils.DecimalPrecision((as.AverageElapsed*float64(as.Count) + float64(elapsed)) / float64((as.Count + count)))
		}
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	// 对每个桶里的数据进行计算
	apiStats := make([]*ApiStat, 0, len(ass))
	for _, as := range ass {
		apiStats = append(apiStats, as)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   apiStats,
	})
}

type ApiMethod struct {
	ID             int     `json:"-"`
	API            string  `json:"api"`
	ServiceType    string  `json:"service_type"`
	RatioElapsed   int     `json:"ratio_elapsed"`
	Elapsed        int     `json:"elapsed"`
	MaxElapsed     int     `json:"max_elapsed"`
	MinElapsed     int     `json:"min_elapsed"`
	Count          int     `json:"count"`
	AverageElapsed float64 `json:"average_elapsed"`
	ErrorCount     int     `json:"error_count"`

	Method string `json:"method"`
	Class  string `json:"class"`
}

func ApiDetail(c echo.Context) error {
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	appName := c.FormValue("app_name")
	api := c.FormValue("api")

	if appName == "" || api == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	q := `SELECT method_id,service_type,elapsed,max_elapsed,min_elapsed,count,err_count FROM method_stats WHERE app_name = ? and api = ? and input_date > ? and input_date < ? `
	iter := misc.TraceCql.Query(q, appName, api, start.Unix(), end.Unix()).Iter()

	var apiID, serType, elapsed, maxE, minE, count, errCount int
	var totalElapsed int
	ad := make(map[int]*ApiMethod)
	for iter.Scan(&apiID, &serType, &elapsed, &maxE, &minE, &count, &errCount) {
		am, ok := ad[apiID]
		if !ok {
			ad[apiID] = &ApiMethod{apiID, api, constant.ServiceType[serType], 0, elapsed, maxE, minE, count, utils.DecimalPrecision(float64(elapsed / count)), errCount, "", ""}
		} else {
			am.Elapsed += elapsed
			// 取最大值
			if maxE > am.MaxElapsed {
				am.MaxElapsed = maxE
			}
			// 取最小值
			if minE < am.MinElapsed {
				am.MinElapsed = minE
			}

			am.Count += count
			am.ErrorCount += errCount
			// 平均 = 过去的平均 * 过去总次数  + 最新的平均 * 最新的次数/ (过去总次数 + 最新次数)
			am.AverageElapsed = utils.DecimalPrecision((am.AverageElapsed*float64(am.Count) + float64(elapsed)) / float64((am.Count + count)))
		}

		totalElapsed += elapsed
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	ads := make([]*ApiMethod, 0, len(ad))
	for _, am := range ad {
		// 计算耗时占比
		am.RatioElapsed = am.Elapsed * 100 / totalElapsed
		// 通过apiID 获取api name

		methodInfo := misc.GetMethodByID(appName, am.ID)
		class, method := misc.SplitMethod(methodInfo)
		am.Method = method
		am.Class = class

		ads = append(ads, am)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ads,
	})
}

func ApiDashboard(c echo.Context) error {
	appName := c.FormValue("app_name")
	api := c.FormValue("api")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	// 查询时间间隔要转换为30的倍数，然后按照倍数聚合相应的点，最终形成30个图绘制点
	//计算间隔
	intv := int(end.Sub(start).Minutes())

	if intv%30 != 0 {
		start = end.Add(-(time.Duration(intv/30+1)*30*time.Minute - time.Minute))
	} else {
		start = start.Add(time.Minute)
	}

	// 把start-end分为30个桶
	timeline := make([]string, 0)
	timeBucks := make(map[string]*Stat)
	current := start
	var step int
	if end.Sub(start).Minutes() <= 60 {
		step = 1
	} else {
		step = int(end.Sub(start).Minutes())/30 + 1
	}

	for {
		if current.Unix() > end.Unix() {
			break
		}
		cs := misc.TimeToChartString(current)
		timeline = append(timeline, cs)
		timeBucks[cs] = &Stat{}
		current = current.Add(time.Duration(step) * time.Minute)
	}

	// 读取相应数据，按照时间填到对应的桶中
	q := misc.TraceCql.Query(`SELECT duration,count,err_count,input_date FROM api_stats WHERE app_name = ?  and api = ? and input_date > ? and input_date < ? `, appName, api, start.Unix(), end.Unix())
	iter := q.Iter()

	// apps := make(map[string]*AppStat)
	var count int
	var tElapsed, errCount int
	var inputDate int64
	for iter.Scan(&tElapsed, &count, &errCount, &inputDate) {
		t := time.Unix(inputDate, 0)
		// 计算该时间落在哪个时间桶里
		i := int(t.Sub(start).Minutes()) / step
		t1 := start.Add(time.Minute * time.Duration(i*step))

		ts := misc.TimeToChartString(t1)
		app := timeBucks[ts]
		app.Count += count
		app.totalElapsed += float64(tElapsed)
		app.errCount += float64(errCount)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	// 对每个桶里的数据进行计算
	for _, app := range timeBucks {
		app.ErrorPercent = utils.DecimalPrecision(100 * app.errCount / float64(app.Count))
		app.AverageElapsed = utils.DecimalPrecision(app.totalElapsed / float64(app.Count))
		app.Count = app.Count / step
	}

	// 把结果数据按照时间点顺序存放
	//请求次数列表
	countList := make([]int, 0)
	//耗时列表
	elapsedList := make([]float64, 0)
	//错误率列表
	errorList := make([]float64, 0)

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
		errorList = append(errorList, app.ErrorPercent)

	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data: DashResult{
			Timeline:    timeline,
			CountList:   countList,
			ElapsedList: elapsedList,
			ErrorList:   errorList,
		},
	})
}
