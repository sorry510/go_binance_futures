package leadaccount

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	binance "go_binance_futures/feature/api/binance"
)

// SnapshotReader is intentionally only the subset needed for Lead risk
// evidence. A nil reader, partial history, invalid account or failed
// reconciliation never yields an executable snapshot.
type SnapshotReader interface {
	GetFuturesAccountContext(context.Context) (*futures.Account, error)
	GetPositionFreshContext(context.Context, binance.PositionParams) ([]*futures.PositionRisk, error)
	GetOpenOrderFreshContext(context.Context, ...string) ([]*futures.Order, error)
	GetPositionModeContext(context.Context) (*futures.PositionMode, error)
	GetIncomeHistoryContext(context.Context, int64, int64, int64) ([]*futures.IncomeHistory, error)
}
type RiskEvidenceSource struct {
	AccountID binance.AccountID
	Reader    SnapshotReader
	Now       func() time.Time
}

func NewRiskEvidenceSource(account *binance.AccountClient) (*RiskEvidenceSource, error) {
	if account == nil || account.ID() != binance.LeadAccountID {
		return nil, fmt.Errorf("Lead risk evidence requires account-bound Client")
	}
	return &RiskEvidenceSource{AccountID: binance.LeadAccountID, Reader: account, Now: time.Now}, nil
}

// Snapshot takes all private data from the bound Lead client. Returning
// Reconciled=false and UnknownOrders=true is deliberate: the Stage 5 WS,
// ownership pending-order scan, and full database account reconcile must
// complete before these two attestations may become true.
func (s *RiskEvidenceSource) Snapshot(ctx context.Context) (RiskSnapshot, error) {
	invalid := RiskSnapshot{AccountID: binance.LeadAccountID, UnknownOrders: true}
	if s == nil || s.AccountID != binance.LeadAccountID || s.Reader == nil {
		return invalid, fmt.Errorf("Lead snapshot source not bound")
	}
	if err := ctx.Err(); err != nil {
		return invalid, err
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	captured := now().UTC()
	out := RiskSnapshot{AccountID: binance.LeadAccountID, CapturedAt: captured, UnknownOrders: true}
	account, err := s.Reader.GetFuturesAccountContext(ctx)
	if err != nil || account == nil {
		return invalid, fmt.Errorf("Lead account balance snapshot unavailable")
	}
	if !account.CanTrade || account.MultiAssetsMargin {
		return invalid, fmt.Errorf("Lead account permission or margin mode unverified")
	}
	count := 0
	for _, asset := range account.Assets {
		if asset == nil {
			continue
		}
		if asset.Asset == "USDT" {
			count++
			wallet, e1 := strconv.ParseFloat(asset.WalletBalance, 64)
			avail, e2 := strconv.ParseFloat(asset.AvailableBalance, 64)
			equity, e3 := strconv.ParseFloat(asset.MarginBalance, 64)
			if e1 != nil || e2 != nil || e3 != nil || !riskFiniteNonnegative(wallet) || !riskFiniteNonnegative(avail) || !riskFiniteNonnegative(equity) {
				return invalid, fmt.Errorf("Lead USDT balance invalid")
			}
			out.WalletUSDT, out.AvailableUSDT, out.EquityUSDT = wallet, avail, equity
		} else if asset.Asset != "" {
			margin, e := strconv.ParseFloat(asset.MarginBalance, 64)
			if e != nil || !riskFiniteNonnegative(margin) || margin > 0 {
				return invalid, fmt.Errorf("Lead non-USDT margin not allowed")
			}
		}
	}
	if count != 1 {
		return invalid, fmt.Errorf("Lead USDT balance missing or duplicated")
	}
	mode, err := s.Reader.GetPositionModeContext(ctx)
	if err != nil || mode == nil || !mode.DualSidePosition {
		return invalid, fmt.Errorf("Lead hedge mode not verified")
	}
	out.HedgeMode = true
	positions, err := s.Reader.GetPositionFreshContext(ctx, binance.PositionParams{})
	if err != nil {
		return invalid, fmt.Errorf("Lead positions unavailable")
	}
	for _, row := range positions {
		if row == nil {
			return invalid, fmt.Errorf("Lead position snapshot contains nil")
		}
		qty, err := strconv.ParseFloat(row.PositionAmt, 64)
		if err != nil || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return invalid, fmt.Errorf("Lead position quantity invalid")
		}
		if math.Abs(qty) < 1e-12 {
			continue
		}
		mark, e1 := strconv.ParseFloat(row.MarkPrice, 64)
		profit, e2 := strconv.ParseFloat(row.UnRealizedProfit, 64)
		lev, e3 := strconv.ParseInt(row.Leverage, 10, 64)
		if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(mark) || !riskFiniteNonnegative(math.Abs(profit)) || lev <= 0 {
			return invalid, fmt.Errorf("Lead position valuation invalid")
		}
		side := strings.ToUpper(strings.TrimSpace(row.PositionSide))
		if side != "LONG" && side != "SHORT" {
			return invalid, fmt.Errorf("Lead non-hedge position rejected")
		}
		out.Positions = append(out.Positions, RiskPosition{Symbol: row.Symbol, Side: side, Quantity: qty, MarkPrice: mark, UnrealizedPNL: profit, Leverage: lev})
	}
	out.PositionsComplete = true
	orders, err := s.Reader.GetOpenOrderFreshContext(ctx)
	if err != nil {
		return invalid, fmt.Errorf("Lead open orders unavailable")
	}
	for _, row := range orders {
		if row == nil {
			return invalid, fmt.Errorf("Lead open order contains nil")
		}
		if row.Status != futures.OrderStatusTypeNew && row.Status != futures.OrderStatusTypePartiallyFilled {
			return invalid, fmt.Errorf("Lead open order status unknown")
		}
		side := strings.ToUpper(string(row.PositionSide))
		buy := row.Side == futures.SideTypeBuy
		sell := row.Side == futures.SideTypeSell
		if !buy && !sell {
			return invalid, fmt.Errorf("Lead order direction invalid")
		}
		if side != "LONG" && side != "SHORT" {
			return invalid, fmt.Errorf("Lead order side invalid")
		}
		if (side == "LONG" && sell) || (side == "SHORT" && buy) {
			continue
		} // close intent
		orig, e1 := strconv.ParseFloat(row.OrigQuantity, 64)
		filled, e2 := strconv.ParseFloat(row.ExecutedQuantity, 64)
		price, e3 := strconv.ParseFloat(row.Price, 64)
		if e1 != nil || e2 != nil || e3 != nil || !riskFinitePositive(orig) || !riskFiniteNonnegative(filled) || filled > orig || !riskFinitePositive(price) {
			// MARKET pending order often has zero price; without verified reference
			// we must fail closed rather than count missing notional as zero.
			return invalid, fmt.Errorf("Lead pending order valuation invalid")
		}
		remaining := orig - filled
		if remaining <= 1e-12 {
			continue
		}
		out.PendingOpens = append(out.PendingOpens, RiskPendingOpen{Symbol: row.Symbol, Side: side, RemainingQuantity: remaining, LimitOrReferencePrice: price})
	}
	out.OrdersComplete = true
	dayStart := captured.Truncate(24 * time.Hour)
	pnl, err := readLeadDailyNetPNL(ctx, s.Reader, dayStart.UnixMilli(), captured.UnixMilli())
	if err != nil {
		return invalid, err
	}
	out.DailyNetRealizedPNLUSDT = pnl
	out.PNLComplete = true
	out.PNLDayStartUTC = dayStart
	out.PNLAsOf = captured
	if err := ctx.Err(); err != nil {
		return invalid, err
	}
	return out, nil
}

