package futuresownership

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/tradefills"

	"github.com/beego/beego/v2/client/orm"
)

const qtyEpsilon = 1e-12

var mutationMu sync.Mutex

type Service struct {
	AccountID string
	Now       func() time.Time
}

func DefaultService() Service { return Service{} }

func (s Service) nowMillis() int64 {
	if s.Now != nil {
		return s.Now().UTC().UnixMilli()
	}
	return time.Now().UTC().UnixMilli()
}

func normalizeOwner(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case OwnerAutoStrategy, OwnerNewCoinRush, OwnerNoticeAutoOrder, OwnerFundingRate, OwnerAgentTrade:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported futures owner %q", value)
	}
}

func normalizeSymbol(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "", fmt.Errorf("symbol is required")
	}
	return value, nil
}

func normalizePositionSide(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value != "LONG" && value != "SHORT" {
		return "", fmt.Errorf("position side must be LONG or SHORT")
	}
	return value, nil
}

func normalizeIntent(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case IntentOpen, IntentClose, IntentTakeProfit, IntentStopLoss:
		return value, nil
	default:
		return "", fmt.Errorf("unsupported order intent %q", value)
	}
}

func activePositionStatus(status string) bool {
	return status == PositionActive || status == PositionSuspended || status == PositionReconcileRequired
}

func liveOrderStatus(status string) bool {
	return status == OrderPending || status == OrderSubmitted || status == OrderPartiallyFilled || status == OrderReconcile
}

func (s Service) ClaimOrder(ctx context.Context, input ClaimOrderInput) (models.FuturesManagedOrder, error) {
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	owner, err := normalizeOwner(input.Owner)
	if err != nil {
		return models.FuturesManagedOrder{}, err
	}
	symbol, err := normalizeSymbol(input.Symbol)
	if err != nil {
		return models.FuturesManagedOrder{}, err
	}
	side, err := normalizePositionSide(input.PositionSide)
	if err != nil {
		return models.FuturesManagedOrder{}, err
	}
	intent, err := normalizeIntent(input.Intent)
	if err != nil {
		return models.FuturesManagedOrder{}, err
	}
	clientID := strings.TrimSpace(input.ClientOrderID)
	if clientID == "" || input.RequestedQty <= qtyEpsilon {
		return models.FuturesManagedOrder{}, fmt.Errorf("client order id and positive requested quantity are required")
	}

	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return models.FuturesManagedOrder{}, accountErr
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	tx, beginErr := orm.NewOrm().Begin()
	if beginErr != nil {
		return models.FuturesManagedOrder{}, beginErr
	}
	defer tx.Rollback()
	var o orm.QueryExecutor = tx
	var existing models.FuturesManagedOrder
	if err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("client_order_id", clientID).One(&existing); err == nil {
		if existing.Owner == owner && existing.AccountID == accountID && existing.Symbol == symbol && existing.PositionSide == side && existing.Intent == intent {
			return existing, nil
		}
		return models.FuturesManagedOrder{}, fmt.Errorf("client order id %q already belongs to another managed order", clientID)
	} else if err != orm.ErrNoRows {
		return models.FuturesManagedOrder{}, err
	}
	var positionID *int64
	if intent == IntentOpen {
		if err := ensureSlotAvailable(o, accountID, symbol, side, ""); err != nil {
			return models.FuturesManagedOrder{}, err
		}
	} else {
		if intent == IntentClose {
			var liveCloses []models.FuturesManagedOrder
			if _, queryErr := o.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("owner", owner).Filter("symbol", symbol).Filter("position_side", side).Filter("intent", IntentClose).All(&liveCloses); queryErr != nil {
				return models.FuturesManagedOrder{}, queryErr
			}
			for _, closeOrder := range liveCloses {
				if liveOrderStatus(closeOrder.Status) {
					return models.FuturesManagedOrder{}, fmt.Errorf("%s %s owner %s already has live close order %s", symbol, side, owner, closeOrder.ClientOrderID)
				}
			}
		}
		position, positionErr := s.getPosition(o, owner, symbol, side)
		if positionErr != nil {
			return models.FuturesManagedOrder{}, fmt.Errorf("%s %s is not managed by owner %s: %w", symbol, side, owner, positionErr)
		}
		if input.RequestedQty > position.ManagedQty+qtyEpsilon {
			return models.FuturesManagedOrder{}, fmt.Errorf("requested mutation qty %.12f exceeds managed qty %.12f for %s %s owner %s", input.RequestedQty, position.ManagedQty, symbol, side, owner)
		}
		positionID = &position.ID
	}
	now := s.nowMillis()
	row := models.FuturesManagedOrder{
		AccountID: accountID,
		Owner:     owner, Symbol: symbol, PositionSide: side, Intent: intent,
		ClientOrderID: clientID, RequestedQty: input.RequestedQty, PositionID: positionID,
		OrderType: strings.ToUpper(strings.TrimSpace(input.OrderType)), Status: OrderPending,
		SourceRef: strings.TrimSpace(input.SourceRef), CreatedAt: now, UpdatedAt: now,
	}
	if _, err := o.Insert(&row); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	return row, nil
}

