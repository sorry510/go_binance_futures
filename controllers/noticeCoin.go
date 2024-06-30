package controllers

import (
	"strconv"

	"go_binance_futures/models"
	"go_binance_futures/spot"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type NoticeCoinController struct {
	web.Controller
}

func (ctrl *NoticeCoinController) Get() {
	paramsType := ctrl.GetString("type", "")
	
	o := orm.NewOrm()
	var symbols []models.NoticeSymbols
	query := o.QueryTable("notice_symbols")
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
	
func (ctrl *NoticeCoinController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.NoticeSymbols)
	ctrl.BindJSON(&symbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	
	// tickSize, stepSize := spot.GetCoinOrderSize(symbols.Symbol)
	// symbols.TickSize = tickSize
	// symbols.StepSize = stepSize
	
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

func (ctrl *NoticeCoinController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.NoticeSymbols)
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

func (ctrl *NoticeCoinController) Post() {
	symbols := new(models.NoticeSymbols)
	ctrl.BindJSON(&symbols)
	
	symbols.Enable = 0 // 默认不开启
	symbols.HasNotice = 0 // 默认未通知
	symbols.AutoOrder = 1 // 默认自动下单
	symbols.ProfitPrice = "0" // 默认止盈价格(0表示不止盈)
	symbols.LossPrice = "0" // 默认止损价格(0表示不止损)
	
	symbols.Leverage = 3
	symbols.MarginType = "ISOLATED"
	symbols.StepSize = "0"
	symbols.TickSize = "0"
	symbols.Usdt = "10"
	symbols.Side = "buy"
	symbols.Quantity = "0"
	
	tickSize, stepSize := spot.GetCoinOrderSize(symbols.Symbol)
	symbols.TickSize = tickSize
	symbols.StepSize = stepSize
	
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

func (ctrl *NoticeCoinController) UpdateEnable() {
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