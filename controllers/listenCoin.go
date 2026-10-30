package controllers

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/technology"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
	"github.com/expr-lang/expr"
)

type ListenCoinController struct {
	web.Controller
}

// localListenSymbolRecord contains only the columns needed for the picker.
// Market data refreshers populate symbols/spot_symbols separately; selecting
// an option must never call Binance, even if the local table is empty.
type localListenSymbolRecord struct {
	Symbol string
	Type   string
}

func filterLocalListenSymbolOptions(rows []localListenSymbolRecord) []string {
	options := make([]string, 0, len(rows))
	seen := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		symbol := strings.ToUpper(strings.TrimSpace(row.Symbol))
		quoteAsset := utils.FuturesSymbolType(symbol, "")
		if symbol == "" || quoteAsset == "" {
			continue
		}
		// "type" in both persisted tables holds the quote asset. Old manual
		// entries may not have it populated; the symbol suffix still decides.
		if storedType := strings.ToUpper(strings.TrimSpace(row.Type)); storedType != "" && storedType != quoteAsset {
			continue
		}
		if _, exists := seen[symbol]; exists {
			continue
		}
		seen[symbol] = struct{}{}
		options = append(options, symbol)
	}
	sort.Strings(options)
	return options
}

// loadLocalListenSymbolOptions reads only local symbol metadata.
// marketType: 1=spot (spot_symbols), 2=USD-M futures (symbols).
// A disabled market symbol can be selected for a new listener: its market
// 'enable' flag is unrelated to the listener's own enable flag.
func loadLocalListenSymbolOptions(o orm.Ormer, marketType int64, exactSymbol string) ([]string, error) {
	var rows []localListenSymbolRecord
	switch marketType {
	case 1:
		var stored []models.SpotSymbols
		query := o.QueryTable("spot_symbols")
		if exactSymbol != "" {
			query = query.Filter("symbol", exactSymbol)
		}
		if _, err := query.All(&stored, "Symbol", "Type"); err != nil {
			return nil, err
		}
		for _, row := range stored {
			rows = append(rows, localListenSymbolRecord{Symbol: row.Symbol, Type: row.Type})
		}
	case 2:
		var stored []models.Symbols
		query := o.QueryTable("symbols")
		if exactSymbol != "" {
			query = query.Filter("symbol", exactSymbol)
		}
		if _, err := query.All(&stored, "Symbol", "Type"); err != nil {
			return nil, err
		}
		for _, row := range stored {
			rows = append(rows, localListenSymbolRecord{Symbol: row.Symbol, Type: row.Type})
		}
	default:
		return nil, fmt.Errorf("unsupported listen symbol market type %d", marketType)
	}
	return filterLocalListenSymbolOptions(rows), nil
}