func ensureSlotAvailable(o orm.QueryExecutor, accountID, symbol, side, ignoreClientOrderID string) error {
	var positions []models.FuturesManagedPosition
	if _, err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", accountID).Filter("symbol", symbol).Filter("position_side", side).All(&positions); err != nil {
		return err
	}
	for _, row := range positions {
		if activePositionStatus(row.Status) && row.ManagedQty > qtyEpsilon {
			return fmt.Errorf("%s %s already has an active managed position owned by %s", symbol, side, row.Owner)
		}
	}
	var orders []models.FuturesManagedOrder
	if _, err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("symbol", symbol).Filter("position_side", side).Filter("intent", IntentOpen).All(&orders); err != nil {
		return err
	}
	for _, row := range orders {
		if row.ClientOrderID == ignoreClientOrderID {
			continue
		}
		if liveOrderStatus(row.Status) {
			return fmt.Errorf("%s %s already has a live managed open order owned by %s", symbol, side, row.Owner)
		}
	}
	return nil
}

// ReserveSubmission is a durable, account-scoped compare-and-swap that runs
// BEFORE any exchange request. It closes the cross-process window where two
// workers could both claim the same still-pending clientOrderID and submit it.
// A reservation survives restart; the next process may only reconcile.
func (s Service) ReserveSubmission(ctx context.Context, clientOrderID string) (bool, error) {
	id, err := s.accountID()
	if err != nil {
		return false, err
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return false, fmt.Errorf("client order id required")
	}
	n, err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", id).Filter("client_order_id", clientOrderID).
		Filter("status", OrderPending).Update(orm.Params{"status": OrderSubmitted, "updated_at": s.nowMillis()})
	return n == 1, err
}

func (s Service) MarkOrderSubmitted(ctx context.Context, clientOrderID, exchangeOrderID string) error {
	return s.updateOrder(ctx, clientOrderID, orm.Params{"status": OrderSubmitted, "exchange_order_id": strings.TrimSpace(exchangeOrderID)})
}

