package futuresownership

import (
	"context"
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/beego/beego/v2/core/logs"

	"github.com/adshao/go-binance/v2/futures"
	binanceapi "go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"
)

type AccountPosition struct {
	Symbol       string
	PositionSide string
	Quantity     float64
	EntryPrice   float64
}

type AccountPositionSource interface {
	Positions(context.Context) ([]AccountPosition, error)
}

// ExactPositionSideSource proves an absent bulk-snapshot key using a separate
// symbol-scoped fresh REST request. Absence alone is not evidence of zero.
type ExactPositionSideSource interface {
	FreshPositionSide(context.Context, string, string) (AccountPosition, error)
}

type ReconcileSummary struct {
	Owner                  string `json:"owner"`
	OrdersChecked          int    `json:"orders_checked"`
	OrdersUnresolved       int    `json:"orders_unresolved"`
	PositionsChecked       int    `json:"positions_checked"`
	PositionsShrunk        int    `json:"positions_shrunk"`
	PositionsClosed        int    `json:"positions_closed"`
	IncompleteAged         int    `json:"incomplete_aged"`
	OldestIncompleteAgeSec int64  `json:"oldest_incomplete_age_sec"`
}

type Reconciler struct {
	Ownership Service
	Executor  Executor
	Account   AccountPositionSource
	// Optional test-only/caller-provided margin asset resolver. In production,
	// the official USD-M exchangeInfo.marginAsset is read via Account.
	SettlementMarginAsset func(context.Context, string) (string, error)
}

func DefaultReconciler() Reconciler {
	ownership := DefaultService()
	return Reconciler{Ownership: ownership, Executor: Executor{Ownership: ownership, Broker: BinanceOrderBroker{}}, Account: BinanceAccountPositionSource{}}
}

// NewAccountReconciler binds positions, ownership writes and order brokerage
// to exactly the same Binance account. A nil account never falls back to main.
func NewAccountReconciler(client *binanceapi.AccountClient) (Reconciler, error) {
	if client == nil {
		return Reconciler{}, fmt.Errorf("account client required")
	}
	executor, err := NewAccountExecutor(client)
	if err != nil {
		return Reconciler{}, err
	}
	return Reconciler{Ownership: executor.Ownership, Executor: executor, Account: BinanceAccountPositionSource{Account: client}}, nil
}
func (r Reconciler) validateBoundAccount() error {
	if err := r.Executor.validateAccountBinding(); err != nil {
		return err
	}
	id, err := r.Ownership.accountID()
	if err != nil {
		return err
	}
	executorID, err := r.Executor.Ownership.accountID()
	if err != nil {
		return err
	}
	if id != executorID {
		return fmt.Errorf("reconciler ownership account %s differs from executor account %s", id, executorID)
	}
	if source, ok := r.Account.(BinanceAccountPositionSource); ok {
		if source.Account == nil {
			if id != "main" {
				return fmt.Errorf("lead reconciliation requires an explicit account position source")
			}
		} else if string(source.Account.ID()) != id {
			return fmt.Errorf("reconciler position source account does not match ownership account")
		}
	}
	return nil
}

var incompleteAlertMu sync.Mutex
var incompleteAlertAt = map[string]int64{}

