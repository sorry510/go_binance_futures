package spotstrategy

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/shopspring/decimal"
)

// replayHoldingCost derives cost basis from all *linked verified* Fill
// records in the owning transaction, including prior partial settlements.
// avg_cost_quote is only a rounded display/cache field; it is never fed back
// as the next trade's accounting basis. This avoids cumulative round-trip
// precision loss when a holding is settled in many small batches.
func replayHoldingCost(ctx context.Context, tx *sql.Tx, h Holding) (decimal.Decimal, decimal.Decimal, error) {
	rows, err := tx.QueryContext(ctx, `SELECT side,CAST(qty AS CHAR),CAST(quote_qty AS CHAR),
  CAST(commission AS CHAR),commission_asset FROM spot_strategy_fills
  WHERE account_id=? AND symbol=? AND holding_id=? AND rest_verified=1
  ORDER BY trade_time,order_id,trade_id`, h.AccountID, h.Symbol, h.ID)
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	qtyTotal, costBasis := decimal.Zero, decimal.Zero
	for rows.Next() {
		var side, qtyText, quoteText, feeText, asset string
		if err = rows.Scan(&side, &qtyText, &quoteText, &feeText, &asset); err != nil {
			break
		}
		qty, e1 := parseAmount(qtyText, false)
		quote, e2 := parseAmount(quoteText, true)
		fee, e3 := parseAmount(feeText, true)
		if e1 != nil || e2 != nil || e3 != nil {
			err = ErrReconciliation
			break
		}
		baseFee, quoteFee := decimal.Zero, decimal.Zero
		if asset == h.BaseAsset {
			baseFee = fee
		}
		if asset == h.QuoteAsset {
			quoteFee = fee
		}
		if side == "BUY" {
			credited := qty.Sub(baseFee)
			if !credited.IsPositive() {
				err = ErrReconciliation
				break
			}
			qtyTotal = qtyTotal.Add(credited)
			costBasis = costBasis.Add(quote).Add(quoteFee)
		} else if side == "SELL" {
			used := qty.Add(baseFee)
			if used.GreaterThan(qtyTotal) || qtyTotal.IsZero() {
				err = ErrReconciliation
				break
			}
			costAllocated := costBasis
			if used.LessThan(qtyTotal) {
				costAllocated = costBasis.Mul(used).DivRound(qtyTotal, 36)
			}
			costBasis = costBasis.Sub(costAllocated)
			qtyTotal = qtyTotal.Sub(used)
		} else {
			err = fmt.Errorf("%w: invalid linked Fill side", ErrReconciliation)
			break
		}
	}
	if rowErr := rows.Err(); err == nil {
		err = rowErr
	}
	if cErr := rows.Close(); err == nil {
		err = cErr
	}
	if err != nil {
		return decimal.Zero, decimal.Zero, err
	}
	if costBasis.IsNegative() {
		return decimal.Zero, decimal.Zero, ErrReconciliation
	}
	return qtyTotal, costBasis, nil
}
