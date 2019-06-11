package app

/* 函数方法、关键事务统计 */
import (
	"net/http"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/pkg/constant"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

func Methods(c echo.Context) error {
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

	q := misc.TraceCql.Query(`SELECT method_id,api,service_type,elapsed,max_elapsed,min_elapsed,count,err_count FROM method_stats WHERE app_name = ? and input_date > ? and input_date < ? `, appName, start.Unix(), end.Unix())
	iter := q.Iter()

	var apiID, serType, elapsed, maxE, minE, count, errCount int
	var totalElapsed int
	var api string
	ad := make(map[int]*ApiMethod)
	for iter.Scan(&apiID, &api, &serType, &elapsed, &maxE, &minE, &count, &errCount) {
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

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ads,
	})
}
