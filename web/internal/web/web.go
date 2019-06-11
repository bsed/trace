package service

import (
	"net/http"
	"time"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/web/internal/admin"
	"github.com/bsed/trace/web/internal/agentd-server"
	"github.com/bsed/trace/web/internal/alerts"
	app "github.com/bsed/trace/web/internal/application"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"go.uber.org/zap"
)

// 后台服务
// Stats 离线计算
type Web struct {
}

// New ...
func New() *Web {
	return &Web{}
}

// Start ...
func (s *Web) Start() error {
	// 初始化Cql连接
	s.initCql()

	// 初始化超级管理员
	admin.InitSuperAdmin()

	// 初始化全体用户列表(缓存以提升性能)
	go admin.LoopLoadUsers()

	agentd.Start()

	go func() {
		e := echo.New()
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowHeaders:     append([]string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept}, "X-Token"),
			AllowCredentials: true,
		}))

		e.Pre(middleware.RemoveTrailingSlash())
		e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Level: 5}))

		// 回调相关
		//同步回调接口
		e.POST("/web/login", session.Login)
		e.POST("/web/logout", session.Logout)
		// 应用查询接口
		//查询应用列表
		e.GET("/web/appList", app.List, s.checkLogin)
		e.GET("/web/appListWithSetting", app.ListWithSetting, s.checkLogin)
		//获取指定应用的一段时间内的状态
		e.GET("/web/appDash", app.Dashboard, s.checkLogin)
		//查询所有的app名
		e.GET("/web/appNames", app.QueryAll)
		e.GET("/web/appNamesWithSetting", app.QueryAllWithSetting, s.checkLogin)

		// 查询APP底下的所有API
		e.GET("/web/appApis", app.QueryApis)

		// agent运行时图表
		e.GET("/web/runtimeDash", app.RuntimeDashboard)
		e.GET("/web/runtimeDashByUnixTime", app.RuntimeDashByUnixTime)

		//应用接口统计
		e.GET("/web/apiStats", app.ApiStats)
		//获取指定接口的详细方法统计
		e.GET("/web/apiDetail", app.ApiDetail)
		//获取接口的图表数据
		e.GET("/web/apiDash", app.ApiDashboard)

		// 应用Method统计
		e.GET("/web/appMethods", app.Methods)

		// 数据库统计
		e.GET("/web/sqlStats", app.SqlStats)
		e.GET("/web/sqlDash", app.SqlDashboard)

		// 异常统计
		e.GET("/web/appException", app.ExceptionStats)
		//获取异常的图表数据
		e.GET("/web/exceptionDash", app.ExceptionDashboard)

		//查询所有服务器名
		e.GET("/web/agentList", app.QueryAgents, s.checkLogin)

		// 应用拓扑图
		e.GET("/web/appServiceMap", app.QueryAPPServiceMap, s.checkLogin)
		// 应用地图
		e.GET("/web/serviceMap", app.QueryServiceMap, s.checkLogin)
		// API拓扑图
		e.GET("/web/apiMap", app.QueryApiMap, s.checkLogin)

		// 链路查询
		e.GET("/web/queryTraces", app.QueryTraces, s.checkLogin)
		e.GET("/web/trace", app.QueryTrace, s.checkLogin)

		// 告警平台
		e.POST("/web/createGroup", alerts.CreateGroup, s.checkLogin)
		e.POST("/web/editGroup", alerts.EditGroup, s.checkLogin)
		e.POST("/web/deleteGroup", alerts.DeleteGroup, s.checkLogin)
		e.GET("/web/queryGroups", alerts.QueryGroups)

		e.POST("/web/createPolicy", alerts.CreatePolicy, s.checkLogin)
		e.POST("/web/editPolicy", alerts.EditPolicy, s.checkLogin)
		e.GET("/web/queryPolicies", alerts.QueryPolicies, s.checkLogin)
		e.GET("/web/queryPolicy", alerts.QueryPolicy)
		e.POST("/web/deletePolicy", alerts.DeletePolicy, s.checkLogin)

		e.POST("/web/createAppAlert", alerts.CreateApp, s.checkLogin)
		e.POST("/web/editAppAlert", alerts.EditApp, s.checkLogin)
		e.POST("/web/deleteAppAlert", alerts.DeleteApp, s.checkLogin)

		e.GET("/web/alertsAppList", alerts.AppList, s.checkLogin)

		e.GET("/web/alertHistory", alerts.History, s.checkLogin)
		e.GET("/web/appAlertsHistory", alerts.AppHistory, s.checkLogin)

		// 管理员面板
		e.GET("/web/admin/userList", admin.UserList, s.checkLogin)
		e.GET("/web/admin/manageUserList", admin.ManageUserList, s.checkLogin)
		e.POST("/web/admin/set", admin.Set, s.checkLogin)
		e.POST("/web/admin/cancel", admin.Cancel, s.checkLogin)
		e.POST("/web/admin/editUser", admin.EditUser, s.checkLogin)

		// 个人设置
		e.POST("/web/setPerson", s.setUser, s.checkLogin)
		e.GET("/web/getAppSetting", s.getAppSetting, s.checkLogin)

		//@to-remove , 开源时删除
		// 传化内部相关,提供监控大盘数据
		e.GET("/web/queryPlatformStatus", app.QueryPlatformStatus)
		e.GET("/web/queryAppBaseStatusList", app.QueryAppBaseStatusList)
		e.GET("/web/queryAppDetailStatus", app.QueryAppDetailStatus)

		// 启动agentd-server，提供agent下载、心跳服务
		// 心跳检测
		e.POST("/web/agentd/update", agentd.Update)
		e.POST("/web/agentd/delete", agentd.Delete, s.checkLogin)
		e.POST("/web/agentd/restart", agentd.Restart, s.checkLogin)
		e.POST("/web/agentd/reset", agentd.Reset, s.checkLogin)

		// 获取所有agentd的信息
		e.GET("/web/agentd/info", agentd.Info, s.checkLogin)
		e.Static("/web/agentd/download", "download/")

		e.Logger.Fatal(e.Start(misc.Conf.Web.Addr))
	}()

	return nil
}