// SetActualOrderID is only called after the Algo API has identified its
// actual exchange matching order and the subsequent order lookup succeeded.
// A different nonzero identity cannot silently replace an earlier mapping.
func (s Service) SetActualOrderID(ctx context.Context, clientOrderID, actualID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id, err := strconv.ParseInt(strings.TrimSpace(actualID), 10, 64)
	if err != nil || id <= 0 {
		return fmt.Errorf("invalid actual exchange order id")
	}
	account, err := s.accountID()
	if err != nil {
		return err
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var order models.FuturesManagedOrder
	if err := tx.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", account).
		Filter("client_order_id", clientOrderID).One(&order); err != nil {
		return err
	}
	if !isAlgoManagedOrderType(order.OrderType) {
		return fmt.Errorf("actual_order_id mapping applies only to Algo orders")
	}
	if order.ActualOrderID != nil && *order.ActualOrderID != actualID {
		return fmt.Errorf("conflicting Algo actual order mapping for %s", clientOrderID)
	}
	order.ActualOrderID = &actualID
	order.UpdatedAt = s.nowMillis()
	if _, err = tx.Update(&order); err != nil {
		return err
	}
	if err = tradefills.Attach(tx, order); err != nil {
		return err
	}
	return tx.Commit()
}

func (s Service) SetOrderStatus(ctx context.Context, clientOrderID, status string) error {
	status = strings.TrimSpace(status)
	switch status {
	case OrderPending, OrderSubmitted, OrderPartiallyFilled, OrderFilled, OrderCanceled, OrderFailed, OrderReconcile:
	default:
		return fmt.Errorf("unsupported managed order status %q", status)
	}
	return s.updateOrder(ctx, clientOrderID, orm.Params{"status": status})
}

func (s Service) MarkOrderReconciled(ctx context.Context, clientOrderID string) error {
	return s.updateOrder(ctx, clientOrderID, orm.Params{"last_reconciled_at": s.nowMillis()})
}

func (s Service) updateOrder(ctx context.Context, clientOrderID string, params orm.Params) error {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return accountErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return fmt.Errorf("client order id is required")
	}
	params["updated_at"] = s.nowMillis()
	count, err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("client_order_id", clientOrderID).Update(params)
	if err != nil {
		return err
	}
	if count == 0 {
		return orm.ErrNoRows
	}
	return nil
}

