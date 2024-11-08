package controllers

import (
	"go_binance_futures/utils"
	"strconv"

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

func (ctrl *LoginController) Post() {
	user := User{}
	err := ctrl.BindJSON(&user)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(101, nil, "参数错误"))
	} else {
		if user.UserName == username && user.Password == password {
			token, err := utils.GenerateToken(user.UserName, expires_int)
			if err != nil {
				ctrl.Ctx.WriteString(err.Error())
				return
			}
			ctrl.Ctx.Resp(utils.ResJson(200, map[string]interface{}{
				"token": "Bearer " +token,
			}))
		} else {
			ctrl.Ctx.Resp(utils.ResJson(101, nil, "账号或密码错误"))
		}
	}
}

type User struct {
	UserName string `json:"username"`
	Password string  `json:"password"`
}