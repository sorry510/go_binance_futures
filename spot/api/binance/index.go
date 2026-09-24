package binance

import (
	"context"
	"strings"

	"go_binance_futures/binanceproxy"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/utils"
	"strconv"
	"time"

	// Loads the global config before the package-level reads below run.
	_ "go_binance_futures/bootstrap"

	"github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
)

var api_key, _ = config.String("binance::api_key")
var api_secret, _ = config.String("binance::api_secret")
var proxy_url, _ = config.String("binance::proxy_url")
var proxyPool *binanceproxy.Pool

var client *binance.Client

func init() {
	var err error
	proxyPool, err = binanceproxy.New(proxy_url)
	if err != nil {
		logs.Error("invalid binance proxy config:", err)
		proxyPool, _ = binanceproxy.New("")
	}

	client = binance.NewClient(api_key, api_secret)
	client.HTTPClient = binanceapiusage.WrapClient(proxyPool.HTTPClient(), binanceapiusage.TransportConfig{
		Product: "spot", Environment: "mainnet", Source: "go_binance",
	})
}

func GetFuturesAccount() (res *binance.Account, err error) {
	res, err = client.NewGetAccountService().Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	logs.Info(utils.ToJson(res))
	return res, err
}

// ExchangeInfo 属于低频静态元数据。全量请求使用长 TTL；新币 Rush 可在
// ticker WS 已确认 Symbol 出现后调用 GetExchangeInfoFreshContext 强制刷新。
func GetExchangeInfo(symbols ...string) (res *binance.ExchangeInfo, err error) {
	ctx := binanceapiusage.WithSource(context.Background(), "spot_exchange_info")
	if len(symbols) > 0 {
		return loadSpotExchangeInfo(ctx, symbols...)
	}
	return getSpotExchangeInfoCached(ctx, func(loadCtx context.Context) (*binance.ExchangeInfo, error) {
		return loadSpotExchangeInfo(loadCtx)
	})
}

func GetExchangeInfoFreshContext(ctx context.Context) (*binance.ExchangeInfo, error) {
	data, err := loadSpotExchangeInfo(ctx)
	if err != nil {
		return nil, err
	}
	storeSpotExchangeInfoCache(data)
	return data, nil
}

func loadSpotExchangeInfo(ctx context.Context, symbols ...string) (*binance.ExchangeInfo, error) {
	res, err := client.NewExchangeInfoService().Symbols(symbols...).Do(ctx)
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	var orderLimit10s, orderLimit1m int64
	for _, rateLimit := range res.RateLimits {
		switch {
		case strings.EqualFold(string(rateLimit.RateLimitType), "REQUEST_WEIGHT") &&
			strings.EqualFold(string(rateLimit.Interval), "MINUTE") &&
			rateLimit.IntervalNum == 1 &&
			rateLimit.Limit > 0:
			binanceapiusage.Default().SetWeightLimit("spot", "mainnet", rateLimit.Limit, "exchange_info")
		case strings.EqualFold(string(rateLimit.RateLimitType), "ORDERS") &&
			strings.EqualFold(string(rateLimit.Interval), "SECOND") &&
			rateLimit.IntervalNum == 10:
			orderLimit10s = rateLimit.Limit
		case strings.EqualFold(string(rateLimit.RateLimitType), "ORDERS") &&
			strings.EqualFold(string(rateLimit.Interval), "MINUTE") &&
			rateLimit.IntervalNum == 1:
			orderLimit1m = rateLimit.Limit
		}
	}
	binanceapiusage.Default().SetOrderLimits("spot", "mainnet", orderLimit10s, orderLimit1m)
	return res, nil
}

// @param symbol 交易对名称，例如：BTCUSDT
// @param interval K线的时间间隔，例如：1m, 3m, 5m, 15m, 30m, 1h等
// @param limit 返回的K线数据条数
// @returns /doc/kine.js
func GetKlineData(symbol string, interval string, limit int) (klines []*binance.Kline, err error) {
	return GetKlineDataContext(context.Background(), symbol, interval, limit)
}

func GetKlineDataContext(ctx context.Context, symbol string, interval string, limit int) (klines []*binance.Kline, err error) {
	return getSpotLiveKlineData(ctx, symbol, interval, limit)
}

// 获取交易价格。全市场 Spot ticker WS fresh 时直接使用本地快照。
func GetTickerPrice(symbol string) (res []*binance.SymbolPrice, err error) {
	return GetTickerPriceContext(binanceapiusage.WithSource(context.Background(), "spot_runtime"), symbol)
}

func GetTickerPriceContext(ctx context.Context, symbol string) (res []*binance.SymbolPrice, err error) {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	source := binanceapiusage.SourceFromContext(ctx)
	if price, ok := GetFreshSpotTickerPrice(symbol); ok {
		binanceapiusage.RecordOptimization(source, "local_ws_hit", 1)
		binanceapiusage.RecordOptimization(source, "prevented_duplicate", 1)
		return []*binance.SymbolPrice{{Symbol: symbol, Price: price}}, nil
	}
	res, err = getSpotTickerRESTCached(ctx, symbol, func(loadCtx context.Context) ([]*binance.SymbolPrice, error) {
		rows, loadErr := client.NewListPricesService().Symbol(symbol).Do(loadCtx)
		if loadErr != nil {
			logs.Error(loadErr)
		}
		return rows, loadErr
	})
	return res, err
}

