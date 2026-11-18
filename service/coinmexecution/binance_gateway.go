package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"github.com/adshao/go-binance/v2/common"
	"math/big"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
)

// BinanceGateway is constructed explicitly, never auto-started from init or
// the existing market WS. Private WS readiness is supplied by UserStream.
type BinanceGateway struct{ stream *UserStream }

func NewBinanceGateway(stream *UserStream) *BinanceGateway { return &BinanceGateway{stream: stream} }
func (g *BinanceGateway) Ready() bool                      { return g != nil && g.stream != nil && g.stream.Active() }
func (g *BinanceGateway) Mode(ctx context.Context) (bool, error) {
	mode, err := binance.GetCoinMPositionModeContext(ctx)
	if err != nil {
		return false, err
	}
	if mode == nil {
		return false, errors.New("nil COIN-M position mode")
	}
	return mode.DualSidePosition, nil
}
func (g *BinanceGateway) Spec(ctx context.Context, symbol string) (feature.CoinMPerpetualSpec, error) {
	info, err := binance.GetDeliveryExchangeInfoContext(ctx)
	if err != nil {
		return feature.CoinMPerpetualSpec{}, err
	}
	for _, symbolInfo := range info.Symbols {
		if symbolInfo.Symbol != symbol {
			continue
		}
		return feature.ParseCoinMPerpetualSpec(symbolInfo)
	}
	return feature.CoinMPerpetualSpec{}, fmt.Errorf("COIN-M exchangeInfo lacks active %s perpetual", symbol)
}
func (g *BinanceGateway) Position(ctx context.Context, symbol, side string) (string, error) {
	all, err := binance.GetDeliveryPositionsContext(ctx)
	if err != nil {
		return "", err
	}
	for _, p := range all {
		if p == nil || p.Symbol != symbol || p.PositionSide != side {
			continue
		}
		value, err := decimal(p.PositionAmt)
		if err != nil {
			return "", err
		}
		return str(new(big.Rat).Abs(value)), nil
	}
	return "0", nil
}
func (g *BinanceGateway) Mark(ctx context.Context, symbol string) (string, error) {
	marks, err := binance.GetCoinMMarkPricesContext(ctx)
	if err != nil {
		return "", err
	}
	now := time.Now().UnixMilli()
	for _, m := range marks {
		if m.Symbol == symbol && binance.CoinMMarkTimeValid(m.Time, now) {
			if _, err := positive(m.MarkPrice); err != nil {
				return "", err
			}
			return m.MarkPrice, nil
		}
	}
	return "", errors.New("fresh COIN-M mark price unavailable")
}
func (g *BinanceGateway) Available(ctx context.Context, asset string) (string, error) {
	balances, err := binance.GetDeliveryBalanceContext(ctx)
	if err != nil {
		return "", err
	}
	for _, b := range balances {
		if b != nil && b.Asset == asset {
			return b.AvailableBalance, nil
		}
	}
	return "", fmt.Errorf("COIN-M available %s balance not found", asset)
}
func (g *BinanceGateway) Create(ctx context.Context, r binance.CoinMTradeOrder) (*OrderAck, error) {
	result, err := binance.CreateCoinMOrderContext(ctx, r)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("nil COIN-M order response")
	}
	return &OrderAck{ClientOrderID: result.ClientOrderID, ExchangeOrderID: result.OrderID,
		Symbol: result.Symbol, PositionSide: string(result.PositionSide), Side: string(result.Side),
		Status: string(result.Status), FilledContracts: result.ExecutedQuantity}, nil
}
func (g *BinanceGateway) Query(ctx context.Context, symbol, id string) (*OrderAck, error) {
	o, err := binance.QueryCoinMOrderContext(ctx, symbol, id)
	if err != nil {
		var apiErr *common.APIError
		if errors.As(err, &apiErr) && apiErr.Code == -2013 {
			return nil, nil
		}
		return nil, err
	}
	if o == nil {
		return nil, errors.New("nil COIN-M order query")
	}
	return &OrderAck{ClientOrderID: o.ClientOrderID, ExchangeOrderID: o.OrderID, Symbol: o.Symbol,
		Side: string(o.Side), PositionSide: string(o.PositionSide), Status: string(o.Status),
		FilledContracts: o.ExecutedQuantity}, nil
}
func (g *BinanceGateway) Cancel(ctx context.Context, symbol, id string) (*OrderAck, error) {
	o, err := binance.CancelCoinMOrderContext(ctx, symbol, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, errors.New("nil COIN-M cancel response")
	}
	return &OrderAck{ClientOrderID: o.ClientOrderID, ExchangeOrderID: o.OrderID, Symbol: o.Symbol,
		Side: string(o.Side), PositionSide: string(o.PositionSide), Status: string(o.Status),
		FilledContracts: o.ExecutedQuantity}, nil
}
func (g *BinanceGateway) Trades(ctx context.Context, symbol string, orderID int64) ([]binance.CoinMTrade, error) {
	return binance.CoinMUserTradesContext(ctx, symbol, orderID)
}
func (g *BinanceGateway) Algo(ctx context.Context, r binance.CoinMAlgoOrderRequest) (*AlgoAck, error) {
	result, err := binance.CreateCoinMAlgoContext(ctx, r)
	if err != nil {
		return nil, err
	}
	return &AlgoAck{ClientAlgoID: result.ClientAlgoID, AlgoID: result.AlgoID,
		Symbol: result.Symbol, Side: result.Side, PositionSide: result.PositionSide, Status: result.AlgoStatus}, nil
}
func (g *BinanceGateway) CancelAlgo(ctx context.Context, symbol, id string) error {
	return binance.CancelCoinMAlgoContext(ctx, symbol, id)
}

