package controllers

import (
	fu "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	spot "go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"
	"strconv"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"github.com/beego/beego/v2/server/web"
)

type EatRateController struct {
	web.Controller
}

func (ctrl *EatRateController) Post() {
	symbols := new(models.EatRateSymbols)
	ctrl.BindJSON(&symbols)
	
	var futureSymbols models.Symbols
	orm.NewOrm().QueryTable("symbols").Filter("symbol", symbols.FuturesSymbol).One(&futureSymbols)
	
	symbols.Enable = 0 // 默认不开启
	symbols.Type = 1 // 正向套利
	symbols.Leverage = 3 // 杠杆类型
	symbols.MarginType = "CROSSED" // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
	symbols.StepSize = futureSymbols.StepSize // 数量精度
	symbols.TickSize = futureSymbols.TickSize // 价格精度
	symbols.Profit = 0.0

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
	paramsSymbol := ctrl.GetString("symbol", "")
	
	o := orm.NewOrm()
	var symbols []models.EatRateSymbols
	query := o.QueryTable("eat_rate_symbols")
	if paramsSymbol != "" {
		query = query.Filter("Symbol", paramsSymbol)
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
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.EatRateSymbols
	o := orm.NewOrm()
	o.QueryTable("eat_rate_symbols").Filter("Id", id).One(&symbols)
	
	totalAmountFloat, _ := strconv.ParseFloat(symbols.TotalAmount, 64)
	
	futuresAmount := totalAmountFloat / float64(symbols.Leverage) // 合约
	spotAmount := totalAmountFloat - futuresAmount // 现货
	spotPriceFloat, err := getSpotPrice(symbols.SpotSymbol) // 现货价格
	if err != nil {
		logs.Info("not found spot symbol:", symbols.SpotSymbol)
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "not found spot"))
	}
	spotQuantity := spotAmount / spotPriceFloat  // 购买数量
	spotQuantity = utils.GetTradePrecision(spotQuantity, symbols.StepSize) // 合理精度的数量
		
	futuresPriceFloat, err := getFuturesPrice(symbols.FuturesSymbol) // 合约价格
	if err != nil {
		logs.Info("not found futures symbol:", symbols.FuturesSymbol)
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "not found futures"))
	}
	futuresQuantity := futuresAmount / futuresPriceFloat * float64(symbols.Leverage)  // 做空数量
	futuresQuantity = utils.GetTradePrecision(futuresQuantity, symbols.StepSize) // 合理数量精度的价格
	
	var spotOrderId int64
	var futuresOrderId int64
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 买入现货
		res, err := spot.BuyMarket(symbols.SpotSymbol, spotQuantity)
		if err != nil {
			logs.Error("spot buy fail, symbol:", symbols.SpotSymbol)
			logs.Error("err:", err.Error())
			return
		}
		spotOrderId = res.OrderID
	}()
	go func() {
		defer wg.Done()
		// 做空合约
		res, err := fu.SellMarket(symbols.FuturesSymbol, futuresQuantity, futures.PositionSideTypeShort)
		if err != nil {
			logs.Error("futures sell short fail, symbol:", symbols.FuturesSymbol)
			logs.Error("err:", err.Error())
			return
		}
		futuresOrderId = res.OrderID
	}()
	wg.Wait()
	
	if spotOrderId == 0 || futuresOrderId == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "buy fail"))
		return
	}
	
	spotOrder, _ := spot.GetOrder(spot.OrderParams{
		Symbol: symbols.SpotSymbol,
		OrderID: spotOrderId,
	})
	
	futuresOrder, _ := fu.GetOrder(fu.OrderParams{
		Symbol: symbols.FuturesSymbol,
		OrderID: futuresOrderId,
	})
	
	// 更新数据表
	symbols.Enable = 1
	symbols.SpotAmount = strconv.FormatFloat(spotAmount, 'f', -1, 64)
	symbols.SpotQuantity = spotQuantity
	// symbols.SpotPrice = strconv.FormatFloat(spotPriceFloat, 'f', -1, 64)
	symbols.SpotPrice = spotOrder.Price
	symbols.SpotOrderId = spotOrderId
	
	symbols.FuturesAmount = strconv.FormatFloat(futuresAmount, 'f', -1, 64)
	symbols.FuturesQuantity = futuresQuantity
	// symbols.FuturesPrice = strconv.FormatFloat(futuresPriceFloat, 'f', -1, 64)
	symbols.FuturesPrice = futuresOrder.Price
	symbols.FuturesOrderId = futuresOrderId
	
	nowTime := time.Now().Unix() * 1000
	symbols.StartTime = nowTime
	symbols.LastProfitTime = nowTime
	
	_, err = o.Update(&symbols)
	if err != nil {
		logs.Error("update fail, symbol:", symbols.Symbol)
		logs.Error("err:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(200, nil))
		return
	}
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

// 平仓关闭
func (ctrl *EatRateController) End() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.EatRateSymbols
	o := orm.NewOrm()
	o.QueryTable("eat_rate_symbols").Filter("Id", id).One(&symbols)
	
	var spotOrderId int64
	var futuresOrderId int64
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 卖出现货
		res, err := spot.SellMarket(symbols.SpotSymbol, symbols.SpotQuantity)
		if err != nil {
			logs.Error("spot buy fail, symbol:", symbols.SpotSymbol)
			logs.Error("err:", err.Error())
			return
		}
		spotOrderId = res.OrderID
	}()
	go func() {
		defer wg.Done()
		// 合约平仓
		res, err := fu.BuyMarket(symbols.FuturesSymbol, symbols.FuturesQuantity, futures.PositionSideTypeShort)
		if err != nil {
			logs.Error("futures sell short fail, symbol:", symbols.FuturesSymbol)
			logs.Error("err:", err.Error())
			return
		}
		futuresOrderId = res.OrderID
	}()
	wg.Wait()
	
	if spotOrderId == 0 || futuresOrderId == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "sell fail"))
		return
	}
	
	// 更新数据表
	symbols.Enable = 0
	symbols.EndTime = time.Now().Unix() * 1000
	_, err := o.Update(&symbols)
	if err != nil {
		logs.Error("update fail, symbol:", symbols.Symbol)
		logs.Error("err:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(200, nil))
		return
	}
	
	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

// 现货价格
func getSpotPrice(symbol string) (float64, error) {
	spotPrice, err := spot.GetTickerPrice(symbol)
	if err != nil {
		return 0.0, err
	}
	spotPriceFloat, _ := strconv.ParseFloat(spotPrice[0].Price, 64)
	return spotPriceFloat, err
}

// 合约价格
func getFuturesPrice(symbol string) (float64, error) {
	futuresPrice, err := fu.GetTickerPrice(symbol)
	if err != nil {
		return 0.0, err
	}
	futuresPriceFloat, _ := strconv.ParseFloat(futuresPrice[0].Price, 64)
	return futuresPriceFloat, err
}

