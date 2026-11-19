package orderhistory

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"sort"
	"strings"

	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"
)

// UF-5 is an immutable, account-bound REST-verified settlement report.
// It does not read the order history display overrides, price snapshots,
// estimated unrealized PnL or exchange APIs.
type PerformanceFilter struct {
	Days         int
	Symbol       string
	PositionSide string
	RuleHash     string
}

var performanceRuleHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (q PerformanceFilter) validated() (PerformanceFilter, error) {
	if q.Days == 0 {
		q.Days = 30
	}
	if q.Days != 7 && q.Days != 30 {
		return q, fmt.Errorf("%w: days", ErrInvalidQuery)
	}
	q.Symbol = strings.ToUpper(strings.TrimSpace(q.Symbol))
	if q.Symbol != "" && !symbolPattern.MatchString(q.Symbol) {
		return q, fmt.Errorf("%w: symbol", ErrInvalidQuery)
	}
	q.PositionSide = strings.ToUpper(strings.TrimSpace(q.PositionSide))
	if q.PositionSide != "" && q.PositionSide != "LONG" && q.PositionSide != "SHORT" {
		return q, fmt.Errorf("%w: position_side", ErrInvalidQuery)
	}
	q.RuleHash = strings.ToLower(strings.TrimSpace(q.RuleHash))
	if q.RuleHash != "" && q.RuleHash != "unknown" && !performanceRuleHash.MatchString(q.RuleHash) {
		return q, fmt.Errorf("%w: rule_hash", ErrInvalidQuery)
	}
	return q, nil
}

// Only the exact auto_strategy:<symbol>:<sha256> format identifies a stable
// rule. Legacy/ambiguous references must be grouped as unknown; source_ref
// itself is never returned to the client.
func performanceHash(sourceRef, symbol string) string {
	parts := strings.Split(strings.TrimSpace(sourceRef), ":")
	if len(parts) != 3 || parts[0] != "auto_strategy" || parts[1] != symbol {
		return "unknown"
	}
	hash := strings.ToLower(parts[2])
	if !performanceRuleHash.MatchString(hash) {
		return "unknown"
	}
	return hash
}

