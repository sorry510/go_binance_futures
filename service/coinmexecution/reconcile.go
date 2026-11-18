package coinmexecution

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/adshao/go-binance/v2/delivery"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
)

func isTerminal(status string) bool {
	return status == "FILLED" || status == "CANCELED" || status == "EXPIRED" || status == "REJECTED"
}
func (p *Processor) applyExchangeUpdate(clientID string, ack *OrderAck) error {
	local, err := loadOrder(p.DB, clientID)
	if err != nil {
		return err
	}
	request := binance.CoinMTradeOrder{ClientOrderID: local.ClientOrderID, Symbol: local.Symbol,
		PositionSide: local.PositionSide, Side: local.Side}
	if err = validateAck(request, ack); err != nil {
		return err
	}
	if local.ExchangeOrderID != "" && local.ExchangeOrderID != strconv.FormatInt(ack.ExchangeOrderID, 10) {
		return errors.New("COIN-M exchange order ID changed after ownership was established")
	}
	if isTerminal(local.Status) && !isTerminal(ack.Status) {
		return nil // Late WS UPDATE may not regress a terminal state.
	}
	_, err = p.DB.Raw("UPDATE coin_m_managed_orders SET exchange_order_id=?,status=?,updated_at=? WHERE id=?",
		strconv.FormatInt(ack.ExchangeOrderID, 10), ack.Status, time.Now().UnixMilli(), local.ID).Exec()
	return err
}
func (p *Processor) markAwaitingFills(local models.CoinMManagedOrder) error {
	_, err := p.DB.Raw("UPDATE coin_m_managed_orders SET status='AWAITING_FILLS',updated_at=? WHERE id=?",
		time.Now().UnixMilli(), local.ID).Exec()
	return err
}
func (p *Processor) observedContracts(orderID int64) (*big.Rat, error) {
	var qty []struct {
		Contracts string `orm:"column(contracts)"`
	}
	if _, err := p.DB.Raw("SELECT contracts FROM coin_m_trade_fills WHERE order_id=?", orderID).QueryRows(&qty); err != nil {
		return nil, err
	}
	total := new(big.Rat)
	for _, fill := range qty {
		n, err := positive(fill.Contracts)
		if err != nil {
			return nil, err
		}
		total.Add(total, n)
	}
	return total, nil
}