// TradeSettings never changes shared account configuration. Existing
// leverage/margin mode must match the CM per-symbol config.
func (g *BinanceGateway) TradeSettings(ctx context.Context, symbol, side string, leverage int64, marginType string) error {
	if leverage < 1 || leverage > 125 {
		return errors.New("invalid configured COIN-M leverage")
	}
	positions, err := binance.GetDeliveryPositionsContext(ctx)
	if err != nil {
		return err
	}
	for _, p := range positions {
		if p == nil || p.Symbol != symbol || p.PositionSide != side {
			continue
		}
		n, err := strconv.ParseInt(p.Leverage, 10, 64)
		if err != nil {
			return err
		}
		if n != leverage {
			return fmt.Errorf("COIN-M exchange leverage %d differs from configured %d; configure Binance before opening", n, leverage)
		}
		actual := strings.ToUpper(p.MarginType)
		if actual == "CROSS" {
			actual = "CROSSED"
		}
		if actual != marginType {
			return fmt.Errorf("COIN-M exchange margin type %s differs from configured %s; configure Binance before opening", p.MarginType, marginType)
		}
		return nil
	}
	return errors.New("COIN-M exchange position settings unavailable")
}
func (g *BinanceGateway) HasOpenOrders(ctx context.Context, symbol string) (bool, error) {
	all, err := binance.GetCoinMOpenOrdersContext(ctx, symbol)
	if err != nil {
		return false, err
	}
	return len(all) > 0, nil
}

func (g *BinanceGateway) OpenAlgos(ctx context.Context, symbol string) ([]binance.CoinMAlgoOrderResponse, error) {
	return binance.ListOpenCoinMAlgosContext(ctx, symbol)
}

func (g *BinanceGateway) QueryAlgo(ctx context.Context, symbol, clientID string) (binance.CoinMAlgoDetail, error) {
	return binance.QueryCoinMAlgoContext(ctx, symbol, clientID)
}
func (g *BinanceGateway) OrderByID(ctx context.Context, symbol string, orderID int64) (*OrderAck, error) {
	order, err := binance.QueryCoinMOrderByIDContext(ctx, symbol, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, errors.New("COIN-M child order query was empty")
	}
	return &OrderAck{ClientOrderID: order.ClientOrderID, ExchangeOrderID: order.OrderID,
		Symbol: order.Symbol, Side: string(order.Side), PositionSide: string(order.PositionSide),
		Status: string(order.Status), FilledContracts: order.ExecutedQuantity}, nil
}
