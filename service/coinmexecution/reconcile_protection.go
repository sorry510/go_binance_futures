package coinmexecution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

// confirmedAlgoChildID accepts only a positive exchange-generated order ID.
// The absence of actualOrderId never authorizes claiming another symbol's Fill.
func confirmedAlgoChildID(raw []byte) (int64, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return 0, errors.New("COIN-M Algo child order ID not yet available")
	}
	value := strings.Trim(string(raw), "\"")
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("COIN-M Algo child order ID not yet confirmed")
	}
	return id, nil
}

// An explicitly absent child can be safely distinguished from a malformed
// nonempty exchange value. A malformed ID is UNKNOWN, never proof of
// successful cancellation.
func algoChildAbsent(raw []byte) bool {
	raw = bytes.TrimSpace(raw)
	return len(raw) == 0 || bytes.Equal(raw, []byte("null")) ||
		bytes.Equal(raw, []byte("0")) || bytes.Equal(raw, []byte{34, 34})
}

func algoChildPresentOrUnknown(raw []byte) error {
	if algoChildAbsent(raw) {
		return nil
	}
	id, err := confirmedAlgoChildID(raw)
	if err != nil {
		return fmt.Errorf("%w: ambiguous COIN-M Algo child identity: %v", ErrUncertain, err)
	}
	return fmt.Errorf("%w: Algo also reports child order %d", ErrUncertain, id)
}

// ReconcileProtection resolves an owned Algo's explicit actualOrderId and
// then verifies the resulting normal child order and its own DAPI userTrades.
// The private WS cannot safely infer child ownership from symbol/side alone.
func (p *Processor) ReconcileProtection(ctx context.Context, clientAlgoID string) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unavailable")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	local, err := loadOrder(p.DB, clientAlgoID)
	if err != nil {
		return local, err
	}
	if local.Intent != "PROTECT" || (local.OrderType != "STOP_MARKET" && local.OrderType != "TAKE_PROFIT_MARKET") {
		return local, errors.New("COIN-M client ID is not a managed protective Algo")
	}
	if ctx == nil {
		return local, errors.New("missing context")
	}
	if err = ctx.Err(); err != nil {
		return local, err
	}
	detail, err := p.API.QueryAlgo(ctx, local.Symbol, local.ClientOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: cannot query managed Algo: %v", ErrUncertain, err)
	}
	if detail.ClientAlgoID != local.ClientOrderID || detail.Symbol != local.Symbol ||
		detail.PositionSide != local.PositionSide || detail.Side != local.Side ||
		detail.OrderType != local.OrderType || detail.AlgoID <= 0 {
		return local, errors.New("COIN-M Algo identity mismatch; will not claim child order")
	}
	if local.Status != "TRIGGERED" && local.Status != "PARTIALLY_FILLED" && local.Status != "FILLED" {
		if local.ExchangeOrderID != "" && local.ExchangeOrderID != strconv.FormatInt(detail.AlgoID, 10) {
			return local, errors.New("COIN-M Algo ID differs from saved managed intent")
		}
	}
	status := strings.ToUpper(detail.AlgoStatus)
	switch status {
	case "NEW", "ACTIVE", "PENDING", "ACCEPTED":
		if local.Status == "TRIGGERED" || local.Status == "FILLED" {
			return local, errors.New("COIN-M Algo status regressed after child trigger")
		}
		_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET exchange_order_id=?,status='ACTIVE',updated_at=? WHERE id=?",
			strconv.FormatInt(detail.AlgoID, 10), time.Now().UnixMilli(), local.ID).Exec()
		if err != nil {
			return local, err
		}
		return loadOrder(p.DB, clientAlgoID)
	case "CANCELED", "CANCELLED", "EXPIRED", "REJECTED":
		if childErr := algoChildPresentOrUnknown(detail.ActualOrderID); childErr != nil {
			return local, childErr
		}
		if local.Status == "TRIGGERED" || local.Status == "FILLED" {
			return local, errors.New("COIN-M received terminal Algo without reconciling prior child Fill")
		}
		_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET status='CANCELED',updated_at=? WHERE id=?",
			time.Now().UnixMilli(), local.ID).Exec()
		if err != nil {
			return local, err
		}
		return loadOrder(p.DB, clientAlgoID)
	case "TRIGGERED", "FINISHED", "EXECUTED":
	default:
		return local, fmt.Errorf("%w: unknown COIN-M Algo status %q", ErrUncertain, detail.AlgoStatus)
	}
	childID, err := confirmedAlgoChildID(detail.ActualOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: %v", ErrUncertain, err)
	}
	child, err := p.API.OrderByID(ctx, local.Symbol, childID)
	if err != nil || child == nil {
		return local, fmt.Errorf("%w: actual child DAPI order lookup: %v", ErrUncertain, err)
	}
	if child.ExchangeOrderID != childID || child.Symbol != local.Symbol ||
		child.PositionSide != local.PositionSide || child.Side != local.Side || child.Status == "" {
		return local, errors.New("COIN-M triggered Algo child does not match owned symbol, hedge side or direction")
	}
	if local.Status == "TRIGGERED" || local.Status == "FILLED" || local.Status == "PARTIALLY_FILLED" {
		if local.ExchangeOrderID != strconv.FormatInt(childID, 10) {
			return local, errors.New("COIN-M triggered child order ID differs from previously verified ID")
		}
	}
	// Do not cancel the sibling until BOTH the remote normal child identity
	// and any previously established owned child ID are verified. Otherwise
	// a mismatched response could strip legitimate TP/SL protection.
	if err = p.cancelSiblingProtectors(ctx, local); err != nil {
		return local, err
	}
	if local.Status == "FILLED" && local.ExchangeOrderID == strconv.FormatInt(childID, 10) {
		return local, nil
	}
	// Exact child order ID is now established independently from query Algo
	// and query normal order. Only now can it be used as the Fill ownership key.
	if _, err = p.DB.Raw("UPDATE coin_m_managed_orders SET exchange_order_id=?,status='TRIGGERED',updated_at=? WHERE id=?",
		strconv.FormatInt(childID, 10), time.Now().UnixMilli(), local.ID).Exec(); err != nil {
		return local, err
	}
	trades, err := p.API.Trades(ctx, local.Symbol, childID)
	if err != nil {
		return local, fmt.Errorf("%w: triggered child userTrades unavailable: %v", ErrUncertain, err)
	}
	if len(trades) >= 1000 {
		return local, fmt.Errorf("%w: child userTrades paginated results incomplete", ErrUncertain)
	}
	for _, trade := range trades {
		if trade.OrderID != childID || trade.Symbol != local.Symbol {
			return local, errors.New("COIN-M Algo child userTrades returned cross-order Fill")
		}
		if _, err = p.applyFill(trade); err != nil {
			return local, err
		}
	}
	after, err := loadOrder(p.DB, clientAlgoID)
	if err != nil {
		return local, err
	}
	observed, err := p.observedContracts(after.ID)
	if err != nil {
		return after, err
	}
	filled, err := decimal(child.FilledContracts)
	if err != nil {
		return after, err
	}
	if observed.Cmp(filled) != 0 {
		_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='AWAITING_FILLS',updated_at=? WHERE id=?",
			time.Now().UnixMilli(), after.ID).Exec()
		return after, fmt.Errorf("%w: triggered child executed=%s, owned Fills=%s", ErrUncertain, child.FilledContracts, str(observed))
	}
	return loadOrder(p.DB, clientAlgoID)
}

