package app

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type Stat struct {
	Name           string  `json:"name"`
	Count          int     `json:"count"`
	Apdex          float64 `json:"apdex"`
	AverageElapsed float64 `json:"average_elapsed"`
	ErrorPercent   float64 `json:"error_percent"`
	ExPercent      int     `json:"ex_percent"`
	totalElapsed   float64

	errCount     float64
	exCount      int
	satisfaction float64
	tolerate     float64

	Alive   int `json:"alive"`   // 存活节点数量
	Unalive int `json:"unalive"` // 不存活节点数量
}

// 获取用户的应用设定
func UserSetting(user string) (int, []string) {
	q := misc.StaticCql.Query(`SELECT app_show,app_names FROM account WHERE id=?`, user)
	var appShow int
	var appNameS string
	err := q.Scan(&appShow, &appNameS)
	if err != nil {
		return 1, nil
	}

	appNames := make([]string, 0)
	err = json.Unmarshal([]byte(appNameS), &appNames)
	if err != nil {
		return 1, nil
	}

	return appShow, appNames
}

// 查询应用底下的所有APi
func QueryApis(c echo.Context) error {
	appName := c.FormValue("app_name")
	if appName == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	q := `SELECT api FROM app_apis WHERE app_name=?`
	iter := misc.StaticCql.Query(q, appName).Iter()

	var api string
	apis := make([]string, 0)
	for iter.Scan(&api) {
		apis = append(apis, api)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   apis,
	})
}

func QueryAll(c echo.Context) error {
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   allAppNames(),
	})
}

func QueryAllWithSetting(c echo.Context) error {
	li := session.GetLoginInfo(c)
	appShow, appNames := UserSetting(li.ID)

	ans := make([]string, 0)
	if appShow == 1 { // 显示全部应用
		ans = allAppNames()
	} else {
		ans = appNames
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   ans,
	})
}

func allAppNames() []string {
	q := misc.StaticCql.Query(`SELECT app_name FROM apps `)
	iter := q.Iter()

	appNames := make([]string, 0)
	var appName string
	for iter.Scan(&appName) {
		appNames = append(appNames, appName)
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	sort.Strings(appNames)
	return appNames
}
