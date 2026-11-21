package backtest

import (
	"regexp"
	"strconv"
	"strings"

	"go_binance_futures/feature/strategy/line"
)

// The dynamic CLOSE fast path is intentionally narrower than general Expr.
// It admits ROI, the immutable entry price, exactly the forming 1h Close[0],
// and completed [1+] kline/technology series. Everything else falls back to
// the original full BuildIntrabar per tick. Never infer this from strategy name.
var (
	adaptiveCloseMemberPattern = regexp.MustCompile(`\b([A-Za-z_][A-Za-z0-9_]*)\.([A-Za-z_][A-Za-z0-9_]*)`)
	adaptiveCloseDynamicTokens = []string{
		"NowTime", "NowPrice", "NowSymbol", "FundingRate", "MarketCondition",
		"NetROI", "Fee", "NetProfit", "Positions", "KdjSimple", "IsAsc", "IsDesc",
		"SystemStartTime", "OpenStrategyHash",
	}
	adaptiveCloseCompletedSeries = []string{
		"kline_", "ma_", "ema_", "macd_", "adx_", "mfi_", "obv_",
		"cci_", "roc_", "kdj_", "rsi_", "kc_", "boll_",
		"donchian_", "atr_", "supertrend_",
	}
)

func canReuseFormingCloseEnvironment(rules []Rule, side string) bool {
	want := "close_long"
	if side == "SHORT" {
		want = "close_short"
	}
	matched := false
	seenFormingClose := false
	for _, rule := range rules {
		if !rule.Enable || rule.Type != want {
			continue
		}
		matched = true
		code := rule.Code
		// Reject dynamic field indirection through strings or interpolation.
		if strings.ContainsAny(code, "'\"\x60") {
			return false
		}
		// Only the precise immutable member is whitelisted. All other
		// Position fields, including MarkPrice/FundingPnL, are tick-dependent.
		code = strings.ReplaceAll(code, "Position.EntryPrice", "")
		if strings.Contains(code, "Position") {
			return false
		}
		// Keep this one current [0] field live at every checked tick. Do not
		// accept High[0], Volume[0], indicator[0], dynamic indices, or slices.
		if strings.Contains(code, "kline_1h.Close[0]") {
			seenFormingClose = true
		}
		code = strings.ReplaceAll(code, "kline_1h.Close[0]", "")
		for _, token := range adaptiveCloseDynamicTokens {
			if strings.Contains(code, token) {
				return false
			}
		}
		matches := adaptiveCloseMemberPattern.FindAllStringSubmatchIndex(code, -1)
		for _, m := range matches {
			name := code[m[2]:m[3]]
			known := false
			for _, prefix := range adaptiveCloseCompletedSeries {
				if strings.HasPrefix(name, prefix) {
					known = true
					break
				}
			}
			if !known || m[1] >= len(code) || code[m[1]] != '[' {
				return false
			}
		}
		indices := bracketIndexPattern.FindAllStringSubmatch(code, -1)
		if len(indices) != strings.Count(code, "[") {
			return false
		}
		for _, index := range indices {
			value, err := strconv.Atoi(strings.TrimSpace(index[1]))
			if err != nil || value < 1 {
				return false
			}
		}
	}
	return matched && seenFormingClose
}

func prepareFormingCloseEnvironment(env map[string]interface{}) bool {
	candle, ok := env["kline_1h"].(line.KLinePrice)
	if !ok || len(candle.Close) == 0 {
		return false
	}
	// The historical environment caches backing indicator/price slices.
	// Clone once so updating forming Close[0] never changes shared caches.
	candle.Close = append([]float64(nil), candle.Close...)
	env["kline_1h"] = candle
	return true
}

func updateFormingCloseEnvironment(env map[string]interface{}, price float64) {
	candle := env["kline_1h"].(line.KLinePrice)
	candle.Close[0] = price
}
