package client

import (
	"net/http"

	"github.com/bsed/trace/agentd/misc"
	"github.com/labstack/echo"
)

func (cli *Client) initAdmin() {
	e := echo.New()
	// 重启agent
	e.GET("/admin/restart", cli.adminRestart)
	e.GET("/admin/reset", cli.adminReset)
	e.Logger.Fatal(e.Start(":" + misc.Conf.Client.AdminPort))
}

func (cli *Client) adminRestart(c echo.Context) error {
	cli.agentStopStart()
	return c.String(http.StatusOK, "success")
}

func (cli *Client) adminReset(c echo.Context) error {
	downloadVersion = ""
	return c.String(http.StatusOK, "success")
}