func (ctrl *ListenCoinController) SpotSymbolOptions() {
	options, err := loadLocalListenSymbolOptions(orm.NewOrm(), 1, "")
	if err != nil {
		logs.Error("query local spot_symbols for listen picker:", err)
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "读取本地现货交易对失败"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": options, "msg": "success"})
}

func (ctrl *ListenCoinController) FuturesSymbolOptions() {
	options, err := loadLocalListenSymbolOptions(orm.NewOrm(), 2, "")
	if err != nil {
		logs.Error("query local symbols for futures listen picker:", err)
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "读取本地合约交易对失败"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{"code": 200, "data": options, "msg": "success"})
}

func (ctrl *ListenCoinController) Get() {
	paramsType := ctrl.GetString("type", "")
	paramsSymbol := strings.TrimSpace(ctrl.GetString("symbol"))
	paramsEnable := ctrl.GetString("enable", "")

	o := orm.NewOrm()
	var symbols []models.ListenSymbols
	query := o.QueryTable("listen_symbols")
	if paramsType != "" {
		query = query.Filter("Type", paramsType)
	}
	if paramsSymbol != "" {
		query = query.Filter("symbol__icontains", strings.ToUpper(paramsSymbol))
	}
	if paramsEnable != "" {
		query = query.Filter("enable", paramsEnable)
	}
	_, err := query.OrderBy("ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

	if paramsType == "1" || paramsType == "2" {
		symbolNames := make([]string, 0, len(symbols))
		for _, s := range symbols {
			symbolNames = append(symbolNames, s.Symbol)
		}

		closeMap := make(map[string]string)
		if len(symbolNames) > 0 {
			if paramsType == "1" {
				// Display only the price persisted by the spot market WS writer.
				var spotSymbols []models.SpotSymbols
				if _, readErr := o.QueryTable("spot_symbols").Filter("symbol__in", symbolNames).All(&spotSymbols, "Symbol", "Close"); readErr != nil {
					logs.Error("read spot listener prices:", readErr)
				}
				for _, ss := range spotSymbols {
					closeMap[ss.Symbol] = ss.Close
				}
			} else {
				var futuresSymbols []models.Symbols
				o.QueryTable("symbols").Filter("symbol__in", symbolNames).All(&futuresSymbols, "Symbol", "Close")
				for _, fs := range futuresSymbols {
					closeMap[fs.Symbol] = fs.Close
				}
			}
		}

		type ListenSymbolWithClose struct {
			models.ListenSymbols
			Close string `json:"close"`
		}
		results := make([]ListenSymbolWithClose, 0, len(symbols))
		for _, s := range symbols {
			results = append(results, ListenSymbolWithClose{
				ListenSymbols: s,
				Close:         closeMap[s.Symbol],
			})
		}
		ctrl.Ctx.Resp(map[string]interface{}{
			"code": 200,
			"data": results,
			"msg":  "success",
		})
		return
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
	})
}

func (ctrl *ListenCoinController) Edit() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.ListenSymbols
	o := orm.NewOrm()
	o.QueryTable("listen_symbols").Filter("Id", id).One(&symbols)

	if err := ctrl.BindJSON(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求参数不正确"))
		return
	}
	// Both spot and futures monitoring are exclusively custom strategies.
	// Keep legacy records editable, but never allow an API edit to restore
	// the retired kline_base/kline_kc modes.
	if symbols.Type == 1 || symbols.Type == 2 {
		symbols.ListenType = "custom"
	}

	_, err := o.Update(&symbols) // _ 是受影响的条数
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "修改失败"))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
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
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "success",
	})
}

func (ctrl *ListenCoinController) Post() {
	symbols := new(models.ListenSymbols)
	if err := ctrl.BindJSON(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请求参数不正确"))
		return
	}
	symbols.Symbol = strings.ToUpper(strings.TrimSpace(symbols.Symbol))
	if symbols.Type != 1 && symbols.Type != 2 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "无效的监听市场类型"))
		return
	}
	options, err := loadLocalListenSymbolOptions(orm.NewOrm(), symbols.Type, symbols.Symbol)
	if err != nil {
		logs.Error("validate listener symbol against local table:", err)
		ctrl.Ctx.Resp(utils.ResJson(500, nil, "读取本地交易对失败"))
		return
	}
	if len(options) != 1 || options[0] != symbols.Symbol {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "请选择本地列表中有效的交易对"))
		return
	}
	if orm.NewOrm().QueryTable("listen_symbols").Filter("type", symbols.Type).Filter("symbol", symbols.Symbol).Exist() {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "该交易对已在监听列表中"))
		return
	}

	symbols.Enable = 0           // 默认不开启
	symbols.KlineInterval = "5m" // 默认k线周期
	symbols.ChangePercent = "10" // 3% 默认变化幅度
	symbols.LastNoticeTime = 0   // 最后一次通知时间
	symbols.NoticeLimitMin = 5   // 最小通知间隔
	// Both markets now create custom listeners by default.
	symbols.ListenType = "custom"

	o := orm.NewOrm()
	id, err := o.Insert(symbols)

	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "新增失败"))
		return
	}
	symbols.ID = id

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
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
	ctx := binanceapiusage.WithSource(ctrl.Ctx.Request.Context(), "manual_api")
	kline_1, err := binance.GetKlineDataContext(ctx, symbol, interval1, limit)
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{}{
			"code": 500,
			"msg":  "kline error",
		})
		return
	}

	high1, low1, close1, _ := line.GetLineFloatPrices(kline_1)
	upper1, ma1, lower1, err := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier1) // kc1
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{}{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}
	upper2, _, lower2, err := line.CalculateKeltnerChannels(high1, low1, close1, period, multiplier2) // kc2
	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{}{
			"code": 500,
			"msg":  err.Error(),
		})
		return
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": map[string]interface{}{
			"upper1": upper1,
			"ma1":    ma1,
			"lower1": lower1,
			"upper2": upper2,
			"lower2": lower2,
			"close1": close1,
			"high1":  high1,
			"low1":   low1,
		},
		"msg": "success",
	})
}

