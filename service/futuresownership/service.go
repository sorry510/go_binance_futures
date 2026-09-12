package futuresownership

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const qtyEpsilon = 1e-12

var mutationMu sync.Mutex

type Service struct {
	Now func() time.Time
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

	mutationMu.Lock()
	defer mutationMu.Unlock()
	o := orm.NewOrm()
	var existing models.FuturesManagedOrder
	if err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientID).One(&existing); err == nil {
		if existing.Owner == owner && existing.Symbol == symbol && existing.PositionSide == side && existing.Intent == intent {
			return existing, nil
		}
		return models.FuturesManagedOrder{}, fmt.Errorf("client order id %q already belongs to another managed order", clientID)
	} else if err != orm.ErrNoRows {
		return models.FuturesManagedOrder{}, err
	}
	if intent == IntentOpen {
		if err := ensureSlotAvailable(o, symbol, side, ""); err != nil {
			return models.FuturesManagedOrder{}, err
		}
	} else {
		if intent == IntentClose {
			var liveCloses []models.FuturesManagedOrder
			if _, queryErr := o.QueryTable(new(models.FuturesManagedOrder)).Filter("owner", owner).Filter("symbol", symbol).Filter("position_side", side).Filter("intent", IntentClose).All(&liveCloses); queryErr != nil {
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
	}
	now := s.nowMillis()
	row := models.FuturesManagedOrder{
		Owner: owner, Symbol: symbol, PositionSide: side, Intent: intent,
		ClientOrderID: clientID, RequestedQty: input.RequestedQty,
		OrderType: strings.ToUpper(strings.TrimSpace(input.OrderType)), Status: OrderPending,
		SourceRef: strings.TrimSpace(input.SourceRef), CreatedAt: now, UpdatedAt: now,
	}
	if _, err := o.Insert(&row); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	return row, nil
}

func ensureSlotAvailable(o orm.Ormer, symbol, side, ignoreClientOrderID string) error {
	var positions []models.FuturesManagedPosition
	if _, err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("symbol", symbol).Filter("position_side", side).All(&positions); err != nil {
		return err
	}
	for _, row := range positions {
		if activePositionStatus(row.Status) && row.ManagedQty > qtyEpsilon {
			return fmt.Errorf("%s %s already has an active managed position owned by %s", symbol, side, row.Owner)
		}
	}
	var orders []models.FuturesManagedOrder
	if _, err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("symbol", symbol).Filter("position_side", side).Filter("intent", IntentOpen).All(&orders); err != nil {
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

func (s Service) MarkOrderSubmitted(ctx context.Context, clientOrderID, exchangeOrderID string) error {
	return s.updateOrder(ctx, clientOrderID, orm.Params{"status": OrderSubmitted, "exchange_order_id": strings.TrimSpace(exchangeOrderID)})
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

func (s Service) updateOrder(ctx context.Context, clientOrderID string, params orm.Params) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return fmt.Errorf("client order id is required")
	}
	params["updated_at"] = s.nowMillis()
	count, err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientOrderID).Update(params)
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
	mutationMu.Lock()
	defer mutationMu.Unlock()
	o := orm.NewOrm()
	var order models.FuturesManagedOrder
	if err := o.QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", strings.TrimSpace(clientOrderID)).One(&order); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if cumulativeFilledQty+qtyEpsilon < order.FilledQty {
		return models.FuturesManagedPosition{}, fmt.Errorf("cumulative filled quantity cannot decrease")
	}
	delta := cumulativeFilledQty - order.FilledQty
	order.FilledQty = cumulativeFilledQty
	if cumulativeFilledQty+qtyEpsilon >= order.RequestedQty {
		order.Status = OrderFilled
	} else if cumulativeFilledQty > qtyEpsilon {
		order.Status = OrderPartiallyFilled
	}
	order.UpdatedAt = s.nowMillis()
	if _, err := o.Update(&order); err != nil {
		return models.FuturesManagedPosition{}, err
	}
	if delta <= qtyEpsilon {
		return s.getPosition(o, order.Owner, order.Symbol, order.PositionSide)
	}
	return s.applyPositionDelta(o, order, delta, entryPrice)
}

