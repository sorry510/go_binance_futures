package spotexecution

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

const decimalPlaces int32 = 18

var one = decimal.NewFromInt(1)
var buyPriceMargin = decimal.RequireFromString("1.01") // 1% price budget reserve

func parseAmount(s string, positive bool) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	if s == "" || strings.ContainsAny(s, "eE+-") {
		return decimal.Zero, fmt.Errorf("invalid decimal %q", s)
	}
	i := strings.Split(s, ".")
	if len(i) > 2 || len(i[0]) == 0 || len(i[0]) > 18 || (len(i) == 2 && (len(i[1]) == 0 || len(i[1]) > 18)) {
		return decimal.Zero, fmt.Errorf("invalid DECIMAL(36,18) amount %q", s)
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return decimal.Zero, fmt.Errorf("invalid amount %q", s)
		}
	}
	v, err := decimal.NewFromString(s)
	if err != nil || (positive && !v.IsPositive()) {
		return decimal.Zero, fmt.Errorf("invalid positive amount %q", s)
	}
	return v, nil
}
func optionalAmount(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return parseAmount(s, false)
}
func fixed(d decimal.Decimal) (string, error) {
	if d.Exponent() < -decimalPlaces {
		return "", fmt.Errorf("too many decimal places")
	}
	out := d.StringFixed(decimalPlaces)
	if len(strings.SplitN(strings.TrimPrefix(out, "-"), ".", 2)[0]) > 18 {
		return "", fmt.Errorf("decimal exceeds database precision")
	}
	return d.String(), nil
}
func floorStep(quantity, step decimal.Decimal) (decimal.Decimal, error) {
	if !step.IsPositive() {
		return decimal.Zero, fmt.Errorf("missing positive Spot LOT_SIZE step")
	}
	return quantity.Div(step).Floor().Mul(step), nil
}
func prepareQty(r Rules, qty, mark decimal.Decimal) (string, error) {
	if !r.Trading || !r.MarketAllowed {
		return "", fmt.Errorf("Spot symbol %s not trading or MARKET not permitted", r.Symbol)
	}
	step, err := parseAmount(r.Step, true)
	if err != nil {
		return "", fmt.Errorf("invalid MARKET_LOT_SIZE/LOT_SIZE: %w", err)
	}
	qty, err = floorStep(qty, step)
	if err != nil {
		return "", err
	}
	min, err := optionalAmount(r.MinQty)
	if err != nil {
		return "", err
	}
	max, err := optionalAmount(r.MaxQty)
	if err != nil {
		return "", err
	}
	if !qty.IsPositive() || qty.LessThan(min) || (max.IsPositive() && qty.GreaterThan(max)) {
		return "", fmt.Errorf("Spot quantity violates lot size")
	}
	notional := qty.Mul(mark)
	minN, err := optionalAmount(r.MinNotional)
	if err != nil {
		return "", err
	}
	maxN, err := optionalAmount(r.MaxNotional)
	if err != nil {
		return "", err
	}
	if (minN.IsPositive() && notional.LessThan(minN)) || (maxN.IsPositive() && notional.GreaterThan(maxN)) {
		return "", fmt.Errorf("Spot order violates MIN_NOTIONAL/NOTIONAL")
	}
	return fixed(qty)
}
func planBuy(r Rules, marketPrice, budget, freeQuote string) (string, error) {
	price, err := parseAmount(marketPrice, true)
	if err != nil {
		return "", err
	}
	quote, err := parseAmount(budget, true)
	if err != nil {
		return "", err
	}
	free, err := parseAmount(freeQuote, false)
	if err != nil {
		return "", err
	}
	if free.LessThan(quote) {
		return "", fmt.Errorf("insufficient free Spot quote %s, need %s", r.Quote, budget)
	}
	if quote.IsZero() {
		return "", fmt.Errorf("zero Spot BUY budget")
	}
	// Quantity orders are used so the ledger's requested_qty is known. A
	// conservative 1% price reserve avoids consuming the entire quote budget.
	qty := quote.DivRound(price.Mul(buyPriceMargin), decimalPlaces)
	return prepareQty(r, qty, price)
}
func planSell(r Rules, marketPrice, owned, freeBase string) (string, error) {
	price, err := parseAmount(marketPrice, true)
	if err != nil {
		return "", err
	}
	strategy, err := parseAmount(owned, true)
	if err != nil {
		return "", err
	}
	free, err := parseAmount(freeBase, false)
	if err != nil {
		return "", err
	}
	// Spot inventory is fungible. Never shrink an order to whatever coins
	// happen to remain in a manually changed wallet: that could sell the
	// user's unrelated holdings. Any deficit forces reconciliation first.
	if free.LessThan(strategy) {
		return "", fmt.Errorf("%w: Spot free base asset below verified strategy-owned inventory", ErrNeedsReview)
	}
	// Sell precisely the verified strategy inventory. Any fixed haircut
	// leaves permanent dust and prevents a CLOSED lifecycle. If base-asset
	// commission is charged IN ADDITION to executed qty, the Fill ledger
	// refuses to debit more than verified stock (manual reconciliation).
	// Never silently round an untradeable residual down and call it CLOSED.
	step, err := parseAmount(r.Step, true)
	if err != nil {
		return "", err
	}
	if !strategy.Div(step).Equal(strategy.Div(step).Floor()) {
		return "", fmt.Errorf("%w: strategy residual cannot be fully sold with current MARKET_LOT_SIZE; reconcile base-fee dust", ErrNeedsReview)
	}
	return prepareQty(r, strategy, price)
}
