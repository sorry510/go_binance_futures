package spotexecution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
	"github.com/shopspring/decimal"
)

// Processor has no timer, scanner, HTTP route, main.go registration or
// autostart. Live Exchange must be explicitly supplied by a later stage.
type Processor struct {
	ex     Exchange
	gate   Gate
	ledger LocalLedger
	mu     sync.Mutex // one in-process submit/reconcile at a time
}

func NewProcessor(ex Exchange, gate Gate, ledger LocalLedger) (*Processor, error) {
	if ex == nil || gate == nil || ledger == nil {
		return nil, fmt.Errorf("Spot executor requires exchange, live gate and ledger")
	}
	return &Processor{ex: ex, gate: gate, ledger: ledger}, nil
}

func (p *Processor) enabled(ctx context.Context, symbol string) error {
	enabled, err := p.gate.Enabled(ctx, symbol)
	if err != nil {
		return fmt.Errorf("%w: cannot confirm Spot Quant switches: %v", ErrDisabled, err)
	}
	if !enabled {
		return ErrDisabled
	}
	return nil
}
func (p *Processor) market(ctx context.Context, symbol string) (Rules, string, error) {
	r, err := p.ex.Rules(ctx, symbol)
	if err != nil {
		return r, "", err
	}
	if r.Symbol != symbol || r.Base+r.Quote != symbol || !(r.Quote == "USDT" || r.Quote == "USDC" || r.Quote == "FDUSD") {
		return r, "", fmt.Errorf("invalid active Spot trading pair identity")
	}
	price, err := p.ex.Price(ctx, symbol)
	if err != nil {
		return r, "", err
	}
	if _, err = parseAmount(price, true); err != nil {
		return r, "", err
	}
	return r, price, nil
}
func validClientID(id string) bool {
	if len(id) == 0 || len(id) > 36 {
		return false
	}
	for _, r := range id {
		if !(r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return false
		}
	}
	return true
}

// ExecuteBuy creates a local order and sends at most ONE Spot MARKET BUY.
// A repeated client ID only queries/reconciles an existing exchange order.
// It is never resent after an ambiguous network result.
func (p *Processor) ExecuteBuy(ctx context.Context, symbol, budget, clientID string) (models.SpotManagedOrder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var empty models.SpotManagedOrder
	if !validClientID(clientID) {
		return empty, fmt.Errorf("invalid Spot Binance client id (max 36 chars)")
	}
	if err := p.enabled(ctx, symbol); err != nil {
		return empty, err
	}
	prior, err := p.ledger.FindOrderByClientID(clientID)
	if err == nil {
		if prior.Symbol != symbol || prior.Side != "BUY" {
			return empty, fmt.Errorf("client id reused with other Spot order")
		}
		return p.reconcile(ctx, prior.ID)
	}
	if !errors.Is(err, orm.ErrNoRows) {
		return empty, err
	}
	// A prior cycle can look CLOSED in its materialized row even when the
	// last exchange SELL was never fully recovered. Verify historical
	// strategy orders via REST before creating a new BUY cycle.
	if err = p.reconcileSymbolOrders(ctx, symbol); err != nil {
		return empty, err
	}
	r, price, err := p.market(ctx, symbol)
	if err != nil {
		return empty, err
	}
	free, err := p.ex.Free(ctx, r.Quote)
	if err != nil {
		return empty, err
	}
	qty, err := planBuy(r, price, budget, free)
	if err != nil {
		return empty, err
	}
	if err = p.enabled(ctx, symbol); err != nil {
		return empty, err
	}
	pos, err := p.ledger.CreatePosition(symbol, r.Base, r.Quote)
	if err != nil {
		return empty, err
	}
	intent := spotledger.OrderIntent{PositionID: pos.ID, Symbol: symbol, Side: "BUY", Type: "MARKET", ClientOrderID: clientID, RequestedQty: qty}
	order, duplicate, err := p.ledger.CreateOrder(intent)
	if err != nil {
		return empty, err
	}
	if duplicate {
		return p.reconcile(ctx, order.ID)
	}
	return p.submitOnce(ctx, order)
}

