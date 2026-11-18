package coinmquant

import (
	"fmt"
	"math/big"
	"strings"

	"go_binance_futures/models"
)

const precision = 18

func exact(raw string, positive bool) (*big.Rat, error) {
	s := strings.TrimSpace(raw)
	if s == "" || len(s) > 128 || strings.ContainsAny(s, "eE/") {
		return nil, fmt.Errorf("invalid decimal %q", s)
	}
	r, ok := new(big.Rat).SetString(s)
	if !ok || (positive && r.Sign() <= 0) {
		return nil, fmt.Errorf("invalid positive decimal %q", s)
	}
	return r, nil
}
func decimalString(r *big.Rat) string {
	if r == nil {
		return ""
	}
	s := r.FloatString(precision)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "" || s == "-" || s == "-0" {
		return "0"
	}
	return s
}
func decimalAdd(a, b *big.Rat) *big.Rat { return new(big.Rat).Add(a, b) }
func decimalZero() *big.Rat             { return new(big.Rat) }

// InversePnL calculates inverse COIN-M profits in the settlement asset:
// LONG: N*C*(1/entry-1/exit); SHORT: N*C*(1/exit-1/entry).
// Neither quantity nor contractSize are a number of BTC/ETH units.
// Uses arbitrary-precision rational arithmetic; only the DISPLAY string is
// rounded to 18 decimals. Funding and commission are separate.
func InversePnL(side, contracts, contractSize, entryPrice, exitPrice string) (string, error) {
	n, err := exact(contracts, true)
	if err != nil {
		return "", err
	}
	c, err := exact(contractSize, true)
	if err != nil {
		return "", err
	}
	entry, err := exact(entryPrice, true)
	if err != nil {
		return "", err
	}
	exit, err := exact(exitPrice, true)
	if err != nil {
		return "", err
	}
	one := big.NewRat(1, 1)
	invEntry := new(big.Rat).Quo(one, entry)
	invExit := new(big.Rat).Quo(one, exit)
	delta := new(big.Rat).Sub(invEntry, invExit)
	switch strings.ToUpper(strings.TrimSpace(side)) {
	case "LONG":
	case "SHORT":
		delta.Neg(delta)
	default:
		return "", fmt.Errorf("unsupported COIN-M side %q", side)
	}
	return decimalString(new(big.Rat).Mul(new(big.Rat).Mul(n, c), delta)), nil
}

// Mark is verified public DAPI /dapi/v1/premiumIndex data. It must NOT be
// silently replaced with the ticker's last-traded close price.
type Mark struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
	Time   int64  `json:"time"`
}

func (m Mark) Check(symbol string) error {
	if m.Symbol != symbol || !strings.HasSuffix(symbol, "USD_PERP") {
		return fmt.Errorf("mark price symbol mismatch")
	}
	_, err := exact(m.Price, true)
	return err
}

type AssetFee struct {
	Asset  string `json:"asset"`
	Amount string `json:"amount"`
}
type PositionSummary struct {
	models.CoinMManagedPosition
	RealizedPnL   string     `json:"realized_pnl"`
	UnrealizedPnL *string    `json:"unrealized_pnl"`
	TotalPnL      *string    `json:"total_pnl"`
	MarkPrice     *string    `json:"mark_price"`
	MarkTime      int64      `json:"mark_time"`
	RealizedUSD   *string    `json:"realized_usd_estimate"`
	UnrealizedUSD *string    `json:"unrealized_usd_estimate"`
	TotalUSD      *string    `json:"total_usd_estimate"`
	OpenFilled    string     `json:"open_filled_contracts"`
	CloseFilled   string     `json:"close_filled_contracts"`
	FeeByAsset    []AssetFee `json:"fees_by_asset"`
	// Status is "verified", "unverified_fills", "unpriced" or "invalid".
	PnLStatus       string `json:"pnl_status"`
	Warning         string `json:"warning"`
	FundingIncluded bool   `json:"funding_included"`
}
type AssetTotal struct {
	Asset           string     `json:"asset"`
	Positions       int        `json:"positions"`
	Verified        int        `json:"verified"`
	Unpriced        int        `json:"unpriced"`
	Pending         int        `json:"pending"`
	Invalid         int        `json:"invalid"`
	RealizedPnL     string     `json:"realized_pnl"`
	UnrealizedPnL   *string    `json:"unrealized_pnl"`
	TotalPnL        *string    `json:"total_pnl"`
	RealizedUSD     *string    `json:"realized_usd_estimate"`
	UnrealizedUSD   *string    `json:"unrealized_usd_estimate"`
	TotalUSD        *string    `json:"total_usd_estimate"`
	Fees            []AssetFee `json:"fees"`
	FundingIncluded bool       `json:"funding_included"`
}

// Display estimates are derived from the corresponding COIN-M mark. They
// are NOT wallet USD balances, transactions, or real USD-settled PnL.
func usdEstimate(amount, mark *big.Rat) *big.Rat {
	return new(big.Rat).Mul(amount, mark)
}
func settledAssetForSymbol(symbol, asset string) bool {
	return strings.TrimSuffix(strings.ToUpper(symbol), "USD_PERP") == strings.ToUpper(asset) &&
		strings.HasSuffix(strings.ToUpper(symbol), "USD_PERP")
}
