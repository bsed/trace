package agentd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/imdevlab/g"
	"github.com/imdevlab/g/utils"
	"go.uber.org/zap"

	"github.com/bsed/trace/pkg/util"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
)

var agentInfos = make(map[string]*util.AgentdInfo)

// 获取agentd上报的信息，同时返回最新的版本号
func Update(c echo.Context) error {
	info := &util.AgentdInfo{}
	fmt.Println(c.FormValue("info"))
	json.Unmarshal(utils.String2Bytes(c.FormValue("info")), &info)

	q := misc.TraceCql.Query(`UPDATE  agentd_info  SET start_time=?,download_times=?,restart_times=?,last_download_time=?,last_restart_time=?,agentd_version=?,agent_version=?,download_version=?,admin_port=?,input_date=? WHERE hostname=?`,
		info.StartTime, info.DownloadTimes, info.RestartTimes, info.LastDownloadTime, info.LastRestartTime, info.AgentdVersion, info.AgentVersion, info.DownloadVersion, info.AdminPort, info.UpdateTime, info.Hostname)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	agentInfos[info.Hostname] = info

	return c.String(http.StatusOK, agentVersion)
}

// 删除条目
func Delete(c echo.Context) error {
	li := session.GetLoginInfo(c)
	// 判断当前用户是否超级管理员
	if li.Priv != g.PRIV_SUPER_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusForbidden,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	hn := c.FormValue("hostname")

	q := misc.TraceCql.Query(`DELETE FROM  agentd_info WHERE hostname=?`, hn)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}

	delete(agentInfos, hn)
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

// 重启指定机器的agent
func Restart(c echo.Context) error {
	li := session.GetLoginInfo(c)
	// 判断当前用户是否超级管理员
	if li.Priv != g.PRIV_SUPER_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusForbidden,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	hn := c.FormValue("hostname")
	info, ok := agentInfos[hn]
	if !ok {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusNotFound,
			Message: g.NotExistE,
			ErrCode: g.NotExistC,
		})
	}

	url := "http://" + info.Hostname + ":" + info.AdminPort + "/admin/restart"
	code, _, err := g.Cli.Get(nil, url)

	if code != 200 || err != nil {
		g.L.Warn("request to restart failed", zap.Error(err), zap.Int("code", code), zap.String("url", url))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			Message: g.ReqFailedE,
			ErrCode: g.ReqFailedC,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

// 重置指定机器的agent：重新下载
func Reset(c echo.Context) error {
	li := session.GetLoginInfo(c)
	// 判断当前用户是否超级管理员
	if li.Priv != g.PRIV_SUPER_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusForbidden,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	hn := c.FormValue("hostname")
	info, ok := agentInfos[hn]
	if !ok {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusNotFound,
			Message: g.NotExistE,
			ErrCode: g.NotExistC,
		})
	}

	url := "http://" + info.Hostname + ":" + info.AdminPort + "/admin/reset"
	code, _, err := g.Cli.Get(nil, url)

	if code != 200 || err != nil {
		g.L.Warn("request to reset failed", zap.Error(err), zap.Int("code", code), zap.String("url", url))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			Message: g.ReqFailedE,
			ErrCode: g.ReqFailedC,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

// 获取所有的agentd信息
func Info(c echo.Context) error {
	agents := make([]*util.AgentdInfo, 0, len(agentInfos))
	for _, agent := range agentInfos {
		agents = append(agents, agent)
	}
	return c.JSON(http.StatusOK, g.Result{
		Status:  http.StatusOK,
		Data:    agents,
		Version: agentVersion,
	})
}
func Start() {
	// 加载所有agentd info 到内存中
	q := misc.TraceCql.Query(`SELECT hostname,start_time,download_times,restart_times,last_download_time,last_restart_time,agentd_version,agent_version,download_version,admin_port,input_date FROM agentd_info`)
	iter := q.Iter()

	var ht, adv, av, dv, ap string
	var dt, rt int
	var st, ldt, lrt, ut int64
	for iter.Scan(&ht, &st, &dt, &rt, &ldt, &lrt, &adv, &av, &dv, &ap, &ut) {
		agentInfos[ht] = &util.AgentdInfo{ht, st, dt, rt, ldt, lrt, adv, av, dv, ap, ut}
	}

	// 加载最新版本信息到内存中
	scanVersion()
}