func (ctrl *ListenCoinController) GetFundingRates() {
	paramsSort := ctrl.GetString("sort")
	symbol := ctrl.GetString("symbol")

	o := orm.NewOrm()
	var symbols []models.SymbolFundingRates
	query := o.QueryTable("symbol_funding_rates")
	if symbol != "" {
		query = query.Filter("symbol__contains", symbol)
	}
	_, err := query.OrderBy("ID").All(&symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "error"))
	}

	sort.SliceStable(symbols, func(i, j int) bool {
		nowFundingRate1, _ := strconv.ParseFloat(symbols[i].NowFundingRate, 64)
		nowFundingRate2, _ := strconv.ParseFloat(symbols[j].NowFundingRate, 64)
		if paramsSort == "+" {
			return nowFundingRate1 >= nowFundingRate2 // 涨幅从大到小排序
		} else if paramsSort == "-" {
			return nowFundingRate1 < nowFundingRate2 // 涨幅从小到大排序
		} else {
			return true
		}
	})

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
	})
}

func (ctrl *ListenCoinController) EditFundingRates() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.SymbolFundingRates
	o := orm.NewOrm()
	o.QueryTable("symbol_funding_rates").Filter("Id", id).One(&symbols)

	ctrl.BindJSON(&symbols)

	_, err := o.Update(&symbols) // _ 是受影响的条数
	if err != nil {
		// 处理错误
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
	})
}

func (ctrl *ListenCoinController) GetFundingRateHistory() {
	symbol := ctrl.GetString("symbol")

	ctx := binanceapiusage.WithSource(ctrl.Ctx.Request.Context(), "manual_api")
	histories, err := binance.GetFundingRateHistoryContext(ctx, binance.FundingRateParams{
		Symbol: symbol,
		Limit:  200,
	})

	if err != nil {
		ctrl.Ctx.Resp(map[string]interface{}{
			"code": 500,
			"msg":  "binance api error",
		})
		return
	}

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": histories,
		"msg":  "success",
	})
}

func (ctrl *ListenCoinController) TestStrategyRule() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.ListenSymbols
	o := orm.NewOrm()
	o.QueryTable("listen_symbols").Filter("Id", id).One(&symbols)

	ctrl.BindJSON(&symbols)

	var strategyConfig technology.StrategyConfig
	err := json.Unmarshal([]byte(symbols.Strategy), &strategyConfig)
	if err != nil {
		logs.Error("Error unmarshalling JSON:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}
	var env map[string]interface{}
	if symbols.Type == 1 {
		env = line.InitSpotParseEnv(symbols.Symbol, symbols.Technology)
	} else {
		env = line.InitParseEnv(symbols.Symbol, symbols.Technology)
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
			ctrl.Ctx.Resp(map[string]interface{}{
				"code": 200,
				"data": map[string]interface{}{
					"pass": output,
					"type": strategy.Type,
				},
				"msg": "success",
			})
		}
	}
}
