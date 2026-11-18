package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/beego/beego/v2/client/orm"
	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

// Gateway is intentionally narrow. The production adapter ONLY uses DAPI.
// All acceptance tests use a fake; constructing a processor starts no WS,
// network activity, or trading cycle.
type Gateway interface {
	Ready() bool                                                        // authenticated COIN-M private user data WS, not market WS
	Mode(context.Context) (bool, error)                                 // hedge/dualSidePosition read only
	Spec(context.Context, string) (feature.CoinMPerpetualSpec, error)   // fresh exchangeInfo
	Position(context.Context, string, string) (string, error)           // position contracts
	TradeSettings(context.Context, string, string, int64, string) error // existing mode/leverage, no mutation
	HasOpenOrders(context.Context, string) (bool, error)                // avoid unmanaged exchange orders
	Mark(context.Context, string) (string, error)                       // fresh markPrice
	Available(context.Context, string) (string, error)                  // settlement asset balance
	Create(context.Context, binance.CoinMTradeOrder) (*OrderAck, error)
	Query(context.Context, string, string) (*OrderAck, error)
	Cancel(context.Context, string, string) (*OrderAck, error)
	Trades(context.Context, string, int64) ([]binance.CoinMTrade, error)
	Algo(context.Context, binance.CoinMAlgoOrderRequest) (*AlgoAck, error)
	CancelAlgo(context.Context, string, string) error
	OpenAlgos(context.Context, string) ([]binance.CoinMAlgoOrderResponse, error)
	QueryAlgo(context.Context, string, string) (binance.CoinMAlgoDetail, error)
	OrderByID(context.Context, string, int64) (*OrderAck, error)
}
type OrderAck struct {
	ClientOrderID                      string
	ExchangeOrderID                    int64
	Symbol, Side, PositionSide, Status string
	FilledContracts                    string
}
type AlgoAck struct {
	ClientAlgoID                       string
	AlgoID                             int64
	Symbol, Side, PositionSide, Status string
}
type Processor struct {
	DB  orm.Ormer
	API Gateway
	mu  sync.Mutex
}

// Caller must pass an existing test or application Ormer. Does not register
// a model, sync schema, or start any execution on construction.
func NewProcessor(db orm.Ormer, api Gateway) *Processor { return &Processor{DB: db, API: api} }

var ErrUncertain = errors.New("COIN-M order submission is uncertain; remote query required, never blind retry")

func decimal(s string) (*big.Rat, error) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 100 || strings.ContainsAny(s, "eE/+") {
		return nil, fmt.Errorf("invalid COIN-M decimal %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		return nil, fmt.Errorf("invalid COIN-M decimal %q", s)
	}
	return r, nil
}
func positive(s string) (*big.Rat, error) {
	r, e := decimal(s)
	if e != nil {
		return nil, e
	}
	if r.Sign() <= 0 {
		return nil, errors.New("COIN-M decimal must be positive")
	}
	return r, nil
}
func str(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := strings.TrimRight(strings.TrimRight(r.FloatString(18), "0"), ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}
func floorToStep(q, step *big.Rat) *big.Rat {
	n := new(big.Rat).Quo(q, step)
	k := new(big.Int).Quo(n.Num(), n.Denom())
	return new(big.Rat).Mul(new(big.Rat).SetInt(k), step)
}
func contractsForBudget(budget string, leverage, face int64, step string) (string, error) {
	b, e := positive(budget)
	if e != nil {
		return "", e
	}
	if leverage < 1 || leverage > 125 || face <= 0 {
		return "", errors.New("invalid COIN-M leverage/face value")
	}
	inc, e := positive(step)
	if e != nil {
		return "", e
	}
	notional := new(big.Rat).Mul(b, new(big.Rat).SetInt64(leverage))
	raw := new(big.Rat).Quo(notional, new(big.Rat).SetInt64(face))
	n := floorToStep(raw, inc)
	if n.Sign() <= 0 {
		return "", errors.New("COIN-M budget too small for one MARKET_LOT_SIZE step")
	}
	return str(n), nil
}
func orderSide(side, intent string) (string, error) {
	switch {
	case side == "LONG" && intent == "OPEN":
		return "BUY", nil
	case side == "LONG" && intent == "CLOSE":
		return "SELL", nil
	case side == "SHORT" && intent == "OPEN":
		return "SELL", nil
	case side == "SHORT" && intent == "CLOSE":
		return "BUY", nil
	}
	return "", fmt.Errorf("invalid COIN-M %s %s", side, intent)
}

type OpenRequest struct {
	Symbol        string
	PositionSide  string // LONG / SHORT
	ClientOrderID string // stable caller-supplied ID, 1 intent == 1 ID
	Type          string // MARKET or LIMIT
	Price         string // LIMIT only
}
type CloseRequest struct {
	PositionID    int64
	ClientOrderID string
	Type          string
	Price         string
}
type ProtectionRequest struct {
	PositionID   int64
	ClientAlgoID string
	Type         string // STOP_MARKET or TAKE_PROFIT_MARKET
	TriggerPrice string
}

func (p *Processor) requireReady(ctx context.Context) error {
	if p == nil || p.DB == nil || p.API == nil {
		return errors.New("COIN-M processor unconfigured")
	}
	if ctx == nil || ctx.Err() != nil {
		return errors.New("COIN-M cancelled context")
	}
	if !p.API.Ready() {
		return errors.New("COIN-M authenticated private WS not connected")
	}
	hedged, err := p.API.Mode(ctx)
	if err != nil {
		return fmt.Errorf("COIN-M position mode read: %w", err)
	}
	if !hedged {
		return errors.New("COIN-M requires hedge dualSidePosition; refusing to change shared UM/CM mode")
	}
	return nil
}
func validClientID(s string) error {
	if s == "" || len(s) > 36 {
		return errors.New("COIN-M client ID must be 1-36 characters")
	}
	for _, b := range s {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-", b) {
			return errors.New("invalid COIN-M client ID")
		}
	}
	return nil
}
func validateAck(req binance.CoinMTradeOrder, ack *OrderAck) error {
	if ack == nil || ack.ExchangeOrderID <= 0 || ack.ClientOrderID != req.ClientOrderID ||
		ack.Symbol != req.Symbol || ack.Side != req.Side ||
		ack.PositionSide != req.PositionSide || ack.Status == "" {
		return errors.New("COIN-M exchange response does not match managed order intent")
	}
	return nil
}
func validatePosition(row models.CoinMManagedPosition) error {
	if row.ID <= 0 || row.Symbol == "" || (row.PositionSide != "LONG" && row.PositionSide != "SHORT") {
		return errors.New("invalid managed COIN-M position")
	}
	return nil
}
