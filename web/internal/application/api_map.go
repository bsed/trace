package app

import (
	"net/http"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

// -- api被应用调用统计表
// CREATE TABLE IF NOT EXISTS api_map (
//     source_name                text,              -- 源应用名
//     source_type                int,               -- 源应用类型

//     target_name                text,              -- 目标应用名
//     target_type                int,               -- 目标应用类型

//     access_count               int,                -- 访问总数
//     access_err_count           int,                -- 访问错误数
//     access_duration            int,                -- 访问总耗时

//     api_id                     int,                -- api id
//     input_date                 bigint,             -- 插入时间
//     PRIMARY KEY (target_name, input_date, api_id, source_name)
// ) WITH gc_grace_seconds = 10800;

const (
	AppNode = 1
	ApiNode = 2
)

// 通过App name来查询service map
func QueryApiMap(c echo.Context) error {
	tname := c.FormValue("app_name")
	start, end, err := misc.StartEndDate(c)
	if err != nil {
		g.L.Info("日期参数不合法", zap.String("start", c.FormValue("start")), zap.String("end", c.FormValue("end")), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusOK,
			ErrCode: g.ParamInvalidC,
			Message: "日期参数不合法",
		})
	}

	nodeMap := make(map[string]*Node)
	linkMap := make(map[string]*NodeLink)

	// 获取当前APP的父应用信息
	q := misc.TraceCql.Query(`SELECT source_name,access_count,access_err_count,access_duration,api FROM api_map WHERE target_name = ? and input_date > ? and input_date < ?  ALLOW FILTERING`, tname, start.Unix(), end.Unix())
	iter := q.Iter()

	var sname, api string
	var accessCount, accessErr, accessDuration int
	for iter.Scan(&sname, &accessCount, &accessErr, &accessDuration, &api) {
		// 应用和api都属于node，分类不同
		// 更新父节点信息
		_, ok := nodeMap[sname]
		if !ok {
			nodeMap[sname] = &Node{Name: sname, Category: AppNode}
		}

		// 更新子节点信息
		name := tname + ":" + api
		_, ok = nodeMap[name]
		if !ok {
			nodeMap[name] = &Node{Name: name, Category: ApiNode}
		}

		// 更新父子之间的link信息
		lname := sname + name
		link, ok := linkMap[lname]
		if !ok {
			linkMap[lname] = &NodeLink{sname, name, accessCount, accessErr, accessDuration, 0}
		} else {
			link.AccessCount += accessCount
			link.ErrorCount += accessErr
			link.duration += accessDuration
		}
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	// 获取应用的下游信息
	sname = tname
	q1 := misc.TraceCql.Query(`SELECT target_name,access_count,access_err_count,access_duration,api FROM api_map WHERE source_name = ? and input_date > ? and input_date < ?  ALLOW FILTERING`, sname, start.Unix(), end.Unix())
	iter1 := q1.Iter()

	for iter1.Scan(&tname, &accessCount, &accessErr, &accessDuration, &api) {
		_, ok := nodeMap[sname]
		if !ok {
			nodeMap[sname] = &Node{Name: sname, Category: AppNode}
		}

		name := tname + ":" + api

		// 更新子节点信息
		_, ok = nodeMap[name]
		if !ok {
			nodeMap[name] = &Node{Name: name, Category: ApiNode}
		}

		// 更新父子之间的link信息
		lname := sname + name
		link, ok := linkMap[lname]
		if !ok {
			linkMap[lname] = &NodeLink{sname, name, accessCount, accessErr, accessDuration, 0}
		} else {
			link.AccessCount += accessCount
			link.ErrorCount += accessErr
			link.duration += accessDuration
		}
	}

	if err := iter1.Close(); err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	nodes := make([]*Node, 0, len(nodeMap))
	links := make([]*NodeLink, 0, len(linkMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}
	for _, link := range linkMap {
		if link.AccessCount > 0 {
			link.AverageDuration = link.duration / link.AccessCount
		}
		links = append(links, link)
	}

	//
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data: ServiceMapResult{
			Nodes: nodes,
			Links: links,
		},
	})
}
