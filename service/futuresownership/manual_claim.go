package futuresownership

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

// These sentinel errors have stable meanings for the UI. Do not expose
// internal DB or exchange details in API responses.
var (
	ErrManualPositionNotFound          = errors.New("manual_claim_position_not_found")
	ErrManualAccountQuantityChanged    = errors.New("manual_claim_account_quantity_changed")
	ErrManualDuplicateExchangePosition = errors.New("manual_claim_duplicate_exchange_position")
	ErrManualInvalidEntryPrice         = errors.New("manual_claim_invalid_entry_price")
)

// ManualClaimInput requires explicit account state confirmation from the UI.
// It NEVER creates an exchange order and never expands an existing live claim.
type ManualClaimInput struct {
	Symbol             string  `json:"symbol"`
	PositionSide       string  `json:"position_side"`
	Owner              string  `json:"owner"`
	Quantity           float64 `json:"quantity"`
	ExpectedAccountQty float64 `json:"expected_account_qty"`
	Reason             string  `json:"reason"`
	Operator           string  `json:"-"`
}

// Only ownership types which participate in the main account's strategy-exit
// loop can be repaired here. Notice/agent_trade require their own protection
// or proposal lifecycle and must not be claimed through a generic dialog.
func manualClaimOwner(value string) (string, error) {
	switch strings.TrimSpace(value) {
	case OwnerAutoStrategy, OwnerNewCoinRush, OwnerFundingRate:
		return value, nil
	default:
		return "", fmt.Errorf("owner cannot be manually assigned from this page")
	}
}

func (r Reconciler) ManualClaimPosition(ctx context.Context, input ManualClaimInput) (models.FuturesManagedPosition, error) {
	if err := r.validateBoundAccount(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	accountID, err := r.Ownership.accountID()
	if err != nil || accountID != "main" {
		return models.FuturesManagedPosition{}, fmt.Errorf("manual ownership repair is restricted to the main account")
	}
	symbol, err := normalizeSymbol(input.Symbol)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	side, err := normalizePositionSide(input.PositionSide)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	owner, err := manualClaimOwner(input.Owner)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if !finitePositive(input.Quantity) || !finitePositive(input.ExpectedAccountQty) ||
		input.Quantity > input.ExpectedAccountQty+qtyEpsilon {
		return models.FuturesManagedPosition{}, fmt.Errorf("invalid manual managed quantity")
	}
	input.Reason = strings.TrimSpace(input.Reason)
	input.Operator = strings.TrimSpace(input.Operator)
	if utf8.RuneCountInString(input.Reason) < 5 || utf8.RuneCountInString(input.Reason) > 512 ||
		utf8.RuneCountInString(input.Operator) == 0 || utf8.RuneCountInString(input.Operator) > 128 {
		return models.FuturesManagedPosition{}, fmt.Errorf("repair reason and authorized operator are required")
	}
	source, ok := r.Account.(interface {
		FreshPositions(context.Context) ([]AccountPosition, error)
	})
	if !ok {
		return models.FuturesManagedPosition{}, fmt.Errorf("fresh exchange account position reader is required")
	}
	// Exchange state is fetched NOW, not trusted from a possibly stale table.
	positions, err := source.FreshPositions(ctx)
	if err != nil {
		return models.FuturesManagedPosition{}, fmt.Errorf("fresh account verification failed: %w", err)
	}
	var actual float64
	var entryPrice float64
	matches := 0
	for _, p := range positions {
		if strings.EqualFold(p.Symbol, symbol) && strings.EqualFold(p.PositionSide, side) {
			matches++
			if matches > 1 {
				// In hedge mode the exchange returns one entry per symbol/side.
				// Do not guess which entry price should control strategy exits.
				return models.FuturesManagedPosition{}, ErrManualDuplicateExchangePosition
			}
			if !finitePositive(math.Abs(p.Quantity)) {
				return models.FuturesManagedPosition{}, fmt.Errorf("invalid exchange account position")
			}
			actual = math.Abs(p.Quantity)
			entryPrice = p.EntryPrice
		}
	}
	if matches == 0 || !finitePositive(actual) {
		return models.FuturesManagedPosition{}, ErrManualPositionNotFound
	}
	if !samePositionQty(actual, input.ExpectedAccountQty) || input.Quantity > actual+qtyEpsilon {
		return models.FuturesManagedPosition{}, ErrManualAccountQuantityChanged
	}
	if !finitePositive(entryPrice) {
		return models.FuturesManagedPosition{}, ErrManualInvalidEntryPrice
	}
	// The service takes the shared ownership mutation lock and a database
	// transaction. Recheck ALL owners and unresolved orders in that transaction.
	return r.Ownership.claimVerifiedManualPosition(ctx, ManualClaimInput{
		Symbol: symbol, PositionSide: side, Owner: owner, Quantity: input.Quantity,
		ExpectedAccountQty: actual, Reason: input.Reason, Operator: input.Operator,
	}, entryPrice)
}

// ManualClaims returns the latest immutable repair audits for the current
// account. It does not return exchange order history.
func (s Service) ManualClaims(ctx context.Context, limit int) ([]models.FuturesOwnershipAdjustment, error) {
	accountID, err := s.accountID()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	var rows []models.FuturesOwnershipAdjustment
	_, err = orm.NewOrm().QueryTable(new(models.FuturesOwnershipAdjustment)).
		Filter("account_id", accountID).OrderBy("-id").Limit(limit).All(&rows)
	return rows, err
}

func finitePositive(value float64) bool {
	return value > qtyEpsilon && !math.IsNaN(value) && !math.IsInf(value, 0)
}
func samePositionQty(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(a, b))
}

