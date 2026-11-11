package spotledger

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Accounting computes an independent read-only Spot position view. Mark price
// must be supplied from a verified Spot quote, never Futures mark/positionRisk.
// No price means no fabricated unrealized or total PnL.
func (l *Ledger) Accounting(positionID int64, markPrice string) (Snapshot, error) {
	var view Snapshot
	// Read a consistent committed snapshot, calculate all financial fields
	// from immutable fills, and roll back this read-only transaction. Never
	// return PnL from spot_managed_positions mutable cache fields.
	tx, unlock, err := l.begin()
	if err != nil {
		return view, err
	}
	defer unlock()
	defer tx.Rollback()
	stored, err := readPosition(tx, positionID, l.lock())
	if err != nil {
		return view, err
	}
	projection, err := projectPosition(tx, stored)
	if err != nil {
		return view, err
	}
	if len(projection.reviewIssues) > 0 {
		return view, fmt.Errorf("%w: %s", ErrNeedsReview, strings.Join(projection.reviewIssues, "; "))
	}
	position := projection.position
	fills := projection.fills
	view.Position = position
	view.RealizedPnL = position.RealizedPnL
	view.UnpricedFees = make(map[string]string)
	feeTotals := make(map[string]decimal.Decimal)
	for _, fill := range fills {
		fee, err := parseDBDecimal(fill.Commission, false)
		if err != nil {
			return view, fmt.Errorf("stored fee %d: %w", fill.ID, err)
		}
		if !fee.IsZero() {
			if fill.CommissionAsset == "" {
				return view, fmt.Errorf("fee currency missing for fill %d", fill.ID)
			}
			feeTotals[fill.CommissionAsset] = feeTotals[fill.CommissionAsset].Add(fee)
		}
	}
	for asset, value := range feeTotals {
		if asset == position.BaseAsset || asset == position.QuoteAsset {
			continue
		}
		view.UnpricedFees[asset], err = numeric(value)
		if err != nil {
			return view, err
		}
	}
	view.FullyPriced = len(view.UnpricedFees) == 0
	if markPrice != "" {
		mark, err := parseDBDecimal(markPrice, true)
		if err != nil {
			return view, fmt.Errorf("invalid Spot mark price: %w", err)
		}
		remaining, err := parseDBDecimal(position.RemainingQty, false)
		if err != nil {
			return view, err
		}
		avg, err := parseDBDecimal(position.AvgCostQuote, false)
		if err != nil {
			return view, err
		}
		realized, err := parseSignedDBDecimal(position.RealizedPnL)
		if err != nil {
			return view, err
		}
		unrealized := mark.Sub(avg).Mul(remaining)
		view.UnrealizedPnL, err = numeric(unrealized)
		if err != nil {
			return view, err
		}
		view.TotalPnL, err = numeric(unrealized.Add(realized))
		if err != nil {
			return view, err
		}
		view.MarkPrice = markPrice
		view.HasMarkPrice = true
	}
	return view, nil
}