// The Binance endpoint has a 1000-entry maximum and millisecond timestamps.
// Re-reading the overlap is necessary to avoid silently dropping rows. When
// all 1000 entries share the cursor millisecond, completeness is unprovable:
// refuse the snapshot rather than guessing a PNL total.
func readLeadDailyNetPNL(ctx context.Context, source SnapshotReader, startMs, endMs int64) (float64, error) {
	if source == nil || endMs < startMs {
		return 0, fmt.Errorf("Lead PNL request unavailable")
	}
	const limit int64 = 1000
	const maxPages = 30
	seen := make(map[string]bool)
	cursor := startMs
	total := 0.0
	for page := 0; page < maxPages; page++ {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		rows, err := source.GetIncomeHistoryContext(ctx, cursor, endMs, limit)
		if err != nil {
			return 0, fmt.Errorf("Lead daily PNL page unavailable")
		}
		if len(rows) > int(limit) {
			return 0, fmt.Errorf("Lead daily PNL page over limit")
		}
		last := cursor
		for _, row := range rows {
			if row == nil || row.Time < cursor || row.Time > endMs || row.TranID <= 0 {
				return 0, fmt.Errorf("Lead daily PNL entry invalid")
			}
			if row.Time > last {
				last = row.Time
			}
			id := strconv.FormatInt(row.TranID, 10) + ":" + row.IncomeType
			if seen[id] {
				continue
			}
			seen[id] = true
			if row.Asset != "USDT" {
				return 0, fmt.Errorf("Lead daily PNL non-USDT asset")
			}
			income, err := strconv.ParseFloat(row.Income, 64)
			if err != nil || math.IsNaN(income) || math.IsInf(income, 0) {
				return 0, fmt.Errorf("Lead daily PNL amount invalid")
			}
			switch row.IncomeType {
			case "REALIZED_PNL", "COMMISSION", "FUNDING_FEE":
				total += income
			default:
				// Transfers and other non-trade credits must never mask a loss.
				// Unknown debit types mean the net trading PNL is not provably complete.
				if income < 0 {
					return 0, fmt.Errorf("Lead daily PNL unknown debit category")
				}
			}
		}
		if len(rows) < int(limit) {
			if !riskFiniteNonnegative(math.Abs(total)) {
				return 0, fmt.Errorf("Lead daily PNL overflow")
			}
			return total, nil
		}
		if last <= cursor {
			return 0, fmt.Errorf("Lead daily PNL page boundary ambiguous")
		}
		cursor = last // inclusive overlap with transaction-ID de-duplication
	}
	return 0, fmt.Errorf("Lead daily PNL pagination exceeded safety limit")
}