func (s Service) ApplyFill(ctx context.Context, clientOrderID string, cumulativeFilledQty, entryPrice float64) (models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if cumulativeFilledQty < 0 {
		return models.FuturesManagedPosition{}, fmt.Errorf("filled quantity cannot be negative")
	}
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return models.FuturesManagedPosition{}, accountErr
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	defer tx.Rollback()
	var o orm.QueryExecutor = tx
	var order models.FuturesManagedOrder
	if err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("client_order_id", strings.TrimSpace(clientOrderID)).One(&order); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if order.AccountID != accountID {
		return models.FuturesManagedPosition{}, fmt.Errorf("order account mismatch")
	}
	if cumulativeFilledQty+qtyEpsilon < order.FilledQty {
		return models.FuturesManagedPosition{}, fmt.Errorf("cumulative filled quantity cannot decrease")
	}
	delta := cumulativeFilledQty - order.FilledQty
	if delta > qtyEpsilon && order.Intent == IntentOpen && order.PositionID == nil && !liveOrderStatus(order.Status) {
		// Do not resurrect a closed/failed legacy OPEN after the schema
		// switch. A terminal unbound order cannot establish a new cycle.
		return models.FuturesManagedPosition{}, fmt.Errorf("terminal unbound OPEN %s requires reconciliation before applying fill", order.ClientOrderID)
	}
	order.FilledQty = cumulativeFilledQty
	if cumulativeFilledQty+qtyEpsilon >= order.RequestedQty {
		order.Status = OrderFilled
	} else if cumulativeFilledQty > qtyEpsilon {
		order.Status = OrderPartiallyFilled
	}
	order.UpdatedAt = s.nowMillis()
	var position models.FuturesManagedPosition
	if delta > qtyEpsilon {
		position, err = s.applyPositionDelta(o, &order, delta, entryPrice)
	} else if order.PositionID != nil {
		position, err = s.getOrderPosition(o, order)
	} else {
		// A zero-fill opening order has no position cycle yet. Old unbound
		// orders also must not be attached to an unrelated active position.
		err = nil
	}
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	// The order's quantity, status and newly assigned position_id commit
	// atomically with the managed position change. No double-count on replay.
	if _, err := o.Update(&order); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	// A WS TRADE can precede the REST response that creates position_id.
	// Link only already verified exact order identities in this transaction.
	if order.PositionID != nil {
		if err := tradefills.Attach(o, order); err != nil {
			return models.FuturesManagedPosition{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	return position, nil
}

// SyncOpenAverageFromFills corrects the legacy position entry snapshot only
// when the complete confirmed OPEN quantity is represented by actual fills.
// Never derive money from an order ACK, a quote, or partial/unknown trades.
func (s Service) SyncOpenAverageFromFills(ctx context.Context, clientOrderID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	account, err := s.accountID()
	if err != nil {
		return err
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	tx, err := orm.NewOrm().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var order models.FuturesManagedOrder
	if err = tx.QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", account).
		Filter("client_order_id", clientOrderID).One(&order); err != nil {
		return err
	}
	if order.Intent != IntentOpen || order.PositionID == nil {
		return nil
	}
	qty, average, err := tradefills.WeightedAverage(tx, order)
	if err != nil {
		return err
	}
	if qty == nil || average == nil {
		return fmt.Errorf("OPEN trade ledger is incomplete")
	}
	expected := new(big.Rat).SetFloat64(order.FilledQty)
	if expected == nil {
		return fmt.Errorf("invalid managed cumulative quantity")
	}
	diff := new(big.Rat).Abs(new(big.Rat).Sub(qty, expected))
	if diff.Cmp(big.NewRat(1, 100000000)) > 0 {
		return fmt.Errorf("OPEN fill coverage does not match managed quantity")
	}
	value, _ := new(big.Float).SetPrec(160).SetRat(average).Float64()
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return fmt.Errorf("invalid exact fill weighted entry price")
	}
	position, err := s.getOrderPosition(tx, order)
	if err != nil {
		return err
	}
	if math.Abs(position.EntryPrice-value) > 1e-12 {
		position.EntryPrice = value
		position.UpdatedAt = s.nowMillis()
		if _, err = tx.Update(&position, "entry_price", "updated_at"); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// getOrderPosition only resolves the pinned cycle, including closed cycles.
// Looking up the newest active position would corrupt repeated trade cycles.
func (s Service) getOrderPosition(o orm.QueryExecutor, order models.FuturesManagedOrder) (models.FuturesManagedPosition, error) {
	if order.PositionID == nil || *order.PositionID <= 0 {
		return models.FuturesManagedPosition{}, fmt.Errorf("managed order %s has no bound position_id; reconcile before applying fill", order.ClientOrderID)
	}
	var position models.FuturesManagedPosition
	if err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("id", *order.PositionID).
		Filter("account_id", order.AccountID).One(&position); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if position.Owner != order.Owner || position.Symbol != order.Symbol || position.PositionSide != order.PositionSide {
		return models.FuturesManagedPosition{}, fmt.Errorf("position_id %d identity mismatch for order %s", *order.PositionID, order.ClientOrderID)
	}
	return position, nil
}

func (s Service) applyPositionDelta(o orm.QueryExecutor, order *models.FuturesManagedOrder, delta, entryPrice float64) (models.FuturesManagedPosition, error) {
	if order.Intent == IntentOpen {
		var position models.FuturesManagedPosition
		if order.PositionID == nil {
			// Only the first actual OPEN fill creates a cycle. Do not
			// silently attach a historical order to another active position.
			if _, err := s.getPosition(o, order.Owner, order.Symbol, order.PositionSide); err == nil {
				return models.FuturesManagedPosition{}, fmt.Errorf("unbound OPEN %s conflicts with an active position; reconcile first", order.ClientOrderID)
			} else if err != orm.ErrNoRows {
				return models.FuturesManagedPosition{}, err
			}
			if err := ensureSlotAvailable(o, order.AccountID, order.Symbol, order.PositionSide, order.ClientOrderID); err != nil {
				return models.FuturesManagedPosition{}, err
			}
			now := s.nowMillis()
			position = models.FuturesManagedPosition{
				AccountID: order.AccountID, Owner: order.Owner, Symbol: order.Symbol, PositionSide: order.PositionSide,
				ManagedQty: delta, EntryPrice: entryPrice, Status: PositionActive, SourceRef: order.SourceRef,
				CreatedAt: now, UpdatedAt: now, SettlementStatus: "open",
			}
			id, err := o.Insert(&position)
			if err != nil {
				return models.FuturesManagedPosition{}, err
			}
			position.ID = id
			order.PositionID = &position.ID
			return position, nil
		}
		var err error
		position, err = s.getOrderPosition(o, *order)
		if err != nil {
			return models.FuturesManagedPosition{}, err
		}
		if !activePositionStatus(position.Status) || position.ManagedQty <= qtyEpsilon {
			return models.FuturesManagedPosition{}, fmt.Errorf("OPEN %s refers to inactive position cycle %d", order.ClientOrderID, position.ID)
		}
		position.ManagedQty += delta
		// Binance's REST ap is the CUMULATIVE order average, not the
		// delta's price. A single OPEN establishes this entire cycle.
		// 2@100 then 2@120 must produce 110, never 105.
		if entryPrice > 0 {
			position.EntryPrice = entryPrice
		}
		position.Status, position.UpdatedAt = PositionActive, s.nowMillis()
		_, err = o.Update(&position)
		return position, err
	}
	// CLOSE/TP/SL must use the immutable position_id reserved at ClaimOrder.
	// A historical unbound order is unsafe to apply to the next trade cycle.
	position, err := s.getOrderPosition(o, *order)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if !activePositionStatus(position.Status) || position.ManagedQty+qtyEpsilon < delta {
		return models.FuturesManagedPosition{}, fmt.Errorf("%s fill exceeds or targets closed managed position cycle %d", order.Intent, position.ID)
	}
	position.ManagedQty = math.Max(0, position.ManagedQty-delta)
	position.UpdatedAt = s.nowMillis()
	if position.ManagedQty <= qtyEpsilon {
		position.ManagedQty, position.Status, position.ClosedAt = 0, PositionClosed, position.UpdatedAt
		if position.SettlementStatus == "open" {
			position.SettlementStatus = "incomplete"
		}
	}
	_, err = o.Update(&position)
	return position, err
}

func (s Service) getPosition(o orm.QueryExecutor, owner, symbol, side string) (models.FuturesManagedPosition, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return models.FuturesManagedPosition{}, accountErr
	}
	var rows []models.FuturesManagedPosition
	_, err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", accountID).Filter("owner", owner).Filter("symbol", symbol).Filter("position_side", side).OrderBy("-id").All(&rows)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	for _, row := range rows {
		if activePositionStatus(row.Status) {
			return row, nil
		}
	}
	return models.FuturesManagedPosition{}, orm.ErrNoRows
}

func (s Service) GetPosition(ctx context.Context, owner, symbol, side string) (models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	owner, err := normalizeOwner(owner)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	symbol, err = normalizeSymbol(symbol)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	side, err = normalizePositionSide(side)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	return s.getPosition(orm.NewOrm(), owner, symbol, side)
}

func (s Service) GetOrder(ctx context.Context, clientOrderID string) (models.FuturesManagedOrder, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return models.FuturesManagedOrder{}, accountErr
	}
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return models.FuturesManagedOrder{}, fmt.Errorf("client order id is required")
	}
	var row models.FuturesManagedOrder
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

// UnsettledClosedPositions includes only post-v22 cycles. Historical CLOSED
// positions have settlement_status=legacy and must never retro-trigger risk.
func (s Service) UnsettledClosedPositions(ctx context.Context, owner string) ([]models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	account, err := s.accountID()
	if err != nil {
		return nil, err
	}
	owner, err = normalizeOwner(owner)
	if err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedPosition
	_, err = orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).
		Filter("account_id", account).Filter("owner", owner).
		Filter("status", PositionClosed).Filter("settlement_status", SettlementIncomplete).
		OrderBy("closed_at", "id").All(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) > 10000 {
		return nil, fmt.Errorf("unsettled positions exceed 10000; audit backlog before processing")
	}
	return rows, nil
}

func (s Service) ActivePositions(ctx context.Context, owner string) ([]models.FuturesManagedPosition, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return nil, accountErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, err := normalizeOwner(owner)
	if err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedPosition
	if _, err := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", accountID).Filter("owner", owner).OrderBy("id").All(&rows); err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, row := range rows {
		if activePositionStatus(row.Status) && row.ManagedQty > qtyEpsilon {
			out = append(out, row)
		}
	}
	return out, nil
}