var _ = binance.CoinMAlgoDetail{}

// cancelSiblingProtectors runs only after the first Algo's actual child
// identity has been validated. It rejects ambiguous sibling status instead
// of treating a REST cancellation ACK as proof the trigger did not race.
func (p *Processor) cancelSiblingProtectors(ctx context.Context, trigger models.CoinMManagedOrder) error {
	var siblings []models.CoinMManagedOrder
	_, err := p.DB.Raw("SELECT "+orderCols+" FROM coin_m_managed_orders WHERE position_id=? AND intent='PROTECT' AND id<>? AND status NOT IN ('CANCELED','FILLED','EXPIRED','REJECTED')", trigger.PositionID, trigger.ID).QueryRows(&siblings)
	if err != nil {
		return err
	}
	for _, sibling := range siblings {
		if sibling.Status != "ACTIVE" {
			return fmt.Errorf("%w: sibling protection %s is not confirmed ACTIVE", ErrUncertain, sibling.ClientOrderID)
		}
		if err = p.API.CancelAlgo(ctx, sibling.Symbol, sibling.ClientOrderID); err != nil {
			_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE id=?", sibling.ID).Exec()
			return fmt.Errorf("%w: sibling protective Algo cancellation: %v", ErrUncertain, err)
		}
		observed, e := p.API.QueryAlgo(ctx, sibling.Symbol, sibling.ClientOrderID)
		if e != nil || observed.ClientAlgoID != sibling.ClientOrderID || observed.Symbol != sibling.Symbol ||
			observed.PositionSide != sibling.PositionSide || observed.Side != sibling.Side ||
			observed.OrderType != sibling.OrderType ||
			(observed.AlgoStatus != "CANCELED" && observed.AlgoStatus != "CANCELLED") {
			_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE id=?", sibling.ID).Exec()
			return fmt.Errorf("%w: sibling protective Algo cancellation not confirmed: %v", ErrUncertain, e)
		}
		if childErr := algoChildPresentOrUnknown(observed.ActualOrderID); childErr != nil {
			_, _ = p.DB.Raw("UPDATE coin_m_managed_orders SET status='UNKNOWN' WHERE id=?", sibling.ID).Exec()
			return childErr
		}
		if _, err = p.DB.Raw("UPDATE coin_m_managed_orders SET status='CANCELED',updated_at=? WHERE id=?", time.Now().UnixMilli(), sibling.ID).Exec(); err != nil {
			return err
		}
	}
	return nil
}
