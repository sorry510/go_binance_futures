package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type RushController struct {
	web.Controller
}

func (ctrl *RushController) Get() {
	
	o := orm.NewOrm()
	var symbols []models.NewSymbols
	_, err := o.QueryTable("new_symbols").OrderBy("ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error"))
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *RushController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.NewSymbols)
	ctrl.BindJSON(&symbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	
	
	// 币还没上线可能会报错
	// marginType := futures.MarginTypeIsolated
	// if symbols.MarginType == "CROSSED" {
	// 	marginType = futures.MarginTypeCrossed
	// }
	// binance.SetLeverage(symbols.Symbol, int(symbols.Leverage))  // 修改合约倍数
	// binance.SetMarginType(symbols.Symbol, marginType) // 修改仓位模式
	
	o := orm.NewOrm()
	_, err := o.Update(symbols) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "修改失败"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *RushController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.NewSymbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	o := orm.NewOrm()
	
	_, err := o.Delete(symbols)
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "删除错误"))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

func (ctrl *RushController) Post() {
	symbols := new(models.NewSymbols)
	ctrl.BindJSON(&symbols)
	
	symbols.Leverage = 3
	symbols.MarginType = "ISOLATED"
	symbols.StepSize = "0"
	symbols.TickSize = "0"
	symbols.Usdt = "10"
	symbols.Type = 1
	symbols.Enable = 0
	symbols.Side = "buy"
	symbols.Quantity = "0"
	
	o := orm.NewOrm()
	id, err := o.Insert(symbols)
	
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "新增失败"))
		return
    }
	symbols.ID = id
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *RushController) UpdateEnable() {
	flag := ctrl.Ctx.Input.Param(":flag")
	
	o := orm.NewOrm()
	_, err := o.Raw("UPDATE newSymbols SET enable = ?", flag).Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "更新错误"))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}