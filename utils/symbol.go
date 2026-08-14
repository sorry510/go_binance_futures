package utils

import "strings"

// FuturesSymbolType returns the supported quote asset for a futures symbol.
func FuturesSymbolType(symbol string, quoteAsset string) string {
	quoteAsset = strings.ToUpper(strings.TrimSpace(quoteAsset))
	switch quoteAsset {
	case "USDT", "USDC", "FDUSD":
		return quoteAsset
	}

	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	for _, suffix := range []string{"USDT", "FDUSD", "USDC"} {
		if strings.HasSuffix(symbol, suffix) {
			return suffix
		}
	}

	return ""
}