// ExecuteSell only sells base units proven by 06-3 verified fills.
// Other wallet holdings are NEVER taken as strategy inventory.
func (p *Processor) ExecuteSell(ctx context.Context, positionID int64, clientID string) (models.SpotManagedOrder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var empty models.SpotManagedOrder
	if !validClientID(clientID) || positionID <= 0 {
		return empty, fmt.Errorf("invalid Spot SELL identity")
	}
	prior, err := p.ledger.FindOrderByClientID(clientID)
	if err == nil {
		if prior.PositionID != positionID || prior.Side != "SELL" {
			return empty, fmt.Errorf("client id reused with other Spot SELL")
		}
		return p.reconcile(ctx, prior.ID)
	}
	if !errors.Is(err, orm.ErrNoRows) {
		return empty, err
	}
	// The switch gate precedes any remote REST query for a NEW SELL.
	// Cancel and explicit Reconcile remain usable when global OFF.
	preview, err := p.ledger.GetPosition(positionID)
	if err != nil {
		return empty, err
	}
	if err = p.enabled(ctx, preview.Symbol); err != nil {
		return empty, err
	}
	// Reconcile only unresolved or recently completed strategy orders.
	if err = p.reconcilePositionOrders(ctx, positionID); err != nil {
		return empty, err
	}
	// Preview the canonical Fill inventory rather than mutable cached qty.
	view, err := p.ledger.Accounting(positionID, "")
	if err != nil {
		return empty, err
	}
	if view.Position.Status != "OPEN" {
		return empty, fmt.Errorf("cannot SELL without strategy-owned open inventory")
	}
	symbol := view.Position.Symbol
	if err := p.enabled(ctx, symbol); err != nil {
		return empty, err
	}
	r, price, err := p.market(ctx, symbol)
	if err != nil {
		return empty, err
	}
	if r.Base != view.Position.BaseAsset || r.Quote != view.Position.QuoteAsset {
		return empty, fmt.Errorf("Spot market identity differs from strategy position")
	}
	free, err := p.ex.Free(ctx, r.Base)
	if err != nil {
		return empty, err
	}
	qty, err := planSell(r, price, view.Position.RemainingQty, free)
	if err != nil {
		return empty, err
	}
	if err = p.enabled(ctx, symbol); err != nil {
		return empty, err
	}
	intent := spotledger.OrderIntent{PositionID: positionID, Symbol: symbol, Side: "SELL", Type: "MARKET", ClientOrderID: clientID, RequestedQty: qty}
	order, duplicate, err := p.ledger.CreateOrder(intent)
	if err != nil {
		return empty, err
	}
	if duplicate {
		return p.reconcile(ctx, order.ID)
	}
	return p.submitOnce(ctx, order)
}

