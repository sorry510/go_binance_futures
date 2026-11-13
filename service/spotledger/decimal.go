// Package spotledger is a local-only Spot BUY/SELL ledger. It never sends
// Binance orders or reads the Spot wallet. Four v25 tables remain isolated.
package spotledger

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

const scale int32 = 18

var exactDecimal = regexp.MustCompile(`^[0-9]{1,18}(?:\.[0-9]{1,18})?$`)

var zero = decimal.Zero

// parseDBDecimal rejects exponent forms and values which would be rounded by
// a DECIMAL(36,18) column, including overflow beyond 18 integer digits.
func parseDBDecimal(value string, positive bool) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if !exactDecimal.MatchString(value) {
		return zero, fmt.Errorf("invalid DECIMAL(36,18) value %q", value)
	}
	v, err := decimal.NewFromString(value)
	if err != nil || (positive && !v.IsPositive()) {
		return zero, fmt.Errorf("invalid positive amount %q", value)
	}
	return v, nil
}

func parseSignedDBDecimal(value string) (decimal.Decimal, error) {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "-") {
		n, err := parseDBDecimal(strings.TrimPrefix(value, "-"), false)
		return n.Neg(), err
	}
	return parseDBDecimal(value, false)
}

func numeric(v decimal.Decimal) (string, error) {
	v = v.Round(scale)
	value := v.StringFixed(scale)
	if len(strings.SplitN(strings.TrimPrefix(value, "-"), ".", 2)[0]) > 18 {
		return "", fmt.Errorf("decimal overflow: %s", value)
	}
	return value, nil
}