// OutstandingOrders retrieves ALL account-scoped nonterminal orders, not
// just the newest 200/500 records. Recovery must never silently miss older
// unresolved submissions after a long outage or restart.
func (s Service) OutstandingOrders(ctx context.Context) ([]models.FuturesManagedOrder, error) {
	account, err := s.accountID()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedOrder
	_, err = orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).
		Filter("account_id", account).
		Filter("status__in", []string{OrderPending, OrderSubmitted, OrderPartiallyFilled, OrderReconcile}).
		OrderBy("id").All(&rows)
	if err != nil {
		return nil, err
	}
	if len(rows) > 10000 {
		return nil, fmt.Errorf("outstanding orders exceed safe recovery scan limit")
	}
	return rows, nil
}

func (s Service) ActiveOrders(ctx context.Context, owner string) ([]models.FuturesManagedOrder, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return nil, accountErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, err := normalizeOwner(owner)
	if err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedOrder
	if _, err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID).Filter("owner", owner).OrderBy("id").All(&rows); err != nil {
		return nil, err
	}
	out := rows[:0]
	for _, row := range rows {
		if liveOrderStatus(row.Status) {
			out = append(out, row)
		}
	}
	return out, nil
}

func (s Service) ReconcilePosition(ctx context.Context, owner, symbol, side string, accountQty float64) (models.FuturesManagedPosition, error) {
	if accountQty < 0 {
		accountQty = math.Abs(accountQty)
	}
	mutationMu.Lock()
	defer mutationMu.Unlock()
	position, err := s.GetPosition(ctx, owner, symbol, side)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	now := s.nowMillis()
	position.LastReconciledAt, position.UpdatedAt = now, now
	if accountQty <= qtyEpsilon {
		position.ManagedQty, position.Status, position.ClosedAt = 0, PositionClosed, now
		if position.SettlementStatus == "open" {
			position.SettlementStatus = "incomplete"
		}
	} else if accountQty+qtyEpsilon < position.ManagedQty {
		position.ManagedQty = accountQty
	}
	_, err = orm.NewOrm().Update(&position)
	return position, err
}