func checkRemote(local models.SpotManagedOrder, remote RemoteOrder) error {
	if remote.Symbol != local.Symbol || remote.ClientOrderID != local.ClientOrderID ||
		remote.Side != local.Side || remote.Type != local.OrderType ||
		remote.ExchangeOrderID == "" {
		return fmt.Errorf("%w: remote Spot order identity mismatch for local order %d", ErrNeedsReview, local.ID)
	}
	q, err := parseAmount(remote.OrigQty, true)
	if err != nil {
		return err
	}
	expected, err := parseAmount(local.RequestedQty, true)
	if err != nil {
		return err
	}
	if !q.Equal(expected) {
		return fmt.Errorf("%w: exchange origQty differs from requestedQty", ErrNeedsReview)
	}
	return nil
}
func (p *Processor) submitOnce(ctx context.Context, local models.SpotManagedOrder) (models.SpotManagedOrder, error) {
	// Query by persisted client ID BEFORE any POST to Binance. A timeout
	// during this query is not "not found" and does not authorize placing.
	remote, err := p.ex.FindOrder(ctx, local.Symbol, local.ClientOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: lookup failed: %v", ErrUncertain, err)
	}
	if remote != nil {
		return p.reconcileRemote(ctx, local, *remote)
	}
	if err := p.enabled(ctx, local.Symbol); err != nil {
		return local, err
	}
	// Final real-time balance and inventory checks before network POST.
	r, price, err := p.market(ctx, local.Symbol)
	if err != nil {
		return local, err
	}
	qty, err := parseAmount(local.RequestedQty, true)
	if err != nil {
		return local, err
	}
	if _, err = prepareQty(r, qty, decimal.RequireFromString(price)); err != nil {
		return local, err
	}
	if local.Side == "BUY" {
		free, err := p.ex.Free(ctx, r.Quote)
		if err != nil {
			return local, err
		}
		available, err := parseAmount(free, false)
		if err != nil {
			return local, err
		}
		// Reserve 1% for market movement before sending quantity order.
		if available.LessThan(qty.Mul(decimal.RequireFromString(price)).Mul(buyPriceMargin)) {
			return local, fmt.Errorf("insufficient free quote for proposed Spot BUY")
		}
	} else {
		view, err := p.ledger.Accounting(local.PositionID, "")
		if err != nil {
			return local, err
		}
		strategy, err := parseAmount(view.Position.RemainingQty, true)
		if err != nil {
			return local, err
		}
		free, err := p.ex.Free(ctx, r.Base)
		if err != nil {
			return local, err
		}
		available, err := parseAmount(free, false)
		if err != nil {
			return local, err
		}
		if available.LessThan(strategy) || !qty.Equal(strategy) {
			return local, fmt.Errorf("%w: Spot SELL must exactly match verified inventory, and wallet free balance must cover it", ErrNeedsReview)
		}
	}
	if err := p.enabled(ctx, local.Symbol); err != nil {
		return local, err
	}
	req := Request{Symbol: local.Symbol, Side: local.Side, Type: "MARKET", ClientOrderID: local.ClientOrderID, Quantity: local.RequestedQty}
	if err := validateRequest(req); err != nil {
		return local, err
	}
	// A failed/timeout POST is ambiguous even when the SDK reports an error.
	// NEVER immediately resubmit the persisted local order.
	remoteOrder, err := p.ex.Place(ctx, req)
	if err != nil {
		// Safe reconciliation is read-only on the exchange side.
		looked, lookupErr := p.ex.FindOrder(ctx, local.Symbol, local.ClientOrderID)
		if lookupErr == nil && looked != nil {
			return p.reconcileRemote(ctx, local, *looked)
		}
		return local, fmt.Errorf("%w: submit error: %v; lookup error: %v", ErrUncertain, err, lookupErr)
	}
	return p.reconcileRemote(ctx, local, remoteOrder)
}

// Reconcile unresolved orders on every attempt, plus terminal orders
// with possible late fills for 24h. Old fully FILLED orders are already
// verified in immutable local Fill rows and must not trigger repeated REST
// lookups after Binance's historical order retention window.
func needsRemoteCheck(o models.SpotManagedOrder) bool {
	switch o.Status {
	case "NEW", "PARTIALLY_FILLED", "PENDING_CANCEL":
		return true
	case "FILLED":
		requested, e1 := parseAmount(o.RequestedQty, true)
		filled, e2 := parseAmount(o.FilledQty, false)
		return e1 != nil || e2 != nil || !requested.Equal(filled)
	case "CANCELED", "EXPIRED":
		return time.Since(time.UnixMilli(o.UpdatedAt)) <= 24*time.Hour
	default:
		return false
	}
}
func (p *Processor) reconcilePositionOrders(ctx context.Context, positionID int64) error {
	orders, err := p.ledger.ListOrders(positionID)
	if err != nil {
		return err
	}
	for _, o := range orders {
		if !needsRemoteCheck(o) {
			continue
		}
		if _, err = p.reconcile(ctx, o.ID); err != nil {
			return err
		}
	}
	return nil
}
func (p *Processor) reconcileSymbolOrders(ctx context.Context, symbol string) error {
	// Only currently active cycles require a live REST roundtrip before a
	// fresh BUY. Closed cycles are already verified from fills by
	// Ledger.CreatePosition; Binance may expire very old order lookups.
	for _, status := range []string{"OPEN", "PENDING_BUY"} {
		for page := 1; ; page++ {
			positions, err := p.ledger.ListPositions(spotledger.PositionFilter{
				Symbol: symbol, Status: status, Page: page, Limit: 100,
			})
			if err != nil {
				return err
			}
			for _, pos := range positions.List {
				if err = p.reconcilePositionOrders(ctx, pos.ID); err != nil {
					return err
				}
			}
			if page*100 >= int(positions.Total) {
				break
			}
		}
	}
	return nil
}

func (p *Processor) Reconcile(ctx context.Context, localOrderID int64) (models.SpotManagedOrder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.reconcile(ctx, localOrderID)
}