// Send an operator-visible WARNING tagged 报警 when a cycle cannot complete
// for 15 minutes. Throttle by exact account+position to avoid minute spam.
// This must never silently switch trading modes or fabricate settlement.
func alarmIncompletePosition(pos models.FuturesManagedPosition, now int64, reason string, summary *ReconcileSummary) {
	if pos.ClosedAt <= 0 || now-pos.ClosedAt < 15*60*1000 {
		return
	}
	age := (now - pos.ClosedAt) / 1000
	summary.IncompleteAged++
	if age > summary.OldestIncompleteAgeSec {
		summary.OldestIncompleteAgeSec = age
	}
	key := pos.AccountID + "/" + strconv.FormatInt(pos.ID, 10)
	incompleteAlertMu.Lock()
	// Bounded throttle state: expire old keys and evict the oldest entry if
	// extreme historical incompletes exceed the memory budget.
	if len(incompleteAlertAt) >= 4096 {
		oldestKey := ""
		oldest := now
		for k, last := range incompleteAlertAt {
			if now-last > 24*60*60*1000 {
				delete(incompleteAlertAt, k)
				continue
			}
			if last <= oldest {
				oldest = last
				oldestKey = k
			}
		}
		if len(incompleteAlertAt) >= 4096 && oldestKey != "" {
			delete(incompleteAlertAt, oldestKey)
		}
	}
	previous := incompleteAlertAt[key]
	shouldLog := now-previous >= 15*60*1000
	if shouldLog {
		incompleteAlertAt[key] = now
	}
	incompleteAlertMu.Unlock()
	if shouldLog {
		logs.Warning("报警: futures settlement incomplete account=", pos.AccountID,
			" owner=", pos.Owner, " symbol=", pos.Symbol, " position_id=", pos.ID,
			" age_sec=", age, " reason=", reason,
			" ; verified fill completeness is required before automatic loss streak adjustment")
	}
}

func clearIncompleteAlarm(account string, positionID int64) {
	incompleteAlertMu.Lock()
	delete(incompleteAlertAt, account+"/"+strconv.FormatInt(positionID, 10))
	incompleteAlertMu.Unlock()
}

