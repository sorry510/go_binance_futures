// Package spotexecution isolates the 06-4 Spot execution contract from
// Futures and legacy Spot rush trading. It is not registered in main/routes:
// creating a processor in 06-4 does not enable live trading.
package spotexecution

import (
	"context"
	"errors"
	"fmt"

	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"
)

var (
	ErrDisabled    = errors.New("spot quant execution disabled")
	ErrUncertain   = errors.New("spot exchange order outcome uncertain: reconcile by client_order_id")
	ErrNeedsReview = spotledger.ErrNeedsReview
	ErrNotFound    = errors.New("spot exchange order not found")
)

type Gate interface {
	// Checked at entry AND immediately before every new exchange order.
	Enabled(ctx context.Context, symbol string) (bool, error)
}

// Rules is a verified, currently TRADING Spot symbol. Only MARKET orders
// are implemented for initial automated execution; all values are exact
// decimal strings, never float64.
type Rules struct {
	Symbol, Base, Quote         string
	Trading                     bool
	MarketAllowed               bool
	Price, Step, MinQty, MaxQty string
	MinNotional, MaxNotional    string
}

type Request struct {
	Symbol, Side, Type, ClientOrderID, Quantity string
}

type RemoteOrder struct {
	Symbol, Side, Type, ClientOrderID, ExchangeOrderID string
	Status, OrigQty, ExecutedQty                       string
}

type Trade struct {
	Symbol, Side, OrderID, TradeID      string
	Qty, Price, QuoteQty, Fee, FeeAsset string
	Time                                int64
}

// Exchange deliberately has no Futures contract or margin operations.
// FindOrder: nil,nil is allowed ONLY for an authoritative exchange
// "order not found"; network errors must be returned as errors.
type Exchange interface {
	Rules(context.Context, string) (Rules, error)
	Price(context.Context, string) (string, error)
	Free(context.Context, string) (string, error)
	FindOrder(context.Context, string, string) (*RemoteOrder, error)
	Place(context.Context, Request) (RemoteOrder, error)
	Cancel(context.Context, string, string) (RemoteOrder, error)
	Trades(context.Context, string, string) ([]Trade, error)
}

// LocalLedger is satisfied by the 06-3 Spot-only ledger; no Futures tables.
type LocalLedger interface {
	CreatePosition(symbol, base, quote string) (models.SpotManagedPosition, error)
	GetPosition(id int64) (models.SpotManagedPosition, error)
	CreateOrder(spotledger.OrderIntent) (models.SpotManagedOrder, bool, error)
	GetOrder(id int64) (models.SpotManagedOrder, error)
	ConfirmNeverAccepted(orderID int64) (models.SpotManagedOrder, error)
	ListOrders(positionID int64) ([]models.SpotManagedOrder, error)
	FindOrderByClientID(id string) (models.SpotManagedOrder, error)
	BindExchangeOrderID(id int64, exchangeID string) (models.SpotManagedOrder, error)
	SetExchangeStatus(id int64, status string) (models.SpotManagedOrder, error)
	RecordFill(spotledger.FillEvent) (spotledger.FillResult, error)
	ListFills(positionID int64) ([]models.SpotTradeFill, error)
	Accounting(positionID int64, mark string) (spotledger.Snapshot, error)
	ReviewPosition(positionID int64) ([]string, error)
	ListPositions(spotledger.PositionFilter) (spotledger.PositionPage, error)
}

func validateRequest(req Request) error {
	if req.Symbol == "" || req.ClientOrderID == "" || req.Type != "MARKET" ||
		(req.Side != "BUY" && req.Side != "SELL") || req.Quantity == "" {
		return fmt.Errorf("only named Spot MARKET BUY/SELL quantity orders are supported")
	}
	return nil
}