func BuyLimit(symbol string, quantity float64, price float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(price, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

func BuyMarket(symbol string, quantity float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeBuy).
		Type(binance.OrderTypeMarket).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

func SellLimit(symbol string, quantity float64, price float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeLimit).
		TimeInForce(binance.TimeInForceTypeGTC).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Price(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

func SellMarket(symbol string, quantity float64) (res *binance.CreateOrderResponse, err error) {
	res, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).
		Type(binance.OrderTypeMarket).
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return
	}
	return res, err
}

// 挂止盈单(现货不支持)
// @see https://binance-docs.github.io/apidocs/spot/cn/#trade-3
// @returns /doc/order.js
func OrderTakeProfit(symbol string, quantity float64, stopPrice float64) (order *binance.CreateOrderResponse, err error) {
	order, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).                             // 止盈单是卖出
		Type(binance.OrderTypeTakeProfit).                      // 类型是止盈
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 当触发stopPrice时，STOP_LOSS和TAKE_PROFIT将执行MARKET订单。
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	return order, err
}

// 挂单止损(现货不支持)
// @see https://binance-docs.github.io/apidocs/spot/cn/#trade-3
// @returns /doc/order.js
func OrderStopLoss(symbol string, quantity float64, stopPrice float64) (order *binance.CreateOrderResponse, err error) {
	order, err = client.NewCreateOrderService().
		Symbol(symbol).
		Side(binance.SideTypeSell).                             // 止损单是卖出
		Type(binance.OrderTypeStopLoss).                        // 类型是止损
		StopPrice(strconv.FormatFloat(stopPrice, 'f', -1, 64)). // 当触发stopPrice时，STOP_LOSS和TAKE_PROFIT将执行MARKET订单。
		Quantity(strconv.FormatFloat(quantity, 'f', -1, 64)).
		// TimeInForce(binance.TimeInForceTypeGTC).
		Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}

	return order, err
}

type OrderParams struct {
	Symbol  string
	OrderID int64
}

type ListOrderParams struct {
	OrderParams
	StartTime int64
	EndTime   int64
	Limit     int
}

// 获取历史订单
// @see https://binance-docs.github.io/apidocs/spot/cn/#user_data-36
func GetOrders(listOrderParams ListOrderParams) (res []*binance.Order, err error) {
	service := client.NewListOrdersService()
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
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// 获取某个订单
func GetOrder(orderParams OrderParams) (res *binance.Order, err error) {
	service := client.NewGetOrderService()
	if orderParams.Symbol != "" {
		service = service.Symbol(orderParams.Symbol)
	}
	if orderParams.OrderID != 0 {
		service = service.OrderID(orderParams.OrderID)
	}
	res, err = service.Do(context.Background())
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	return res, err
}

// websocket 订阅全市场最新价格变化，只有币价格变化才会推送(24小时变化)
// @doc https://developers.binance.com/docs/zh-CN/binance-spot-api-docs/web-socket-streams#%E6%8C%89symbol%E7%9A%84%E5%AE%8C%E6%95%B4ticker
var flagWsSpot = 0

func UpdateCoinByWs(retryNum int64) {
	for {
		if retryNum > 0 {
			logs.Info("spot ws restart num:", retryNum)
		}

		// binance.BaseWsMainURL = "wss://testnet.binance.vision/ws"
		var lock = false
		var o = orm.NewOrm()
		runErrCh := make(chan error, 1)
		doneC, _, err := wsSpotAllMarketsStatServe(func(event binance.WsAllMarketsStatEvent) {
			receivedAt := time.Now()
			// Spot market WebSocket is mandatory infrastructure for runtime prices
			// and the local spot_symbols market snapshot.
			storeSpotTickerEvents(event, receivedAt)

			if flagWsSpot == 0 {
				logs.Info("spot ws start")
				flagWsSpot = 1
			}
			if !lock {
				lock = true
				for _, ticker := range event {
					if ticker == nil {
						continue
					}
					o.Raw(
						"UPDATE `spot_symbols` set `percentChange` = ?, `close` = ?, `open` = ?, `low` = ?, `high` = ?, `updateTime` = ?, `baseVolume` = ?, `quoteVolume` = ?, `closeQty` = ?,  `tradeCount` = ?, `lastClose` = close, `lastUpdateTime` = updateTime WHERE `symbol` = ?",
						ticker.PriceChangePercent,
						ticker.LastPrice, // 当前价格
						ticker.OpenPrice,
						ticker.LowPrice,
						ticker.HighPrice,
						ticker.Time,
						ticker.BaseVolume,  // 成交量
						ticker.QuoteVolume, // 成交额
						ticker.CloseQty,    // 最新成交价格上的成交量
						ticker.Count,       // 成交数

						ticker.Symbol,
					).Exec()
				}
				lock = false
			}
		}, func(err error) {
			logs.Error("spot ws run error:", err)
			select {
			case runErrCh <- err:
			default:
			}
		})
		if err != nil {
			flagWsSpot = 0
			logs.Error("spot ws start error:", err)
			retryNum++
			time.Sleep(time.Second * 3)
			continue
		}

		if doneC == nil {
			flagWsSpot = 0
			logs.Error("spot ws closed immediately: done channel is nil")
			retryNum++
			time.Sleep(time.Second * 3)
			continue
		}

		<-doneC
		flagWsSpot = 0

		select {
		case runErr := <-runErrCh:
			if runErr != nil {
				logs.Error("spot ws closed after run error, restarting")
			} else {
				logs.Error("spot ws closed, restarting")
			}
		default:
			logs.Error("spot ws done channel closed, restarting")
		}

		retryNum++
		time.Sleep(time.Second * 3)
	}
}
