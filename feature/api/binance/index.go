package binance

import (
	"context"
	"fmt"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)


var api_key, _ = config.String("binance::api_key")
var api_secret, _ = config.String("binance::api_secret")
var proxy_url, _ = config.String("binance::proxy_url")
var pusher = notify.GetNotifyChannel()

const (
	defaultFastMoveThresholdPct = 20.0
	defaultFastMoveRecoverPct   = 18.0
	defaultFastMoveCooldownSec  = int64(15 * 60)
	defaultFastMoveWindows      = "3m,5m,10m"
	futuresWsFlushInterval      = time.Second
	futuresWsBatchSize          = 500
	wsNoDataAlertThreshold      = 3 * time.Minute
	wsNoDataAlertInterval       = 10 * time.Minute
	wsNoDataCheckInterval       = 30 * time.Second
	futuresOrderTypeTakeProfitMarket futures.OrderType = "TAKE_PROFIT_MARKET"
	futuresOrderTypeStopMarket       futures.OrderType = "STOP_MARKET"
)

type wsPricePoint struct {
	TimeMs int64
	Price  float64
}

type wsFastMoveWindow struct {
	Name       string
	DurationMs int64
}

type wsFastMoveState struct {
	LastNotifyTs int64
	Armed        bool
}

var wsFastMoveWindows = []wsFastMoveWindow{
	{Name: "3m", DurationMs: 3 * 60 * 1000},
	{Name: "5m", DurationMs: 5 * 60 * 1000},
	{Name: "10m", DurationMs: 10 * 60 * 1000},
}

var wsTickerPriceHistory = make(map[string][]wsPricePoint)
var wsFastMoveNoticeState = make(map[string]map[string]*wsFastMoveState)
var wsLatestTickerMap = make(map[string]futures.WsMarketTickerEvent)
var wsLatestTickerMu sync.Mutex
var futuresWsFlushOnce sync.Once

var futuresClient *futures.Client
var deliveryClient *delivery.Client

func init() {
	if proxy_url == "" {
		futuresClient = futures.NewClient(api_key, api_secret)
		deliveryClient = delivery.NewClient(api_key, api_secret)
	} else {
		futuresClient = futures.NewProxiedClient(api_key, api_secret, proxy_url)
		deliveryClient = delivery.NewClient(api_key, api_secret) // 暂不支持代理
	}
}

type OrderParams struct {
	Symbol    string
	OrderID   int64
}

type ListOrderParams struct {
	OrderParams
	StartTime int64
	EndTime   int64
	Limit     int
}

