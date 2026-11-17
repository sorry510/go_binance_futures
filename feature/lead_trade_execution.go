package feature

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/beego/beego/v2/core/logs"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/feature/strategy"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	"go_binance_futures/service/futuresownership"
	"go_binance_futures/types"
)

// Lead-03: explicit bridge from the shared strategy runner to the existing
// LeadExecutionAdapter; the adapter (not this layer) owns authorization, order
// reservation, exchange writes and Fill reconciliation. The production runner
// remains mock-only until a separately reviewed startup/authorization stage.
type leadStrategyOrderPort interface {
	BoundTo(*binance.AccountClient) bool
	PreflightStrategyOpen(context.Context, *models.Symbols, string, float64, float64, string) error
	ExecuteStrategyOpen(context.Context, futuresownership.OrderRequest, *models.Symbols, float64) (futuresownership.ExchangeOrder, error)
	ExecuteManagedClose(context.Context, futuresownership.OrderRequest) (futuresownership.ExchangeOrder, error)
	CancelExpiredStrategyOpens(context.Context, int64) error
}

var ErrLeadStrategyTicket = errors.New("Lead opening attempt lacks a matching verified preflight")

type leadOpenTicket struct {
	coin           models.Symbols
	symbol, side   string
	orderType      string
	quantity       float64
	referencePrice float64
}

func leadFinitePositive(value float64) bool {
	return value > 0 && !math.IsNaN(value) && !math.IsInf(value, 0)
}

func leadExchangeReceipt(receipt futuresownership.ExchangeOrder) (*futures.CreateOrderResponse, error) {
	id, err := strconv.ParseInt(receipt.ExchangeOrderID, 10, 64)
	if err != nil || id <= 0 || !strings.HasPrefix(receipt.ClientOrderID, "lead_") ||
		(receipt.Status != "NEW" && receipt.Status != "PARTIALLY_FILLED" && receipt.Status != "FILLED") {
		return nil, ErrLeadStrategyTicket
	}
	// A Binance acknowledgement is NOT a confirmed Fill. The managed ledger
	// independently verifies and applies cumulative fills during reconciliation.
	return &futures.CreateOrderResponse{OrderID: id, ClientOrderID: receipt.ClientOrderID}, nil
}