// ReconcileByClientID performs only REST reads and ledger recovery. It is
// safe with the execution switches OFF and never resubmits an old request.
func (p *Processor) ReconcileByClientID(ctx context.Context, clientID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	order, err := p.ledger.FindOrderByClientID(clientID)
	if err != nil {
		return err
	}
	_, err = p.reconcile(ctx, order.ID)
	return err
}
func (p *Processor) reconcile(ctx context.Context, id int64) (models.SpotManagedOrder, error) {
	order, err := p.ledger.GetOrder(id)
	if err != nil {
		return order, err
	}
	remote, err := p.ex.FindOrder(ctx, order.Symbol, order.ClientOrderID)
	if err != nil {
		return order, fmt.Errorf("%w: order lookup: %v", ErrUncertain, err)
	}
	if remote == nil {
		return order, fmt.Errorf("%w: authoritative exchange lookup did not find client id %s; explicit confirmation required", ErrNotFound, order.ClientOrderID)
	}
	return p.reconcileRemote(ctx, order, *remote)
}

// ConfirmNeverAccepted is an EXPLICIT operator action, never invoked by
// automatic retry or recovery. Even authoritative -2013 is not enough
// immediately after a timeout; allow at least 30 minutes for delayed
// exchange propagation, perform two fresh remote queries, then close the
// empty local BUY cycle as REJECTED without changing its client ID.
// This is intentionally not a generic manual position/amount override.
func (p *Processor) ConfirmNeverAccepted(ctx context.Context, localOrderID int64, operatorConfirmed bool) (models.SpotManagedOrder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	var empty models.SpotManagedOrder
	if !operatorConfirmed {
		return empty, fmt.Errorf("%w: explicit operator acknowledgement required", ErrNeedsReview)
	}
	order, err := p.ledger.GetOrder(localOrderID)
	if err != nil {
		return empty, err
	}
	if order.Status != "NEW" || order.ExchangeOrderID != "" || order.Side != "BUY" {
		return order, fmt.Errorf("%w: only unresolved unbound BUY may be released", ErrNeedsReview)
	}
	if time.Since(time.UnixMilli(order.CreatedAt)) < 30*time.Minute {
		return order, fmt.Errorf("%w: allow 30 minutes for ambiguous exchange acceptance", ErrUncertain)
	}
	for i := 0; i < 2; i++ {
		remote, err := p.ex.FindOrder(ctx, order.Symbol, order.ClientOrderID)
		if err != nil {
			return order, fmt.Errorf("%w: verification lookup failed: %v", ErrUncertain, err)
		}
		if remote != nil {
			_, reconcileErr := p.reconcileRemote(ctx, order, *remote)
			if reconcileErr != nil {
				return order, reconcileErr
			}
			return order, fmt.Errorf("%w: remote order exists; cannot confirm absence", ErrNeedsReview)
		}
	}
	return p.ledger.ConfirmNeverAccepted(order.ID)
}

