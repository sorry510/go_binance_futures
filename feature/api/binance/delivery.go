package binance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"go_binance_futures/service/binanceapiusage"

	"github.com/adshao/go-binance/v2/delivery"
	"github.com/beego/beego/v2/adapter/logs"
	"github.com/beego/beego/v2/client/orm"
)

// GetDeliveryAccount is read-only and always uses the selected DAPI network.
// A missing or mixed-environment credential never falls back to mainnet.
func GetDeliveryAccount() (*delivery.Account, error) {
	return GetDeliveryAccountContext(context.Background())
}

func GetDeliveryAccountContext(ctx context.Context) (*delivery.Account, error) {
	if err := coinMPrivateReadiness(); err != nil {
		return nil, err
	}
	return deliveryClient.NewGetAccountService().Do(ctx)
}

// These are explicit READ-ONLY COIN-M account methods. No CM position-mode
// mutation, leveraged trade request, or signature fallback is exposed in CM-2.
func GetDeliveryBalanceContext(ctx context.Context) ([]*delivery.Balance, error) {
	if err := coinMPrivateReadiness(); err != nil {
		return nil, err
	}
	return deliveryClient.NewGetBalanceService().Do(ctx)
}

func GetDeliveryPositionsContext(ctx context.Context) ([]*delivery.PositionRisk, error) {
	if err := coinMPrivateReadiness(); err != nil {
		return nil, err
	}
	return deliveryClient.NewGetPositionRiskService().Do(ctx)
}

func GetDeliveryPositionModeContext(ctx context.Context) (*delivery.PositionMode, error) {
	if err := coinMPrivateReadiness(); err != nil {
		return nil, err
	}
	return deliveryClient.NewGetPositionModeService().Do(ctx)
}

func GetDeliveryExchangeInfo() (*delivery.ExchangeInfo, error) {
	return GetDeliveryExchangeInfoContext(context.Background())
}

// GetDeliveryExchangeInfoContext is a PUBLIC DAPI call (no API key required).
// It carries the actual environment into the REST weight/order collectors.
func GetDeliveryExchangeInfoContext(ctx context.Context) (*delivery.ExchangeInfo, error) {
	if coinMSetupErr != nil {
		return nil, fmt.Errorf("COIN-M DAPI setup failed: %w", coinMSetupErr)
	}
	if deliveryClient == nil || coinMRuntime.Environment == "" || deliveryClient.BaseURL != coinMRuntime.RESTURL {
		return nil, errors.New("COIN-M market client unavailable or endpoint mismatch")
	}
	info, err := deliveryClient.NewExchangeInfoService().Do(ctx)
	if err != nil {
		return nil, err
	}
	applyCoinMExchangeRateLimits(info, coinMRuntime.Environment)
	return info, nil
}

func applyCoinMExchangeRateLimits(info *delivery.ExchangeInfo, environment string) {
	if info == nil || environment == "" {
		return
	}
	var orders10s, orders1m int64
	for _, limit := range info.RateLimits {
		switch {
		case strings.EqualFold(limit.RateLimitType, "REQUEST_WEIGHT") &&
			strings.EqualFold(limit.Interval, "MINUTE") &&
			limit.IntervalNum == 1 && limit.Limit > 0:
			binanceapiusage.Default().SetWeightLimit("delivery", environment, limit.Limit, "exchange_info")
		case strings.EqualFold(limit.RateLimitType, "ORDERS") &&
			strings.EqualFold(limit.Interval, "SECOND") &&
			limit.IntervalNum == 10 && limit.Limit > 0:
			orders10s = limit.Limit
		case strings.EqualFold(limit.RateLimitType, "ORDERS") &&
			strings.EqualFold(limit.Interval, "MINUTE") &&
			limit.IntervalNum == 1 && limit.Limit > 0:
			orders1m = limit.Limit
		}
	}
	if orders10s > 0 || orders1m > 0 {
		binanceapiusage.Default().SetOrderLimits("delivery", environment, orders10s, orders1m)
	}
}

var deliveryWSStarted atomic.Bool
var deliveryWSActive atomic.Bool
var deliveryWSLastEvent atomic.Int64

// DeliveryWSStarted means a connection was attempted, not that it is healthy.
func DeliveryWSStarted() bool { return deliveryWSStarted.Load() }

// DeliveryMarketWSFresh is for market-price display, NOT authorization for
// private order placement (which will require a separate user-data stream).
func DeliveryMarketWSFresh() bool {
	last := deliveryWSLastEvent.Load()
	return deliveryWSActive.Load() && last > 0 && time.Since(time.UnixMilli(last)) < 30*time.Second
}

// UpdateDeliveryCoinByWs is a market-only COIN-M WS loop. It does not subscribe
// to user-data, manage positions or submit Binance orders.
func UpdateDeliveryCoinByWs() {
	if coinMSetupErr != nil || deliveryClient == nil {
		logs.Warning("COIN-M market WS disabled: selected DAPI environment unavailable")
		return
	}
	if err := validateCoinMWSConfiguration(coinMRuntime); err != nil {
		logs.Error("COIN-M market WS disabled: ", err)
		return
	}
	for {
		deliveryWSStarted.Store(true)
		done, _, err := wsDeliveryAllMarketTickerServe(func(tickers delivery.WsAllMarketTickerEvent) {
			n, writeErr := persistCoinMTickers(orm.NewOrm(), tickers)
			if writeErr != nil {
				logs.Warning("COIN-M ticker persistence: ", writeErr)
			}
			if n > 0 {
				deliveryWSLastEvent.Store(time.Now().UnixMilli())
			}
		}, func(wsErr error) {
			deliveryWSActive.Store(false)
			logs.Warning("COIN-M ticker stream disconnected: ", wsErr)
		})
		if err == nil {
			deliveryWSActive.Store(true)
			logs.Info("COIN-M ticker stream connected: environment=", coinMRuntime.Environment)
			<-done // SDK's 24-hour reconnect (and network disconnect) exits here.
		} else {
			logs.Warning("COIN-M ticker WS connect error: ", err)
		}
		deliveryWSActive.Store(false)
		deliveryWSLastEvent.Store(0)
		time.Sleep(5 * time.Second)
		if err := validateCoinMWSConfiguration(coinMRuntime); err != nil {
			logs.Error("COIN-M ticker WS environment changed; refusing reconnect: ", err)
			return
		}
	}
}
