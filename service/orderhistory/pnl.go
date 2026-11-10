package orderhistory

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"go_binance_futures/models"
)

// Price snapshots come from the local futures symbols table. They are last
// prices, not Binance mark prices, and are therefore an indicative valuation.
type symbolQuote struct {
	Symbol     string `orm:"column(symbol)"`
	Close      string `orm:"column(close)"`
	UpdateTime int64  `orm:"column(updateTime)"`
	Type       string `orm:"column(type)"`
}

type ProfitBreakdown struct {
	Realized   string `json:"realized"`
	Unrealized string `json:"unrealized"`
	Total      string `json:"total"`
}

func (s Store) symbolPrices(ctx context.Context, positions []models.FuturesManagedPosition) (map[string]symbolQuote, error) {
	names := make([]string, 0, len(positions))
	seen := make(map[string]bool, len(positions))
	for _, p := range positions {
		if !seen[p.Symbol] {
			seen[p.Symbol] = true
			names = append(names, p.Symbol)
		}
	}
	prices := make(map[string]symbolQuote, len(names))
	if len(names) == 0 {
		return prices, nil
	}
	placeholders := make([]string, len(names))
	args := make([]interface{}, len(names))
	for i, name := range names {
		placeholders[i], args[i] = "?", name
	}
	var rows []symbolQuote
	query := "SELECT symbol, close, updateTime, type FROM symbols WHERE symbol IN (" +
		strings.Join(placeholders, ",") + ")"
	if _, err := s.db().Raw(query, args...).QueryRows(&rows); err != nil {
		return nil, fmt.Errorf("read symbols price snapshots: %w", err)
	}
	for _, row := range rows {
		prices[row.Symbol] = row
	}
	return prices, ctx.Err()
}

// applyPricePnL is a price-spread PnL view independent of REST certification.
// Binance USD-M linear contract PnL: LONG=(exit-entry)*qty, SHORT=reverse.
// Excludes both exchange fees (which may be paid in BNB) and funding.
// Historical realized legs use executed entry/exit VWAP; open legs use the
// currently managed quantity, not all lifetime opening fills.
func applyPricePnL(p models.FuturesManagedPosition, out *Position, quote symbolQuote) {
	asset := strings.ToUpper(strings.TrimSpace(quote.Type))
	if asset != "USDT" && asset != "USDC" {
		if strings.HasSuffix(p.Symbol, "USDT") {
			asset = "USDT"
		} else if strings.HasSuffix(p.Symbol, "USDC") {
			asset = "USDC"
		} else {
			return
		}
	}
	out.DisplayPnLAsset = asset
	if current, err := dec(quote.Close); err == nil && current.Sign() > 0 {
		out.CurrentPrice = ptr(current)
		out.PriceUpdatedAt = quote.UpdateTime
	}
	var entry *big.Rat
	if out.OpenAvgPrice != nil {
		entry, _ = dec(*out.OpenAvgPrice)
	}
	if entry == nil || entry.Sign() <= 0 {
		entry, _ = dec(strconv.FormatFloat(p.EntryPrice, 'f', 12, 64))
	}
	if entry == nil || entry.Sign() <= 0 {
		return
	}
	opened, openErr := dec(out.OpenQty)
	closed, closeErr := dec(out.CloseQty)
	remaining, remainErr := dec(out.RemainingQty)
	if openErr != nil || closeErr != nil || remainErr != nil || remaining.Sign() < 0 {
		return
	}
	if opened.Sign() == 0 && closed.Sign() == 0 && remaining.Sign() == 0 {
		return
	}
	direction := int64(1)
	if p.PositionSide == "SHORT" {
		direction = -1
	} else if p.PositionSide != "LONG" {
		return
	}
	realized := new(big.Rat)
	if closed.Sign() > 0 {
		if out.CloseAvgPrice == nil {
			return
		}
		closePrice, err := dec(*out.CloseAvgPrice)
		if err != nil || closePrice.Sign() <= 0 {
			return
		}
		realized.Mul(new(big.Rat).Sub(closePrice, entry), closed)
		realized.Mul(realized, big.NewRat(direction, 1))
	}
	out.RealizedPnL = displayProfitPtr(realized)
	unrealized := new(big.Rat)
	if remaining.Sign() > 0 {
		if out.CurrentPrice == nil {
			return // Do not silently claim a total when an open leg is unpriced.
		}
		current, _ := dec(*out.CurrentPrice)
		unrealized.Mul(new(big.Rat).Sub(current, entry), remaining)
		unrealized.Mul(unrealized, big.NewRat(direction, 1))
	}
	out.UnrealizedPnL = displayProfitPtr(unrealized)
	total := new(big.Rat).Add(realized, unrealized)
	out.TotalPnL = displayProfitPtr(total)

	// Nominal return: price PnL divided by initial opening notional,
	// intentionally not leveraged ROI (historic leverage is not reliable).
	basis := new(big.Rat).Add(closed, remaining)
	if opened.Cmp(basis) > 0 {
		basis = opened
	}
	if basis.Sign() > 0 {
		notional := new(big.Rat).Mul(basis, entry)
		out.DisplayReturnPct = displayProfitPtr(new(big.Rat).Mul(new(big.Rat).Quo(total, notional), big.NewRat(100, 1)))
	}
}

// Limit displayed PnL to eight decimal places: weighted-average rational
// division can produce tiny tails not present in exchange prices.
func displayProfitValue(r *big.Rat) string {
	if r == nil {
		return "0"
	}
	s := r.FloatString(8)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}
	return s
}

func displayProfitPtr(r *big.Rat) *string {
	v := displayProfitValue(r)
	return &v
}

type profitAccumulator struct {
	realized, unrealized, total *big.Rat
}

func sumDisplayedPnL(positions []Position, grouped map[string]*profitAccumulator) int {
	unpriced := 0
	for _, p := range positions {
		if p.TotalPnL == nil || p.RealizedPnL == nil || p.UnrealizedPnL == nil || p.DisplayPnLAsset == "" {
			unpriced++
			continue
		}
		r, er := dec(*p.RealizedPnL)
		u, eu := dec(*p.UnrealizedPnL)
		t, et := dec(*p.TotalPnL)
		if er != nil || eu != nil || et != nil {
			unpriced++
			continue
		}
		s := grouped[p.DisplayPnLAsset]
		if s == nil {
			s = &profitAccumulator{new(big.Rat), new(big.Rat), new(big.Rat)}
			grouped[p.DisplayPnLAsset] = s
		}
		s.realized.Add(s.realized, r)
		s.unrealized.Add(s.unrealized, u)
		s.total.Add(s.total, t)
	}
	return unpriced
}