func (p *Processor) reconcileRemote(ctx context.Context, local models.SpotManagedOrder, remote RemoteOrder) (models.SpotManagedOrder, error) {
	if err := checkRemote(local, remote); err != nil {
		return local, err
	}
	order, err := p.ledger.BindExchangeOrderID(local.ID, remote.ExchangeOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: bind order: %v", ErrNeedsReview, err)
	}
	// Mark exchange state before importing trades: a CANCELED terminal
	// remains CANCELED even if a late fill completes the requested quantity.
	order, err = p.ledger.SetExchangeStatus(local.ID, remote.Status)
	if err != nil {
		return order, fmt.Errorf("%w: exchange status conflict: %v", ErrNeedsReview, err)
	}
	events, err := p.ex.Trades(ctx, local.Symbol, remote.ExchangeOrderID)
	if err != nil {
		return order, fmt.Errorf("%w: cannot query myTrades: %v", ErrNeedsReview, err)
	}
	if len(events) >= 1000 {
		return order, fmt.Errorf("%w: myTrades result may be truncated; paginate before continuing", ErrNeedsReview)
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Time == events[j].Time {
			return events[i].TradeID < events[j].TradeID
		}
		return events[i].Time < events[j].Time
	})
	for _, tr := range events {
		if tr.Symbol != local.Symbol || tr.OrderID != remote.ExchangeOrderID || tr.Side != local.Side {
			return order, fmt.Errorf("%w: myTrades identity mismatch", ErrNeedsReview)
		}
		result, err := p.ledger.RecordFill(spotledger.FillEvent{
			OrderID: local.ID, Symbol: tr.Symbol, ExchangeOrderID: tr.OrderID,
			TradeID: tr.TradeID, Side: tr.Side, Qty: tr.Qty, Price: tr.Price,
			QuoteQty: tr.QuoteQty, Commission: tr.Fee, CommissionAsset: tr.FeeAsset,
			TradeTime: tr.Time,
		})
		if err != nil {
			return order, fmt.Errorf("%w: failed to record verified Spot trade: %v", ErrNeedsReview, err)
		}
		order = result.Order
	}
	recorded, err := p.ledger.ListFills(local.PositionID)
	if err != nil {
		return order, err
	}
	sum := decimal.Zero
	for _, f := range recorded {
		if f.OrderID != local.ID {
			continue
		}
		n, err := parseAmount(f.Qty, true)
		if err != nil {
			return order, err
		}
		sum = sum.Add(n)
	}
	remoteFilled, err := parseAmount(remote.ExecutedQty, false)
	if err != nil {
		return order, fmt.Errorf("%w: invalid remote executed quantity", ErrNeedsReview)
	}
	if !sum.Equal(remoteFilled) {
		return order, fmt.Errorf("%w: myTrades quantity %s differs from exchange executed %s", ErrNeedsReview, sum, remoteFilled)
	}
	issues, err := p.ledger.ReviewPosition(local.PositionID)
	if err != nil {
		return order, err
	}
	if len(issues) != 0 {
		return order, fmt.Errorf("%w: %s", ErrNeedsReview, strings.Join(issues, "; "))
	}
	return p.ledger.GetOrder(local.ID)
}

// Cancel is a recovery operation, not a new position. It remains available
// when either switch is OFF; it can never turn into a new BUY/SELL.
func (p *Processor) Cancel(ctx context.Context, orderID int64) (models.SpotManagedOrder, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	local, err := p.ledger.GetOrder(orderID)
	if err != nil {
		return local, err
	}
	if local.Status == "CANCELED" || local.Status == "FILLED" || local.Status == "EXPIRED" || local.Status == "REJECTED" {
		return p.reconcile(ctx, local.ID)
	}
	remote, err := p.ex.FindOrder(ctx, local.Symbol, local.ClientOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: lookup before cancel: %v", ErrUncertain, err)
	}
	if remote == nil {
		return local, fmt.Errorf("%w: cannot cancel an unconfirmed remote order", ErrUncertain)
	}
	if err := checkRemote(local, *remote); err != nil {
		return local, err
	}
	if _, err = p.ex.Cancel(ctx, local.Symbol, local.ClientOrderID); err != nil {
		return local, fmt.Errorf("%w: cancel request ambiguous: %v", ErrUncertain, err)
	}
	return p.reconcile(ctx, local.ID)
}

// RecoverOutstanding scans active Spot managed positions on explicit
// request (e.g., after WS disconnect or restart); it never claims wallet
// holdings, resubmits unknown orders, or scans old closed history.
func (p *Processor) RecoverOutstanding(ctx context.Context) error {
	// Snapshot active IDs BEFORE any reconciliation, because one pending
	// cycle can become CLOSED and shift OFFSET pages while iterating.
	activeIDs := make([]int64, 0)
	for _, status := range []string{"OPEN", "PENDING_BUY"} {
		for page := 1; ; page++ {
			items, err := p.ledger.ListPositions(spotledger.PositionFilter{Status: status, Page: page, Limit: 100})
			if err != nil {
				return err
			}
			for _, pos := range items.List {
				activeIDs = append(activeIDs, pos.ID)
			}
			if page*100 >= int(items.Total) {
				break
			}
		}
	}
	issues := []string{}
	for _, positionID := range activeIDs {
		orders, err := p.ledger.ListOrders(positionID)
		if err != nil {
			return err
		}
		for _, o := range orders {
			if !needsRemoteCheck(o) {
				continue
			}
			_, err = p.Reconcile(ctx, o.ID)
			if err != nil {
				issues = append(issues, fmt.Sprintf("%s/%s: %v", o.Symbol, o.ClientOrderID, err))
			}
		}
	}
	if len(issues) > 0 {
		return fmt.Errorf("%w: %s", ErrNeedsReview, strings.Join(issues, " | "))
	}
	return nil
}
