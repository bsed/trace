package alerts

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

func CreateGroup(c echo.Context) error {
	name := c.FormValue("name")
	channel := c.FormValue("channel")
	usersS := c.FormValue("users")

	if name == "" || channel == "" || usersS == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	var users []string
	err := json.Unmarshal([]byte(usersS), &users)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 插入
	q1 := misc.StaticCql.Query(`INSERT INTO  alerts_group (id,name,owner,channel,users,update_date) VALUES (uuid(),?,?,?,?,?)`, name, li.ID, channel, users, time.Now().Unix())
	err = q1.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

func EditGroup(c echo.Context) error {
	id := c.FormValue("id")
	name := c.FormValue("name")
	channel := c.FormValue("channel")
	usersS := c.FormValue("users")

	if id == "" || name == "" || channel == "" || usersS == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	var users []string
	err := json.Unmarshal([]byte(usersS), &users)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 更新
	q1 := misc.StaticCql.Query(`UPDATE alerts_group SET name=?,channel=?,users=?,update_date=? WHERE id=? and owner=?`,
		name, channel, users, time.Now().Unix(), id, li.ID)
	err = q1.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

func DeleteGroup(c echo.Context) error {
	id := c.FormValue("id")

	if id == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 删除
	q1 := misc.StaticCql.Query(`DELETE FROM  alerts_group WHERE id=? and owner=?`,
		id, li.ID)
	err := q1.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

type Group struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	OwnerID   string   `json:"owner_id"`
	OwnerName string   `json:"owner_name"`
	Channel   string   `json:"channel"`
	Users     []string `json:"users"`
}

func QueryGroups(c echo.Context) error {
	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 若该用户是管理员，可以获取所有组
	var iter *gocql.Iter
	if li.Priv == g.PRIV_NORMAL {
		iter = misc.StaticCql.Query(`SELECT id,name,owner,channel,users FROM alerts_group WHERE owner=?`, li.ID).Iter()
	} else {
		iter = misc.StaticCql.Query(`SELECT id,name,owner,channel,users FROM alerts_group`).Iter()
	}

	var id, name, owner, channel string
	var users []string

	groups := make([]*Group, 0)
	for iter.Scan(&id, &name, &owner, &channel, &users) {
		ownerNameR, _ := session.UsersMap.Load(owner)
		groups = append(groups, &Group{id, name, owner, ownerNameR.(*session.User).Name, channel, users})
	}

	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   groups,
	})
}
