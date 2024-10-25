package controllers

import (
	"go_binance_futures/feature/api/binance"

	"github.com/beego/beego/v2/server/web"
)

type AccountController struct {
	web.Controller
}

func (ctrl *AccountController) GetBinanceFuturesAccount() {
	account, err := binance.GetFuturesAccount()
	
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 400,
			"msg": err.Error(),
		})
	}

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"account": account,
		},
		"msg": "success",
	})
}
