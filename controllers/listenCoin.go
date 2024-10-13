package controllers

import (
	"strconv"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type ListenCoinController struct {
	web.Controller
}

func (ctrl *ListenCoinController) Get() {
	paramsType := ctrl.GetString("type", "")
	
	o := orm.NewOrm()
	var symbols []models.ListenSymbols
	query := o.QueryTable("listen_symbols")
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
	
func (ctrl *ListenCoinController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.ListenSymbols
	o := orm.NewOrm()
	o.QueryTable("listen_symbols").Filter("Id", id).One(&symbols)
	
	ctrl.BindJSON(&symbols)
	
	_, err := o.Update(&symbols) // _ 是受影响的条数
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

func (ctrl *ListenCoinController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.ListenSymbols)
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

func (ctrl *ListenCoinController) Post() {
	symbols := new(models.ListenSymbols)
	ctrl.BindJSON(&symbols)
	
	symbols.Enable = 0 // 默认不开启
	symbols.KlineInterval = "1m" // 默认k线周期
	symbols.ChangePercent = "1.1" // 1.1% 默认变化幅度
	symbols.LastNoticeTime = 0 // 最后一次通知时间
	symbols.NoticeLimitMin = 5 // 最小通知间隔
	symbols.ListenType = "kline_base"

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

func (ctrl *ListenCoinController) UpdateEnable() {
	flag := ctrl.Ctx.Input.Param(":flag")
	
	o := orm.NewOrm()
	_, err := o.Raw("UPDATE listen_symbols SET enable = ?", flag).Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "更新错误"))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *ListenCoinController) GetKcLineChart() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.ListenSymbols
	o := orm.NewOrm()
	o.QueryTable("listen_symbols").Filter("Id", id).One(&symbols)
	
	symbol := symbols.Symbol
	limit := 150
	period := 50
	interval1 := symbols.KlineInterval
	multiplier1 := 2.75 // 窄通道
	multiplier2 := 3.75 // 宽通道
	kline_1, err := binance.GetKlineData(symbol, interval1, limit)
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{} {
			"code": 500,
			"msg": "kline error",
		})
		return
	}
	
	high1, low1, close1 := line.GetLineFloatPrices(kline_1)
	upper1, ma1, lower1 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier1) // kc1
	upper2, _, lower2 := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier2) // kc2
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"upper1": upper1,
			"ma1": ma1,
			"lower1": lower1,
			"upper2": upper2,
			"lower2": lower2,
			"close1": close1,
			"high1": high1,
			"low1": low1,
		},
		"msg": "success",
	})
}