package app

/* SQL统计 */

import (
	"net/http"
	"sort"
	"time"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type SqlStat struct {
	ID             int     `json:"id"`
	SQL            string  `json:"sql"`
	MaxElapsed     int     `json:"max_elapsed"`
	MinElapsed     int     `json:"min_elapsed"`
	Count          int     `json:"count"`
	AverageElapsed float64 `json:"average_elapsed"`
	ErrorCount     int     `json:"error_count"`

	inputDate int64
}

type SqlStatList []*SqlStat

func (a SqlStatList) Len() int { // 重写 Len() 方法
	return len(a)
}
func (a SqlStatList) Swap(i, j int) { // 重写 Swap() 方法
	a[i], a[j] = a[j], a[i]
}
func (a SqlStatList) Less(i, j int) bool { // 重写 Less() 方法， 从大到小排序
	return a[i].inputDate < a[j].inputDate
}

func SqlStats(c echo.Context) error {
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	appName := c.FormValue("app_name")
	if appName == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	q := misc.TraceCql.Query(`SELECT sql,max_elapsed,min_elapsed,elapsed,count,err_count FROM sql_stats WHERE app_name = ? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
	iter := q.Iter()

	var sqlID, maxE, minE, count, errCount, elapsed int
	ad := make(map[int]*SqlStat)
	for iter.Scan(&sqlID, &maxE, &minE, &elapsed, &count, &errCount) {
		am, ok := ad[sqlID]
		if !ok {
			ad[sqlID] = &SqlStat{sqlID, "", maxE, minE, count, utils.DecimalPrecision(float64(elapsed / count)), errCount, 0}
		} else {
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
			am.AverageElapsed, _ = utils.DecimalPrecision((am.AverageElapsed*float64(am.Count) + float64(elapsed)) / float64((am.Count + count)))
		}
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	ads := make([]*SqlStat, 0, len(ad))
	for _, am := range ad {
		am.SQL = misc.GetSqlByID(appName, am.ID)
		ads = append(ads, am)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ads,
	})
}

type SqlDashResult struct {
	Suc         bool      `json:"suc"` //是否有数据
	Timeline    []string  `json:"timeline"`
	CountList   []int     `json:"count_list"`
	ElapsedList []float64 `json:"elapsed_list"`
	ApdexList   []float64 `json:"apdex_list"`
	ErrorList   []int     `json:"error_list"`
}

func SqlDashboard(c echo.Context) error {
	appName := c.FormValue("app_name")
	sqlID := c.FormValue("sql_id")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	// 把start-end分为30个桶
	timeline := make([]string, 0)

	// 读取相应数据，按照时间填到对应的桶中
	q := misc.TraceCql.Query(`SELECT elapsed,count,err_count,input_date FROM sql_stats WHERE app_name = ?  and sql = ? and input_date > ? and input_date < ?`, appName, sqlID, start.Unix(), end.Unix())
	iter := q.Iter()

	// apps := make(map[string]*AppStat)
	var count int
	var tElapsed, errCount int
	var inputDate int64
	// 把结果数据按照时间点顺序存放
	//请求次数列表
	countList := make([]int, 0)
	//耗时列表
	elapsedList := make([]float64, 0)
	//错误率列表
	errorList := make([]int, 0)

	sl := make(SqlStatList, 0)
	for iter.Scan(&tElapsed, &count, &errCount, &inputDate) {
		sl = append(sl, &SqlStat{
			Count:          count,
			ErrorCount:     errCount,
			AverageElapsed: utils.DecimalPrecision(float64(tElapsed) / float64(count)),
			inputDate:      inputDate,
		})
		// ts :=
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	sort.Sort(sl)

	for _, s := range sl {
		timeline = append(timeline, misc.TimeToChartString(time.Unix(s.inputDate, 0)))
		countList = append(countList, s.Count)
		errorList = append(errorList, s.ErrorCount)
		elapsedList = append(elapsedList, s.AverageElapsed)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data: SqlDashResult{
			Timeline:    timeline,
			CountList:   countList,
			ElapsedList: elapsedList,
			ErrorList:   errorList,
		},
	})
}
