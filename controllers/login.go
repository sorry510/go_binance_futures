package controllers

import (
	"crypto/subtle"
	"net/http"
	"strconv"
	"time"

	"go_binance_futures/utils"

	// Loads the global config before the package-level reads below run.
	_ "go_binance_futures/bootstrap"

	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/server/web"
)

var username, _ = config.String("web::username")
var password, _ = config.String("web::password")
var expires, _ = config.String("web::expires")
var expires_int, _ = strconv.Atoi(expires)

type LoginController struct {
	web.Controller
}

// sendLoginRateLimited reports HTTP 429 to both browsers and API clients.
func (ctrl *LoginController) sendLoginRateLimited(wait time.Duration) {
	seconds := retryAfterSeconds(wait)
	ctrl.Ctx.Output.Header("Retry-After", strconv.Itoa(seconds))
	ctrl.Ctx.Output.Header("Cache-Control", "no-store")
	ctrl.Ctx.Output.SetStatus(http.StatusTooManyRequests)
	_ = ctrl.Ctx.Output.JSON(map[string]interface{}{
		"code": http.StatusTooManyRequests,
		"msg":  "登录失败次数过多，请稍后再试",
		"data": map[string]int{"retry_after": seconds},
	}, false, false)
}

func (ctrl *LoginController) Post() {
	ip := loginRemoteIP(ctrl.Ctx.Request)
	if wait := webLoginLimiter.reserve(ip, time.Now()); wait > 0 {
		ctrl.sendLoginRateLimited(wait)
		return
	}

	user := User{}
	if err := ctrl.BindJSON(&user); err != nil {
		if wait := webLoginLimiter.failed(ip, time.Now()); wait > 0 {
			ctrl.sendLoginRateLimited(wait)
			return
		}
		ctrl.Ctx.Resp(utils.ResJson(101, nil, "参数错误"))
		return
	}

	// Avoid data-dependent string comparison for credentials.
	userOK := subtle.ConstantTimeCompare([]byte(user.UserName), []byte(username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(user.Password), []byte(password)) == 1
	if !userOK || !passOK {
		if wait := webLoginLimiter.failed(ip, time.Now()); wait > 0 {
			ctrl.sendLoginRateLimited(wait)
			return
		}
		ctrl.Ctx.Resp(utils.ResJson(101, nil, "账号或密码错误"))
		return
	}

	token, err := utils.GenerateToken(user.UserName, expires_int)
	if err != nil {
		ctrl.Ctx.Output.SetStatus(http.StatusInternalServerError)
		_ = ctrl.Ctx.Output.JSON(map[string]interface{}{
			"code": 500, "msg": "登录服务暂时不可用", "data": nil,
		}, false, false)
		return
	}
	webLoginLimiter.succeeded(ip)
	ctrl.Ctx.Output.Header("Cache-Control", "no-store")
	ctrl.Ctx.Resp(utils.ResJson(200, map[string]interface{}{
		"token": "Bearer " + token,
	}))
}

type User struct {
	UserName string `json:"username"`
	Password string `json:"password"`
}
