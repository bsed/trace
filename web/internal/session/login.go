package session

import (
	"encoding/json"
	"net/http"
	"net/url"

	"time"

	"github.com/imdevlab/g/utils"
	"github.com/bsed/trace/web/internal/misc"

	"go.uber.org/zap"

	"github.com/imdevlab/g"
	"github.com/labstack/echo"
	"github.com/valyala/fasthttp"
)

var defaultRole = "normal"

type Session struct {
	User       *UserInfo
	CreateTime time.Time
}

type UserInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
	Priv     string `json:"priv"`
	SsoToken string `json:"ssoToken"`
}

func Login(c echo.Context) error {
	subToken := c.FormValue("subToken")
	// 通过subtoken获取用户和ssotoken
	body := "{'subToken':'" + subToken + "'}"
	url := misc.Conf.Login.SsoLogin

	b := requestToSso(body, url)
	tokenInfo := &TokenInfo{}
	err := json.Unmarshal(b, tokenInfo)
	if err != nil {
		g.L.Info("解析sso用户信息失败", zap.Error(err), zap.String("body", string(b)))
		return nil
	}

	uid := tokenInfo.Data.SubTokenObj.UserSession.UserId

	// 查询该用户是否是管理员
	var priv string
	q := `SELECT priv FROM admin WHERE id=?`
	err = misc.StaticCql.Query(q, uid).Scan(&priv)
	if priv == "" {
		priv = g.PRIV_NORMAL
	}

	email := tokenInfo.Data.SubTokenObj.UserSession.OaSession.Email
	mobile := tokenInfo.Data.SubTokenObj.UserSession.OaSession.Mobile
	if mobile == "" {
		mobile, email = getUserFromTone(uid, tokenInfo.Data.SubTokenObj.UserSession.UserName)
	}

	user := &UserInfo{
		ID:       uid,
		Name:     tokenInfo.Data.SubTokenObj.UserSession.UserName,
		Avatar:   "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif",
		SsoToken: tokenInfo.Data.SubTokenObj.SsoToken,
		Email:    email,
		Mobile:   mobile,
		Priv:     priv,
	}

	if user.ID == "" {
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusUnauthorized,
			Message: "sub token无效，请重新登陆",
		})
	}

	//sub token验证成功，保存session
	Sessions.Store(user.SsoToken, &Session{
		User:       user,
		CreateTime: time.Now(),
	})

	// 更新数据库中的user表
	q = `UPDATE account SET name=?,mobile=?,email=?,last_login_date=? WHERE id=?`
	err = misc.StaticCql.Query(q, user.Name, user.Mobile, user.Email, utils.Time2StringSecond(time.Now()), user.ID).Exec()
	if err != nil {
		g.L.Info("插入用户信息失败", zap.Error(err))
	}
	// 更新登录次数
	q = `UPDATE login_count SET count  = count + 1 WHERE id=?`
	err = misc.StaticCql.Query(q, user.ID).Exec()
	if err != nil {
		g.L.Info("更新登录次数失败", zap.Error(err))
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   user,
	})
}

func LoginMock(c echo.Context) error {
	user := &UserInfo{
		ID:       "13269",
		Name:     "孙飞",
		Avatar:   "https://wpimg.wallstcn.com/f778738c-e4f8-4870-b634-56703b4acafe.gif",
		SsoToken: "0af8c18eed353af38b2e2524f4850f76",
	}

	//sub token验证成功，保存session
	Sessions.Store(user.SsoToken, &Session{
		User:       user,
		CreateTime: time.Now(),
	})

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   user,
	})
}

func Logout(c echo.Context) error {
	token := getToken(c)
	// 删除用户的session
	Sessions.Delete(token)

	// 请求sso 注销token
	body := "{'ssoToken':'" + token + "', 'clientNo':'1' }"
	url := misc.Conf.Login.SsoLogout
	requestToSso(body, url)
	return c.JSON(http.StatusOK, g.Result{
		Status:  http.StatusOK,
		Message: "退出登陆成功",
	})
}

func GetUserInfo(c echo.Context) error {
	token := getToken(c)
	sess, ok := Sessions.Load(token)
	if !ok {
		// 用户未登陆或者session失效
		return c.JSON(http.StatusOK, g.Result{
			Status:  http.StatusUnauthorized,
			ErrCode: g.NeedLoginC,
			Message: g.NeedLoginE,
		})
	}

	return c.JSON(http.StatusOK, g.Result{
		Status: http.StatusOK,
		Data:   sess.(*Session).User,
	})
}

//subToken认证，返回的用户信息
type TokenInfo struct {
	Code int `json:"code"`
	Data struct {
		SubTokenObj struct {
			Message     string `json:"message"`
			SsoToken    string `json:"ssoToken"`
			Status      int    `json:"status"`
			UserSession struct {
				OaSession struct {
					Dept   string `json:"fdDept"`
					Email  string `json:"fdEmail"`
					Mobile string `json:"fdMobileNo"`
				}
				Facility        string `json:"facility"`
				HeadImgUrl      string `json:"headImgUrl"`
				SysDepartmentId string `json:"sysDepartmentId"`
				UserId          string `json:"userId"`
				UserName        string `json:"userName"`
				UserType        string `json:"userType"`
			} `json:"userSession"`
		} `json:"subTokenObj"`
	} `json:"data"`
	Message string `json:"message"`
}

func requestToSso(body string, url string) []byte {
	req := &fasthttp.Request{}
	resp := &fasthttp.Response{}

	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	req.SetBodyString(body)

	req.SetRequestURI(url)
	var cli = &fasthttp.Client{}
	err := cli.DoTimeout(req, resp, 15*time.Second)
	if err != nil {
		g.L.Info("获取sso用户信息失败", zap.Error(err))
		return nil
	}

	return resp.Body()
}

func getToken(c echo.Context) string {
	return c.Request().Header.Get("X-Token")
}

func GetLoginInfo(c echo.Context) *UserInfo {
	token := getToken(c)
	sess, ok := Sessions.Load(token)
	if !ok {
		// 用户未登陆或者session失效
		return nil
	}

	return sess.(*Session).User
}

type ToneUserRes struct {
	Data map[string]string `json:"data"`
}

func getUserFromTone(id, name string) (string, string) {
	var p = url.Values{}
	p.Add("userId", id)
	p.Add("username", name)

	url := misc.Conf.Login.Tone + "/tone/open/employee/queryEmployeeByUserId?" + p.Encode()
	code, r, err := g.Cli.Get(nil, url)

	if code != 200 || err != nil {
		g.L.Warn("request to tone failed", zap.Error(err), zap.Int("code", code), zap.String("url", url))
		return "", ""
	}

	res := &ToneUserRes{}
	json.Unmarshal(r, &res)

	return res.Data["mobileNumber"], res.Data["email"]
}
