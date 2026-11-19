package orderhistory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// UF-3: read-only, account-scoped anomaly locator. No exchange calls, order
// submission, reconciler mutation, or history/presentation overrides.
type AnomalyFilter struct {
	AccountID    string
	Owner        string
	Symbol       string
	PositionSide string
	Category     string
	Page         int
	Limit        int
}
type AnomalyItem struct {
	Category         string  `json:"category" orm:"column(category)"`
	Source           string  `json:"source" orm:"column(source)"`
	AccountID        string  `json:"account_id" orm:"column(account_id)"`
	Owner            string  `json:"owner" orm:"column(owner)"`
	Symbol           string  `json:"symbol" orm:"column(symbol)"`
	PositionSide     string  `json:"position_side" orm:"column(position_side)"`
	PositionID       *int64  `json:"position_id" orm:"column(position_id)"`
	ManagedOrderID   *int64  `json:"managed_order_id" orm:"column(managed_order_id)"`
	ProposalID       string  `json:"proposal_id,omitempty" orm:"column(proposal_id)"`
	ClientOrderID    string  `json:"client_order_id,omitempty" orm:"column(client_order_id)"`
	ExchangeOrderID  string  `json:"exchange_order_id,omitempty" orm:"column(exchange_order_id)"`
	Intent           string  `json:"intent,omitempty" orm:"column(intent)"`
	Status           string  `json:"status" orm:"column(status)"`
	SettlementStatus string  `json:"settlement_status,omitempty" orm:"column(settlement_status)"`
	RequestedQty     float64 `json:"requested_qty" orm:"column(requested_qty)"`
	FilledQty        float64 `json:"filled_qty" orm:"column(filled_qty)"`
	UpdatedAt        int64   `json:"updated_at" orm:"column(updated_at)"`
	LastReconciledAt int64   `json:"last_reconciled_at" orm:"column(last_reconciled_at)"`
}
type AnomalyPage struct {
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	List  []AnomalyItem `json:"list"`
	// Not the system health summary: this can include aged orders and incomplete
	// settlements, and can be filtered by account, Owner and category.
	Scope string `json:"scope"`
}

var validAnomalyCategories = map[string]bool{
	"":                             true,
	"reconcile_required":           true, // combined Main position/order health group
	"managed_position_reconcile":   true,
	"managed_order_reconcile":      true,
	"fill_verification_pending":    true,
	"aged_managed_order":           true,
	"settlement_incomplete":        true,
	"proposal_execution_uncertain": true,
	"proposal_protection_failed":   true,
}

func (q AnomalyFilter) validate() (AnomalyFilter, error) {
	q.AccountID = strings.ToLower(strings.TrimSpace(q.AccountID))
	if q.AccountID == "" {
		q.AccountID = "main"
	}
	if q.AccountID != "main" && q.AccountID != "lead" {
		return q, fmt.Errorf("%w: account", ErrInvalidQuery)
	}
	q.Owner = strings.ToLower(strings.TrimSpace(q.Owner))
	if q.Owner == "all" {
		q.Owner = ""
	}
	if q.Owner != "" && !ownerPattern.MatchString(q.Owner) {
		return q, fmt.Errorf("%w: owner", ErrInvalidQuery)
	}
	q.Symbol = strings.ToUpper(strings.TrimSpace(q.Symbol))
	if q.Symbol != "" && !symbolPattern.MatchString(q.Symbol) {
		return q, fmt.Errorf("%w: symbol", ErrInvalidQuery)
	}
	q.PositionSide = strings.ToUpper(strings.TrimSpace(q.PositionSide))
	if q.PositionSide == "ALL" {
		q.PositionSide = ""
	}
	if q.PositionSide != "" && q.PositionSide != "LONG" && q.PositionSide != "SHORT" {
		return q, fmt.Errorf("%w: position_side", ErrInvalidQuery)
	}
	q.Category = strings.ToLower(strings.TrimSpace(q.Category))
	if !validAnomalyCategories[q.Category] {
		return q, fmt.Errorf("%w: category", ErrInvalidQuery)
	}
	if q.Page < 1 || q.Page > 1000000 || q.Limit < 1 || q.Limit > 100 {
		return q, fmt.Errorf("%w: pagination", ErrInvalidQuery)
	}
	return q, nil
}

// Health's fill-verification-pending condition is intentionally mirrored
// here. Do not classify every unverified fill as a harmless "pending" event:
// mismatched quantity, absent exchange IDs or orphaned cycles still need
// reconciliation, and are treated as higher priority anomalies.
const uf3FillPending = `
	o.requested_qty>0 AND ABS(o.filled_qty-o.requested_qty)<=0.00000001
	AND o.exchange_order_id IS NOT NULL AND o.exchange_order_id<>''
	AND p.status IN ('active','closed')
	AND (SELECT COUNT(*) FROM futures_trade_fills f
		 WHERE f.account_id=o.account_id AND f.symbol=o.symbol
		   AND f.order_id=o.exchange_order_id)>0
	AND (SELECT COUNT(*) FROM futures_trade_fills f
		 WHERE f.account_id=o.account_id AND f.symbol=o.symbol
		   AND f.order_id=o.exchange_order_id AND f.rest_verified=0)>0
	AND ABS(COALESCE((SELECT SUM(f.qty) FROM futures_trade_fills f
		 WHERE f.account_id=o.account_id AND f.symbol=o.symbol
		   AND f.order_id=o.exchange_order_id),0)-o.filled_qty)<=0.00000001
`

