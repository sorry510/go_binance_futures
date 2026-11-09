package controllers

import (
	"sort"
	"strconv"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/spot"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/server/web"
)

type SpotController struct {
	web.Controller
}

func (ctrl *SpotController) Get() {
	paramsSort := ctrl.GetString("sort")
	symbol_type := ctrl.GetString("symbol_type")
	symbol := ctrl.GetString("symbol")
	enable := ctrl.GetString("enable")
	pin := ctrl.GetString("pin")

	o := orm.NewOrm()
	var symbols []models.SpotSymbols
	query := o.QueryTable("spot_symbols")
	if symbol_type != "" {
		query = query.Filter("type", symbol_type)
	}
	if symbol != "" {
		query = query.Filter("symbol__contains", symbol)
	}
	if enable != "" {
		query = query.Filter("enable", enable)
	}
	if pin != "" {
		query = query.Filter("pin", 1)
	}
	_, err := query.OrderBy("-Pin", "ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error"))
	}

	if strings.HasPrefix(paramsSort, "percent_change") {
		sort.SliceStable(symbols, func(i, j int) bool {
			base := symbols[i].Pin >= symbols[j].Pin
			if paramsSort == "percent_change+" {
				return base && symbols[i].PercentChange >= symbols[j].PercentChange // 涨幅从大到小排序
			} else if paramsSort == "percent_change-" {
				return base && symbols[i].PercentChange < symbols[j].PercentChange // 涨幅从小到大排序
			} else {
				return true
			}
		})
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
	})
}

func (ctrl *SpotController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.SpotSymbols
	o := orm.NewOrm()
	if err := o.QueryTable(symbols.TableName()).Filter("Id", id).One(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "spot symbol not found"))
		return
	}
	oldSymbol, oldQuote := symbols.Symbol, symbols.Type
	oldStrategy, oldTechnology := symbols.Strategy, symbols.Technology
	if err := ctrl.BindJSON(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid Spot symbol configuration"))
		return
	}
	// Legacy editors cannot silently replace the symbol/quote or executable
	// rules of a separately enabled Spot strategy.
	if symbols.StrategyTradeEnabled == 1 &&
		(symbols.Symbol != oldSymbol || symbols.Type != oldQuote ||
			symbols.Strategy != oldStrategy || symbols.Technology != oldTechnology) {
		ctrl.Ctx.Resp(utils.ResJson(409, nil, "disable Spot strategy before modifying legacy strategy or trading pair"))
		return
	}
	// Restrict legacy editors to their former configuration fields. Never
	// write the v25 private strategy_trade_enabled/budget fields through this
	// whole-model update (even on a malformed or stale request).
	if _, err := o.Update(&symbols, "Symbol", "Enable", "TickSize", "StepSize",
		"Usdt", "Profit", "Loss", "Technology", "Strategy", "StrategyType",
		"Pin", "Sort", "Type"); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": symbols, "msg": "success"})
}

func (ctrl *SpotController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.SpotSymbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	o := orm.NewOrm()

	_, err := o.Delete(symbols)
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "success",
	})
}

func (ctrl *SpotController) Post() {
	symbols := new(models.SpotSymbols)
	ctrl.BindJSON(&symbols)
	symbols.PercentChange = 0
	symbols.Close = "0"
	symbols.Open = "0"
	symbols.Low = "0"
	symbols.High = "0"

	symbols.StepSize = "0.1"
	symbols.TickSize = "0.1"
	symbols.Usdt = "10"
	symbols.StrategyTradeEnabled = 0
	symbols.StrategyBudgetQuote = "0"
	symbols.Profit = "100"
	symbols.Loss = "100"

	o := orm.NewOrm()
	id, err := o.Insert(symbols)

	go func() {
		spot.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()

	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	symbols.ID = id

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
	})
}

func (ctrl *SpotController) UpdateEnable() {
	flag := ctrl.Ctx.Input.Param(":flag")

	o := orm.NewOrm()
	_, err := o.Raw("UPDATE spot_symbols SET enable = ?", flag).Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *SpotController) BatchEdit() {
	params := new(SpotBatchEditParams)
	if err := ctrl.BindJSON(params); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid Spot batch payload"))
		return
	}
	// Parameterized allowlist: these are legacy configuration fields only.
	// Never allow a bulk update of v25 private strategy trading switches.
	patch := orm.Params{}
	if params.Usdt != "" {
		patch["usdt"] = params.Usdt
	}
	if params.Profit != "" {
		patch["profit"] = params.Profit
	}
	if params.Loss != "" {
		patch["loss"] = params.Loss
	}
	if params.StrategyType != "" {
		patch["strategy_type"] = params.StrategyType
	}
	if params.StrategyTemplateId != 0 {
		var template models.StrategyTemplates
		if err := orm.NewOrm().QueryTable("strategy_templates").Filter("Id", params.StrategyTemplateId).One(&template); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, "strategy template unavailable"))
			return
		}
		patch["technology"] = template.Technology
		patch["strategy"] = template.Strategy
	}
	if len(patch) > 0 {
		if _, err := orm.NewOrm().QueryTable("spot_symbols").Filter("strategy_trade_enabled", 0).Update(patch); err != nil {
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "msg": "success"})
}
