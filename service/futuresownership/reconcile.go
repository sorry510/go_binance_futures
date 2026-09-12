package futuresownership

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	binanceapi "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

type AccountPosition struct {
	Symbol       string
	PositionSide string
	Quantity     float64
}

type AccountPositionSource interface {
	Positions(context.Context) ([]AccountPosition, error)
}

type ReconcileSummary struct {
	Owner            string `json:"owner"`
	OrdersChecked    int    `json:"orders_checked"`
	OrdersUnresolved int    `json:"orders_unresolved"`
	PositionsChecked int    `json:"positions_checked"`
	PositionsShrunk  int    `json:"positions_shrunk"`
	PositionsClosed  int    `json:"positions_closed"`
}

type Reconciler struct {
	Ownership Service
	Executor  Executor
	Account   AccountPositionSource
}

func DefaultReconciler() Reconciler {
	ownership := DefaultService()
	return Reconciler{Ownership: ownership, Executor: Executor{Ownership: ownership, Broker: BinanceOrderBroker{}}, Account: BinanceAccountPositionSource{}}
}

func (r Reconciler) ReconcileOwner(ctx context.Context, owner string) (ReconcileSummary, error) {
	owner, err := normalizeOwner(owner)
	if err != nil {
		return ReconcileSummary{}, err
	}
	summary := ReconcileSummary{Owner: owner}
	orders, err := r.Ownership.ActiveOrders(ctx, owner)
	if err != nil {
		return summary, err
	}
	for _, order := range orders {
		summary.OrdersChecked++
		if _, err := r.Executor.Reconcile(ctx, order.Symbol, order.ClientOrderID); err != nil {
			summary.OrdersUnresolved++
		}
	}
	managed, err := r.Ownership.ActivePositions(ctx, owner)
	if err != nil {
		return summary, err
	}
	if len(managed) == 0 {
		summary.OrdersUnresolved += r.cancelOrphanMutations(ctx, owner, managed)
		return summary, nil
	}
	if r.Account == nil {
		return summary, fmt.Errorf("account position source is required")
	}
	accountPositions, err := r.Account.Positions(ctx)
	if err != nil {
		return summary, err
	}
	accountQty := make(map[string]float64, len(accountPositions))
	for _, position := range accountPositions {
		accountQty[positionKey(position.Symbol, position.PositionSide)] = math.Abs(position.Quantity)
	}
	for _, position := range managed {
		summary.PositionsChecked++
		before := position.ManagedQty
		updated, err := r.Ownership.ReconcilePosition(ctx, owner, position.Symbol, position.PositionSide, accountQty[positionKey(position.Symbol, position.PositionSide)])
		if err != nil {
			return summary, err
		}
		if updated.Status == PositionClosed && position.Status != PositionClosed {
			summary.PositionsClosed++
		} else if updated.ManagedQty+qtyEpsilon < before {
			summary.PositionsShrunk++
		}
	}
	remaining, err := r.Ownership.ActivePositions(ctx, owner)
	if err != nil {
		return summary, err
	}
	summary.OrdersUnresolved += r.cancelOrphanMutations(ctx, owner, remaining)
	return summary, nil
}

func (r Reconciler) cancelOrphanMutations(ctx context.Context, owner string, positions []models.FuturesManagedPosition) int {
	active := make(map[string]bool, len(positions))
	for _, position := range positions {
		active[positionKey(position.Symbol, position.PositionSide)] = true
	}
	orders, err := r.Ownership.ActiveOrders(ctx, owner)
	if err != nil {
		return 1
	}
	unresolved := 0
	for _, order := range orders {
		if order.Intent == IntentOpen || active[positionKey(order.Symbol, order.PositionSide)] {
			continue
		}
		if strings.TrimSpace(order.ExchangeOrderID) == "" {
			unresolved++
			continue
		}
		if err := r.Executor.Cancel(ctx, owner, order); err != nil {
			unresolved++
		}
	}
	return unresolved
}

func (r Reconciler) ReconcileAll(ctx context.Context) ([]ReconcileSummary, error) {
	owners := []string{OwnerAutoStrategy, OwnerNewCoinRush, OwnerNoticeAutoOrder, OwnerFundingRate, OwnerAgentTrade}
	out := make([]ReconcileSummary, 0, len(owners))
	for _, owner := range owners {
		summary, err := r.ReconcileOwner(ctx, owner)
		out = append(out, summary)
		if err != nil {
			return out, fmt.Errorf("reconcile owner %s: %w", owner, err)
		}
	}
	return out, nil
}

type BinanceAccountPositionSource struct{}

func (BinanceAccountPositionSource) Positions(ctx context.Context) ([]AccountPosition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := binanceapi.GetPosition(binanceapi.PositionParams{})
	if err != nil {
		return nil, err
	}
	out := make([]AccountPosition, 0, len(rows))
	for _, row := range rows {
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.Abs(qty) <= qtyEpsilon {
			continue
		}
		out = append(out, AccountPosition{Symbol: row.Symbol, PositionSide: string(row.PositionSide), Quantity: qty})
	}
	return out, nil
}

func positionKey(symbol, side string) string {
	return strings.ToUpper(strings.TrimSpace(symbol)) + "|" + strings.ToUpper(strings.TrimSpace(side))
}

type AccountPositionOwnership struct {
	Symbol           string  `json:"symbol"`
	PositionSide     string  `json:"position_side"`
	AccountQty       float64 `json:"account_qty"`
	Owner            string  `json:"owner"`
	ManagedQty       float64 `json:"managed_qty"`
	ManagedStatus    string  `json:"managed_status,omitempty"`
	SourceRef        string  `json:"source_ref,omitempty"`
	LastReconciledAt int64   `json:"last_reconciled_at,omitempty"`
}

func (r Reconciler) PositionOverview(ctx context.Context) ([]AccountPositionOwnership, error) {
	if r.Account == nil {
		return nil, fmt.Errorf("account position source is required")
	}
	account, err := r.Account.Positions(ctx)
	if err != nil {
		return nil, err
	}
	managed, err := r.Ownership.ListPositions(ctx, "", true)
	if err != nil {
		return nil, err
	}
	managedByKey := make(map[string]models.FuturesManagedPosition, len(managed))
	for _, row := range managed {
		managedByKey[positionKey(row.Symbol, row.PositionSide)] = row
	}
	out := make([]AccountPositionOwnership, 0, len(account)+len(managed))
	seen := make(map[string]bool, len(account))
	for _, position := range account {
		key := positionKey(position.Symbol, position.PositionSide)
		seen[key] = true
		view := AccountPositionOwnership{Symbol: strings.ToUpper(position.Symbol), PositionSide: strings.ToUpper(position.PositionSide), AccountQty: math.Abs(position.Quantity), Owner: "unmanaged"}
		if row, ok := managedByKey[key]; ok {
			view.Owner, view.ManagedQty, view.ManagedStatus = row.Owner, row.ManagedQty, row.Status
			view.SourceRef, view.LastReconciledAt = row.SourceRef, row.LastReconciledAt
		}
		out = append(out, view)
	}
	for _, row := range managed {
		key := positionKey(row.Symbol, row.PositionSide)
		if seen[key] {
			continue
		}
		out = append(out, AccountPositionOwnership{Symbol: row.Symbol, PositionSide: row.PositionSide, Owner: row.Owner, ManagedQty: row.ManagedQty, ManagedStatus: row.Status, SourceRef: row.SourceRef, LastReconciledAt: row.LastReconciledAt})
	}
	return out, nil
}