func (s Service) claimVerifiedManualPosition(ctx context.Context, input ManualClaimInput, entryPrice float64) (models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	accountID, err := s.accountID()
	if err != nil || accountID != "main" {
		return models.FuturesManagedPosition{}, fmt.Errorf("manual repair account mismatch")
	}
	if !finitePositive(entryPrice) {
		return models.FuturesManagedPosition{}, ErrManualInvalidEntryPrice
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()

	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	defer tx.Rollback()
	var rows []models.FuturesManagedPosition
	if _, err = tx.QueryTable(new(models.FuturesManagedPosition)).
		Filter("account_id", accountID).Filter("symbol", input.Symbol).
		Filter("position_side", input.PositionSide).OrderBy("-id").All(&rows); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	var previous models.FuturesManagedPosition
	for _, row := range rows {
		if activePositionStatus(row.Status) && row.ManagedQty > qtyEpsilon {
			return models.FuturesManagedPosition{}, fmt.Errorf("position is already managed by %s", row.Owner)
		}
		if row.Status != PositionClosed || row.ManagedQty > qtyEpsilon {
			return models.FuturesManagedPosition{}, fmt.Errorf("position has an unresolved ownership record; reconcile first")
		}
	}
	if len(rows) > 0 {
		previous = rows[0]
	}
	var orders []models.FuturesManagedOrder
	if _, err = tx.QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", accountID).Filter("symbol", input.Symbol).
		Filter("position_side", input.PositionSide).All(&orders); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	for _, order := range orders {
		if liveOrderStatus(order.Status) {
			return models.FuturesManagedPosition{}, fmt.Errorf("position has unresolved order %s; reconcile before manual repair", order.ClientOrderID)
		}
	}
	now := s.nowMillis()
	audit := models.FuturesOwnershipAdjustment{
		AccountID: accountID, Symbol: input.Symbol, PositionSide: input.PositionSide,
		Owner: input.Owner, AccountQty: input.ExpectedAccountQty, ManagedQty: input.Quantity,
		PreviousOwner: previous.Owner, PreviousStatus: previous.Status,
		PreviousSourceRef: previous.SourceRef, PreviousQty: previous.ManagedQty,
		Operator: input.Operator, Reason: input.Reason, CreatedAt: now,
	}
	if audit.ID, err = tx.Insert(&audit); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	position := previous
	position.AccountID = accountID
	position.Owner, position.Symbol, position.PositionSide = input.Owner, input.Symbol, input.PositionSide
	position.ManagedQty, position.EntryPrice = input.Quantity, entryPrice
	position.Status = PositionActive
	position.SourceRef = fmt.Sprintf("manual_repair:%d", audit.ID)
	if input.Owner == OwnerFundingRate {
		// The shared main strategy-exit loop intentionally filters funding
		// rate legs by this prefix; without it, a repair looks managed in
		// the table but is silently skipped by automatic exits.
		position.SourceRef = fmt.Sprintf("funding_rate:manual_repair:%d", audit.ID)
	}
	position.UpdatedAt, position.LastReconciledAt = now, now
	position.ClosedAt = 0
	if position.ID == 0 {
		position.CreatedAt = now
		position.ID, err = tx.Insert(&position)
	} else {
		var affected int64
		affected, err = tx.QueryTable(new(models.FuturesManagedPosition)).
			Filter("id", previous.ID).Filter("account_id", accountID).
			Filter("status", previous.Status).Filter("managed_qty", previous.ManagedQty).
			Filter("updated_at", previous.UpdatedAt).
			Update(orm.Params{"owner": position.Owner, "managed_qty": position.ManagedQty,
				"entry_price": position.EntryPrice, "status": position.Status,
				"source_ref": position.SourceRef, "updated_at": position.UpdatedAt,
				"last_reconciled_at": position.LastReconciledAt, "closed_at": int64(0)})
		if err == nil && affected != 1 {
			err = fmt.Errorf("ownership changed concurrently; refresh and retry")
		}
	}
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	audit.PositionID = position.ID
	if _, err = tx.Update(&audit, "position_id"); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	return position, nil
}