// ReconcileObservedPosition is for verified periodic account snapshots.
// A single zero observation must not permanently release ownership: an
// asynchronous fill or a lagging WS mirror can temporarily omit the position.
// All callers MUST pass a freshly verified exchange quantity before shrinking
// or closing ownership. A cached WS/local quantity is not sufficient evidence.
// This method deliberately does not claim unknown manual account positions.
func (s Service) ReconcileObservedPosition(ctx context.Context, owner, symbol, side string, accountQty float64) (models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if math.IsNaN(accountQty) || math.IsInf(accountQty, 0) {
		return models.FuturesManagedPosition{}, fmt.Errorf("invalid observed account quantity")
	}
	accountQty = math.Abs(accountQty)
	mutationMu.Lock()
	defer mutationMu.Unlock()

	position, err := s.GetPosition(ctx, owner, symbol, side)
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	previous := position
	now := s.nowMillis()
	if accountQty <= qtyEpsilon {
		if position.Status != PositionReconcileRequired {
			// First zero: reserve the owned quantity until a second verification.
			position.Status = PositionReconcileRequired
			position.LastReconciledAt, position.UpdatedAt = now, now
		} else if now-position.LastReconciledAt >= 10_000 {
			position.ManagedQty, position.Status, position.ClosedAt = 0, PositionClosed, now
			if position.SettlementStatus == "open" {
				position.SettlementStatus = "incomplete"
			}
			position.LastReconciledAt, position.UpdatedAt = now, now
		} else {
			return position, nil
		}
	} else {
		// Never adopt a manual increase; clear a false zero when it reappears.
		if accountQty+qtyEpsilon < position.ManagedQty {
			position.ManagedQty = accountQty
		}
		if position.Status == PositionReconcileRequired {
			position.Status = PositionActive
		}
		position.LastReconciledAt, position.UpdatedAt = now, now
	}
	// Another process may apply a fill while this cycle holds a stale snapshot.
	// Compare previous state instead of overwriting newer ownership.
	n, err := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).
		Filter("id", previous.ID).Filter("account_id", previous.AccountID).
		Filter("status", previous.Status).Filter("managed_qty", previous.ManagedQty).
		Filter("updated_at", previous.UpdatedAt).
		Update(orm.Params{
			"managed_qty": position.ManagedQty, "status": position.Status,
			"closed_at": position.ClosedAt, "last_reconciled_at": position.LastReconciledAt,
			"updated_at": position.UpdatedAt, "settlement_status": position.SettlementStatus,
		})
	if err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if n != 1 {
		return models.FuturesManagedPosition{}, fmt.Errorf("ownership changed during reconciliation for %s %s; retry after fresh snapshot", symbol, side)
	}
	return position, nil
}