// Close ...
func (s *Web) Close() error {
	return nil
}

func (web *Web) checkLogin(f echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		li := session.GetLoginInfo(c)
		if li == nil {
			return c.JSON(http.StatusOK, g.Result{
				Status:  http.StatusUnauthorized,
				ErrCode: g.NeedLoginC,
				Message: g.NeedLoginE,
			})
		}

		return f(c)
	}
}

func (web *Web) initCql() {
	cqlCluster := gocql.NewCluster(misc.Conf.Storage.Cluster...)
	cqlCluster.Keyspace = "tracing_static"
	cqlCluster.Timeout = 10 * time.Second
	cqlCluster.ReconnectInterval = 500 * time.Millisecond

	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cqlCluster.NumConns = misc.Conf.Storage.NumConns

	// cluster.RetryPolicy = &RetryPolicy{NumRetries: -1, Interval: 2}

	cql, err := cqlCluster.CreateSession()
	if err != nil {
		g.L.Fatal("Init web cql connections error", zap.String("error", err.Error()))
	}
	misc.StaticCql = cql

	cqlCluster1 := gocql.NewCluster(misc.Conf.Storage.Cluster...)
	cqlCluster1.Keyspace = "tracing_data"
	cqlCluster1.Timeout = 20 * time.Second
	cqlCluster1.ReconnectInterval = 500 * time.Millisecond

	//设置连接池的数量,默认是2个（针对每一个host,都建立起NumConns个连接）
	cqlCluster1.NumConns = 20

	// cluster.RetryPolicy = &RetryPolicy{NumRetries: -1, Interval: 2}

	cql1, err := cqlCluster1.CreateSession()
	if err != nil {
		g.L.Fatal("Init web cql connections error", zap.String("error", err.Error()))
	}
	misc.TraceCql = cql1
}
