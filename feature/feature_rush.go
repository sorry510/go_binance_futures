package feature

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/lang"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/service/binanceapiusage"
	"go_binance_futures/utils"
	"math"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

var flagFuturesRush = 0

const (
	futuresRushExchangeInfoRefreshInterval = 2 * time.Second
	futuresRushFallbackProbeInterval       = 5 * time.Second
)

var futuresRushExchangeInfoLastRefreshAt atomic.Int64

func TryRush(systemConfig models.Config) {
	if systemConfig.FutureNewEnable == 1 {
		if flagFuturesRush == 0 {
			logs.Info("futures rush bot start")
			flagFuturesRush = 1
		}
	} else {
		if flagFuturesRush == 1 {
			logs.Info("futures rush bot stop")
			flagFuturesRush = 0
		}
		return
	}
	o := orm.NewOrm()
	var coins []models.NewSymbols
	o.QueryTable("new_symbols").OrderBy("ID").Filter("enable", 1).Filter("type", 2).All(&coins) // 允许抢购的合约币

	notHasSizeSymbols := []string{}

	for _, coin := range coins {
		if coin.StepSize != "0" {
			_, err := tryBuyMarket(coin, coin.StepSize)
			if err == nil {
				coin.Enable = 0 // 更新为禁用
			}
			orm.NewOrm().Update(&coin)
		} else {
			notHasSizeSymbols = append(notHasSizeSymbols, coin.Symbol)
		}
	}
	if len(notHasSizeSymbols) == 0 {
		// logs.Info("没有币需要更新交易精度")
		return
	}

	// Do not poll ExchangeInfo every 100ms while a contract is not listed.
	// The all-market ticker WS is the listing signal; only after at least one
	// target symbol is visible do we force-refresh ExchangeInfo for lot size.
	hasFreshListedSymbol := false
	for _, symbol := range notHasSizeSymbols {
		if _, ok := binance.GetFreshFuturesTickerPrice(symbol); ok {
			hasFreshListedSymbol = true
			break
		}
	}
	probeInterval := futuresRushExchangeInfoRefreshInterval
	if binance.FuturesTickerWSFresh() {
		if !hasFreshListedSymbol {
			return
		}
	} else {
		// WS is disabled/stale: retain Rush functionality, but never return to
		// the legacy 100ms ExchangeInfo polling loop.
		probeInterval = futuresRushFallbackProbeInterval
	}
	if !reserveFuturesRushExchangeInfoRefresh(probeInterval) {
		return
	}

	ctx := binanceapiusage.WithSource(context.Background(), "new_coin_rush")
	res, err := binance.GetExchangeInfoFreshContext(ctx)
	if err != nil {
		logs.Error("GetExchangeInfoError:", err)
		return
	}
	symbolMap := make(map[string]string)
	for _, item := range res.Symbols {
		lotSizeFilter := item.LotSizeFilter()
		if lotSizeFilter != nil {
			symbolMap[item.Symbol] = lotSizeFilter.StepSize
		}
	}

	for _, coin := range coins {
		// 找到了币的精度，说明币可能上线了
		if stepSize, ok := symbolMap[coin.Symbol]; ok {
			logs.Info("lotSize:", stepSize)
			_, err := tryBuyMarket(coin, stepSize)

			if err == nil {
				coin.Enable = 0 // 更新为禁用
				logs.Info("合约抢购成功，关闭交易")
			}
			coin.StepSize = stepSize
			orm.NewOrm().Update(&coin)
		} else {
			logs.Info("还未上线此合约币种,未确定交易价格数量精度:", coin.Symbol)
		}
	}
}