// Reconcile may query remote status and exact owned userTrades, NEVER re-POST
// an uncertain order. Re-running is idempotent by trade id and client id.
func (p *Processor) Reconcile(ctx context.Context, clientID string) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unconfigured")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	local, err := loadOrder(p.DB, clientID)
	if err != nil {
		return local, err
	}
	if err = ctx.Err(); err != nil {
		return local, err
	}
	remote, err := p.API.Query(ctx, local.Symbol, local.ClientOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: remote order lookup failed: %v", ErrUncertain, err)
	}
	if remote == nil {
		return local, fmt.Errorf("%w: remote order not yet visible, never resubmit", ErrUncertain)
	}
	if err = p.applyExchangeUpdate(clientID, remote); err != nil {
		return local, err
	}
	local, err = loadOrder(p.DB, clientID)
	if err != nil {
		return local, err
	}
	trades, err := p.API.Trades(ctx, local.Symbol, remote.ExchangeOrderID)
	if err != nil {
		return local, fmt.Errorf("%w: userTrades fetch failed: %v", ErrUncertain, err)
	}
	if len(trades) >= 1000 {
		// One page is not proof that the order has no more trades.
		// Persist the incomplete-evidence state instead of retaining an
		// exchange FILLED status which could appear verified to readers.
		if err = p.markAwaitingFills(local); err != nil {
			return local, err
		}
		local, err = loadOrder(p.DB, clientID)
		if err != nil {
			return local, err
		}
		return local, fmt.Errorf("%w: userTrades reached page limit, requires additional pagination", ErrUncertain)
	}
	for _, tr := range trades {
		if tr.OrderID != remote.ExchangeOrderID || tr.Symbol != local.Symbol {
			return local, errors.New("COIN-M userTrades returned cross-order/cross-symbol record")
		}
		if _, err = p.applyFill(tr); err != nil {
			return local, err
		}
	}
	observed, err := p.observedContracts(local.ID)
	if err != nil {
		return local, err
	}
	expected, err := decimal(remote.FilledContracts)
	if err != nil {
		return local, err
	}
	if observed.Cmp(expected) > 0 {
		return local, errors.New("local COIN-M Fill sum exceeds reported exchange execution")
	}
	if observed.Cmp(expected) < 0 {
		_ = p.markAwaitingFills(local)
		local, _ = loadOrder(p.DB, clientID)
		return local, fmt.Errorf("%w: exchange executed=%s but verified Fills=%s", ErrUncertain, remote.FilledContracts, str(observed))
	}
	// Fully cancelled OPEN with zero Fills leaves a zero-contract local cycle;
	// it may be terminated only after the exchange confirms status.
	if local.Intent == "OPEN" && (remote.Status == "CANCELED" || remote.Status == "REJECTED" || remote.Status == "EXPIRED") && observed.Sign() == 0 {
		_, err = p.DB.Raw("UPDATE coin_m_managed_positions SET status='CANCELED',updated_at=? WHERE id=? AND opened_contracts=0",
			time.Now().UnixMilli(), local.PositionID).Exec()
		if err != nil {
			return local, err
		}
	}
	if local.Intent == "CLOSE" && isTerminal(remote.Status) {
		pos, e := loadPosition(p.DB, local.PositionID)
		if e != nil {
			return local, e
		}
		if pos.Status == "CLOSING" {
			_, err = p.DB.Raw("UPDATE coin_m_managed_positions SET status='OPEN',updated_at=? WHERE id=?",
				time.Now().UnixMilli(), local.PositionID).Exec()
			if err != nil {
				return local, err
			}
		}
	}
	return loadOrder(p.DB, clientID)
}
func (p *Processor) Cancel(ctx context.Context, clientID string) (models.CoinMManagedOrder, error) {
	if p == nil || p.DB == nil || p.API == nil {
		return models.CoinMManagedOrder{}, errors.New("COIN-M processor unconfigured")
	}
	p.mu.Lock()
	local, err := loadOrder(p.DB, clientID)
	if err != nil {
		p.mu.Unlock()
		return local, err
	}
	if isTerminal(local.Status) {
		p.mu.Unlock()
		return local, nil
	}
	if local.OrderType != "MARKET" && local.OrderType != "LIMIT" {
		p.mu.Unlock()
		return local, errors.New("COIN-M Algo orders must be canceled with CancelProtection")
	}
	if !p.API.Ready() {
		p.mu.Unlock()
		return local, errors.New("COIN-M private WS disconnected")
	}
	ack, err := p.API.Cancel(ctx, local.Symbol, local.ClientOrderID)
	if err == nil {
		err = p.applyExchangeUpdate(clientID, ack)
	}
	p.mu.Unlock()
	if err != nil {
		return local, fmt.Errorf("%w: cancel may have succeeded, reconcile by ID: %v", ErrUncertain, err)
	}
	return p.Reconcile(ctx, clientID)
}
func (p *Processor) OnWSOrder(event *delivery.WsUserDataEvent) error {
	if event == nil || event.Event != delivery.UserDataEventTypeOrderTradeUpdate {
		return nil
	}
	update := event.OrderTradeUpdate
	if update.ClientOrderID == "" {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	local, found, err := localOrder(p.DB, update.ClientOrderID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	} // Manual or other product: never claim.
	if local.OrderType != "MARKET" && local.OrderType != "LIMIT" {
		return errors.New("COIN-M conditional child order cannot be silently attributed")
	}
	ack := &OrderAck{ClientOrderID: update.ClientOrderID, ExchangeOrderID: update.ID,
		Symbol: update.Symbol, PositionSide: string(update.PositionSide), Side: string(update.Side),
		Status: string(update.Status), FilledContracts: update.AccumulatedFilledQty}
	if err = p.applyExchangeUpdate(local.ClientOrderID, ack); err != nil {
		return err
	}
	if update.ExecutionType != delivery.OrderExecutionTypeTrade || update.LastFilledQty == "" || update.LastFilledQty == "0" {
		return nil
	}
	trade := binance.CoinMTrade{
		Symbol: update.Symbol, OrderID: update.ID, TradeID: update.TradeID,
		Side: string(update.Side), PositionSide: string(update.PositionSide),
		Price: update.LastFilledPrice, Quantity: update.LastFilledQty,
		RealizedPnL: update.RealizedPnL, MarginAsset: update.MarginAsset,
		Commission: update.Commission, CommissionAsset: update.CommissionAsset,
		Time: update.TradeTime,
	}
	if _, err = p.applyFill(trade); err != nil {
		return fmt.Errorf("COIN-M private WS Fill requires REST reconciliation: %w", err)
	}
	return nil
}
