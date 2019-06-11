package admin

import (
	"net/http"
	"time"

	"github.com/imdevlab/g"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

func InitSuperAdmin() {
	q := misc.StaticCql.Query(`UPDATE  admin  SET priv=? WHERE id=?`, "super_admin", "13269")
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
	}
}

func ManageUserList(c echo.Context) error {
	// 查询所有用户
	q := `SELECT id,name,mobile,email,last_login_date FROM account`
	iter := misc.StaticCql.Query(q).Iter()

	users := make(map[string]*session.User)
	var id, name, mobile, email, priv, lld string

	for iter.Scan(&id, &name, &mobile, &email, &lld) {
		users[id] = &session.User{
			ID:            id,
			Name:          name,
			Mobile:        mobile,
			Email:         email,
			Priv:          g.PRIV_NORMAL,
			LastLoginDate: lld,
		}
	}

	// 查询相应权限
	q = `SELECT id,priv FROM admin`
	iter = misc.StaticCql.Query(q).Iter()

	for iter.Scan(&id, &priv) {
		u, ok := users[id]
		if ok {
			u.Priv = priv
		}
	}
	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	// 查询用户的登录次数
	q = `SELECT id,count FROM login_count`
	iter = misc.StaticCql.Query(q).Iter()

	var count int
	for iter.Scan(&id, &count) {
		u, ok := users[id]
		if ok {
			u.LoginCount = count
		}
	}
	if err := iter.Close(); err != nil {
		g.L.Warn("close iter error:", zap.Error(err))
	}

	nusers := make([]*session.User, 0)
	for _, u := range users {
		nusers = append(nusers, u)
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   nusers,
	})
}

func UserList(c echo.Context) error {
	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   session.UsersList,
	})
}

func Set(c echo.Context) error {
	userID := c.FormValue("user_id")
	li := session.GetLoginInfo(c)
	// 判断当前用户是否超级管理员
	if li.Priv != g.PRIV_SUPER_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusForbidden,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	// 将目标用户设置为管理员
	q := misc.StaticCql.Query(`INSERT INTO admin (id,priv) VALUES (?,?)`, userID, g.PRIV_ADMIN)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
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

func Cancel(c echo.Context) error {
	userID := c.FormValue("user_id")
	li := session.GetLoginInfo(c)
	// 判断当前用户是否超级管理员
	if li.Priv != g.PRIV_SUPER_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusForbidden,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	// 将目标用户设置为管理员
	q := misc.StaticCql.Query(`DELETE FROM admin WHERE id=?`, userID)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
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

func LoopLoadUsers() {
	for {
		// 查询所有用户
		q := `SELECT id,name,mobile,email,last_login_date FROM account`
		iter := misc.StaticCql.Query(q).Iter()

		users := make([]*session.User, 0)
		var id, name, mobile, email, lld string

		for iter.Scan(&id, &name, &mobile, &email, &lld) {
			u := &session.User{
				ID:            id,
				Name:          name,
				Mobile:        mobile,
				Email:         email,
				LastLoginDate: lld,
			}
			users = append(users, u)
			session.UsersMap.Store(id, u)
		}
		if err := iter.Close(); err != nil {
			g.L.Warn("close iter error:", zap.Error(err))
		}

		session.UsersList = users

		time.Sleep(60 * time.Second)
	}
}

func EditUser(c echo.Context) error {
	id := c.FormValue("id")
	email := c.FormValue("email")
	mobile := c.FormValue("mobile")

	// 更新数据库中的user表
	q := misc.StaticCql.Query(`UPDATE account SET mobile=?,email=? WHERE id=?`, mobile, email, id)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
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
