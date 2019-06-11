package alerts

import (
	"net/http"
	"strconv"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g/utils"
	"go.uber.org/zap"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/pkg/util"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
)

type AlertHistory struct {
	ID        string   `json:"id"`
	Type      int      `json:"tp"`
	AppName   string   `json:"app_name"`
	Channel   string   `json:"channel"`
	InputDate string   `json:"input_date"`
	SqlID     int      `json:"sql_id"`
	API       string   `json:"api"`
	Alert     string   `json:"alert"`
	Value     float64  `json:"value"`
	Users     []string `json:"users"`
}

func History(c echo.Context) error {
	// 获取当前用户
	// li := session.GetLoginInfo(c)

	// 若该用户是管理员，可以获取所有组
	// var iter *gocql.Iter
	// if li.Priv == g.PRIV_NORMAL {
	// 	iter = misc.Cql.Query(`SELECT id,name,owner,alerts,update_date FROM alerts_policy WHERE owner=?`, li.ID).Iter()
	// } else {
	// 	iter = misc.Cql.Query(`SELECT id,name,owner,alerts,update_date FROM alerts_policy`).Iter()
	// }

	offset, _ := strconv.Atoi(c.FormValue("id"))
	limit, _ := strconv.Atoi(c.FormValue("limit"))
	if limit == 0 {
		limit = 2000
	}
	var q *gocql.Query
	if offset == 0 {
		q = misc.TraceCql.Query(`SELECT id,type,app_name,api,sql,alert_value,channel,users,input_date,alert FROM alert_history where const_id=1 limit ?`, limit)
	} else {
		q = misc.TraceCql.Query(`SELECT id,type,app_name,api,sql,alert_value,channel,users,input_date,alert FROM alert_history where token(const_id)=token(1) and id<? limit ? ALLOW FILTERING`, offset, limit)
	}

	var id, appName, channel, api string
	var inputDate int64
	var sqlID, tp int
	var alertValue float64
	var users []string
	alert := &util.Alert{}
	ah := make([]*AlertHistory, 0)

	iter := q.Iter()
	for iter.Scan(&id, &tp, &appName, &api, &sqlID, &alertValue, &channel, &users, &inputDate, &alert) {
		ah = append(ah, &AlertHistory{id, tp, appName, channel, utils.UnixToTimestring(inputDate), sqlID, api, alert.Name, utils.DecimalPrecision(alertValue), users})
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ah,
	})
}

func AppHistory(c echo.Context) error {
	appName := c.FormValue("app_name")
	limit, _ := strconv.Atoi(c.FormValue("limit"))
	if limit == 0 {
		limit = 5
	}

	q := misc.TraceCql.Query(`SELECT id,type,api,sql,alert_value,input_date,alert FROM alert_history where const_id=1 and app_name=? limit ?`, appName, limit)
	var id, api string
	var inputDate int64
	var sqlID, tp int
	var alertValue float64
	alert := &util.Alert{}
	ah := make([]*AlertHistory, 0)

	iter := q.Iter()
	for iter.Scan(&id, &tp, &api, &sqlID, &alertValue, &inputDate, &alert) {
		ah = append(ah, &AlertHistory{id, tp, appName, "", utils.UnixToTimestring(inputDate), sqlID, api, alert.Name, utils.DecimalPrecision(alertValue), nil})
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ah,
	})
}