// bindLeadOrderHooks configures one Lead strategy cycle. It does NOT call
// runAccountTradeCycle, launch a scheduler or grant the Stage 7 trade permit.
func bindLeadOrderHooks(r *accountTradeRunner, io *leadTradeIO, port leadStrategyOrderPort) error {
	if r == nil || r.AccountID != binance.LeadAccountID ||
		r.Config == nil || r.LeadSymbols == nil || io == nil || !io.valid() ||
		!r.LeadSymbols.BoundTo(io.account) || port == nil || !port.BoundTo(io.account) {
		return errLeadTradeIOUnverified
	}
	if err := io.bindReadHooks(r); err != nil {
		return err
	}
	r.SyncPositions = func(live []types.FuturesPosition) ([]ownedTradePosition, error) {
		// The previous cycle cannot silently carry a stale quarantine.
		r.leadPendingSymbols = nil
		owned, pending, err := io.managedPositionsBySymbol(context.Background(), live)
		if err != nil {
			return nil, err
		}
		r.leadPendingSymbols = pending
		return owned, nil
	}

	var mu sync.Mutex
	tickets := map[string]leadOpenTicket{}
	key := func(symbol, side string) string {
		return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.ToUpper(side)
	}
	r.LoadSymbols = GetAllSymbols
	r.SelectCoins = func(cfg *models.Config, all []*models.Symbols) ([]*models.Symbols, error) {
		return selectLeadTradeCoins(cfg, all, r.LeadSymbols)
	}
	r.EvaluateEntry = func(coin *models.Symbols, positions []types.FuturesPosition) strategy.OpenResult {
		return (line.TradeLineCustom{}).GetCanLongOrShortWithPositions(strategy.OpenParams{Symbols: coin}, positions)
	}
	r.EvaluateExit = evaluateSharedTradeExit
	r.Depth = func(ctx context.Context, symbol string, limit int) (float64, float64, error) {
		return binance.GetDepthAvgPriceContext(ctx, symbol, limit)
	}
	// Lead CheckOpen validates existing leverage/margin against the actual
	// exchange account. Never use Main's mutable trade-config API client.
	r.EnsureConfig = func(context.Context, *models.Symbols) error { return nil }
	// Same Main timeout policy, but only the Lead-bound managed order
	// executor is allowed to cancel; never invoke Main's global helper.
	r.CancelExpired = func(timeoutSeconds int64) {
		if timeoutSeconds <= 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := port.CancelExpiredStrategyOpens(ctx, timeoutSeconds); err != nil {
			logs.Warning("Lead managed opening order cancellation deferred: ", err)
		}
	}
	r.PreflightOpen = func(ctx context.Context, coin *models.Symbols, side futures.PositionSideType, qty, quote float64) error {
		// No previous ticket may survive ANY failed preflight, even when
		// the new coin/quantity/price is invalid before a key is available.
		mu.Lock()
		clear(tickets)
		mu.Unlock()
		if err := ctx.Err(); err != nil {
			return err
		}
		if coin == nil || !leadFinitePositive(qty) || !leadFinitePositive(quote) {
			return ErrLeadStrategyTicket
		}
		symbol := strings.ToUpper(strings.TrimSpace(coin.Symbol))
		sideName := string(side)
		slot := key(symbol, sideName)
		orderType := r.Config.FutureOrderType
		if !r.AllowNewOpens || r.leadPendingSymbols[symbol] || !r.LeadSymbols.BoundTo(io.account) || !r.LeadSymbols.Allows(symbol) ||
			(side != futures.PositionSideTypeLong && side != futures.PositionSideTypeShort) ||
			(orderType != "MARKET" && orderType != "LIMIT") {
			return ErrLeadStrategyTicket
		}
		preCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := port.PreflightStrategyOpen(preCtx, coin, sideName, qty, quote, orderType); err != nil {
			return err
		}
		mu.Lock()
		tickets[slot] = leadOpenTicket{
			coin: *coin, symbol: symbol, side: sideName,
			orderType: orderType, quantity: qty, referencePrice: quote,
		}
		mu.Unlock()
		return nil
	}
	r.SubmitOpen = func(snapshot *tradeCycleAccountSnapshot, symbol string, quantity, price float64,
		side futures.SideType, positionSide futures.PositionSideType, orderType futures.OrderType, hash string,
	) (*futures.CreateOrderResponse, error) {
		slot := key(symbol, string(positionSide))
		mu.Lock()
		ticket, ok := tickets[slot]
		delete(tickets, slot) // preflight is single-use, even if submit fails
		mu.Unlock()
		if snapshot == nil || !ok || ticket.symbol != symbol || ticket.side != string(positionSide) ||
			ticket.orderType != string(orderType) || ticket.orderType != r.Config.FutureOrderType ||
			!leadFinitePositive(quantity) || ticket.quantity != quantity || !r.AllowNewOpens ||
			r.leadPendingSymbols[ticket.symbol] || !r.LeadSymbols.BoundTo(io.account) || !r.LeadSymbols.Allows(ticket.symbol) ||
			(orderType == futures.OrderTypeMarket && price != 0) ||
			(orderType == futures.OrderTypeLimit && price != ticket.referencePrice) ||
			(positionSide == futures.PositionSideTypeLong && side != futures.SideTypeBuy) ||
			(positionSide == futures.PositionSideTypeShort && side != futures.SideTypeSell) ||
			(orderType != futures.OrderTypeMarket && orderType != futures.OrderTypeLimit) {
			return nil, ErrLeadStrategyTicket
		}
		request := futuresownership.OrderRequest{
			Owner:  futuresownership.OwnerAutoStrategy,
			Symbol: ticket.symbol, PositionSide: ticket.side, Intent: futuresownership.IntentOpen,
			Side: string(side), OrderType: ticket.orderType,
			Quantity: quantity, Price: price,
			SourceRef: autoStrategySourceRef(ticket.symbol, hash),
		}
		// Final authorization/risk/ownership reservation happens inside
		// ExecuteStrategyOpen, not at the earlier strategy signal evaluation.
		execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		receipt, err := port.ExecuteStrategyOpen(execCtx, request, &ticket.coin, ticket.referencePrice)
		if err != nil {
			return nil, err
		}
		response, err := leadExchangeReceipt(receipt)
		if err != nil {
			return nil, err
		}
		snapshot.RecordPendingOpen(ticket.symbol, side, positionSide, response.OrderID)
		return response, nil
	}
	r.SubmitClose = func(owner, sourceRef, symbol string, quantity float64, positionSide futures.PositionSideType) (*futures.CreateOrderResponse, error) {
		if owner != futuresownership.OwnerAutoStrategy || !leadFinitePositive(quantity) ||
			strings.TrimSpace(symbol) == "" || strings.TrimSpace(sourceRef) == "" ||
			(positionSide != futures.PositionSideTypeLong && positionSide != futures.PositionSideTypeShort) {
			return nil, ErrLeadStrategyTicket
		}
		side := futures.SideTypeSell
		if positionSide == futures.PositionSideTypeShort {
			side = futures.SideTypeBuy
		}
		request := futuresownership.OrderRequest{
			Owner: owner, SourceRef: sourceRef, Symbol: strings.ToUpper(strings.TrimSpace(symbol)),
			PositionSide: string(positionSide), Side: string(side),
			Intent: futuresownership.IntentClose, OrderType: string(futures.OrderTypeMarket), Quantity: quantity,
		}
		// In contrast to opening, do NOT require AllowNewOpens or a currently
		// allowed Lead whitelist. Existing verified Managed exits must continue.
		execCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		receipt, err := port.ExecuteManagedClose(execCtx, request)
		if err != nil {
			return nil, err
		}
		return leadExchangeReceipt(receipt)
	}
	// No "order opened" notification on acceptance alone; a real Fill must
	// be confirmed before reporting a successful trade in Lead-03-3.
	r.NotifyOpen = func(notify.FuturesOrderParams) {}
	r.NotifyClose = func(notify.FuturesOrderParams) {}
	r.Sleep = time.Sleep
	return nil
}
