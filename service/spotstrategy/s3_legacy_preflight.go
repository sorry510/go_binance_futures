//go:build spot_legacy_doctor

package spotstrategy

import (
	"context"
	"database/sql"
	"sort"
	"time"
)

// LegacyPreflight is a read-only evidence snapshot. It NEVER authorizes a
// strategy mode switch or real Binance Spot order. A clean snapshot is not
// sufficient to prove manual holdings and legacy exit obligations are closed.
type LegacyPreflight struct {
	AccountID      string               `json:"account_id"`
	ObservedAt     int64                `json:"observed_at"`
	Mode           string               `json:"mode"`
	ReadComplete   bool                 `json:"read_complete"`
	ReadyForLive   bool                 `json:"ready_for_live"`
	EvidenceSource string               `json:"evidence_source"`
	Balances       []LegacyAsset        `json:"balances"`
	OpenOrders     []LegacyOpenOrder    `json:"open_orders"`
	LocalPending   []LegacyPendingOrder `json:"local_pending"`
	Holdings       []LegacyHolding      `json:"holdings"`
	Blockers       []string             `json:"blockers"`
}
type LegacyAsset struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}
type LegacyOpenOrder struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"client_order_id"`
	OrderID       string `json:"order_id"`
	Ownership     string `json:"ownership"`
	Status        string `json:"status"`
}
type LegacyPendingOrder struct {
	Symbol        string `json:"symbol"`
	ClientOrderID string `json:"client_order_id"`
	Status        string `json:"status"`
}
type LegacyHolding struct {
	Symbol   string `json:"symbol"`
	Status   string `json:"status"`
	OwnedQty string `json:"owned_qty"`
}

// InspectLegacySpotRisk is intentionally not connected to any scheduler or
// HTTP route. A future operator-scoped, authorized audit can call it using
// the signed S2 read-only client. Tests pass isolated DB/client fixtures.
// The only Client methods used are Balances and OpenOrders with symbol="".
func InspectLegacySpotRisk(ctx context.Context, db *sql.DB, client Client) (LegacyPreflight, error) {
	report := LegacyPreflight{
		AccountID: AccountMain, ObservedAt: time.Now().UnixMilli(),
		ReadyForLive: false, ReadComplete: false, EvidenceSource: "unattested_readonly_snapshot",
		Balances: []LegacyAsset{}, OpenOrders: []LegacyOpenOrder{},
		LocalPending: []LegacyPendingOrder{}, Holdings: []LegacyHolding{},
		Blockers: []string{"operator_approval_missing", "legacy_exit_ownership_unverified",
			"signed_response_provenance_unverified"},
	}
	if db == nil || client == nil {
		return report, ErrReconciliation
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	var mode string
	modes, err := db.QueryContext(ctx, "SELECT spot_auto_trade_mode FROM config ORDER BY id LIMIT 2")
	if err != nil {
		return report, ErrReconciliation
	}
	var n int
	for modes.Next() {
		n++
		if scanErr := modes.Scan(&mode); scanErr != nil {
			err = scanErr
			break
		}
	}
	if rowErr := modes.Err(); err == nil {
		err = rowErr
	}
	if closeErr := modes.Close(); err == nil {
		err = closeErr
	}
	if err != nil || n != 1 {
		return report, ErrReconciliation
	}
	report.Mode = mode
	if mode != ModeLegacy && mode != ModeStrategy && mode != ModeOff {
		return report, ErrReconciliation
	}
	if mode != ModeLegacy {
		report.Blockers = append(report.Blockers, "legacy_mode_not_active")
	}
	// Account-level records owned by other local actors are not strategy
	// inventory and cannot silently disappear during a mode change.
	var foreignOrders, foreignHoldings int64
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_orders
 WHERE account_id=? AND owner<>?`, AccountMain, OwnerAuto).Scan(&foreignOrders); err != nil {
		return report, ErrReconciliation
	}
	if err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM spot_strategy_holdings
 WHERE account_id=? AND owner<>?`, AccountMain, OwnerAuto).Scan(&foreignHoldings); err != nil {
		return report, ErrReconciliation
	}
	if foreignOrders > 0 {
		report.Blockers = append(report.Blockers, "other_owner_local_orders")
	}
	if foreignHoldings > 0 {
		report.Blockers = append(report.Blockers, "other_owner_local_holdings")
	}
	type localRecord struct{ symbol, clientID, orderID, status string }
	localByClient := map[string]localRecord{}
	rows, err := db.QueryContext(ctx, `SELECT symbol,client_order_id,
  COALESCE(exchange_order_id,''),status FROM spot_strategy_orders
  WHERE account_id=? AND owner=?`, AccountMain, OwnerAuto)
	if err != nil {
		return report, ErrReconciliation
	}
	for rows.Next() {
		var o localRecord
		if scanErr := rows.Scan(&o.symbol, &o.clientID, &o.orderID, &o.status); scanErr != nil {
			err = scanErr
			break
		}
		if _, duplicate := localByClient[o.clientID]; duplicate {
			err = ErrReconciliation
			break
		}
		localByClient[o.clientID] = o
		if !isTerminal(o.status) {
			report.LocalPending = append(report.LocalPending, LegacyPendingOrder{Symbol: o.symbol,
				ClientOrderID: o.clientID, Status: o.status})
		}
	}
	if rowErr := rows.Err(); err == nil {
		err = rowErr
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return report, ErrReconciliation
	}
	if len(report.LocalPending) > 0 {
		report.Blockers = append(report.Blockers, "strategy_pending_orders")
	}
	sort.Slice(report.LocalPending, func(i, j int) bool {
		if report.LocalPending[i].Symbol != report.LocalPending[j].Symbol {
			return report.LocalPending[i].Symbol < report.LocalPending[j].Symbol
		}
		return report.LocalPending[i].ClientOrderID < report.LocalPending[j].ClientOrderID
	})
	rows, err = db.QueryContext(ctx, `SELECT symbol,status,CAST(owned_qty AS CHAR)
 FROM spot_strategy_holdings WHERE account_id=? AND owner=?`, AccountMain, OwnerAuto)
	if err != nil {
		return report, ErrReconciliation
	}
	for rows.Next() {
		var h LegacyHolding
		if scanErr := rows.Scan(&h.Symbol, &h.Status, &h.OwnedQty); scanErr != nil {
			err = scanErr
			break
		}
		if _, parseErr := parseAmount(h.OwnedQty, true); parseErr != nil {
			err = parseErr
			break
		}
		report.Holdings = append(report.Holdings, h)
	}
	if rowErr := rows.Err(); err == nil {
		err = rowErr
	}
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return report, ErrReconciliation
	}
	if len(report.Holdings) > 0 {
		report.Blockers = append(report.Blockers, "strategy_holding_requires_review")
	}
	sort.Slice(report.Holdings, func(i, j int) bool { return report.Holdings[i].Symbol < report.Holdings[j].Symbol })
	balances, err := client.Balances(ctx, AccountMain)
	if err != nil {
		return report, ErrReconciliation
	}
	if balances == nil {
		return report, ErrReconciliation
	}
	for asset, b := range balances {
		if asset == "" || len(asset) > 16 {
			return report, ErrReconciliation
		}
		free, fErr := parseAmount(b.Free, true)
		locked, lErr := parseAmount(b.Locked, true)
		if fErr != nil || lErr != nil {
			return report, ErrReconciliation
		}
		if free.IsZero() && locked.IsZero() {
			continue
		}
		report.Balances = append(report.Balances, LegacyAsset{Asset: asset, Free: free.String(), Locked: locked.String()})
	}
	if len(report.Balances) > 0 {
		report.Blockers = append(report.Blockers, "spot_assets_require_ownership_review")
	}
	sort.Slice(report.Balances, func(i, j int) bool { return report.Balances[i].Asset < report.Balances[j].Asset })
	exchangeOrders, err := client.OpenOrders(ctx, AccountMain, "")
	if err != nil {
		return report, ErrReconciliation
	}
	exchangeClients := map[string]bool{}
	exchangeIDs := map[string]bool{}
	for _, o := range exchangeOrders {
		if o.Symbol == "" || o.ClientOrderID == "" || o.OrderID == "" ||
			!clientIDPattern.MatchString(o.ClientOrderID) {
			return report, ErrReconciliation
		}
		if exchangeClients[o.ClientOrderID] || exchangeIDs[o.Symbol+":"+o.OrderID] {
			return report, ErrReconciliation
		}
		exchangeClients[o.ClientOrderID] = true
		exchangeIDs[o.Symbol+":"+o.OrderID] = true
		ownership := "unmanaged"
		if o.Status != "NEW" && o.Status != "PARTIALLY_FILLED" {
			ownership = "unexpected_open_order_status"
		} else if local, found := localByClient[o.ClientOrderID]; found {
			if local.symbol != o.Symbol || (local.orderID != "" && local.orderID != o.OrderID) {
				ownership = "identity_conflict"
			} else if isTerminal(local.status) {
				ownership = "terminal_conflict"
			} else {
				ownership = "local_strategy_pending"
			}
		}
		report.OpenOrders = append(report.OpenOrders, LegacyOpenOrder{
			Symbol: o.Symbol, ClientOrderID: o.ClientOrderID, OrderID: o.OrderID,
			Ownership: ownership, Status: o.Status,
		})
	}
	sort.Slice(report.OpenOrders, func(i, j int) bool {
		if report.OpenOrders[i].Symbol != report.OpenOrders[j].Symbol {
			return report.OpenOrders[i].Symbol < report.OpenOrders[j].Symbol
		}
		return report.OpenOrders[i].ClientOrderID < report.OpenOrders[j].ClientOrderID
	})
	for _, o := range report.OpenOrders {
		switch o.Ownership {
		case "unmanaged":
			report.Blockers = append(report.Blockers, "external_open_orders")
		case "identity_conflict":
			report.Blockers = append(report.Blockers, "exchange_order_identity_conflict")
		case "terminal_conflict":
			report.Blockers = append(report.Blockers, "terminal_order_still_open")
		case "unexpected_open_order_status":
			report.Blockers = append(report.Blockers, "exchange_open_status_invalid")
		}
	}
	// Deduplicate blockers while preserving stable order. Even with no orders,
	// empty assets and no local positions the historic/manual ownership question
	// and lack of signed response evidence MUST NOT be silently waived.
	seen := map[string]bool{}
	unique := make([]string, 0, len(report.Blockers))
	for _, b := range report.Blockers {
		if !seen[b] {
			seen[b] = true
			unique = append(unique, b)
		}
	}
	report.Blockers = unique
	report.ReadComplete = true
	// This is a diagnostic snapshot, NEVER the release decision.
	report.ReadyForLive = false
	return report, nil
}

// LegacyPreflightBlocked cannot authorize funds by trusting a caller-supplied
// report, even if an attacker constructs a struct with ReadyForLive=true. A
// read-only snapshot is not a durable operator grant and must never be used
// as a decision to switch strategy mode or execute a signed Spot write.
func LegacyPreflightBlocked(_ LegacyPreflight, _ error) bool {
	return true
}
