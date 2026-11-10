package spotstrategy

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// HoldingReport derives realized PnL and fee currencies from the immutable
// linked Fill identities. Quote prices and quantities use decimal arithmetic.
// Third-asset fees (BNB, etc) are NEVER misrepresented as USDT/USDC profit.
func (e *Engine) HoldingReport(ctx context.Context, holdingID int64, priceQuote string) (HoldingSummary, error) {
	h, err := readHolding(e.db.QueryRowContext(ctx, "SELECT "+holdingCols+" FROM spot_strategy_holdings WHERE id=?", holdingID))
	if err != nil {
		return HoldingSummary{}, err
	}
	if err = ValidateScope(h.AccountID, h.Owner); err != nil {
		return HoldingSummary{}, ErrUnmanagedOrder
	}
	r := HoldingSummary{Holding: h, Fees: map[string]string{}, Complete: true}
	rows, err := e.db.QueryContext(ctx, `SELECT side,CAST(qty AS CHAR),CAST(quote_qty AS CHAR),
 CAST(commission AS CHAR),commission_asset,rest_verified
 FROM spot_strategy_fills WHERE account_id=? AND symbol=? AND holding_id=?
 ORDER BY trade_time,trade_id`, h.AccountID, h.Symbol, h.ID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	owned, cost, realGross, net := decimal.Zero, decimal.Zero, decimal.Zero, decimal.Zero
	fees := map[string]decimal.Decimal{}
	reasons := map[string]struct{}{}
	count := 0
	for rows.Next() {
		count++
		var side, qtyText, quoteText, feeText, feeAsset string
		var verified int
		if err = rows.Scan(&side, &qtyText, &quoteText, &feeText, &feeAsset, &verified); err != nil {
			return r, err
		}
		qty, er1 := parseAmount(qtyText, false)
		quote, er2 := parseAmount(quoteText, true)
		fee, er3 := parseAmount(feeText, true)
		if er1 != nil || er2 != nil || er3 != nil {
			return r, ErrReconciliation
		}
		if verified != 1 {
			// Do not mix unverified or manually edited data into quantities,
			// cost basis, gross PnL or fees presented as settled money.
			reasons["fill_source_not_verified"] = struct{}{}
			continue
		}
		fees[feeAsset] = fees[feeAsset].Add(fee)
		baseFee, quoteFee := decimal.Zero, decimal.Zero
		if feeAsset == h.BaseAsset {
			baseFee = fee
		}
		if feeAsset == h.QuoteAsset {
			quoteFee = fee
		}
		if fee.IsPositive() && feeAsset != h.BaseAsset && feeAsset != h.QuoteAsset {
			reasons["third_asset_fee_requires_rate_at_trade_time"] = struct{}{}
		}
		if side == "BUY" {
			qtyNet := qty.Sub(baseFee)
			if !qtyNet.IsPositive() {
				return r, ErrReconciliation
			}
			owned = owned.Add(qtyNet)
			cost = cost.Add(quote).Add(quoteFee)
		} else if side == "SELL" {
			consumed := qty.Add(baseFee)
			if owned.LessThan(consumed) || !owned.IsPositive() {
				return r, ErrReconciliation
			}
			// Use the entire remaining exact basis for a full exit; for
			// partial exits retain 36 fractional decimal places, not the
			// rounded display avg_cost_quote cache.
			allocated := cost
			if consumed.LessThan(owned) {
				allocated = cost.Mul(consumed).DivRound(owned, 36)
			}
			netSale := quote.Sub(quoteFee)
			realGross = realGross.Add(quote.Sub(allocated))
			net = net.Add(netSale.Sub(allocated))
			owned = owned.Sub(consumed)
			cost = cost.Sub(allocated)
		} else {
			return r, ErrReconciliation
		}
	}
	if err = rows.Err(); err != nil {
		return r, err
	}
	if count == 0 {
		return r, ErrReconciliation
	}
	stored, err := parseAmount(h.OwnedQty, true)
	if err != nil || !stored.Equal(owned) {
		reasons["holding_quantity_not_reconciled"] = struct{}{}
	}
	r.Complete = len(reasons) == 0
	if !r.Complete {
		keys := make([]string, 0, len(reasons))
		for reason := range reasons {
			keys = append(keys, reason)
		}
		sort.Strings(keys)
		r.IncompleteReason = strings.Join(keys, ";")
	}
	if _, unverified := reasons["fill_source_not_verified"]; unverified {
		// This is not an "unverified estimate": suppress all accounting
		// amounts, including the Holding's stored owned_qty/avg_cost fields.
		// An external UI must not mistake dirty records for owned inventory.
		r.Holding.OwnedQty = ""
		r.Holding.AvgCostQuote = ""
		r.CostBasisQuote = ""
		r.RealizedGrossQuote = ""
		r.RealizedNetQuote = ""
		r.UnrealizedEstimateQuote = ""
	} else {
		r.CostBasisQuote = cost.String()
		r.RealizedGrossQuote = realGross.String()
		if r.Complete {
			r.RealizedNetQuote = net.String()
		}
	}
	for asset, n := range fees {
		if asset != "" && !n.IsZero() {
			r.Fees[asset] = n.String()
		}
	}
	if priceQuote != "" {
		price, priceErr := parseAmount(priceQuote, false)
		if priceErr != nil {
			return r, fmt.Errorf("invalid Spot quote estimate: %w", priceErr)
		}
		if r.Complete {
			r.UnrealizedEstimateQuote = owned.Mul(price).Sub(cost).String()
		}
	}
	return r, nil
}

func (e *Engine) GetActiveHolding(ctx context.Context, symbol string) (Holding, error) {
	h, err := readHolding(e.db.QueryRowContext(ctx, "SELECT "+holdingCols+" FROM spot_strategy_holdings WHERE active_slot=?",
		AccountMain+":"+OwnerAuto+":"+symbol))
	if err == sql.ErrNoRows {
		return Holding{}, ErrUnmanagedOrder
	}
	if err != nil {
		return Holding{}, err
	}
	if err = ValidateScope(h.AccountID, h.Owner); err != nil {
		return Holding{}, ErrUnmanagedOrder
	}
	return h, nil
}