func (s Service) applyPositionDelta(o orm.Ormer, order models.FuturesManagedOrder, delta, entryPrice float64) (models.FuturesManagedPosition, error) {
	position, err := s.getPosition(o, order.Owner, order.Symbol, order.PositionSide)
	if order.Intent == IntentOpen {
		if err == orm.ErrNoRows {
			if err := ensureSlotAvailable(o, order.Symbol, order.PositionSide, order.ClientOrderID); err != nil {
				return models.FuturesManagedPosition{}, err
			}
			now := s.nowMillis()
			position = models.FuturesManagedPosition{Owner: order.Owner, Symbol: order.Symbol, PositionSide: order.PositionSide, ManagedQty: delta, EntryPrice: entryPrice, Status: PositionActive, SourceRef: order.SourceRef, CreatedAt: now, UpdatedAt: now}
			_, err = o.Insert(&position)
			return position, err
		}
		if err != nil {
			return models.FuturesManagedPosition{}, err
		}
		oldQty := position.ManagedQty
		position.ManagedQty += delta
		if entryPrice > 0 && position.ManagedQty > qtyEpsilon {
			position.EntryPrice = ((position.EntryPrice * oldQty) + (entryPrice * delta)) / position.ManagedQty
		}
		position.Status, position.UpdatedAt = PositionActive, s.nowMillis()
		_, err = o.Update(&position)
		return position, err
	}
	if err != nil {
		return models.FuturesManagedPosition{}, fmt.Errorf("cannot apply %s fill without managed position: %w", order.Intent, err)
	}
	position.ManagedQty = math.Max(0, position.ManagedQty-delta)
	position.UpdatedAt = s.nowMillis()
	if position.ManagedQty <= qtyEpsilon {
		position.ManagedQty, position.Status, position.ClosedAt = 0, PositionClosed, position.UpdatedAt
	}
	_, err = o.Update(&position)
	return position, err
}

func (s Service) getPosition(o orm.Ormer, owner, symbol, side string) (models.FuturesManagedPosition, error) {
	var rows []models.FuturesManagedPosition
	_, err := o.QueryTable(new(models.FuturesManagedPosition)).Filter("owner", owner).Filter("symbol", symbol).Filter("position_side", side).OrderBy("-id").All(&rows)
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
	if err := ctx.Err(); err != nil {
		return models.FuturesManagedOrder{}, err
	}
	clientOrderID = strings.TrimSpace(clientOrderID)
	if clientOrderID == "" {
		return models.FuturesManagedOrder{}, fmt.Errorf("client order id is required")
	}
	var row models.FuturesManagedOrder
	err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("client_order_id", clientOrderID).One(&row)
	return row, err
}

func (s Service) ActivePositions(ctx context.Context, owner string) ([]models.FuturesManagedPosition, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, err := normalizeOwner(owner)
	if err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedPosition
	if _, err := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition)).Filter("owner", owner).OrderBy("id").All(&rows); err != nil {
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

func (s Service) ActiveOrders(ctx context.Context, owner string) ([]models.FuturesManagedOrder, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner, err := normalizeOwner(owner)
	if err != nil {
		return nil, err
	}
	var rows []models.FuturesManagedOrder
	if _, err := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder)).Filter("owner", owner).OrderBy("id").All(&rows); err != nil {
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
	} else if accountQty+qtyEpsilon < position.ManagedQty {
		position.ManagedQty = accountQty
	}
	_, err = orm.NewOrm().Update(&position)
	return position, err
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := orm.NewOrm().QueryTable(new(models.FuturesManagedPosition))
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	q := orm.NewOrm().QueryTable(new(models.FuturesManagedOrder))
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
