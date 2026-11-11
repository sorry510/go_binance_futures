package spotexecution

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go_binance_futures/service/spotledger"

	"github.com/beego/beego/v2/client/orm"
)

type ExecutionReport struct {
	Symbol, ClientOrderID, ExchangeOrderID, Side, Status string
	ExecutionType                                        string
	TradeID                                              string
	LastQty, LastPrice, LastQuoteQty, Fee, FeeAsset      string
	TradeTime                                            int64
}

// HandleExecutionReport only records events linked to an EXISTING managed
// client order ID. Manual/wallet orders are ignored and never claimed.
// WS events are reconciled against REST myTrades after reconnect.
func (p *Processor) HandleExecutionReport(ctx context.Context, r ExecutionReport) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if !validClientID(r.ClientOrderID) {
		return false, nil
	}
	order, err := p.ledger.FindOrderByClientID(r.ClientOrderID)
	if errors.Is(err, orm.ErrNoRows) {
		return false, nil
	} // unrelated manual Spot trade
	if err != nil {
		return false, err
	}
	if r.Symbol != order.Symbol || r.Side != order.Side {
		return true, fmt.Errorf("%w: Spot user stream order identity mismatch", ErrNeedsReview)
	}
	if r.ExchangeOrderID == "" {
		return true, fmt.Errorf("%w: missing exchange order id", ErrNeedsReview)
	}
	if _, err := strconv.ParseInt(r.ExchangeOrderID, 10, 64); err != nil {
		return true, fmt.Errorf("%w: invalid exchange order id", ErrNeedsReview)
	}
	if _, err = p.ledger.BindExchangeOrderID(order.ID, r.ExchangeOrderID); err != nil {
		return true, fmt.Errorf("%w: conflicting WS exchange order id: %v", ErrNeedsReview, err)
	}
	// The SDK reports exchange authoritative status, including terminal.
	if r.Status != "" {
		if _, err = p.ledger.SetExchangeStatus(order.ID, r.Status); err != nil {
			return true, fmt.Errorf("%w: status conflict from Spot user stream: %v", ErrNeedsReview, err)
		}
	}
	if r.ExecutionType == "TRADE" {
		if strings.TrimSpace(r.TradeID) == "" || r.TradeTime <= 0 ||
			r.LastQty == "" || r.LastPrice == "" || r.LastQuoteQty == "" || strings.TrimSpace(r.Fee) == "" {
			return true, fmt.Errorf("%w: incomplete Spot executionReport", ErrNeedsReview)
		}
		commission, feeErr := parseAmount(r.Fee, false)
		if feeErr != nil || (commission.IsPositive() && strings.TrimSpace(r.FeeAsset) == "") {
			return true, fmt.Errorf("%w: invalid commission or missing fee asset in Spot executionReport", ErrNeedsReview)
		}
		_, err = p.ledger.RecordFill(spotledger.FillEvent{
			OrderID: order.ID, Symbol: r.Symbol, ExchangeOrderID: r.ExchangeOrderID,
			TradeID: r.TradeID, Side: r.Side, Qty: r.LastQty, Price: r.LastPrice,
			QuoteQty: r.LastQuoteQty, Commission: r.Fee, CommissionAsset: r.FeeAsset,
			TradeTime: r.TradeTime,
		})
		if err != nil {
			return true, fmt.Errorf("%w: user stream fill: %v", ErrNeedsReview, err)
		}
	}
	issues, err := p.ledger.ReviewPosition(order.PositionID)
	if err != nil {
		return true, err
	}
	if len(issues) > 0 {
		return true, fmt.Errorf("%w: Spot stream requires myTrades reconciliation: %s", ErrNeedsReview, strings.Join(issues, "; "))
	}
	return true, nil
}
