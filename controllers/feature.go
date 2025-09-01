package controllers

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"

	"go_binance_futures/feature"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)


type BatchEditParams struct {
	Usdt string `json:"usdt"`
	Profit string `json:"profit"`
	Loss string `json:"loss"`
	Leverage string `json:"leverage"`
	MarginType string `json:"marginType"`
	StrategyType string `json:"strategyType"`
	StrategyTemplateId int64 `json:"strategyTemplateId"`
}
type FeatureController struct {
	web.Controller
}

func (ctrl *FeatureController) Get() {
	paramsSort := ctrl.GetString("sort")
	symbol_type := ctrl.GetString("symbol_type")
	symbol := ctrl.GetString("symbol")
	enable := ctrl.GetString("enable")
	margin_type := ctrl.GetString("margin_type")
	pin := ctrl.GetString("pin")
	paramsPage := ctrl.GetString("page", "1")
	paramsLimit := ctrl.GetString("limit", "20")
	page, _ := strconv.Atoi(paramsPage)
	limit, _ := strconv.Atoi(paramsLimit)
	offset := (page - 1) * limit
	
	o := orm.NewOrm()
	var symbols []models.Symbols
	query := o.QueryTable("symbols")
	countQuery := o.QueryTable("symbols")
	if symbol_type != "" {
		query = query.Filter("type", symbol_type)
		countQuery = countQuery.Filter("type", symbol_type)
	}
	if symbol != "" {
		query = query.Filter("symbol__contains", symbol)
		countQuery = countQuery.Filter("symbol__contains", symbol)
	}
	if enable != "" {
		query = query.Filter("enable", enable)
		countQuery = countQuery.Filter("enable", enable)
	}
	if margin_type != "" {
		query = query.Filter("marginType", margin_type)
		countQuery = countQuery.Filter("marginType", margin_type)
	}
	if pin != "" {
		query = query.Filter("pin", 1)
		countQuery = countQuery.Filter("pin", 1)
	}
	_, err := query.OrderBy("-Pin", "ID").Limit(limit, offset).All(&symbols,
		"ID", "Symbol", "PercentChange", "Close", "Open", "Low", "High", "Enable", "UpdateTime", "QuoteVolume", "TradeCount", "Leverage", "MarginType",
		"StepSize", "TickSize", "Usdt", "Profit", "Loss", "StrategyType", "Pin", "Sort", "Type",
	)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	total, err := countQuery.Count()
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
	}
	
	// TODO: 分页之后这里有问题
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

	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": map[string]interface{} {
			"total": total,
			"list": symbols,
		},
		"msg": "success",
	})
}

func (ctrl *FeatureController) Show() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.Symbols
	o := orm.NewOrm()
	if err := o.QueryTable("symbols").Filter("Id", id).One(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Id", id).One(&symbols)
	
	go func() {
		// feature.UpdateSymbolTradeInfo(symbols) // 更新合约倍率和仓位模式
		feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()
	
	ctrl.BindJSON(&symbols)
	
	_, err := o.Update(&symbols) // _ 是受影响的条数
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Delete() {
	id := ctrl.Ctx.Input.Param(":id")
	symbols := new(models.Symbols)
	intId, _ := strconv.ParseInt(id, 10, 64)
	symbols.ID = intId
	o := orm.NewOrm()
	
	_, err := o.Delete(symbols)
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
    }
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

func (ctrl *FeatureController) Post() {
	symbols := new(models.Symbols)
	ctrl.BindJSON(&symbols)
	symbols.PercentChange = 0
	symbols.Close = "0"
	symbols.Open = "0"
	symbols.Low = "0"
	symbols.High = "0"
	
	symbols.Leverage = 3
	symbols.MarginType = "CROSSED" // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	symbols.StepSize = "0.1"
	symbols.TickSize = "0.1"
	symbols.Usdt = "10"
	symbols.Profit = "100"
	symbols.Loss = "100"
	symbols.KlineInterval = "1d"
	symbols.BaseVolume = 0.0
	symbols.QuoteVolume = 0.0
	symbols.CloseQty = 0.0
	symbols.TradeCount = 0.0
	
	
	o := orm.NewOrm()
	id, err := o.Insert(symbols)
	
	go func() {
		// feature.UpdateSymbolTradeInfo(symbols) // 更新合约倍率和仓位模式
		feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	}()
	
    if err != nil {
        // 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
    }
	symbols.ID = id
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbols,
		"msg": "success",
	})
}

