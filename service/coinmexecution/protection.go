package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/feature"
	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

// PlaceProtection creates a COIN-M CONDITIONAL Algo with closePosition=true.
// No STOP/TP order ever reaches the ordinary /dapi/v1/order endpoint.
// This does not claim future child fills without independent exchange evidence.
func (p *Processor) PlaceProtection(ctx context.Context, req ProtectionRequest) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unavailable")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := validClientID(req.ClientAlgoID); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if req.Type != "STOP_MARKET" && req.Type != "TAKE_PROFIT_MARKET" {
		return models.CoinMManagedOrder{}, errors.New("COIN-M protective order requires STOP_MARKET or TAKE_PROFIT_MARKET")
	}
	if req.PositionID <= 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M protection requires position ID")
	}
	existing, found, err := localOrder(p.DB, req.ClientAlgoID)
	if err != nil {
		return existing, err
	}
	if found {
		if existing.PositionID != req.PositionID || existing.OrderType != req.Type || existing.Intent != "PROTECT" {
			return existing, errors.New("COIN-M protection client ID belongs to another intent")
		}
		return existing, nil // Never blindly POST after a timeout/restart.
	}
	if err := p.requireReady(ctx); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	pos, err := loadPosition(p.DB, req.PositionID)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = validatePosition(pos); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if pos.Status != "OPEN" {
		return models.CoinMManagedOrder{}, errors.New("COIN-M protection requires verified OPEN position")
	}
	remain, err := positive(pos.RemainingContracts)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	var conflicts int64
	if err = p.DB.Raw("SELECT COUNT(*) FROM coin_m_managed_orders WHERE position_id=? AND intent!='PROTECT' AND status IN ('PENDING_SUBMIT','UNKNOWN','NEW','PARTIALLY_FILLED','ACTIVE','AWAITING_FILLS')", pos.ID).QueryRow(&conflicts); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if conflicts > 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M pending order/protection must settle first")
	}
	var existingAlgos []models.CoinMManagedOrder
	_, err = p.DB.Raw("SELECT "+orderCols+" FROM coin_m_managed_orders WHERE position_id=? AND intent='PROTECT' AND status NOT IN ('CANCELED','FILLED','EXPIRED','REJECTED')", pos.ID).QueryRows(&existingAlgos)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if len(existingAlgos) > 1 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M supports at most one managed STOP plus one TAKE_PROFIT")
	}
	for _, old := range existingAlgos {
		if old.Status != "ACTIVE" || old.OrderType == req.Type {
			return models.CoinMManagedOrder{}, errors.New("COIN-M same-type or uncertain protective order is already active")
		}
	}
	if err = ctx.Err(); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	algos, err := p.API.OpenAlgos(ctx, pos.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, fmt.Errorf("%w: COIN-M open protective order lookup failed: %v", ErrUncertain, err)
	}
	// A second protective Algo may be added only when every live exchange
	// Algo exactly matches the one already managed in this cycle.
	if len(algos) != len(existingAlgos) {
		return models.CoinMManagedOrder{}, errors.New("COIN-M exchange protective Algo inventory differs from Managed ledger")
	}
	for _, remoteAlgo := range algos {
		matched := false
		for _, owned := range existingAlgos {
			if remoteAlgo.ClientAlgoID == owned.ClientOrderID &&
				strconv.FormatInt(remoteAlgo.AlgoID, 10) == owned.ExchangeOrderID &&
				remoteAlgo.Symbol == owned.Symbol &&
				remoteAlgo.PositionSide == owned.PositionSide &&
				remoteAlgo.Side == owned.Side &&
				remoteAlgo.OrderType == owned.OrderType {
				matched = true
				break
			}
		}
		if !matched {
			return models.CoinMManagedOrder{}, errors.New("unmanaged or mismatched exchange Algo blocks protection")
		}
	}
	remote, err := p.API.Position(ctx, pos.Symbol, pos.PositionSide)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	n, err := positive(remote)
	if err != nil || n.Cmp(remain) != 0 {
		return models.CoinMManagedOrder{}, errors.New("COIN-M remote position differs from managed remaining contracts")
	}
	spec, err := p.API.Spec(ctx, pos.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if spec.Symbol != pos.Symbol || spec.ContractSize <= 0 || spec.MarginAsset != pos.MarginAsset {
		return models.CoinMManagedOrder{}, errors.New("COIN-M protective spec is missing/mismatched")
	}
	if err = feature.ValidateCoinMOrderFilters(spec, str(remain), req.TriggerPrice, false); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	trigger, err := positive(req.TriggerPrice)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	mark, err := p.API.Mark(ctx, pos.Symbol)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	current, err := positive(mark)
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	isLong := pos.PositionSide == "LONG"
	if (req.Type == "STOP_MARKET" && ((isLong && trigger.Cmp(current) >= 0) || (!isLong && trigger.Cmp(current) <= 0))) ||
		(req.Type == "TAKE_PROFIT_MARKET" && ((isLong && trigger.Cmp(current) <= 0) || (!isLong && trigger.Cmp(current) >= 0))) {
		return models.CoinMManagedOrder{}, errors.New("COIN-M protective trigger is on wrong side of mark price")
	}
	side, _ := orderSide(pos.PositionSide, "CLOSE")
	sent := binance.CoinMAlgoOrderRequest{Symbol: pos.Symbol, PositionSide: pos.PositionSide,
		Side: side, Type: req.Type, TriggerPrice: req.TriggerPrice, ClientAlgoID: req.ClientAlgoID}
	tx, err := p.DB.Begin()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	_, err = tx.Raw("INSERT INTO coin_m_managed_orders (position_id,symbol,position_side,intent,side,order_type,client_order_id,exchange_order_id,requested_contracts,filled_contracts,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,NULL,?,?,?, ?,?)",
		pos.ID, pos.Symbol, pos.PositionSide, "PROTECT", side, req.Type, req.ClientAlgoID,
		str(remain), "0", "PENDING_SUBMIT", now, now).Exec()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = tx.Commit(); err != nil {
		return models.CoinMManagedOrder{}, err
	}
	if err = ctx.Err(); err != nil {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE client_order_id=?", req.ClientAlgoID).Exec()
		local, _ := loadOrder(p.DB, req.ClientAlgoID)
		return local, fmt.Errorf("%w: protected intent persisted but context cancelled: %v", ErrUncertain, err)
	}
	ack, postErr := p.API.Algo(ctx, sent)
	if postErr != nil {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE client_order_id=?", req.ClientAlgoID).Exec()
		local, _ := loadOrder(p.DB, req.ClientAlgoID)
		return local, fmt.Errorf("%w: COIN-M Algo POST may have succeeded: %v", ErrUncertain, postErr)
	}
	if ack == nil || ack.AlgoID <= 0 || ack.ClientAlgoID != req.ClientAlgoID || ack.Symbol != pos.Symbol ||
		ack.Side != side || ack.PositionSide != pos.PositionSide {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE client_order_id=?", req.ClientAlgoID).Exec()
		return models.CoinMManagedOrder{}, ErrUncertain
	}
	_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET exchange_order_id=?,status='ACTIVE',updated_at=? WHERE client_order_id=?",
		strconv.FormatInt(ack.AlgoID, 10), time.Now().UnixMilli(), req.ClientAlgoID).Exec()
	if err != nil {
		return models.CoinMManagedOrder{}, err
	}
	return loadOrder(p.DB, req.ClientAlgoID)
}

