package app

/* 应用首页列表 */
import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

func List(c echo.Context) error {
	napps := make([]*Stat, 0)

	now := time.Now()
	// 查询缓存数据是否存在和过期
	// if web.cache.appList == nil || now.Sub(web.cache.appListUpdate).Seconds() > CacheUpdateIntv {
	// 取过去30分钟的数据
	start := now.Unix() - 30*60
	q := misc.TraceCql.Query(`SELECT app_name,total_elapsed,count,err_count,satisfaction,tolerate FROM api_stats WHERE input_date > ? and input_date < ? `, start, now.Unix())
	iter := q.Iter()

	apps := make(map[string]*Stat)
	var appName string
	var count int
	var tElapsed, errCount, satisfaction, tolerate int

	for iter.Scan(&appName, &tElapsed, &count, &errCount, &satisfaction, &tolerate) {
		app, ok := apps[appName]
		if !ok {
			apps[appName] = &Stat{
				Name:         appName,
				Count:        count,
				totalElapsed: float64(tElapsed),
				errCount:     float64(errCount),
				satisfaction: float64(satisfaction),
				tolerate:     float64(tolerate),
			}
		} else {
			app.Count += count
			app.totalElapsed += float64(tElapsed)
			app.errCount += float64(errCount)
			app.satisfaction += float64(satisfaction)
			app.tolerate += float64(tolerate)
		}
	}

	for _, app := range apps {
		app.ErrorPercent = 100 * utils.DecimalPrecision(app.errCount/float64(app.Count))
		app.AverageElapsed = utils.DecimalPrecision(app.totalElapsed / float64(app.Count))
		app.Apdex = utils.DecimalPrecision((app.satisfaction + app.tolerate/2) / float64(app.Count))
		napps = append(napps, app)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	// 	// 更新缓存
	// 	if len(napps) > 0 {
	// 		web.cache.appList = napps
	// 		web.cache.appListUpdate = now
	// 	}
	// } else {
	// 	napps = web.cache.appList
	// 	fmt.Println("query from cache:", napps)
	// }

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   napps,
	})
}

func ListWithSetting(c echo.Context) error {
	li := session.GetLoginInfo(c)
	appShow, appNames := UserSetting(li.ID)

	// 获取最近X分钟
	intv, _ := strconv.ParseInt(c.FormValue("start"), 10, 64)

	// 获取app stat
	stats := appList(intv, appShow, appNames)

	// 获取JVM异常统计
	exceptions := AllExceptionStats(intv)

	for _, stat := range stats {
		e, ok := exceptions[stat.Name]
		if ok {
			if stat.Count != 0 {
				stat.ExPercent = 100 * e / stat.Count
			}
		}
	}

	// 合并stats 和 jvm异常
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   stats,
	})
}

func appList(intv int64, appShow int, appNames []string) []*Stat {
	now := time.Now()
	start := now.Unix() - intv*60
	stats := make([]*Stat, 0)

	statsMap := make(map[string]*Stat)
	var q *gocql.Query
	var appName string
	var count int
	var tElapsed, errCount, satisfaction, tolerate int

	if appShow == 1 {
		q = misc.TraceCql.Query(`SELECT app_name,duration,count,err_count,satisfaction,tolerate FROM api_stats WHERE input_date > ? and input_date < ? `, start, now.Unix())
		iter := q.Iter()

		for iter.Scan(&appName, &tElapsed, &count, &errCount, &satisfaction, &tolerate) {
			app, ok := statsMap[appName]
			if !ok {
				statsMap[appName] = &Stat{
					Name:         appName,
					Count:        count,
					totalElapsed: float64(tElapsed),
					errCount:     float64(errCount),
					satisfaction: float64(satisfaction),
					tolerate:     float64(tolerate),
				}
			} else {
				app.Count += count
				app.totalElapsed += float64(tElapsed)
				app.errCount += float64(errCount)
				app.satisfaction += float64(satisfaction)
				app.tolerate += float64(tolerate)
			}
		}

		if err := iter.Close(); err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		}
	} else {
		for _, an := range appNames {
			err := misc.TraceCql.Query(`SELECT app_name,total_elapsed,count,err_count,satisfaction,tolerate FROM api_stats WHERE app_name =? and input_date > ? and input_date < ? `, an, start, now.Unix()).Scan(&appName, &tElapsed, &count, &errCount, &satisfaction, &tolerate)
			if err != nil {
				log.Println("select app stats error:", err)
			}

			app, ok := statsMap[appName]
			if !ok {
				statsMap[appName] = &Stat{
					Name:         appName,
					Count:        count,
					totalElapsed: float64(tElapsed),
					errCount:     float64(errCount),
					satisfaction: float64(satisfaction),
					tolerate:     float64(tolerate),
				}
			} else {
				app.Count += count
				app.totalElapsed += float64(tElapsed)
				app.errCount += float64(errCount)
				app.satisfaction += float64(satisfaction)
				app.tolerate += float64(tolerate)
			}
		}

	}

	for _, stat := range statsMap {
		stat.ErrorPercent = utils.DecimalPrecision(100 * stat.errCount / float64(stat.Count))
		stat.AverageElapsed = utils.DecimalPrecision(stat.totalElapsed / float64(stat.Count))
		stat.Apdex = utils.DecimalPrecision((stat.satisfaction + stat.tolerate/2) / float64(stat.Count))
		stats = append(stats, stat)
	}

	// 获取所有应用，不在之前列表的应用，所有数据置为0
	allApps := allAppNames()
	for _, name := range allApps {
		_, ok := statsMap[name]
		if !ok {
			stat := &Stat{
				Name:  name,
				Apdex: 1,
			}
			stats = append(stats, stat)
		}
	}

	// 获取应用的服务器节点状态
	alive, unalive := countAgentsAlive()
	for _, stat := range stats {
		stat.Alive = alive[stat.Name]
		stat.Unalive = unalive[stat.Name]
	}

	return stats
}
