package controllers

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	fu "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	futuresownership "go_binance_futures/service/futuresownership"
	spot "go_binance_futures/spot/api/binance"
	"go_binance_futures/utils"

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

	symbols.Enable = 0                        // 默认不开启
	symbols.Type = 1                          // 正向套利
	symbols.Leverage = 3                      // 杠杆类型
	symbols.MarginType = "CROSSED"            // 杠杆类型 ISOLATED(逐仓), CROSSED(全仓)
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

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
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

	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
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
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"data": symbols,
		"msg":  "success",
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
	ctrl.Ctx.Resp(map[string]interface{}{
		"code": 200,
		"msg":  "success",
	})
}

// 开启吃资金费率
func (ctrl *EatRateController) Start() {
	id := ctrl.Ctx.Input.Param(":id")
	var symbols models.EatRateSymbols
	o := orm.NewOrm()
	if err := o.QueryTable("eat_rate_symbols").Filter("Id", id).One(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "funding-rate arbitrage config not found"))
		return
	}

	totalAmountFloat, parseErr := strconv.ParseFloat(symbols.TotalAmount, 64)
	if parseErr != nil || totalAmountFloat <= 0 || symbols.Leverage <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "invalid total amount or leverage"))
		return
	}

	futuresAmount := totalAmountFloat / float64(symbols.Leverage) // 合约
	spotAmount := totalAmountFloat - futuresAmount                // 现货
	spotPriceFloat, err := getSpotPrice(symbols.SpotSymbol)       // 现货价格
	if err != nil || spotPriceFloat <= 0 {
		logs.Info("not found spot symbol:", symbols.SpotSymbol)
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "not found spot"))
		return
	}
	spotQuantity := spotAmount / spotPriceFloat                            // 购买数量
	spotQuantity = utils.GetTradePrecision(spotQuantity, symbols.StepSize) // 合理精度的数量

	futuresPriceFloat, err := getFuturesPrice(symbols.FuturesSymbol) // 合约价格
	if err != nil || futuresPriceFloat <= 0 {
		logs.Info("not found futures symbol:", symbols.FuturesSymbol)
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "not found futures"))
		return
	}
	futuresQuantity := futuresAmount / futuresPriceFloat * float64(symbols.Leverage) // 做空数量
	futuresQuantity = utils.GetTradePrecision(futuresQuantity, symbols.StepSize)     // 合理数量精度的价格
	if spotQuantity <= 0 || futuresQuantity <= 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "trade quantity is zero after precision rounding"))
		return
	}
	if err := validateEatRateFuturesOpenSlot(context.Background(), symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

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
		// 做空合约。Ownership 账本统一保存正数量；SELL + SHORT 表示开空。
		res, err := submitEatRateFuturesOpen(context.Background(), symbols, futuresQuantity)
		if err != nil {
			logs.Error("futures sell short fail, symbol:", symbols.FuturesSymbol)
			logs.Error("err:", err.Error())
			return
		}
		futuresOrderId, _ = strconv.ParseInt(res.ExchangeOrderID, 10, 64)
		if res.FilledQty > 0 {
			futuresQuantity = res.FilledQty
		}
	}()
	wg.Wait()

	if spotOrderId == 0 || futuresOrderId == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "buy fail"))
		return
	}

	spotOrder, _ := spot.GetOrder(spot.OrderParams{
		Symbol:  symbols.SpotSymbol,
		OrderID: spotOrderId,
	})

	futuresOrder, _ := fu.GetOrder(fu.OrderParams{
		Symbol:  symbols.FuturesSymbol,
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
	if err := o.QueryTable("eat_rate_symbols").Filter("Id", id).One(&symbols); err != nil {
		ctrl.Ctx.Resp(utils.ResJson(404, nil, "funding-rate arbitrage config not found"))
		return
	}
	closeQty, err := prepareEatRateFuturesClose(context.Background(), symbols)
	if err != nil {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, err.Error()))
		return
	}

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
		// 合约平仓。BUY + SHORT 只减少本条 funding_rate ownership 的 managed quantity。
		res, err := submitEatRateFuturesClose(context.Background(), symbols, closeQty)
		if err != nil {
			logs.Error("futures buy short close fail, symbol:", symbols.FuturesSymbol)
			logs.Error("err:", err.Error())
			return
		}
		futuresOrderId, _ = strconv.ParseInt(res.ExchangeOrderID, 10, 64)
	}()
	wg.Wait()

	if spotOrderId == 0 || futuresOrderId == 0 {
		ctrl.Ctx.Resp(utils.ResJson(400, nil, "sell fail"))
		return
	}

	// 更新数据表
	symbols.Enable = 0
	symbols.EndTime = time.Now().Unix() * 1000
	_, err = o.Update(&symbols)
	if err != nil {
		logs.Error("update fail, symbol:", symbols.Symbol)
		logs.Error("err:", err.Error())
		ctrl.Ctx.Resp(utils.ResJson(200, nil))
		return
	}

	ctrl.Ctx.Resp(utils.ResJson(200, nil))
}

