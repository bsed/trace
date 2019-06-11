package alerts

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/imdevlab/g/utils"

	"github.com/gocql/gocql"
	"github.com/imdevlab/g"
	"github.com/bsed/trace/pkg/util"
	ecode "github.com/bsed/trace/web/internal/error_code"
	"github.com/bsed/trace/web/internal/misc"
	"github.com/bsed/trace/web/internal/session"
	"github.com/labstack/echo"
	"go.uber.org/zap"
)

type Policy struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	OwnerID    string        `json:"owner_id"`
	OwnerName  string        `json:"owner_name"`
	Alerts     []*util.Alert `json:"alerts"`
	UpdateDate string        `json:"update_date"`
}

func CreatePolicy(c echo.Context) error {
	policyRaw := c.FormValue("policy")

	policy := &Policy{}
	err := json.Unmarshal([]byte(policyRaw), &policy)
	if err != nil || policy.Name == "" || len(policy.Alerts) == 0 {
		g.L.Info("create policy params error", zap.String("policy", policyRaw), zap.Error(err))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 同一个用户下的模版名不能重复
	n := misc.StaticCql.Query(`SELECT id FROM  alerts_policy WHERE name=? and owner=? ALLOW FILTERING`, policy.Name, li.ID).Iter().NumRows()
	if n > 0 {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: ecode.PolicyNameExistC,
			Message: ecode.PolicyNameExistE,
		})
	}
	// 插入
	q := misc.StaticCql.Query(`INSERT INTO  alerts_policy (id,name,owner,alerts,update_date) VALUES (uuid(),?,?,?,?)`, policy.Name, li.ID, policy.Alerts, time.Now().Unix())
	err = q.Exec()
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

func EditPolicy(c echo.Context) error {
	policyRaw := c.FormValue("policy")
	policy := &Policy{}
	err := json.Unmarshal([]byte(policyRaw), &policy)
	if err != nil || policy.Name == "" || len(policy.Alerts) == 0 {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusBadRequest,
			ErrCode: g.ParamInvalidC,
			Message: g.ParamInvalidE,
		})
	}

	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 查询目标policy是否已经存在
	var owner string
	err = misc.StaticCql.Query(`SELECT owner FROM  alerts_policy WHERE id=?`, policy.ID).Scan(&owner)
	if err != nil {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	// 必须是owner本人或者管理员才能编辑
	if owner != li.ID && li.Priv != g.PRIV_SUPER_ADMIN && li.Priv != g.PRIV_ADMIN {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.ForbiddenC,
			Message: g.ForbiddenE,
		})
	}

	// 插入
	now := time.Now().Unix()
	q := misc.StaticCql.Query(`UPDATE  alerts_policy SET name=?,alerts=?,update_date=? WHERE id=? and owner=? IF EXISTS`, policy.Name, policy.Alerts, now, policy.ID, owner)
	err = q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	// 将关联到该策略的所有app的update时间进行更新
	iter := misc.StaticCql.Query(`SELECT name,owner FROM alerts_app WHERE policy_id=?`, policy.ID).Iter()
	var appName, appOwner string
	for iter.Scan(&appName, &appOwner) {
		q1 := misc.StaticCql.Query(`UPDATE alerts_app SET update_date=? WHERE name=? and owner=?`, now, appName, appOwner)
		err = q1.Exec()
		if err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
			return c.JSON(http.StatusOK, g.Result{
				Status:  http.StatusInternalServerError,
				ErrCode: g.DatabaseC,
				Message: g.DatabaseE,
			})
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

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}

func QueryPolicies(c echo.Context) error {
	// 获取当前用户
	li := session.GetLoginInfo(c)

	// 若该用户是管理员，可以获取所有组
	var q *gocql.Query

	if li.Priv == g.PRIV_NORMAL {
		q = misc.StaticCql.Query(`SELECT id,name,owner,alerts,update_date FROM alerts_policy WHERE owner=?`, li.ID)
	} else {
		q = misc.StaticCql.Query(`SELECT id,name,owner,alerts,update_date FROM alerts_policy`)
	}

	iter := q.Iter()
	var id, name, owner string
	var updateDate int64
	var alerts []*util.Alert

	policies := make([]*Policy, 0)
	for iter.Scan(&id, &name, &owner, &alerts, &updateDate) {
		ownerNameR, _ := session.UsersMap.Load(owner)
		t := utils.Time2StringSecond(time.Unix(updateDate, 0))
		policies = append(policies, &Policy{id, name, owner, ownerNameR.(*session.User).Name, alerts, t})
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
		Data:   policies,
	})
}

func QueryPolicy(c echo.Context) error {
	pid := c.FormValue("id")

	var id, name, owner string
	var updateDate int64
	var alerts []*util.Alert

	q := misc.StaticCql.Query(`SELECT id,name,owner,alerts,update_date FROM alerts_policy WHERE id=?`, pid)
	err := q.Scan(&id, &name, &owner, &alerts, &updateDate)
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	ownerNameR, _ := session.UsersMap.Load(owner)
	t := utils.Time2StringSecond(time.Unix(updateDate, 0))

	policy := &Policy{id, name, owner, ownerNameR.(*session.User).Name, alerts, t}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   policy,
	})
}

func DeletePolicy(c echo.Context) error {
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
	q := misc.StaticCql.Query(`DELETE FROM  alerts_policy WHERE id=? and owner=?`,
		id, li.ID)
	err := q.Exec()
	if err != nil {
		g.L.Warn("access database error", zap.Error(err), zap.String("query", q.String()))
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusInternalServerError,
			ErrCode: g.DatabaseC,
			Message: g.DatabaseE,
		})
	}

	// 将相关联的应用的策略更新
	// 将关联到该策略的所有app的update时间进行更新
	iter := misc.StaticCql.Query(`SELECT name,owner FROM alerts_app WHERE policy_id=?`, id).Iter()
	var appName, owner string
	for iter.Scan(&appName, &owner) {
		q1 := misc.StaticCql.Query(`UPDATE alerts_app SET policy_id=?,update_date=? WHERE name=? and owner=?`, "", time.Now().Unix(), appName, owner)
		err = q1.Exec()
		if err != nil {
			g.L.Warn("access database error", zap.Error(err), zap.String("query", q1.String()))
			return c.JSON(http.StatusOK, g.Result{
				Status:  http.StatusInternalServerError,
				ErrCode: g.DatabaseC,
				Message: g.DatabaseE,
			})
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

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
	})
}
