package orderhistory

// BNB commission is estimated at the CURRENT local symbols.BNBUSDT.close.
// No historical candles, external APIs, data writes, settlement certification
// or real-money risk-control changes are involved.
import (
	"math"
	"math/big"
	"strings"

	"go_binance_futures/models"
)

// Projection eligibility uses summarize()'s existing identity/REST/quantity
// checks. Only a cycle whose sole original problem was lack of settlement may
// get a NORMAL execution verdict with ESTIMATED fee-adjusted PnL.
func applyBNBFeeEstimate(p models.FuturesManagedPosition, out *Position, proof Position, fills []fillRow, quote symbolQuote) {
	if p.Status != "closed" || p.SettlementStatus != "incomplete" || proof.IncompleteReason != "cycle_not_settled" ||
		math.Abs(p.ManagedQty) > 1e-12 || p.ClosedAt <= 0 || !strings.HasSuffix(p.Symbol, "USDT") {
		return
	}
	opened, err1 := dec(out.OpenQty)
	closed, err2 := dec(out.CloseQty)
	if err1 != nil || err2 != nil || opened.Sign() <= 0 || opened.Cmp(closed) != 0 || len(fills) == 0 {
		return
	}
	// This is a best-effort display estimate. A missing/invalid local
	// quote must not turn an incomplete cycle into a normal one.
	if quote.Symbol != "BNBUSDT" || strings.ToUpper(strings.TrimSpace(quote.Type)) != "USDT" {
		return
	}
	bnbPrice, err := dec(quote.Close)
	if err != nil || bnbPrice.Sign() <= 0 {
		return
	}
	gross, fee := new(big.Rat), new(big.Rat)
	hasBNB := false
	for _, f := range fills {
		if f.RESTVerified != 1 || (f.PnLAsset != "" && f.PnLAsset != "USDT") {
			return
		}
		commission, e1 := dec(f.Commission)
		pnl, e2 := dec(f.RealizedPnL)
		if e1 != nil || e2 != nil || commission.Sign() < 0 {
			return
		}
		gross.Add(gross, pnl)
		if commission.Sign() == 0 {
			continue
		}
		switch f.CommissionAsset {
		case "USDT":
			fee.Add(fee, commission)
		case "BNB":
			hasBNB = true
			fee.Add(fee, new(big.Rat).Mul(commission, bnbPrice))
		default:
			return
		}
	}
	if !hasBNB {
		return
	}
	net := new(big.Rat).Sub(gross, fee)
	out.BNBFeeEstimated = true
	out.EstimatedFeeUSDT = displayProfitPtr(fee)
	out.EstimatedNetPnL = displayProfitPtr(net)
	out.BNBFeePriceUSDT = displayProfitPtr(bnbPrice)
	out.BNBFeePriceUpdatedAt = quote.UpdateTime
	// Normal trade execution. The real financial settlement status remains
	// incomplete, and NetPnL remains nil (never disguised as verified).
	out.Completeness = "estimated"
	out.IncompleteReason = ""
	out.DisplayPnLAsset = "USDT"
	out.RealizedPnL = displayProfitPtr(net)
	out.UnrealizedPnL = displayProfitPtr(new(big.Rat))
	out.TotalPnL = displayProfitPtr(net)
	if out.OpenAvgPrice != nil {
		entry, err := dec(*out.OpenAvgPrice)
		if err == nil && entry.Sign() > 0 {
			notional := new(big.Rat).Mul(entry, opened)
			out.DisplayReturnPct = displayProfitPtr(new(big.Rat).Mul(new(big.Rat).Quo(net, notional), big.NewRat(100, 1)))
		}
	}
}
