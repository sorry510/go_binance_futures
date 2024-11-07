package controllers

import (
	"go_binance_futures/models"
	"go_binance_futures/utils"
	"strconv"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type EatRateController struct {
	web.Controller
}

func (ctrl *EatRateController) Post() {
	symbols := new(models.EatRateSymbols)
	ctrl.BindJSON(&symbols)
	
	symbols.Enable = 0 // 默认不开启
	symbols.Type = 1 // 正向套利
	symbols.Leverage = 3 // 杠杆类型
	symbols.MarginType = "CROSSED" // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	symbols.StepSize = "0.1"
	symbols.TickSize = "0.1"
	symbols.Profit = "0"

	o := orm.NewOrm()
	id, err := o.Insert(symbols)
	
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "add failed"))
		return
    }
	symbols.ID = id
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *EatRateController) Get() {
	paramsType := ctrl.GetString("type", "")
	
	o := orm.NewOrm()
	var symbols []models.EatRateSymbols
	query := o.QueryTable("eat_rate_symbols")
	if paramsType != "" {
		query = query.Filter("Type", paramsType)
	}
	_, err := query.OrderBy("ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}
	
func (ctrl *EatRateController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.EatRateSymbols
	o := orm.NewOrm()
	o.QueryTable("eat_rate_symbols").Filter("Id", id).One(&symbols)
	
	ctrl.BindJSON(&symbols)
	
	_, err := o.Update(&symbols) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "edit failed"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *EatRateController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.EatRateSymbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	o := orm.NewOrm()
	
	_, err := o.Delete(symbols)
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "delete failed"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

// 开启吃资金费率
func (ctrl *EatRateController) Start() {
	// flag := ctrl.Ctx.Input.Param(":flag")
	
	// o := orm.NewOrm()
	// _, err := o.Raw("UPDATE eat_rate_symbols SET enable = ?", flag).Exec()
	// if err != nil {
	// 	// 处理错误
	// 	ctrl.Ctx.Resp(utils.ResJson(400, nil, "eat failed"))
	// 	return
	// }
	// ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

// 平仓关闭
func (ctrl *EatRateController) End() {
	// flag := ctrl.Ctx.Input.Param(":flag")
	
	// o := orm.NewOrm()
	// _, err := o.Raw("UPDATE eat_rate_symbols SET enable = ?", flag).Exec()
	// if err != nil {
	// 	// 处理错误
	// 	ctrl.Ctx.Resp(utils.ResJson(400, nil, "close failed"))
	// 	return
	// }
	// ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