// UNION keeps a single ordered/paginated view across independent sources,
// including OPEN intents without a position_id. No N+1 ORM loads, no whole
// history projection or unknown client-side account-ownership assumptions.
const uf3AnomalyUnion = `
SELECT
 CASE WHEN o.status='reconcile_required' AND (` + uf3FillPending + `)
      THEN 'fill_verification_pending'
      WHEN o.status='reconcile_required' THEN 'managed_order_reconcile'
      ELSE 'aged_managed_order' END AS category,
 'managed_order' AS source,o.account_id,o.owner,o.symbol,o.position_side,
 o.position_id,o.id AS managed_order_id,'' AS proposal_id,
 o.client_order_id,COALESCE(o.exchange_order_id,'') AS exchange_order_id,
 o.intent,o.status,'' AS settlement_status,o.requested_qty,o.filled_qty,
 o.updated_at,o.last_reconciled_at
 FROM futures_managed_orders o
 LEFT JOIN futures_managed_positions p ON p.id=o.position_id AND p.account_id=o.account_id
 WHERE o.status='reconcile_required' OR
 (o.status IN ('pending','submitted','partially_filled') AND o.updated_at>0 AND o.updated_at<?)
UNION ALL
SELECT CASE WHEN p.status='reconcile_required' THEN 'managed_position_reconcile'
 ELSE 'settlement_incomplete' END AS category,
 'managed_position' AS source,p.account_id,p.owner,p.symbol,p.position_side,
 p.id AS position_id,NULL AS managed_order_id,'' AS proposal_id,
 '' AS client_order_id,'' AS exchange_order_id,
 '' AS intent,p.status,p.settlement_status,0 AS requested_qty,
 p.managed_qty AS filled_qty,p.updated_at,p.last_reconciled_at
 FROM futures_managed_positions p
 WHERE p.status='reconcile_required' OR
 (p.status='closed' AND p.settlement_status='incomplete')
UNION ALL
SELECT CASE WHEN a.status='execution_uncertain' THEN 'proposal_execution_uncertain'
 ELSE 'proposal_protection_failed' END AS category,
 'agent_proposal' AS source,'main' AS account_id,'agent_trade' AS owner,
 a.symbol,a.side AS position_side,NULL AS position_id,NULL AS managed_order_id,
 a.proposal_id,'' AS client_order_id,'' AS exchange_order_id,
 '' AS intent,a.status,'' AS settlement_status,0 AS requested_qty,
 0 AS filled_qty,a.updated_at,0 AS last_reconciled_at
 FROM agent_trade_proposals a
 WHERE a.status IN ('execution_uncertain','protection_failed')
`

func (s Store) Anomalies(ctx context.Context, q AnomalyFilter) (AnomalyPage, error) {
	q, err := q.validate()
	if err != nil {
		return AnomalyPage{}, err
	}
	if err = ctx.Err(); err != nil {
		return AnomalyPage{}, err
	}
	cutoff := s.now().Add(-10 * time.Minute).UnixMilli()
	where := []string{"account_id=?"}
	params := []any{cutoff, q.AccountID}
	if q.Owner != "" {
		where = append(where, "owner=?")
		params = append(params, q.Owner)
	}
	if q.Symbol != "" {
		where = append(where, "symbol=?")
		params = append(params, q.Symbol)
	}
	if q.PositionSide != "" {
		where = append(where, "position_side=?")
		params = append(params, q.PositionSide)
	}
	if q.Category == "reconcile_required" {
		where = append(where, "category IN ('managed_position_reconcile','managed_order_reconcile')")
	} else if q.Category != "" {
		where = append(where, "category=?")
		params = append(params, q.Category)
	}
	condition := strings.Join(where, " AND ")
	source := "(" + uf3AnomalyUnion + ") anomalies"
	page := AnomalyPage{Page: q.Page, Limit: q.Limit, List: []AnomalyItem{}, Scope: "local_ledger_and_agent_proposals"}
	if err = s.db().Raw("SELECT COUNT(*) FROM "+source+" WHERE "+condition, params...).QueryRow(&page.Total); err != nil {
		return AnomalyPage{}, fmt.Errorf("read anomalies count: %w", err)
	}
	if page.Total == 0 {
		return page, nil
	}
	if err = ctx.Err(); err != nil {
		return AnomalyPage{}, err
	}
	limitParams := append(append([]any{}, params...), q.Limit, (q.Page-1)*q.Limit)
	_, err = s.db().Raw("SELECT * FROM "+source+" WHERE "+condition+" ORDER BY updated_at DESC,source ASC,managed_order_id DESC,position_id DESC,proposal_id DESC LIMIT ? OFFSET ?", limitParams...).QueryRows(&page.List)
	if err != nil {
		return AnomalyPage{}, fmt.Errorf("read anomalies: %w", err)
	}
	return page, nil
}