func tryBuyMarket(coin models.NewSymbols, stepSize string) (res *futures.CreateOrderResponse, err error) {
	symbol := coin.Symbol
	side := strings.ToLower(strings.TrimSpace(coin.Side))
	if side != "buy" && side != "sell" {
		return nil, fmt.Errorf("unsupported futures rush side %q for %s", coin.Side, symbol)
	}
	usdt := coin.Usdt
	usdt_float64, _ := strconv.ParseFloat(usdt, 64) // 交易金额
	buyPrice := 0.0
	if coin.ExpectPrice != "0" {
		// 定义的挂单价格
		buyPrice, _ = strconv.ParseFloat(coin.ExpectPrice, 64) // 挂单价格
	} else {
		// 获取最新的交易价格
		priceCtx := binanceapiusage.WithSource(context.Background(), "new_coin_rush")
		resPrice, err1 := binance.GetTickerPriceContext(priceCtx, symbol)
		if err1 != nil {
			logs.Info("还未上线此合约币种,未确定交易价格symbol:", symbol)
			return nil, err1
		}
		buyPrice, _ = strconv.ParseFloat(resPrice[0].Price, 64) // 预计交易价格
		if buyPrice < 0.000000001 {
			logs.Info("还未正式上线此合约币种,没有交易盘价格", symbol)
			return nil, errors.New("无交易价格")
		}
	}
	if math.IsNaN(buyPrice) || math.IsInf(buyPrice, 0) || buyPrice < 0.000000001 {
		return nil, fmt.Errorf("invalid futures rush price %q for %s", coin.ExpectPrice, symbol)
	}
	logs.Info("尝试开始合约抢币symbol:", symbol)
	logs.Info("预计交易价格为:", buyPrice)
	// Rush runs every 100ms. Reuse a recently confirmed identical margin/leverage
	// configuration so failed order attempts do not resend both mutations every tick.
	marginType := futures.MarginTypeCrossed
	if coin.MarginType == "ISOLATED" {
		marginType = futures.MarginTypeIsolated
	}
	configCtx := binanceapiusage.WithSource(context.Background(), "new_coin_rush")
	if configErr := binance.EnsureTradeConfigContext(configCtx, symbol, marginType, int(coin.Leverage)); configErr != nil {
		logs.Warning("rush trade config refresh:", symbol, configErr)
	}

	leverage_float64 := float64(coin.Leverage)               // 合约倍数
	quantity := (usdt_float64 / buyPrice) * leverage_float64 // 购买数量
	quantity = utils.GetTradePrecision(quantity, stepSize)   // 合理精度的数量
	// logs.Info("symbol:", symbol, "buyPrice:", buyPrice, "quantity:", quantity)

	if side == "buy" {
		if coin.ExpectPrice != "0" {
			// 挂单价格
			res, err = submitNewCoinRushOpen(fmt.Sprintf("new_coin_rush:%d", coin.ID), symbol, quantity, buyPrice, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeLimit)
		} else {
			// 市价
			res, err = submitNewCoinRushOpen(fmt.Sprintf("new_coin_rush:%d", coin.ID), symbol, quantity, 0, futures.SideTypeBuy, futures.PositionSideTypeLong, futures.OrderTypeMarket)
		}
	} else {
		if coin.ExpectPrice != "0" {
			// 挂单价格
			res, err = submitNewCoinRushOpen(fmt.Sprintf("new_coin_rush:%d", coin.ID), symbol, quantity, buyPrice, futures.SideTypeSell, futures.PositionSideTypeShort, futures.OrderTypeLimit)
		} else {
			// 市价
			res, err = submitNewCoinRushOpen(fmt.Sprintf("new_coin_rush:%d", coin.ID), symbol, quantity, 0, futures.SideTypeSell, futures.PositionSideTypeShort, futures.OrderTypeMarket)
		}
	}

	positionSide := "long"
	if side == "sell" {
		positionSide = "short"
	}
	if err != nil {
		logs.Info("rush error symbol: ", symbol)
		logs.Info("err in feature_rush: ", err.Error())
	} else {
		pusher.SetModuleName("new_coin_rush").FuturesOpenOrder(notify.FuturesOrderParams{
			Title:        lang.Lang("futures.new_coin_rush_notice_title"),
			Symbol:       symbol,
			Side:         side,
			PositionSide: positionSide,
			Price:        buyPrice,
			Quantity:     quantity,
			Leverage:     leverage_float64,
			Status:       "success",
		})
	}
	return res, err
}

func reserveFuturesRushExchangeInfoRefresh(interval time.Duration) bool {
	if interval <= 0 {
		interval = futuresRushExchangeInfoRefreshInterval
	}
	now := time.Now().UnixMilli()
	for {
		last := futuresRushExchangeInfoLastRefreshAt.Load()
		if last > 0 && now-last < interval.Milliseconds() {
			return false
		}
		if futuresRushExchangeInfoLastRefreshAt.CompareAndSwap(last, now) {
			return true
		}
	}
}