func (r Reconciler) ReconcileOwner(ctx context.Context, owner string) (ReconcileSummary, error) {
	if err := r.validateBoundAccount(); err != nil {
		return ReconcileSummary{}, err
	}
	ctx = binanceapiusage.WithSource(ctx, "ownership_reconcile")
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
	unsettled, err := r.Ownership.UnsettledClosedPositions(ctx, owner)
	if err != nil {
		return summary, err
	}
	if len(managed) == 0 && len(unsettled) == 0 {
		summary.OrdersUnresolved += r.cancelOrphanMutations(ctx, owner, managed)
		return summary, nil
	}
	if r.Account == nil {
		// Do not infer zero holdings from absence of an exchange reader.
		summary.OrdersUnresolved += len(unsettled)
		if len(managed) != 0 {
			return summary, fmt.Errorf("account position source is required")
		}
		return summary, nil
	}
	// Reconciliation can shrink/close durable ownership. A cached account
	// snapshot is insufficient evidence for that destructive transition.
	var accountPositions []AccountPosition
	if source, ok := r.Account.(BinanceAccountPositionSource); ok {
		accountPositions, err = source.FreshPositions(ctx)
	} else {
		accountPositions, err = r.Account.Positions(ctx)
	}
	if err != nil {
		return summary, err
	}
	accountQty := make(map[string]float64, len(accountPositions))
	for _, position := range accountPositions {
		accountQty[positionKey(position.Symbol, position.PositionSide)] = math.Abs(position.Quantity)
	}
	// Binance may return an incomplete bulk snapshot. Verify every absent
	// managed key individually against its symbol-scoped fresh REST response.
	// Cap distinct extra REST calls per owner pass to avoid rate-limit storms.
	unverified := make(map[string]bool)
	needed := make(map[string]AccountPosition)
	{

		for _, position := range managed {
			key := positionKey(position.Symbol, position.PositionSide)
			if _, found := accountQty[key]; !found {
				needed[key] = AccountPosition{Symbol: position.Symbol, PositionSide: position.PositionSide}
			}
		}
		for _, position := range unsettled {
			key := positionKey(position.Symbol, position.PositionSide)
			if _, found := accountQty[key]; !found {
				needed[key] = AccountPosition{Symbol: position.Symbol, PositionSide: position.PositionSide}
			}
		}
		keys := make([]string, 0, len(needed))
		for key := range needed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		exact, available := r.Account.(ExactPositionSideSource)
		if !available {
			// A custom/wrapped source that cannot prove a side-specific
			// zero is NEVER authoritative for destructive ownership edits,
			// even when its bulk Positions contains an explicit zero.
			for _, p := range managed {
				unverified[positionKey(p.Symbol, p.PositionSide)] = true
			}
			for _, p := range unsettled {
				unverified[positionKey(p.Symbol, p.PositionSide)] = true
			}
		}
		// Rotate a deterministic sorted batch each minute. With more than
		// six missing keys no particular symbol is starved forever.
		offset := 0
		if len(keys) > 0 {
			offset = int((r.Ownership.nowMillis() / 60000) % int64(len(keys)))
		}
		for i := range keys {
			key := keys[(offset+i)%len(keys)]
			if !available || i >= 6 {
				unverified[key] = true
				continue
			}
			target := needed[key]
			item, fetchErr := exact.FreshPositionSide(ctx, target.Symbol, target.PositionSide)
			if fetchErr != nil || positionKey(item.Symbol, item.PositionSide) != key ||
				math.IsNaN(item.Quantity) || math.IsInf(item.Quantity, 0) {
				unverified[key] = true
				continue
			}
			accountQty[key] = math.Abs(item.Quantity)
		}
	}
	if len(unverified) > 0 {
		logs.Warning("报警: ownership snapshot missing exact position-side verification",
			" owner=", owner, " unverified_count=", len(unverified),
			" ; no inferred zero allowed for these managed sides")
	}
	// A suspicious empty position snapshot plus unresolved exchange orders
	// must not be used as proof of zero for account-managed ownership.
	uncertainEmpty := len(accountPositions) == 0 && summary.OrdersUnresolved > 0
	for _, position := range managed {
		summary.PositionsChecked++
		before := position.ManagedQty
		// A still-uncertain open can fill just after this snapshot. Retain its
		// ownership until the exchange order is reconciled.
		unresolvedOpen := false
		for _, order := range orders {
			if order.Intent != IntentOpen || order.Symbol != position.Symbol || order.PositionSide != position.PositionSide {
				continue
			}
			refreshed, loadErr := r.Ownership.GetOrder(ctx, order.ClientOrderID)
			if loadErr != nil || liveOrderStatus(refreshed.Status) {
				unresolvedOpen = true
				break
			}
		}
		// A source without an exact-side proof cannot safely shrink the
		// cycle even to a positive quantity. Preserve the durable ownership.
		if unverified[positionKey(position.Symbol, position.PositionSide)] &&
			accountQty[positionKey(position.Symbol, position.PositionSide)]+qtyEpsilon < before {
			summary.OrdersUnresolved++
			continue
		}
		if (unresolvedOpen || uncertainEmpty || unverified[positionKey(position.Symbol, position.PositionSide)]) && accountQty[positionKey(position.Symbol, position.PositionSide)] <= qtyEpsilon {
			if unverified[positionKey(position.Symbol, position.PositionSide)] {
				summary.OrdersUnresolved++
			}
			continue
		}
		updated, err := r.Ownership.ReconcileObservedPosition(ctx, owner, position.Symbol, position.PositionSide, accountQty[positionKey(position.Symbol, position.PositionSide)])
		if err != nil {
			return summary, err
		}
		if updated.ManagedQty+qtyEpsilon < before || updated.Status == PositionClosed {
			// External reductions are never labeled strategy fills, but an
			// existing TP/SL must not remain larger than the real position.
			if cancelErr := r.Executor.cancelExcessProtections(ctx, updated.ID); cancelErr != nil {
				summary.OrdersUnresolved++
			}
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
	// A CLOSED position has disappeared from ActivePositions, so it must be
	// retried separately until the immutable trade ledger proves a complete
	// one-cycle settlement. Zero here is a fresh exchange REST snapshot.
	for _, position := range unsettled {
		// Read-only REST trade recovery is independent of the ability to
		// prove a zero position side. A missing/ambiguous PositionRisk row
		// MUST block settlement, never block fetching immutable trade facts.
		if backfillErr := r.reconcileClosedCycleFills(ctx, position); backfillErr != nil {
			summary.OrdersUnresolved++
			alarmIncompletePosition(position, r.Ownership.nowMillis(),
				"closed-cycle trade backfill: "+backfillErr.Error(), &summary)
			continue
		}
		reason := ""
		if unverified[positionKey(position.Symbol, position.PositionSide)] {
			reason = "symbol-scoped fresh position snapshot missing or ambiguous"
		}
		if uncertainEmpty {
			reason = "empty snapshot and unresolved exchange orders"
		}
		if accountQty[positionKey(position.Symbol, position.PositionSide)] > qtyEpsilon {
			alarmIncompletePosition(position, r.Ownership.nowMillis(), "exchange still has nonzero position", &summary)
			summary.OrdersUnresolved++
			continue
		}
		if reason != "" {
			summary.OrdersUnresolved++
			alarmIncompletePosition(position, r.Ownership.nowMillis(), reason, &summary)
			continue
		}
		margin, assetErr := r.resolveSettlementMarginAsset(ctx, position.Symbol)
		if assetErr != nil || margin == "" {
			summary.OrdersUnresolved++
			alarmIncompletePosition(position, r.Ownership.nowMillis(), "exchange settlement asset unavailable", &summary)
			continue
		}
		result, settleErr := r.Ownership.settleVerifiedFlat(ctx, position.ID, margin)
		if settleErr != nil || result.Status != SettlementSettled {
			summary.OrdersUnresolved++
			why := result.Reason
			if settleErr != nil {
				why = settleErr.Error()
			}
			alarmIncompletePosition(position, r.Ownership.nowMillis(), why, &summary)
		} else {
			clearIncompleteAlarm(position.AccountID, position.ID)
		}
	}
	return summary, nil
}

// reconcileClosedCycleFills is read-only at the exchange and idempotent on
// the local ledger. Inactive orders do not come through the ActiveOrders loop.
// Never call Executor.Reconcile here: it could change a terminal order status
// from a stale lookup. Only official userTrades can upgrade REST verification.
func (r Reconciler) reconcileClosedCycleFills(ctx context.Context, pos models.FuturesManagedPosition) error {
	var orders []models.FuturesManagedOrder
	account, err := r.Ownership.accountID()
	if err != nil {
		return err
	}
	if account != pos.AccountID {
		return fmt.Errorf("closed-cycle account binding mismatch")
	}
	_, err = orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", account).Filter("position_id", pos.ID).OrderBy("id").All(&orders)
	if err != nil {
		return err
	}
	if len(orders) == 0 {
		return fmt.Errorf("closed cycle has no linked orders")
	}
	for _, order := range orders {
		if order.Owner != pos.Owner || order.Symbol != pos.Symbol || order.PositionSide != pos.PositionSide {
			return fmt.Errorf("closed-cycle order identity mismatch: %s", order.ClientOrderID)
		}
		if order.FilledQty <= qtyEpsilon {
			continue
		}
		// This verifies both completeness and provenance, including an
		// original FILLED order with stale WS-only rest_verified=0 rows.
		if err := r.Executor.reconcileTradeFills(ctx, order.ClientOrderID, ExchangeOrder{
			ExchangeOrderID: order.ExchangeOrderID,
			FilledQty:       order.FilledQty,
		}); err != nil {
			return fmt.Errorf("confirm %s: %w", order.ClientOrderID, err)
		}
	}
	return nil
}

func (r Reconciler) resolveSettlementMarginAsset(ctx context.Context, symbol string) (string, error) {
	if r.SettlementMarginAsset != nil {
		return r.SettlementMarginAsset(ctx, symbol)
	}
	source, ok := r.Account.(BinanceAccountPositionSource)
	if !ok {
		return "", fmt.Errorf("no official USD-M contract metadata provider")
	}
	var info *futures.ExchangeInfo
	var err error
	if source.Account != nil {
		info, err = source.Account.GetLeadExchangeInfoContext(ctx)
	} else {
		info, err = binanceapi.GetExchangeInfoContext(ctx)
	}
	if err != nil {
		return "", err
	}
	if info == nil {
		return "", fmt.Errorf("missing exchangeInfo metadata")
	}
	for _, row := range info.Symbols {
		if row.Symbol == symbol && row.ContractType == futures.ContractTypePerpetual &&
			row.MarginAsset != "" {
			return row.MarginAsset, nil
		}
	}
	return "", fmt.Errorf("no matching perpetual contract margin asset")
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
	if err := r.validateBoundAccount(); err != nil {
		return nil, err
	}
	owners := []string{OwnerAutoStrategy, OwnerNewCoinRush, OwnerNoticeAutoOrder, OwnerFundingRate, OwnerAgentTrade}
	accountID, err := r.Ownership.accountID()
	if err != nil {
		return nil, err
	}
	if accountID == "lead" {
		owners = []string{OwnerAutoStrategy}
	}
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

type BinanceAccountPositionSource struct {
	Account *binanceapi.AccountClient
}

func (source BinanceAccountPositionSource) Positions(ctx context.Context) ([]AccountPosition, error) {
	return source.positions(ctx, false)
}

func (source BinanceAccountPositionSource) FreshPositions(ctx context.Context) ([]AccountPosition, error) {
	return source.positions(ctx, true)
}

// FreshPositionSide requires an explicitly returned symbol/side risk row;
// unlike the all-account Positions view it accepts verified zero quantities.
func (source BinanceAccountPositionSource) FreshPositionSide(ctx context.Context, symbol, side string) (AccountPosition, error) {
	params := binanceapi.PositionParams{Symbol: strings.ToUpper(strings.TrimSpace(symbol))}
	var rows []*futures.PositionRisk
	var err error
	if source.Account != nil {
		rows, err = source.Account.GetPositionFreshContext(ctx, params)
	} else {
		rows, err = binanceapi.GetPositionFreshContext(ctx, params)
	}
	if err != nil {
		return AccountPosition{}, err
	}
	for _, row := range rows {
		if row == nil {
			return AccountPosition{}, fmt.Errorf("nil exact position row")
		}
		if row.Symbol != params.Symbol || !strings.EqualFold(string(row.PositionSide), side) {
			continue
		}
		qty, e := strconv.ParseFloat(row.PositionAmt, 64)
		if e != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return AccountPosition{}, fmt.Errorf("invalid exact position quantity for %s %s", symbol, side)
		}
		return AccountPosition{Symbol: row.Symbol, PositionSide: string(row.PositionSide), Quantity: qty}, nil
	}
	return AccountPosition{}, fmt.Errorf("symbol-scoped Binance REST did not return %s %s side; cannot infer zero", symbol, side)
}

func (source BinanceAccountPositionSource) positions(ctx context.Context, fresh bool) ([]AccountPosition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []*futures.PositionRisk
	var err error
	if source.Account != nil {
		if fresh {
			rows, err = source.Account.GetPositionFreshContext(ctx, binanceapi.PositionParams{})
		} else {
			rows, err = source.Account.GetPositionContext(ctx, binanceapi.PositionParams{})
		}
	} else if fresh {
		rows, err = binanceapi.GetPositionFreshContext(ctx, binanceapi.PositionParams{})
	} else {
		rows, err = binanceapi.GetPositionContext(ctx, binanceapi.PositionParams{})
	}
	if err != nil {
		return nil, err
	}
	out := make([]AccountPosition, 0, len(rows))
	for _, row := range rows {
		if row == nil {
			return nil, fmt.Errorf("invalid nil account position")
		}
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, fmt.Errorf("invalid exchange position quantity %q for %s", row.PositionAmt, row.Symbol)
		}
		if math.Abs(qty) <= qtyEpsilon {
			continue
		}
		entry, parseErr := strconv.ParseFloat(row.EntryPrice, 64)
		if parseErr != nil || math.IsNaN(entry) || math.IsInf(entry, 0) || entry < 0 {
			return nil, fmt.Errorf("invalid exchange entry price for %s %s: %q", row.Symbol, row.PositionSide, row.EntryPrice)
		}
		out = append(out, AccountPosition{Symbol: row.Symbol, PositionSide: string(row.PositionSide), Quantity: qty, EntryPrice: entry})
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