// CancelProtection is restricted to a previously owned protective clientAlgoId.
// No exchange-wide cancel-all and no cancellation by arbitrary symbol.
func (p *Processor) CancelProtection(ctx context.Context, clientAlgoID string) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unavailable")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	order, err := loadOrder(p.DB, clientAlgoID)
	if err != nil {
		return order, err
	}
	if order.Intent != "PROTECT" || (order.OrderType != "STOP_MARKET" && order.OrderType != "TAKE_PROFIT_MARKET") {
		return order, errors.New("COIN-M order is not managed protective Algo")
	}
	if order.Status == "CANCELED" {
		return order, nil
	}
	if order.Status == "TRIGGERED" || order.Status == "FILLED" ||
		order.Status == "PARTIALLY_FILLED" || order.Status == "AWAITING_FILLS" {
		return order, errors.New("COIN-M Algo has triggered; reconcile child order instead of canceling")
	}
	if err = p.requireReady(ctx); err != nil {
		return order, err
	}
	if err = p.API.CancelAlgo(ctx, order.Symbol, order.ClientOrderID); err != nil {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE id=?", order.ID).Exec()
		return order, fmt.Errorf("%w: cancel COIN-M protection: %v", ErrUncertain, err)
	}
	// Cancellation ACK can race a trigger. Verify signed Algo state.
	verified, verifyErr := p.API.QueryAlgo(ctx, order.Symbol, order.ClientOrderID)
	if verifyErr != nil || verified.ClientAlgoID != order.ClientOrderID ||
		verified.Symbol != order.Symbol || verified.PositionSide != order.PositionSide ||
		verified.Side != order.Side || verified.OrderType != order.OrderType || verified.AlgoID <= 0 ||
		(order.ExchangeOrderID != "" && order.ExchangeOrderID != strconv.FormatInt(verified.AlgoID, 10)) ||
		(strings.ToUpper(verified.AlgoStatus) != "CANCELED" &&
			strings.ToUpper(verified.AlgoStatus) != "CANCELLED") {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN',updated_at=? WHERE id=?", time.Now().UnixMilli(), order.ID).Exec()
		return order, fmt.Errorf("%w: cancelled protection not confirmed: %v", ErrUncertain, verifyErr)
	}
	if childErr := algoChildPresentOrUnknown(verified.ActualOrderID); childErr != nil {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN',updated_at=? WHERE id=?", time.Now().UnixMilli(), order.ID).Exec()
		return order, childErr
	}
	_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET status='CANCELED',updated_at=? WHERE id=?", time.Now().UnixMilli(), order.ID).Exec()
	if err != nil {
		return order, err
	}
	return loadOrder(p.DB, clientAlgoID)
}
func checkFaceValue(left string, right int64) bool {
	a, e := positive(left)
	return e == nil && a.Cmp(new(big.Rat).SetInt64(right)) == 0
}