func (s Service) CloseQuantity(ctx context.Context, owner, symbol, side string, accountQty float64) (float64, error) {
	position, err := s.GetPosition(ctx, owner, symbol, side)
	if err != nil {
		return 0, err
	}
	if accountQty < 0 {
		accountQty = math.Abs(accountQty)
	}
	return math.Min(position.ManagedQty, accountQty), nil
}

func (s Service) SuspendPosition(ctx context.Context, owner, symbol, side string) error {
	position, err := s.GetPosition(ctx, owner, symbol, side)
	if err != nil {
		return err
	}
	position.Status, position.UpdatedAt = PositionSuspended, s.nowMillis()
	_, err = orm.NewOrm().Update(&position)
	return err
}

func (s Service) ListPositions(ctx context.Context, owner string, activeOnly bool) ([]models.FuturesManagedPosition, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return nil, accountErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).Filter("account_id", accountID)
	if strings.TrimSpace(owner) != "" {
		normalized, err := normalizeOwner(owner)
		if err != nil {
			return nil, err
		}
		q = q.Filter("owner", normalized)
	}
	var rows []models.FuturesManagedPosition
	if _, err := q.OrderBy("-id").All(&rows); err != nil {
		return nil, err
	}
	if !activeOnly {
		return rows, nil
	}
	out := make([]models.FuturesManagedPosition, 0, len(rows))
	for _, row := range rows {
		if activePositionStatus(row.Status) && row.ManagedQty > qtyEpsilon {
			out = append(out, row)
		}
	}
	return out, nil
}

func (s Service) ListOrders(ctx context.Context, owner string, limit int) ([]models.FuturesManagedOrder, error) {
	accountID, accountErr := s.accountID()
	if accountErr != nil {
		return nil, accountErr
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("account_id", accountID)
	if strings.TrimSpace(owner) != "" {
		normalized, err := normalizeOwner(owner)
		if err != nil {
			return nil, err
		}
		q = q.Filter("owner", normalized)
	}
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows []models.FuturesManagedOrder
	_, err := q.OrderBy("-id").Limit(limit).All(&rows)
	return rows, err
}