// @returns /doc/futuresAccount.js
func GetFuturesAccount() (res *futures.Account, err error) {
	res, err = futuresClient.NewGetAccountService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// func GetSymbolConfig() (res []*futures.ExchangeInfoSymbol, err error) {
// 	// res, err := futuresClient.NewC
// 	if err != nil {
// 		return nil, err
// 	}
// 	return res, err
// }

type PositionParams struct {
	Symbol string
}

// @returns /doc/position.js
func GetPosition(positionParams PositionParams) (res []*futures.PositionRisk, err error){
	query := futuresClient.NewGetPositionRiskService()
	if (positionParams.Symbol != "") {
		query = query.Symbol(positionParams.Symbol)
	}
	res, err = query.Do(context.Background())
	if err != nil {
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

// @see https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/trade/rest-api/Position-Information-V3
// 这个版本仅返回有持仓或挂单的交易对，但是缺少了 Leverage 字段
func GetPositionV3(positionParams PositionParams) (res []*futures.PositionRiskV3, err error){
	query := futuresClient.NewGetPositionRiskV3Service()
	if (positionParams.Symbol != "") {
		query = query.Symbol(positionParams.Symbol)
	}
	res, err = query.Do(context.Background())
	if err != nil {
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

type IncomeParams struct {
	Symbol     string
	IncomeType string
	StartTime  int64
	EndTime    int64
	Limit      int64
}

// @returns /doc/income.js
func GetIncome(incomeParams IncomeParams) (res []*futures.IncomeHistory, err error){
	query := futuresClient.NewGetIncomeHistoryService()
	if (incomeParams.Symbol != "") {
		query = query.Symbol(incomeParams.Symbol)
	}
	if (incomeParams.IncomeType != "") {
		query = query.IncomeType(incomeParams.IncomeType)
	}
	if (incomeParams.StartTime != 0) {
		query = query.StartTime(incomeParams.StartTime)
	}
	if (incomeParams.EndTime != 0) {
		query = query.EndTime(incomeParams.EndTime)
	}
	if (incomeParams.Limit != 0) {
		query = query.Limit(incomeParams.Limit)
	}
	res, err = query.Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

func GetDepth(symbol string, limits ...int) (res *futures.DepthResponse, err error) {
	limit := 100 // 默认值
    if len(limits) != 0 {
        limit = limits[0]
    }
	res, err = futuresClient.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 获取交易价格
func GetTickerPrice(symbol string) (res []*futures.SymbolPrice, err error) {
	res, err = futuresClient.NewListPricesService().Symbol(symbol).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// limit 5, 10, 20, 50, 100, 500, 1000
// @see https://binance-docs.github.io/apidocs/futures/cn/#38a975b802
func GetDepthAvgPrice(symbol string, limits ...int) (buyPrice float64, sellPrice float64, err error) {
	limit := 50 // 默认值
    if len(limits) != 0 {
        limit = limits[0]
    }
	res, err := futuresClient.NewDepthService().Symbol(symbol).Limit(limit).Do(context.Background())
	if err != nil {
		logs.Error(err)
		return 0.0, 0.0, err
	}
	buyPrice, sellPrice = avgPrice(res)
	// logs.Info(utils.ToJson(res))
	// logs.Info(buyPrice, sellPrice)
	return buyPrice, sellPrice, err
}

func avgPrice(data *futures.DepthResponse) (buyPrice float64, sellPrice float64) {
	var buyNum float64
	var sellNum float64
	for _, item := range data.Bids {
		price1, _ := strconv.ParseFloat(item.Price, 64)
		quantity1, _ := strconv.ParseFloat(item.Quantity, 64)
		buyPrice += price1 * quantity1
		buyNum += quantity1
	}
	for _, item := range data.Asks {
		price2, _ := strconv.ParseFloat(item.Price, 64)
		quantity2, _ := strconv.ParseFloat(item.Quantity, 64)
		sellPrice += price2 * quantity2
		sellNum += quantity2
	}
	if len(data.Bids) == 0 || len(data.Asks) == 0 {
		return 0.0, 0.0
	}
	return buyPrice / buyNum, sellPrice / sellNum
}

// @param symbol 交易对名称，例如：BTCUSDT
// @param interval K线的时间间隔，例如：1m, 3m, 5m, 15m, 30m, 1h等
// @param limit 返回的K线数据条数
// @returns /doc/kine.js
func GetKlineData(symbol string, interval string, limit int) (klines []*futures.Kline, err error) {
	klines, err = futuresClient.NewKlinesService().Symbol(symbol).Interval(interval).Limit(limit).Do(context.Background())
	if err != nil {
		return nil, err
	}
	sort.Slice(klines, func(i, j int) bool {
		return klines[i].OpenTime > klines[j].OpenTime // 按照时间降序()
	})
	// logs.Info(utils.ToJson(klines))
	return klines, err
}

// 限价买入
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func BuyLimit(symbol string, quantity float64, price float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeBuy).
		PositionSide(positionSide).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(price, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

// 限价卖出
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func SellLimit(symbol string, quantity float64, price float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		PositionSide(positionSide).
		Type(futures.OrderTypeLimit).
		TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(price, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

// 市价买入
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func BuyMarket(symbol string, quantity float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeBuy).
		PositionSide(positionSide).
		Type(futures.OrderTypeMarket).
		// TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

// 市价卖出
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func SellMarket(symbol string, quantity float64, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(futures.SideTypeSell).
		PositionSide(positionSide).
		Type(futures.OrderTypeMarket).
		// TimeInForce(futures.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

// 撤销订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-6
func CancelOrder(symbol string, orderId int64) (res *futures.CancelOrderResponse, err error){
	res, err = futuresClient.NewCancelOrderService().Symbol(symbol).OrderID(orderId).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// 交易合约倍数
// @param string symbol
// @param Number 1-125
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-10
func SetLeverage(symbol string, leverage int) (res *futures.SymbolLeverage, err error) {
	res, err = futuresClient.NewChangeLeverageService().Symbol(symbol).Leverage(leverage).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// 合约模式 全仓与 逐仓
// @param string symbol
// @param futures.MarginType Isolated(逐仓), Crossed(全仓)
func SetMarginType(symbol string, marginType futures.MarginType) (err error) {
	err = futuresClient.NewChangeMarginTypeService().Symbol(symbol).MarginType(marginType).Do(context.Background())
	if err != nil {
		return err
	}
	return err
}

// 获取历史订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#user_data-7
func GetOrders(listOrderParams ListOrderParams) (res []*futures.Order, err error) {
	service := futuresClient.NewListOrdersService()
	if listOrderParams.Symbol != "" {
		service = service.Symbol(listOrderParams.Symbol)
	}
	if listOrderParams.OrderID != 0 {
		service = service.OrderID(listOrderParams.OrderID)
	}
	if listOrderParams.StartTime != 0 {
		service = service.StartTime(listOrderParams.StartTime)
	}
	if listOrderParams.EndTime != 0 {
		service = service.EndTime(listOrderParams.EndTime)
	}
	if listOrderParams.Limit != 0 {
		service = service.Limit(listOrderParams.Limit)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// 获取某个订单
func GetOrder(orderParams OrderParams) (res *futures.Order, err error) {
	service := futuresClient.NewGetOrderService()
	if orderParams.Symbol != "" {
		service = service.Symbol(orderParams.Symbol)
	}
	if orderParams.OrderID != 0 {
		service = service.OrderID(orderParams.OrderID)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// @param startTime ex:(time.Now().Unix() - 60 * 60 * 24) // 获取最近1天的交易订单
// @see https://binance-docs.github.io/apidocs/futures/cn/#user_data-7
func GetLimitStartTimeOrders(startTime int64) (res []*futures.Order, err error) {
	service := futuresClient.NewListOrdersService()
	res, err = service.StartTime(startTime).Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// 查看当前全部挂单(权重40)
// @see https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/trade/rest-api/Current-All-Open-Orders
func GetOpenOrder(symbols ...string) (res []*futures.Order, err error) {
	service := futuresClient.NewListOpenOrdersService()
	if len(symbols) > 0 {
		service = service.Symbol(symbols[0])
	}
	res, err = service.Do(context.Background())
	if err != nil {
		return nil, err
	}
	return res, err
}

// 获取交易规则和交易对
// @see https://binance-docs.github.io/apidocs/futures/cn/#0f3f2d5ee7
func GetExchangeInfo()(res *futures.ExchangeInfo, err error) {
	res, err = futuresClient.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		return nil, err
	}
	// logs.Info(utils.ToJson(res))
	return res, err
}

// 挂止盈单
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func OrderTakeProfit(symbol string, stopPrice float64, side futures.SideType, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(positionSide).
		Type(futuresOrderTypeTakeProfitMarket). // 止盈市价单
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 触发价格
		ClosePosition(true). // 是否市价全平(和quantity参数互斥)
		// Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

// 挂单止损
// @see https://binance-docs.github.io/apidocs/futures/cn/#trade-3
// @returns /doc/order.js
func OrderStopLoss(symbol string, stopPrice float64, side futures.SideType, positionSide futures.PositionSideType) (order *futures.CreateOrderResponse, err error) {
	order, err = futuresClient.NewCreateOrderService().
		Symbol(symbol).
		Side(side).
		PositionSide(positionSide).
		Type(futuresOrderTypeStopMarket). // 止损限价单
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 触发价格
		ClosePosition(true). // 是否市价全平(和quantity参数互斥)
		// Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		return nil, err
	}
		
	return order, err
}

type FundingRateParams struct {
	Symbol    string
	StartTime int64
	EndTime   int64
	Limit     int
}

// 最新标记价格和资金费率
// @see https://binance-docs.github.io/apidocs/futures/cn/#69f9b0b2f3
// @returns /doc/fundingRate.js
func GetFundingRate(params FundingRateParams) (res []*futures.PremiumIndex, err error) {
	service := futuresClient.NewPremiumIndexService()
	if params.Symbol != "" {
		service = service.Symbol(params.Symbol)
	}
	res, err = service.Do(context.Background())
	return res, err
}

// 资金费率历史记录(整点时刻4或8h的历史记录) 限制 500/5min/IP
// @see https://binance-docs.github.io/apidocs/futures/cn/#31dbeb24c4
// @returns /doc/fundingRateHistory.js
// 根据时间变化而来的数据，交易对可能不唯一有多条，数据顺序是时间由旧到新
func GetFundingRateHistory(params FundingRateParams) (res []*futures.FundingRate, err error) {
	service := futuresClient.NewFundingRateService()
	if params.Symbol != "" {
		service = service.Symbol(params.Symbol)
	}
	if params.StartTime != 0 {
		service = service.StartTime(params.StartTime)
	}
	if params.EndTime != 0 {
		service = service.EndTime(params.EndTime)
	}
	if params.Limit != 0 {
		service = service.Limit(params.Limit)
	}
	res, err = service.Do(context.Background())
	return res, err
}

// websocket 订阅全市场最新价格变化，只有币价格变化才会推送(24小时变化)
var flagWsFutures = 0

func startFuturesWsFlushTask(systemConfig *models.Config) {
	futuresWsFlushOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(futuresWsFlushInterval)
			defer ticker.Stop()

			for range ticker.C {
				flushLatestWsTickers(systemConfig)
			}
		}()
	})
}

func flushLatestWsTickers(systemConfig *models.Config) {
	wsLatestTickerMu.Lock()
	if len(wsLatestTickerMap) == 0 {
		wsLatestTickerMu.Unlock()
		return
	}

	tickers := make([]futures.WsMarketTickerEvent, 0, len(wsLatestTickerMap))
	for symbol, ticker := range wsLatestTickerMap {
		tickers = append(tickers, ticker)
		delete(wsLatestTickerMap, symbol)
	}
	wsLatestTickerMu.Unlock()

	o := orm.NewOrm()
	for start := 0; start < len(tickers); start += futuresWsBatchSize {
		end := start + futuresWsBatchSize
		if end > len(tickers) {
			end = len(tickers)
		}

		chunk := tickers[start:end]
		query, args := buildBatchUpdateSymbolsSQL(chunk)
		if query == "" {
			continue
		}
		logs.Debug(fmt.Sprintf("futures ws batch update symbols: processing %d/%d", end, len(tickers)))

		if _, err := o.Raw(query, args...).Exec(); err != nil {
			logs.Error("futures ws batch update symbols error:", err)
			wsLatestTickerMu.Lock()
			for _, ticker := range tickers[start:] {
				wsLatestTickerMap[ticker.Symbol] = ticker
			}
			wsLatestTickerMu.Unlock()
			return
		}
	}

	go func() {
		for index := range tickers {
			ticker := &tickers[index]
			priceChangeNotice(systemConfig, ticker)
			fastMoveNoticeByWindow(systemConfig, ticker)
		}
	}()
}

func buildBatchUpdateSymbolsSQL(tickers []futures.WsMarketTickerEvent) (string, []interface{}) {
	if len(tickers) == 0 {
		return "", nil
	}

	valueParts := make([]string, 0, len(tickers))
	args := make([]interface{}, 0, len(tickers)*28)
	for _, ticker := range tickers {
		// logs.Info("ws symbol:", ticker.Symbol, "closePrice:", ticker.ClosePrice)
		suffixType := ""
		if strings.HasSuffix(ticker.Symbol, "USDT") {
			suffixType = "USDT"
		} else if strings.HasSuffix(ticker.Symbol, "FDUSD") {
			suffixType = "FDUSD"
		} else if strings.HasSuffix(ticker.Symbol, "USDC") {
			suffixType = "USDC"
		}

		valueParts = append(valueParts, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			ticker.Symbol,
			ticker.PriceChangePercent,
			ticker.ClosePrice,
			ticker.OpenPrice,
			ticker.LowPrice,
			ticker.HighPrice,
			0,
			ticker.Time,
			ticker.ClosePrice,
			ticker.Time,
			ticker.BaseVolume,
			ticker.QuoteVolume,
			ticker.CloseQty,
			ticker.TradeCount,
			3,
			"CROSSED",
			"",
			"",
			"10",
			"20",
			"20",
			"1d",
			"",
			"",
			"global",
			0,
			0,
			suffixType,
		)
	}

	query := fmt.Sprintf(
		"INSERT INTO `symbols` "+
			"(`symbol`, `percentChange`, `close`, `open`, `low`, `high`, `enable`, `updateTime`, `lastClose`, `lastUpdateTime`, `baseVolume`, `quoteVolume`, `closeQty`, `tradeCount`, `leverage`, `marginType`, `tickSize`, `stepSize`, `usdt`, `profit`, `loss`, `kline_interval`, `technology`, `strategy`, `strategy_type`, `pin`, `sort`, `type`) "+
			"VALUES %s "+
			"ON DUPLICATE KEY UPDATE "+
			"`lastClose` = `close`, "+
			"`lastUpdateTime` = `updateTime`, "+
			"`percentChange` = VALUES(`percentChange`), "+
			"`close` = VALUES(`close`), "+
			"`open` = VALUES(`open`), "+
			"`low` = VALUES(`low`), "+
			"`high` = VALUES(`high`), "+
			"`updateTime` = VALUES(`updateTime`), "+
			"`baseVolume` = VALUES(`baseVolume`), "+
			"`quoteVolume` = VALUES(`quoteVolume`), "+
			"`closeQty` = VALUES(`closeQty`), "+
			"`tradeCount` = VALUES(`tradeCount`)",
		strings.Join(valueParts, ", "),
	)
	return query, args
}

func UpdateCoinByWs(systemConfig *models.Config, retryNum int64) {
	startFuturesWsFlushTask(systemConfig)

	for {
		if retryNum > 0 {
			logs.Info("futures ws restart num:", retryNum)
		}

		runErrCh := make(chan error, 1)
		monitorStopC := make(chan struct{})
		var lastRecvAt atomic.Int64
		var lastAlertAt atomic.Int64
		// futures.WebsocketKeepalive = true
		
		doneC, _, err := futures.WsAllMarketTickerServe(func(event futures.WsAllMarketTickerEvent) {
			lastRecvAt.Store(time.Now().UnixMilli())
			lastAlertAt.Store(0)
			if systemConfig.WsFuturesEnable == 1 {
				if flagWsFutures == 0 {
					logs.Info("futures ws start")
					flagWsFutures = 1
				}
			} else {
				if flagWsFutures == 1 {
					logs.Info("futures ws stop")
					flagWsFutures = 0
				}
				return
			}
			for _, ticker := range event {
				wsLatestTickerMu.Lock()
				wsLatestTickerMap[ticker.Symbol] = *ticker
				wsLatestTickerMu.Unlock()
				// if (ticker.Symbol == "BTCUSDT") {
				// 	logs.Info("futures ws update symbol:", ticker.Symbol)
				// }
			}
		}, func(err error) {
			logs.Error("futures ws run error:", err)
			select {
			case runErrCh <- err:
			default:
			}
		})
		if err != nil {
			flagWsFutures = 0
			logs.Error("futures ws start error:", err)
			retryNum++
			time.Sleep(time.Second * 3) // 3 秒间隔
			continue
		}

		if doneC == nil {
			flagWsFutures = 0
			logs.Error("futures ws closed immediately: done channel is nil")
			retryNum++
			time.Sleep(time.Second * 3)
			continue
		}

		lastRecvAt.Store(time.Now().UnixMilli())
		go watchFuturesWsNoData(systemConfig, &lastRecvAt, &lastAlertAt, monitorStopC)

		<-doneC
		close(monitorStopC)
		flagWsFutures = 0

		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				logs.Error("futures ws closed after run error, restarting")
			} else {
				logs.Error("futures ws closed, restarting")
			}
		default:
			logs.Error("futures ws done channel closed, restarting")
		}

		retryNum++
		time.Sleep(time.Second * 3)
	}
}

func watchFuturesWsNoData(systemConfig *models.Config, lastRecvAt *atomic.Int64, lastAlertAt *atomic.Int64, stopC <-chan struct{}) {
	ticker := time.NewTicker(wsNoDataCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopC:
			return
		case <-ticker.C:
			if systemConfig.WsFuturesEnable != 1 {
				continue
			}

			nowMs := time.Now().UnixMilli()
			lastRecvMs := lastRecvAt.Load()
			if lastRecvMs <= 0 || nowMs-lastRecvMs < wsNoDataAlertThreshold.Milliseconds() {
				continue
			}

			lastAlertMs := lastAlertAt.Load()
			if lastAlertMs > 0 && nowMs-lastAlertMs < wsNoDataAlertInterval.Milliseconds() {
				continue
			}

			if !lastAlertAt.CompareAndSwap(lastAlertMs, nowMs) {
				continue
			}

			noDataMinutes := float64(nowMs-lastRecvMs) / float64(time.Minute/time.Millisecond)
			logs.Error("futures ws no data received for", noDataMinutes, "minutes")
			sendFuturesWsNoDataAlert(noDataMinutes)
		}
	}
}

func sendFuturesWsNoDataAlert(noDataMinutes float64) {
	alertPusher := pusher.SetModuleName("futures_market_listen")
	content := fmt.Sprintf(`
## FuturesWS 无数据告警
#### 超过 %.1f 分钟未收到任何数据包
#### 告警阈值：2 分钟
#### 告警频率：5 分钟一次

> author <sorry510sf@gmail.com>`, noDataMinutes)

	switch p := alertPusher.(type) {
	case notify.DingDing:
		notify.DingDingApi(content, p)
	case notify.Slack:
		notify.SlackApi(content, p)
	default:
		alertPusher.FuturesPriceChangeNotice(notify.FuturesNoticeParams{
			Symbol:        "FuturesWS",
			Title:         " 无数据告警",
			ChangePercent: noDataMinutes,
			Price:         0,
		})
	}
}

var symbolPriceNoticeMap = make(map[string]int64) // 价格变动通知，key: symbol, value: timestamp
func priceChangeNotice(systemConfig *models.Config, ticker *futures.WsMarketTickerEvent) {
	if systemConfig.WsFuturesEnable == 0 || systemConfig.WsFuturesPriceChangeLimit == 0 {
		return
	}
	lastTime, ok := symbolPriceNoticeMap[ticker.Symbol]
	if ok && time.Now().Unix() - lastTime < 3600 * 4 {
		// 4小时内已经通知过了，避免重复通知
		return
	}
	if changePercent, err := strconv.ParseFloat(ticker.PriceChangePercent, 64); err == nil {
		if changePercent >= float64(systemConfig.WsFuturesPriceChangeLimit) || changePercent <= float64(-systemConfig.WsFuturesPriceChangeLimit) {
			closePriceFloat, _ := strconv.ParseFloat(ticker.ClosePrice, 64)
			logs.Info("futures price change notice, symbol:", ticker.Symbol, " changePercent:", changePercent)
			title := lang.Lang("futures.up_or_down")
			pusher.SetModuleName("futures_market_listen").FuturesPriceChangeNotice(notify.FuturesNoticeParams{
				Title:         title,
				Symbol:        ticker.Symbol,
				Price:         closePriceFloat,
				ChangePercent: changePercent,
			})
			direction := "up"
			if changePercent < 0 {
				direction = "down"
			}
			saveMarketNoticeLog(ticker.Symbol, "price_change", "24h", direction, closePriceFloat, 0, changePercent, float64(systemConfig.WsFuturesPriceChangeLimit), title, ticker.Time)
			symbolPriceNoticeMap[ticker.Symbol] = time.Now().Unix()
		}
	}
}

var blackSymbols = map[string]bool{
	"USDCUSDT": true,
	"XAGUSDT":  true,
	"XAUUSDT":  true,
}

func saveMarketNoticeLog(symbol string, noticeType string, window string, direction string, price float64, basePrice float64, changePercent float64, thresholdPercent float64, content string, eventTime int64) {
	if eventTime <= 0 {
		eventTime = time.Now().UnixMilli()
	}

	item := models.FuturesMarketNoticeLog{
		Symbol:           symbol,
		NoticeType:       noticeType,
		Window:           window,
		Direction:        direction,
		Source:           "ws_all_market_ticker",
		Price:            price,
		BasePrice:        basePrice,
		ChangePercent:    changePercent,
		ThresholdPercent: thresholdPercent,
		Content:          content,
		CreateTime:       eventTime,
		UpdateTime:       eventTime,
	}

	if _, err := orm.NewOrm().Insert(&item); err != nil {
		logs.Error("save market notice log error:", err)
	}
}

func fastMoveNoticeByWindow(systemConfig *models.Config, ticker *futures.WsMarketTickerEvent) {
	if systemConfig.WsFuturesEnable == 0 || systemConfig.WsFuturesFastMoveEnable == 0 {
		return
	}
	// 只关注 USDT 交易对，避免一些非主流交易对价格波动过大导致频繁通知
	if ticker == nil || ticker.Symbol == "" || !strings.Contains(ticker.Symbol, "USDT") {
		return
	}
	// 黑名单中的交易对不发送快速波动通知，避免一些稳定币交易对价格波动过大导致频繁通知
	if _, ok := blackSymbols[ticker.Symbol]; ok {
		return
	}
	
	windows := parseFastMoveWindows(systemConfig.WsFuturesFastMoveWindows)
	if len(windows) == 0 {
		return
	}
	maxWindowMs := getMaxWindowMs(windows)
	thresholdPct := getFastMoveThreshold(systemConfig)
	recoverPct := getFastMoveRecover(systemConfig, thresholdPct)
	cooldownSec := getFastMoveCooldownSec(systemConfig)

	price, err := strconv.ParseFloat(ticker.ClosePrice, 64)
	if err != nil || price <= 0 {
		return
	}
	currentTimeMs := ticker.Time
	if currentTimeMs <= 0 {
		currentTimeMs = time.Now().UnixMilli()
	}

	history := append(wsTickerPriceHistory[ticker.Symbol], wsPricePoint{TimeMs: currentTimeMs, Price: price})
	minTimeMs := currentTimeMs - maxWindowMs
	trimIndex := 0
	for i := 0; i < len(history); i++ {
		if history[i].TimeMs >= minTimeMs {
			trimIndex = i
			break
		}
		trimIndex = i + 1
	}
	if trimIndex > 0 {
		history = history[trimIndex:]
	}
	wsTickerPriceHistory[ticker.Symbol] = history

	if _, ok := wsFastMoveNoticeState[ticker.Symbol]; !ok {
		wsFastMoveNoticeState[ticker.Symbol] = make(map[string]*wsFastMoveState)
	}
	stateMap := wsFastMoveNoticeState[ticker.Symbol]
	nowSec := currentTimeMs / 1000
	// logs.Info("fast move check, symbol:", ticker.Symbol, " price:", price, " historyLen:", len(history))

	for _, window := range windows {
		basePrice, ok := findWindowBasePrice(history, currentTimeMs-window.DurationMs)
		if !ok || basePrice <= 0 {
			continue
		}

		changePercent := (price - basePrice) / basePrice * 100
		absChange := changePercent
		if absChange < 0 {
			absChange = -absChange
		}

		state, ok := stateMap[window.Name]
		if !ok {
			state = &wsFastMoveState{Armed: true}
			stateMap[window.Name] = state
		}

		if absChange <= recoverPct {
			state.Armed = true
		}

		if absChange < thresholdPct {
			continue
		}
		if !state.Armed {
			continue
		}
		if state.LastNotifyTs > 0 && nowSec-state.LastNotifyTs < cooldownSec {
			continue
		}

		directionText := "快速上涨"
		if changePercent < 0 {
			directionText = "快速下跌"
		}
		title := fmt.Sprintf(" %s%s(>=%.0f%%)", window.Name, directionText, thresholdPct)
		logs.Info("futures fast move notice, symbol:", ticker.Symbol, " window:", window.Name, " changePercent:", changePercent)
		pusher.SetModuleName("futures_market_listen").FuturesPriceChangeNotice(notify.FuturesNoticeParams{
			Title:         title,
			Symbol:        ticker.Symbol,
			Price:         price,
			ChangePercent: changePercent,
		})
		direction := "up"
		if changePercent < 0 {
			direction = "down"
		}
		saveMarketNoticeLog(ticker.Symbol, "fast_move", window.Name, direction, price, basePrice, changePercent, thresholdPct, title, currentTimeMs)

		state.LastNotifyTs = nowSec
		state.Armed = false
	}
}

func findWindowBasePrice(history []wsPricePoint, targetTimeMs int64) (float64, bool) {
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].TimeMs <= targetTimeMs {
			return history[i].Price, true
		}
	}
	return 0, false
}

func getFastMoveThreshold(systemConfig *models.Config) float64 {
	if systemConfig.WsFuturesFastMoveThreshold > 0 {
		return float64(systemConfig.WsFuturesFastMoveThreshold)
	}
	return defaultFastMoveThresholdPct
}

func getFastMoveRecover(systemConfig *models.Config, threshold float64) float64 {
	if systemConfig.WsFuturesFastMoveRecover > 0 {
		recoverVal := float64(systemConfig.WsFuturesFastMoveRecover)
		if recoverVal >= threshold {
			return threshold - 1
		}
		return recoverVal
	}

	recoverVal := threshold - 2
	if recoverVal < 0 {
		return 0
	}
	return recoverVal
}

func getFastMoveCooldownSec(systemConfig *models.Config) int64 {
	if systemConfig.WsFuturesFastMoveCooldownSec > 0 {
		return int64(systemConfig.WsFuturesFastMoveCooldownSec)
	}
	return defaultFastMoveCooldownSec
}

func parseFastMoveWindows(raw string) []wsFastMoveWindow {
	text := strings.TrimSpace(raw)
	if text == "" {
		text = defaultFastMoveWindows
	}

	items := strings.Split(text, ",")
	res := make([]wsFastMoveWindow, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		name := strings.TrimSpace(item)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		duration, ok := parseWindowDurationMs(name)
		if !ok || duration <= 0 {
			continue
		}
		seen[name] = struct{}{}
		res = append(res, wsFastMoveWindow{Name: name, DurationMs: duration})
	}

	if len(res) == 0 {
		return wsFastMoveWindows
	}
	return res
}

func parseWindowDurationMs(text string) (int64, bool) {
	if len(text) < 2 {
		return 0, false
	}
	unit := text[len(text)-1]
	numText := text[:len(text)-1]
	val, err := strconv.ParseInt(numText, 10, 64)
	if err != nil || val <= 0 {
		return 0, false
	}

	switch unit {
	case 's':
		return val * 1000, true
	case 'm':
		return val * 60 * 1000, true
	case 'h':
		return val * 60 * 60 * 1000, true
	default:
		return 0, false
	}
}

func getMaxWindowMs(windows []wsFastMoveWindow) int64 {
	var maxMs int64
	for _, item := range windows {
		if item.DurationMs > maxMs {
			maxMs = item.DurationMs
		}
	}
	if maxMs <= 0 {
		return 30 * 60 * 1000
	}
	return maxMs
}

// websocket user data 使用
func GetListenKey() (listenKey string, err error) {
	return futuresClient.NewStartUserStreamService().Do(context.Background())
}

func UpdateListenKey(listenKey string) (err error) {
	return futuresClient.NewKeepaliveUserStreamService().ListenKey(listenKey).Do(context.Background())
}

// @see https://binance-docs.github.io/apidocs/futures/cn/#listenkey-user_stream
// 不能实时推送仓位信息，只有仓位变化才会推送, 结合查询接口联合使用
func WsUserData() {
	listenKey, err := GetListenKey()
	if err != nil {
		logs.Error("GetListenKey:", err.Error())
		return
	}
	logs.Info("futures_user_data ws start: auto update db futures position")
	o := orm.NewOrm()
	doneC, _, err := futures.WsUserDataServe(listenKey, func(event *futures.WsUserDataEvent) {
		if (event.Event == "ACCOUNT_UPDATE") {
			for _, v := range event.AccountUpdate.Positions {
				floatAmount, _ := strconv.ParseFloat(v.Amount, 64)
				var position models.FuturesPosition
				var symbols models.Symbols
				o.QueryTable("symbols").Filter("symbol", v.Symbol).One(&symbols)
				o.QueryTable("futures_positions").Filter("symbol", v.Symbol).Filter("side", v.Side).One(&position)
				position.Symbol = v.Symbol
				position.Side = string(v.Side)
				position.Amount = strconv.FormatFloat(floatAmount, 'f', -1, 64) // 清仓时推送持仓数量为 0
				position.MarginType = string(v.MarginType)
				position.Leverage = symbols.Leverage // 杠杆倍数没在这里推送(默认为1), 只能暂时调用接口获取 TODO
				position.IsolatedWallet = v.IsolatedWallet
				position.EntryPrice = v.EntryPrice
				position.MarkPrice = v.MarkPrice
				position.UnrealizedProfit = v.UnrealizedPnL
				position.AccumulatedRealized = v.AccumulatedRealized
				position.MaintenanceMarginRequired = v.MaintenanceMarginRequired
				position.UpdateTime = event.Time
				if position.ID == 0 {
					position.CreateTime = event.Time
					o.Insert(&position)
				} else {
					o.Update(&position)
				}
			}
		}  else if (event.Event == "ORDER_TRADE_UPDATE") {
			order := event.OrderTradeUpdate
			var orderModel models.FuturesOrder
			o.QueryTable("futures_orders").Filter("order_id", order.ID).One(&orderModel)
			orderModel.Symbol = order.Symbol
			orderModel.ClientOrderId = order.ClientOrderID
			orderModel.OrderId = strconv.FormatInt(order.ID, 10)
			orderModel.Side = string(order.Side)
			orderModel.PositionSide = string(order.PositionSide)
			orderModel.Type = string(order.Type)
			orderModel.Status = string(order.Status)
			orderModel.Price = order.OriginalPrice
			orderModel.OrigQty = order.OriginalQty
			orderModel.ExecutedQty = order.AccumulatedFilledQty
			orderModel.AveragePrice = order.AveragePrice
			orderModel.StopPrice = order.StopPrice
			orderModel.CommissionAsset = order.CommissionAsset
			orderModel.Commission = order.Commission
			orderModel.RealizedPnL = order.RealizedPnL
			
			orderModel.UpdateTime = event.Time
			if orderModel.ID == 0 {
				orderModel.CreateTime = event.Time
				o.Insert(&orderModel)
			} else {
				o.Update(&orderModel)
			}
		} else if (event.Event == "ACCOUNT_CONFIG_UPDATE") {
			config := event.AccountConfigUpdate
			if config.Leverage == 0 {
				// 其它推送不处理
				return
			}
			var positionModels []models.FuturesPosition
			o.QueryTable("futures_positions").Filter("symbol", config.Symbol).All(&positionModels)
			for _, positionModel := range positionModels {
				positionModel.Leverage = config.Leverage
				o.Update(&positionModel)
			}
		} else if (event.Event == "listenKeyExpired") {
			// 如果 listenKey 过期，重新获取 listenKey
			logs.Info("futures_user_data ws listenKeyExpired")
			UpdateListenKey(listenKey)
		}
	}, func(err error) {
		logs.Error("futures_user_data ws run error:", err)
		time.Sleep(time.Second * 3) // 3 秒间隔
		WsUserData()
	})
	go func() {
		for {
			time.Sleep(time.Minute * 20) // key 1小时过期， 20 分钟更新一次
			UpdateListenKey(listenKey)
		}	
	}()
	
	if err != nil {
		logs.Error("futures_user_data ws start error:", err)
		time.Sleep(time.Second * 30) // 30 秒间隔
		WsUserData()
		return
	}
	<-doneC
}

// @see https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/websocket-market-streams/Kline-Candlestick-Streams
func WsKlineData(symbol string, interval string, callback func(kline futures.WsKline)) {
	_, _, err := futures.WsKlineServe(symbol, interval, func(event *futures.WsKlineEvent) {
		callback(event.Kline)
	}, func(err error) {
		logs.Error("futures ws kline data error:", err)
	})
	if err != nil {
		logs.Error("futures ws kline data start error:", err)
		return
	}
}