type PerformanceBreakdown struct {
	Name   string `json:"name"`
	Trades int    `json:"trades"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Zero   int    `json:"zero"`
	NetPnL string `json:"net_pnl"`
}
type PerformanceAsset struct {
	Asset            string                 `json:"asset"`
	Trades           int                    `json:"trades"`
	Wins             int                    `json:"wins"`
	Losses           int                    `json:"losses"`
	Zero             int                    `json:"zero"`
	WinRatePct       string                 `json:"win_rate_pct"`
	GrossPnL         string                 `json:"gross_pnl"`
	Commission       string                 `json:"commission"`
	NetPnL           string                 `json:"net_pnl"`
	AverageNetPnL    string                 `json:"average_net_pnl"`
	ProfitFactor     *string                `json:"profit_factor"` // nil if no loss sample
	ProfitFactorNote string                 `json:"profit_factor_note,omitempty"`
	SettledDrawdown  string                 `json:"settled_net_pnl_drawdown"` // currency amount, NOT equity drawdown
	BySymbol         []PerformanceBreakdown `json:"by_symbol"`
	BySide           []PerformanceBreakdown `json:"by_side"`
	ByRule           []PerformanceBreakdown `json:"by_rule"`
}
type PerformanceReport struct {
	AccountID       string             `json:"account_id"`
	Owner           string             `json:"owner"`
	Days            int                `json:"days"`
	StartClosedAt   int64              `json:"start_closed_at"`
	EndClosedAt     int64              `json:"end_closed_at"`
	Symbol          string             `json:"symbol,omitempty"`
	PositionSide    string             `json:"position_side,omitempty"`
	RuleHash        string             `json:"rule_hash,omitempty"`
	ObservedAt      int64              `json:"observed_at"`
	CandidateCycles int                `json:"candidate_cycles"`
	VerifiedCycles  int                `json:"verified_cycles"`
	ExcludedCycles  int                `json:"excluded_cycles"`
	ExcludedReasons map[string]int     `json:"excluded_reasons"`
	Assets          []PerformanceAsset `json:"assets"`
	Scope           string             `json:"scope"`
}

type performanceEntry struct {
	asset, symbol, side, hash string
	id, closedAt              int64
	net, gross, fee           *big.Rat
}

type performanceCounter struct {
	trades, wins, losses, zero                                      int
	gross, fee, net, positive, negative, cumulative, peak, drawdown *big.Rat
}

func newPerformanceCounter() *performanceCounter {
	return &performanceCounter{
		gross: new(big.Rat), fee: new(big.Rat), net: new(big.Rat),
		positive: new(big.Rat), negative: new(big.Rat),
		cumulative: new(big.Rat), peak: new(big.Rat), drawdown: new(big.Rat),
	}
}
func (a *performanceCounter) add(e performanceEntry) {
	a.trades++
	a.gross.Add(a.gross, e.gross)
	a.fee.Add(a.fee, e.fee)
	a.net.Add(a.net, e.net)
	switch e.net.Sign() {
	case 1:
		a.wins++
		a.positive.Add(a.positive, e.net)
	case -1:
		a.losses++
		a.negative.Sub(a.negative, e.net)
	default:
		a.zero++
	}
	a.cumulative.Add(a.cumulative, e.net)
	if a.cumulative.Cmp(a.peak) > 0 {
		a.peak.Set(a.cumulative)
	}
	current := new(big.Rat).Sub(a.peak, a.cumulative)
	if current.Cmp(a.drawdown) > 0 {
		a.drawdown.Set(current)
	}
}
func (a *performanceCounter) breakdown(name string) PerformanceBreakdown {
	return PerformanceBreakdown{Name: name, Trades: a.trades, Wins: a.wins, Losses: a.losses, Zero: a.zero, NetPnL: val(a.net)}
}
func performanceGrouped(entries []performanceEntry, key func(performanceEntry) string) []PerformanceBreakdown {
	group := map[string]*performanceCounter{}
	for _, e := range entries {
		k := key(e)
		if group[k] == nil {
			group[k] = newPerformanceCounter()
		}
		group[k].add(e)
	}
	result := make([]PerformanceBreakdown, 0, len(group))
	for name, grouped := range group {
		result = append(result, grouped.breakdown(name))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func performanceAssetResult(asset string, entries []performanceEntry) PerformanceAsset {
	summary := newPerformanceCounter()
	// The input is already sorted by (closed_at, cycle_id); starting peak is
	// zero. This is the drawdown of cumulative settled net PnL, not account
	// equity or an estimate of mark-to-market maximum drawdown.
	for _, e := range entries {
		summary.add(e)
	}
	result := PerformanceAsset{
		Asset: asset, Trades: summary.trades, Wins: summary.wins, Losses: summary.losses, Zero: summary.zero,
		GrossPnL: val(summary.gross), Commission: val(summary.fee), NetPnL: val(summary.net),
		SettledDrawdown: val(summary.drawdown),
		BySymbol:        performanceGrouped(entries, func(e performanceEntry) string { return e.symbol }),
		BySide:          performanceGrouped(entries, func(e performanceEntry) string { return e.side }),
		ByRule:          performanceGrouped(entries, func(e performanceEntry) string { return e.hash }),
	}
	result.WinRatePct = val(new(big.Rat).Mul(big.NewRat(int64(summary.wins), int64(summary.trades)), big.NewRat(100, 1)))
	result.AverageNetPnL = val(new(big.Rat).Quo(summary.net, big.NewRat(int64(summary.trades), 1)))
	if summary.negative.Sign() > 0 {
		pf := val(new(big.Rat).Quo(summary.positive, summary.negative))
		result.ProfitFactor = &pf
	} else {
		result.ProfitFactorNote = "no_loss_samples"
	}
	return result
}

func performanceExcludedReason(p models.FuturesManagedPosition, problem string, activatedAt int64) string {
	if activatedAt <= 0 {
		return "history_not_activated"
	}
	if p.CreatedAt < activatedAt {
		return "pre_activation"
	}
	if p.SettlementStatus == "legacy" {
		return "legacy"
	}
	if p.SettlementStatus != "settled" {
		return "not_settled"
	}
	if p.Status != "closed" || math.Abs(p.ManagedQty) > 1e-12 ||
		p.ClosedAt <= 0 || p.ClosedAt < p.CreatedAt ||
		p.SettlementProcessedAt < p.ClosedAt {
		return "settlement_conflict"
	}
	switch problem {
	case "fee_and_pnl_not_rest_verified":
		return "rest_not_verified"
	case "no_linked_managed_orders", "no_confirmed_exchange_fills",
		"settled_cycle_missing_executions", "order_fill_quantity_mismatch",
		"opening_and_closing_fills_unbalanced", "filled_order_missing_real_exchange_id":
		return "missing_or_mismatched_fills"
	case "mixed_commission_assets", "nonzero_fee_missing_asset", "unknown_settlement_asset",
		"mixed_pnl_assets", "settled_fill_missing_pnl_asset":
		return "unknown_or_mixed_asset"
	case "managed_order_cycle_identity_mismatch", "duplicate_client_order",
		"duplicate_exchange_matching_order", "fill_order_cycle_identity_mismatch":
		return "identity_mismatch"
	default:
		return "incomplete_evidence"
	}
}

// Performance uses 100-cycle batches. It replays the same strict per-cycle
// summarize proof as real order history, before any presentation override or
// market-price estimates can enter the result. Does not call List (which
// unnecessarily computes full-range unrealized PnL) or external services.
func (s Store) Performance(ctx context.Context, raw PerformanceFilter) (PerformanceReport, error) {
	q, err := raw.validated()
	if err != nil {
		return PerformanceReport{}, err
	}
	if err = ctx.Err(); err != nil {
		return PerformanceReport{}, err
	}
	t0, err := s.activatedAt("main")
	if err != nil {
		return PerformanceReport{}, err
	}
	now := s.now().UTC()
	end := now.UnixMilli()
	start := now.AddDate(0, 0, -q.Days).UnixMilli()
	report := PerformanceReport{
		AccountID: "main", Owner: "auto_strategy", Days: q.Days,
		StartClosedAt: start, EndClosedAt: end, ObservedAt: end,
		Symbol: q.Symbol, PositionSide: q.PositionSide, RuleHash: q.RuleHash,
		ExcludedReasons: map[string]int{}, Assets: []PerformanceAsset{},
		Scope: "rest_verified_settled_main_auto_strategy_only",
	}
	query := s.db().QueryTable(new(models.FuturesManagedPosition)).
		Filter("account_id", "main").Filter("owner", "auto_strategy").
		Filter("closed_at__gte", start).Filter("closed_at__lte", end)
	if q.Symbol != "" {
		query = query.Filter("symbol", q.Symbol)
	}
	if q.PositionSide != "" {
		query = query.Filter("position_side", q.PositionSide)
	}
	// When sourceRef is requested we intentionally do NOT use an SQL LIKE
	// filter: only a validated hash derived from the exact format is eligible.
	// This also allows "unknown" to be identified safely without leaking
	// a raw source_ref to the API.
	count, err := query.Count()
	if err != nil {
		return PerformanceReport{}, fmt.Errorf("read settlement cohort count: %w", err)
	}
	entries := map[string][]performanceEntry{}
	const batchSize = 100
	for offset := int64(0); offset < count; offset += batchSize {
		if err = ctx.Err(); err != nil {
			return PerformanceReport{}, err
		}
		var batch []models.FuturesManagedPosition
		_, err = query.OrderBy("closed_at", "id").Limit(batchSize, offset).All(&batch)
		if err != nil {
			return PerformanceReport{}, fmt.Errorf("read settlement cohort: %w", err)
		}
		if len(batch) == 0 {
			break
		}
		rows, err := s.performanceEvidence(ctx, batch, t0)
		if err != nil {
			return PerformanceReport{}, err
		}
		for i, p := range batch {
			hash := performanceHash(p.SourceRef, p.Symbol)
			if q.RuleHash != "" && hash != q.RuleHash {
				continue
			}
			report.CandidateCycles++
			row := rows[i]
			if row.Completeness != "verified" || row.NetPnL == nil || row.GrossPnL == nil ||
				(row.PnLAsset != "USDT" && row.PnLAsset != "USDC") ||
				p.Status != "closed" || math.Abs(p.ManagedQty) > 1e-12 ||
				p.ClosedAt <= 0 || p.ClosedAt < p.CreatedAt ||
				p.SettlementStatus != "settled" ||
				p.SettlementProcessedAt < p.ClosedAt || p.SettlementProcessedAt > end {
				reason := performanceExcludedReason(p, row.IncompleteReason, t0)
				if p.SettlementStatus == "settled" && p.SettlementProcessedAt > end {
					reason = "settlement_conflict"
				}
				report.ExcludedReasons[reason]++
				report.ExcludedCycles++
				continue
			}
			net, netErr := dec(*row.NetPnL)
			gross, grossErr := dec(*row.GrossPnL)
			if netErr != nil || grossErr != nil {
				report.ExcludedReasons["invalid_amount"]++
				report.ExcludedCycles++
				continue
			}
			fee := new(big.Rat).Sub(gross, net)
			entries[row.PnLAsset] = append(entries[row.PnLAsset], performanceEntry{
				asset: row.PnLAsset, symbol: p.Symbol, side: p.PositionSide, hash: hash,
				id: p.ID, closedAt: p.ClosedAt, net: net, gross: gross, fee: fee,
			})
			report.VerifiedCycles++
		}
	}
	for asset, items := range entries {
		// Entry iteration follows database order, but make that ordering
		// explicit for deterministic peak-to-trough drawdown.
		sort.Slice(items, func(i, j int) bool {
			if items[i].closedAt != items[j].closedAt {
				return items[i].closedAt < items[j].closedAt
			}
			return items[i].id < items[j].id
		})
		report.Assets = append(report.Assets, performanceAssetResult(asset, items))
	}
	sort.Slice(report.Assets, func(i, j int) bool { return report.Assets[i].Asset < report.Assets[j].Asset })
	return report, nil
}

// Settlement's authoritative proof requires opening and closing exact REST
// quantities to be equal, even when a display-oriented tolerance might have
// treated them as visually equal. Never certify a near-flat unmatched cycle.
func performanceExactFlat(fills []fillRow) bool {
	opened, closed := new(big.Rat), new(big.Rat)
	for _, f := range fills {
		qty, err := dec(f.Qty)
		if err != nil || qty.Sign() <= 0 {
			return false
		}
		switch {
		case f.Intent == "open":
			opened.Add(opened, qty)
		case closingIntent(f.Intent):
			closed.Add(closed, qty)
		default:
			return false
		}
	}
	return opened.Sign() > 0 && opened.Cmp(closed) == 0
}

func (s Store) performanceEvidence(ctx context.Context, positions []models.FuturesManagedPosition, t0 int64) ([]Position, error) {
	ids := make([]int64, len(positions))
	for i, p := range positions {
		ids[i] = p.ID
	}
	var orders []models.FuturesManagedOrder
	_, err := s.db().QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", "main").Filter("position_id__in", ids).
		OrderBy("created_at", "id").All(&orders)
	if err != nil {
		return nil, fmt.Errorf("read performance orders: %w", err)
	}
	slots := make([]string, len(ids))
	args := []any{tradefills.MarketUSDM, "main"}
	for i, id := range ids {
		slots[i] = "?"
		args = append(args, id)
	}
	sql := `SELECT market,account_id,symbol,order_id,trade_id,position_id,client_order_id,intent,
 position_side,qty,price,commission,commission_asset,realized_pnl,pnl_asset,trade_time,rest_verified
 FROM futures_trade_fills WHERE market=? AND account_id=? AND position_id IN (` +
		strings.Join(slots, ",") + `) ORDER BY trade_time,order_id,trade_id`
	var fills []fillRow
	_, err = s.db().Raw(sql, args...).QueryRows(&fills)
	if err != nil {
		return nil, fmt.Errorf("read performance Fills: %w", err)
	}
	byOrders := map[int64][]models.FuturesManagedOrder{}
	byFills := map[int64][]fillRow{}
	for _, o := range orders {
		if o.PositionID != nil {
			byOrders[*o.PositionID] = append(byOrders[*o.PositionID], o)
		}
	}
	for _, f := range fills {
		if f.PositionID != nil {
			byFills[*f.PositionID] = append(byFills[*f.PositionID], f)
		}
	}
	result := make([]Position, len(positions))
	for i, p := range positions {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		// No project() here: it applies presentation overrides and
		// current-price valuations; neither is valid for this report.
		row, _ := summarize(p, byOrders[p.ID], byFills[p.ID], false, s.now().UnixMilli(), t0)
		if row.Completeness == "verified" && !performanceExactFlat(byFills[p.ID]) {
			row.Completeness = "incomplete"
			row.IncompleteReason = "opening_and_closing_fills_unbalanced"
			row.NetPnL, row.GrossPnL = nil, nil
			row.PnLAsset = ""
		}
		result[i] = row
	}
	return result, nil
}