func (ctrl *FeatureController) UpdateEnable() {
	flag := ctrl.Ctx.Input.Param(":flag")
	
	o := orm.NewOrm()
	_, err := o.Raw("UPDATE symbols SET enable = ?", flag).Exec()
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func (ctrl *FeatureController) BatchEdit() {
	params := new(BatchEditParams)
	ctrl.BindJSON(&params)
	query := "UPDATE symbols SET"
	bindData := []interface{}{}
	
	if params.Usdt != "" {
		query += " usdt = ?,"
		bindData = append(bindData, params.Usdt)
	}
	if (params.Profit != "") {
		query += " profit = ?,"
		bindData = append(bindData, params.Profit)
	}
	if (params.Loss != "") {
		query += " loss = ?,"
		bindData = append(bindData, params.Loss)
	}
	if (params.Leverage != "") {
		query += " leverage = ?,"
		bindData = append(bindData, params.Leverage)
	}
	if (params.MarginType != "") {
		query += " marginType = ?,"
		bindData = append(bindData, params.MarginType)
	}
	if (params.StrategyType != "") {
		query += " strategy_type = ?,"
		bindData = append(bindData, params.StrategyType)
	}
	if (params.StrategyTemplateId != 0) {
		var template models.StrategyTemplates
		orm.NewOrm().QueryTable("strategy_templates").Filter("Id", params.StrategyTemplateId).One(&template)
		query += " technology = ?,"
		bindData = append(bindData, template.Technology)
		query += " strategy = ?,"
		bindData = append(bindData, template.Strategy)
	}
	
	if strings.HasSuffix(query, ",") {
		query = strings.TrimSuffix(query, ",")
		// query += " WHERE enable = 1" // 只更新开启的交易对
		_, err := orm.NewOrm().Raw(query, bindData...).Exec()
		if err != nil {
			// 处理错误
			ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
			return
		}
	}
	
	// go func() {
	// 	feature.UpdateSymbolsTradePrecision() // 更新合约交易精度
	// }()
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"msg": "success",
	})
}

func (ctrl *FeatureController) TestStrategyRule() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.Symbols
	o := orm.NewOrm()
	o.QueryTable("symbols").Filter("Id", id).One(&symbols)
	
	ctrl.BindJSON(&symbols)
	
	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(symbols.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	env := line.InitParseEnv(symbols.Symbol, symbols.Technology)
	
	env["ROI"] = 10.2
	env["Position"] = types.FuturesPositionCode{
		Symbol: symbols.Symbol,
		EntryPrice: 68000.0,
		MarkPrice: 72000.0,
		Amount: -0.02,
		UnrealizedProfit: 100.2,
		Leverage: 3,
		Side: "SHORT",
		Mock: true,
		CreateTime: 1234567890000,
		SourceType: "api",
	}
	positions, _ := feature.GetTransformPositions()
	for _, position := range positions {
		if position.Symbol != symbols.Symbol {
			continue
		}
		positionAmtFloat, _ := strconv.ParseFloat(position.Amount, 64)
		positionAmtFloatAbs := math.Abs(positionAmtFloat) // 空单为负数,纠正为绝对值
		if positionAmtFloatAbs < 0.0000000001 {// 没有持仓的
			continue
		}
		if (position.Side == "LONG" && strategyConfig[0].Type == "close_long") || (position.Side == "SHORT" && strategyConfig[0].Type == "close_short") {
			unRealizedProfit, _ := strconv.ParseFloat(position.UnrealizedProfit, 64)
			leverage_float64 := float64(position.Leverage)
			entryPrice_float64, _ := strconv.ParseFloat(position.EntryPrice, 64)
			markPrice_float64, _ := strconv.ParseFloat(position.MarkPrice, 64)
			nowProfit := (unRealizedProfit / (positionAmtFloatAbs * markPrice_float64)) * leverage_float64 * 100 // 当前收益率(正为盈利，负为亏损)
			env["ROI"] = nowProfit
			env["Position"] = types.FuturesPositionCode{
				Symbol: symbols.Symbol,
				EntryPrice: entryPrice_float64,
				MarkPrice: markPrice_float64,
				Amount:positionAmtFloat,
				UnrealizedProfit: unRealizedProfit,
				Leverage: int64(leverage_float64),
				Side: position.Side,
				Mock: false,
				CreateTime: position.CreateTime,
				SourceType: position.SourceType,
			}
		}
	}
	
	for _, strategy := range strategyConfig {
		if strategy.Enable {
			program, err := expr.Compile(strategy.Code, expr.Env(env))
			if err != nil {
				logs.Error("Error Strategy Compile:", err.Error())
				ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
				return
			}
			output, err := expr.Run(program, env)
			if err != nil {
				logs.Error("Error Strategy Run:", err.Error())
				ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
				return
			}
			ctrl.Ctx.Resp(map[string]interface{} {
				"code": 200,
				"data": map[string]interface{} {
					"pass": output,
					"type": strategy.Type,
				},
				"msg": "success",
			})
		}
	}
}

func (ctrl *FeatureController) GetOptions() {
	o := orm.NewOrm()
	var symbols []models.Symbols
	_, err := o.QueryTable("symbols").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	
	symbolsArr := make([]string, 0, len(symbols))
	for _, symbol := range symbols {
		symbolsArr = append(symbolsArr, symbol.Symbol)
	}
	
	ctrl.Ctx.Resp(map[string]interface{} {
		"code": 200,
		"data": symbolsArr,
		"msg": "success",
	})
}