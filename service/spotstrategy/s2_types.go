package spotstrategy

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

var clientIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,36}$`)
var signalKeyPattern = regexp.MustCompile(`^[A-Za-z0-9:_-]{1,64}$`)

var (
	ErrLiveWriteDisabled   = errors.New("S2 real Spot writes are disabled")
	ErrReconciliation      = errors.New("spot order reconciliation required")
	ErrDuplicate           = errors.New("duplicate Spot order or pending symbol")
	ErrUnmanagedOrder      = errors.New("unknown Spot order cannot be managed by this strategy")
	ErrTradePageIncomplete = errors.New("Spot trade pagination incomplete")
)

const (
	StateReserved   = "reserved"
	StateSubmitting = "submitting"
	StateUnknown    = "unknown"
	StateNew        = "new"
	StatePartial    = "partially_filled"
	StateFilled     = "filled"
	StateCanceled   = "canceled"
	StateExpired    = "expired"
	StateRejected   = "rejected"
	StateReconcile  = "reconcile_required"
)

type Balance struct{ Free, Locked string }
type Instrument struct {
	Symbol, BaseAsset, QuoteAsset, Status                              string
	SpotAllowed                                                        bool
	LotMin, LotMax, LotStep, MarketLotMin, MarketLotMax, MarketLotStep string
	PriceMin, PriceMax, TickSize, MinNotional, MaxNotional             string
}
type OrderRequest struct {
	AccountID, Owner, Symbol, Side, Type, ClientOrderID string
	Quantity, Price, EstimatePrice                      string
	SignalKey                                           string
}
type OrderSnapshot struct {
	Symbol, OrderID, ClientOrderID, Side, Status string
	RequestedQty, ExecutedQty, ExecutedQuoteQty  string
}
type Trade struct {
	Symbol, OrderID, TradeID, Side                    string
	Qty, Price, QuoteQty, Commission, CommissionAsset string
	Time                                              int64
}

// Client is deliberately minimal. The Binance adapter implements only
// read-only operations. The S2 trade methods are Fake-only and are not
// connected to any HTTP route or live order scheduler.
type Client interface {
	Balances(context.Context, string) (map[string]Balance, error)
	Instrument(context.Context, string) (Instrument, error)
	OpenOrders(context.Context, string, string) ([]OrderSnapshot, error)
	QueryOrder(context.Context, string, string, string, string) (OrderSnapshot, error)
	Trades(context.Context, string, string, string, int64, int) ([]Trade, error)
	Place(context.Context, OrderRequest) (OrderSnapshot, error)
	Cancel(context.Context, string, string, string) error
}

func parseAmount(input string, allowZero bool) (decimal.Decimal, error) {
	v := strings.TrimSpace(input)
	if v == "" || len(v) > 55 || strings.ContainsAny(v, "eE+ -") || !decimalText.MatchString(v) {
		return decimal.Zero, fmt.Errorf("invalid exact decimal %q", input)
	}
	n, err := decimal.NewFromString(v)
	if err != nil || n.IsNegative() || (!allowZero && n.IsZero()) {
		return decimal.Zero, fmt.Errorf("invalid decimal amount %q", input)
	}
	return n, nil
}
func isTerminal(status string) bool {
	switch status {
	case StateFilled, StateCanceled, StateExpired, StateRejected:
		return true
	}
	return false
}
func nextState(exchangeStatus string) (string, error) {
	switch strings.ToUpper(exchangeStatus) {
	case "NEW":
		return StateNew, nil
	case "PARTIALLY_FILLED":
		return StatePartial, nil
	case "FILLED":
		return StateFilled, nil
	case "CANCELED":
		return StateCanceled, nil
	case "EXPIRED", "EXPIRED_IN_MATCH":
		return StateExpired, nil
	case "REJECTED":
		return StateRejected, nil
	default:
		return StateReconcile, fmt.Errorf("unknown exchange order status %q", exchangeStatus)
	}
}
func validateTrade(t Trade, symbol, orderID, side string) error {
	if t.Symbol != symbol || t.OrderID != orderID || t.Side != side || t.TradeID == "" || t.Time <= 0 {
		return fmt.Errorf("%w: trade identity mismatched", ErrReconciliation)
	}
	qty, err := parseAmount(t.Qty, false)
	if err != nil {
		return err
	}
	price, err := parseAmount(t.Price, false)
	if err != nil {
		return err
	}
	quote, err := parseAmount(t.QuoteQty, true)
	if err != nil {
		return err
	}
	fee, err := parseAmount(t.Commission, true)
	if err != nil {
		return err
	}
	if quote.IsZero() && !qty.IsZero() && !price.IsZero() {
		return fmt.Errorf("%w: zero quote for nonzero fill", ErrReconciliation)
	}
	if fee.IsPositive() && strings.TrimSpace(t.CommissionAsset) == "" {
		return fmt.Errorf("%w: fee asset unknown", ErrReconciliation)
	}
	if len(t.CommissionAsset) > 16 {
		return fmt.Errorf("%w: fee asset invalid", ErrReconciliation)
	}
	return nil
}