func eatRateSourceRef(row models.EatRateSymbols) string {
	return fmt.Sprintf("eat_rate:%d", row.ID)
}

func validateEatRateFuturesOpenSlot(ctx context.Context, row models.EatRateSymbols) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// This legacy controller used to bypass V3-5. Preserve the same SHORT
	// strategy but refuse to merge with an unmanaged/account position or open
	// order, otherwise ownership could no longer distinguish the quantities.
	positions, err := fu.GetPosition(fu.PositionParams{Symbol: strings.ToUpper(strings.TrimSpace(row.FuturesSymbol))})
	if err != nil {
		return err
	}
	for _, position := range positions {
		qty, _ := strconv.ParseFloat(position.PositionAmt, 64)
		if strings.EqualFold(string(position.PositionSide), "SHORT") && math.Abs(qty) > 1e-12 {
			return fmt.Errorf("%s SHORT already exists in account; funding-rate ownership will not merge it", row.FuturesSymbol)
		}
	}
	openOrders, err := fu.GetOpenOrder()
	if err != nil {
		return err
	}
	for _, order := range openOrders {
		if strings.EqualFold(order.Symbol, row.FuturesSymbol) && strings.EqualFold(string(order.PositionSide), "SHORT") && strings.EqualFold(string(order.Side), "SELL") {
			return fmt.Errorf("%s SHORT already has an account open order; funding-rate ownership will not merge it", row.FuturesSymbol)
		}
	}
	return nil
}

func submitEatRateFuturesOpen(ctx context.Context, row models.EatRateSymbols, quantity float64) (futuresownership.ExchangeOrder, error) {
	if math.IsNaN(quantity) || math.IsInf(quantity, 0) || quantity <= 1e-12 {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("invalid futures quantity %.12f", quantity)
	}
	if err := validateEatRateFuturesOpenSlot(ctx, row); err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	return futuresownership.DefaultExecutor().Execute(ctx, futuresownership.OrderRequest{
		Owner: futuresownership.OwnerFundingRate, Symbol: row.FuturesSymbol, PositionSide: "SHORT",
		Intent: futuresownership.IntentOpen, Side: string(futures.SideTypeSell), OrderType: string(futures.OrderTypeMarket),
		Quantity: quantity, SourceRef: eatRateSourceRef(row),
	})
}

func prepareEatRateFuturesClose(ctx context.Context, row models.EatRateSymbols) (float64, error) {
	ownership := futuresownership.DefaultService()
	position, err := ownership.GetPosition(ctx, futuresownership.OwnerFundingRate, row.FuturesSymbol, "SHORT")
	if err != nil {
		return 0, fmt.Errorf("funding-rate managed SHORT not found: %w", err)
	}
	if position.SourceRef != eatRateSourceRef(row) {
		return 0, fmt.Errorf("managed SHORT source %q does not belong to %q", position.SourceRef, eatRateSourceRef(row))
	}
	positions, err := fu.GetPosition(fu.PositionParams{Symbol: strings.ToUpper(strings.TrimSpace(row.FuturesSymbol))})
	if err != nil {
		return 0, err
	}
	accountQty := 0.0
	for _, account := range positions {
		if !strings.EqualFold(string(account.PositionSide), "SHORT") {
			continue
		}
		qty, parseErr := strconv.ParseFloat(account.PositionAmt, 64)
		if parseErr != nil {
			return 0, parseErr
		}
		accountQty = math.Abs(qty)
		break
	}
	closeQty := math.Min(position.ManagedQty, accountQty)
	if closeQty <= 1e-12 {
		_, _ = ownership.ReconcilePosition(ctx, futuresownership.OwnerFundingRate, row.FuturesSymbol, "SHORT", 0)
		return 0, fmt.Errorf("%s SHORT has no managed account quantity left to close", row.FuturesSymbol)
	}
	return closeQty, nil
}

func submitEatRateFuturesClose(ctx context.Context, row models.EatRateSymbols, requestedQty float64) (futuresownership.ExchangeOrder, error) {
	currentQty, err := prepareEatRateFuturesClose(ctx, row)
	if err != nil {
		return futuresownership.ExchangeOrder{}, err
	}
	closeQty := math.Min(math.Abs(requestedQty), currentQty)
	if closeQty <= 1e-12 {
		return futuresownership.ExchangeOrder{}, fmt.Errorf("invalid managed close quantity %.12f", requestedQty)
	}
	return futuresownership.DefaultExecutor().Execute(ctx, futuresownership.OrderRequest{
		Owner: futuresownership.OwnerFundingRate, Symbol: row.FuturesSymbol, PositionSide: "SHORT",
		Intent: futuresownership.IntentClose, Side: string(futures.SideTypeBuy), OrderType: string(futures.OrderTypeMarket),
		Quantity: closeQty, SourceRef: eatRateSourceRef(row),
	})
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
