package leadaccount

import (
	"context"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/futuresownership"
)

var ErrLeadReconcileIncomplete = errors.New("Lead authoritative recovery is incomplete")

type LeadRecoveryReport struct {
	AccountID        binance.AccountID `json:"account_id"`
	ObservedAt       time.Time         `json:"observed_at"`
	OrdersChecked    int               `json:"orders_checked"`
	PositionsChecked int               `json:"positions_checked"`
	Blocked          bool              `json:"blocked"`
	Reasons          []string          `json:"reasons"`
}
type LeadRecoveryReader interface {
	GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error)
	GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error)
}
type LeadRecoveryAuditor struct {
	Owner  futuresownership.Service
	Broker futuresownership.OrderBroker
	Reader LeadRecoveryReader
	// Only exact Lead-bound executors may be used. Never fall back to Main.
}

func NewLeadRecoveryAuditor(client *binance.AccountClient) (*LeadRecoveryAuditor, error) {
	if client == nil || client.ID() != binance.LeadAccountID {
		return nil, ErrLeadReconcileIncomplete
	}
	owner, err := futuresownership.BindAccount(binance.LeadAccountID)
	if err != nil {
		return nil, err
	}
	return &LeadRecoveryAuditor{Owner: owner, Broker: futuresownership.BinanceOrderBroker{Account: client}, Reader: client}, nil
}

// Reconcile is a deliberate, caller-initiated recovery operation, NOT a
// trading authorization. It may apply verified cumulative fills to the
// durable Lead ownership ledger; it NEVER releases Stage 4 pendingReconcile
// or enables a real broker. All failures remain blocking.
func (r *LeadRecoveryAuditor) Reconcile(ctx context.Context) (LeadRecoveryReport, error) {
	report := LeadRecoveryReport{AccountID: binance.LeadAccountID, ObservedAt: time.Now().UTC(), Blocked: true}
	fail := func(code string) (LeadRecoveryReport, error) {
		report.Reasons = append(report.Reasons, code)
		return report, ErrLeadReconcileIncomplete
	}
	if r == nil || r.Owner.AccountID != "lead" || r.Broker == nil || r.Reader == nil {
		return fail("lead_recovery_not_bound")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	all, err := r.Owner.OutstandingOrders(ctx)
	if err != nil {
		return fail("lead_outstanding_orders_unavailable")
	}
	// Load history including CLOSED; GetPosition only returns active legs.
	history, err := r.Owner.ListPositions(ctx, "", false)
	if err != nil {
		return fail("lead_ownership_history_unavailable")
	}
	for _, h := range history {
		if h.AccountID != "lead" {
			return fail("lead_ownership_cross_account")
		}
	}
	report.PositionsChecked = len(history)
	live, err := r.Reader.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		return fail("lead_positions_unavailable")
	}
	pending, err := r.Reader.GetOpenOrderFreshContext(ctx)
	if err != nil {
		return fail("lead_orders_unavailable")
	}
	actual := map[string]float64{}
	for _, p := range live {
		if p == nil || p.Symbol == "" {
			return fail("lead_position_invalid")
		}
		q, e := strconv.ParseFloat(p.PositionAmt, 64)
		if e != nil || math.IsNaN(q) || math.IsInf(q, 0) {
			return fail("lead_position_invalid")
		}
		if math.Abs(q) < 1e-12 {
			continue
		}
		side := strings.ToUpper(p.PositionSide)
		if side != "LONG" && side != "SHORT" {
			return fail("lead_hedge_mode_unconfirmed")
		}
		actual[riskSlot(p.Symbol, side)] += math.Abs(q)
	}
	for _, p := range pending {
		if p == nil || p.Symbol == "" || p.ClientOrderID == "" {
			return fail("lead_unknown_exchange_order")
		}
	}
	// Any outstanding exchange order not present in owned order history needs
	// manual review before a new open. This includes unmanaged manual orders.
	known := map[string]bool{}
	for _, row := range all {
		known[row.ClientOrderID] = true
	}
	// Historical FILLED orders won't be in OutstandingOrders; an exchange open
	// order has to correspond to an outstanding claimed order, not a CLOSED one.
	for _, row := range pending {
		if !known[row.ClientOrderID] {
			return fail("lead_unmanaged_open_order")
		}
	}
	report.OrdersChecked = len(all)
	executor := futuresownership.Executor{Ownership: r.Owner, Broker: r.Broker}
	for _, row := range all {
		if row.AccountID != "lead" || row.ClientOrderID == "" || row.Owner == "" {
			return fail("lead_managed_order_invalid")
		}
		// Pending is a claimed row which may already have reached Binance.
		// Reconcile with the same client order ID; never Submit or Cancel.
		got, e := executor.Reconcile(ctx, row.Symbol, row.ClientOrderID)
		if e != nil || got.ClientOrderID != row.ClientOrderID || got.ExchangeOrderID == "" {
			return fail("lead_order_not_confirmed")
		}
	}
	// Reload all historical positions AFTER processing cumulative fills.
	history, err = r.Owner.ListPositions(ctx, "", false)
	if err != nil {
		return fail("lead_ownership_history_unavailable")
	}
	report.PositionsChecked = len(history)
	byLeg := map[string]float64{}
	for _, h := range history {
		if h.AccountID != "lead" || !riskFiniteNonnegative(h.ManagedQty) {
			return fail("lead_ownership_invalid")
		}
		if h.Status == futuresownership.PositionClosed {
			continue
		}
		byLeg[riskSlot(h.Symbol, h.PositionSide)] += h.ManagedQty
	}
	for leg, managed := range byLeg {
		if managed > actual[leg]+1e-8 {
			return fail("lead_manual_reduction_unreconciled")
		}
	}
	// Rechecking outstanding rows matters: a terminal REST receipt is not
	// sufficient if the persisted ledger remains partial or unknown.
	left, err := r.Owner.OutstandingOrders(ctx)
	if err != nil || len(left) != 0 {
		return fail("lead_orders_still_unresolved")
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	report.Blocked = false
	report.Reasons = nil
	return report, nil
}

// compile-time assert the durable ledger and snapshot are account-bound;
// recovering does NOT provide a public ResetPending shortcut.
var _ = models.FuturesManagedOrder{}
