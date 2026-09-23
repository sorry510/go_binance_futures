package scanner

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"go_binance_futures/models"
)

func TestSmartLocalV2DeterministicAndHardFilters(t *testing.T) {
	now := time.Now().UnixMilli()
	items := []*models.Symbols{
		smartSymbol("AAAUSDT", 1, 8, 120_000_000, 220_000, now),
		smartSymbol("BBBUSDT", 1, 8, 120_000_000, 220_000, now),
		smartSymbol("DISABLEDUSDT", 0, 5, 500_000_000, 500_000, now),
		smartSymbol("STALEUSDT", 1, 5, 500_000_000, 500_000, now-60_000),
		smartSymbol("LOWVOLUSDT", 1, 5, 5_000_000, 500_000, now),
		smartSymbol("COOLDOWNUSDT", 1, 5, 500_000_000, 500_000, now),
		smartSymbol("BTCUSDT", 1, 4, 2_000_000_000, 900_000, now),
	}
	cooldown := map[string]bool{"COOLDOWNUSDT": true}
	opts := SmartLocalV2Options{Limit: 10, MaxDataAgeMs: 30_000, MinQuoteVolume: 10_000_000, IncludeExcluded: true}

	first := SmartLocalV2FromSymbols(items, cooldown, opts)
	second := SmartLocalV2FromSymbols(items, cooldown, opts)

	if !reflect.DeepEqual(first.Candidates, second.Candidates) {
		t.Fatalf("same local input must be deterministic: first=%+v second=%+v", first.Candidates, second.Candidates)
	}
	if first.Meta["rest_api_used"] != false {
		t.Fatalf("rest_api_used=%v want=false", first.Meta["rest_api_used"])
	}

	got := make([]string, 0, len(first.Candidates))
	for _, item := range first.Candidates {
		got = append(got, item.Symbol)
	}
	if len(got) != 3 {
		t.Fatalf("candidates=%v want 3", got)
	}
	// BTC/ETH are allowed for auto-trade selector even though the generic scanner
	// excludes them by default for altcoin opportunity scans.
	if !containsString(got, "BTCUSDT") {
		t.Fatalf("BTCUSDT should be eligible for smart_local_v2: %v", got)
	}

	excluded := map[string]string{}
	for _, item := range first.Excluded {
		excluded[item.Symbol] = item.Reason
	}
	for _, symbol := range []string{"DISABLEDUSDT", "STALEUSDT", "LOWVOLUSDT", "COOLDOWNUSDT"} {
		if excluded[symbol] == "" {
			t.Fatalf("%s missing exclusion: %+v", symbol, first.Excluded)
		}
	}
}

func TestSmartLocalV2RuntimeOmitsExcludedDebugData(t *testing.T) {
	now := time.Now().UnixMilli()
	result := SmartLocalV2FromSymbols([]*models.Symbols{
		smartSymbol("DISABLEDUSDT", 0, 5, 500_000_000, 500_000, now),
		smartSymbol("GOODUSDT", 1, 5, 500_000_000, 500_000, now),
	}, nil, SmartLocalV2Options{Limit: 5})
	if len(result.Excluded) != 0 {
		t.Fatalf("runtime selector should omit debug exclusions: %+v", result.Excluded)
	}
	if len(result.Candidates) != 1 || result.Candidates[0].Symbol != "GOODUSDT" {
		t.Fatalf("unexpected candidates: %+v", result.Candidates)
	}
}

func TestSmartLocalV2StableTieBreakUsesSymbol(t *testing.T) {
	now := time.Now().UnixMilli()
	// Same score and quote volume: Symbol must make ordering stable regardless
	// of the caller's input order.
	a := smartSymbol("AAAUSDT", 1, 10, 100_000_000, 100_000, now)
	b := smartSymbol("BBBUSDT", 1, 10, 100_000_000, 100_000, now)

	result := SmartLocalV2FromSymbols([]*models.Symbols{b, a}, nil, SmartLocalV2Options{Limit: 5})
	if len(result.Candidates) != 2 {
		t.Fatalf("candidate count=%d", len(result.Candidates))
	}
	if result.Candidates[0].Symbol != "AAAUSDT" || result.Candidates[1].Symbol != "BBBUSDT" {
		t.Fatalf("tie order=%v, want AAAUSDT then BBBUSDT", []string{result.Candidates[0].Symbol, result.Candidates[1].Symbol})
	}
}

func TestGenericPrefilterStillExcludesBenchmarksByDefault(t *testing.T) {
	now := time.Now().UnixMilli()
	item := smartSymbol("BTCUSDT", 1, 4, 2_000_000_000, 900_000, now)
	result := PrefilterTop30FromSymbols([]*models.Symbols{item}, PrefilterOptions{Limit: 5})
	if len(result.Candidates) != 0 {
		t.Fatalf("generic prefilter benchmark compatibility changed: %+v", result.Candidates)
	}
}

func smartSymbol(name string, enable int, pct, quoteVolume, tradeCount float64, updateTime int64) *models.Symbols {
	return &models.Symbols{
		Symbol:         name,
		Enable:         enable,
		PercentChange:  pct,
		Close:          "1.08",
		Open:           "1.00",
		Low:            "0.98",
		High:           "1.10",
		QuoteVolume:    quoteVolume,
		TradeCount:     tradeCount,
		Type:           "USDT",
		UpdateTime:     updateTime,
		LastClose:      "1.07",
		LastUpdateTime: updateTime - 1000,
	}
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func TestGenericPrefilterDoesNotUseSmartLocalV2TradeCountOrTieBreak(t *testing.T) {
	now := time.Now().UnixMilli()
	b := smartSymbol("BBBUSDT", 1, 10, 100_000_000, 5_000, now)
	a := smartSymbol("AAAUSDT", 1, 10, 100_000_000, 500_000, now)

	result := PrefilterTop30FromSymbols([]*models.Symbols{b, a}, PrefilterOptions{Limit: 5, IncludeBenchmarks: true})
	if len(result.Candidates) != 2 {
		t.Fatalf("candidate count=%d", len(result.Candidates))
	}
	if result.Candidates[0].Symbol != "BBBUSDT" || result.Candidates[1].Symbol != "AAAUSDT" {
		t.Fatalf("generic prefilter ordering changed: %+v", result.Candidates)
	}
	if result.Candidates[0].Score != result.Candidates[1].Score {
		t.Fatalf("generic prefilter must ignore V4-3 TradeCount score: %+v", result.Candidates)
	}

	smart := SmartLocalV2FromSymbols([]*models.Symbols{b, a}, nil, SmartLocalV2Options{Limit: 5})
	if len(smart.Candidates) != 2 || smart.Candidates[0].Symbol != "AAAUSDT" {
		t.Fatalf("smart_local_v2 should apply TradeCount scoring: %+v", smart.Candidates)
	}
}

func TestSmartLocalV2LimitClampsToSixty(t *testing.T) {
	now := time.Now().UnixMilli()
	items := make([]*models.Symbols, 0, 70)
	for i := 0; i < 70; i++ {
		name := fmt.Sprintf("C%02dUSDT", i)
		items = append(items, smartSymbol(name, 1, 5, 100_000_000+float64(i), 100_000, now))
	}
	result := SmartLocalV2FromSymbols(items, nil, SmartLocalV2Options{Limit: 100})
	if len(result.Candidates) != 60 {
		t.Fatalf("candidate count=%d want=60", len(result.Candidates))
	}
	if got := result.Meta["requested_limit"]; got != 100 {
		t.Fatalf("requested_limit=%v want=100", got)
	}
	if got := result.Meta["effective_limit"]; got != 60 {
		t.Fatalf("effective_limit=%v want=60", got)
	}
}

func TestGenericPrefilterStillCapsAtThirty(t *testing.T) {
	now := time.Now().UnixMilli()
	items := make([]*models.Symbols, 0, 40)
	for i := 0; i < 40; i++ {
		name := fmt.Sprintf("G%02dUSDT", i)
		items = append(items, smartSymbol(name, 1, 5, 100_000_000+float64(i), 100_000, now))
	}
	result := PrefilterTop30FromSymbols(items, PrefilterOptions{
		Limit:             60,
		IncludeBenchmarks: true,
	})
	if len(result.Candidates) != 30 {
		t.Fatalf("generic candidate count=%d want=30", len(result.Candidates))
	}
	if got := result.Meta["limit"]; got != 30 {
		t.Fatalf("generic effective limit=%v want=30", got)
	}
}

func TestSmartLocalV2UsesFiveMillionQuoteVolumeThreshold(t *testing.T) {
	now := time.Now().UnixMilli()
	result := SmartLocalV2FromSymbols([]*models.Symbols{
		smartSymbol("PASSUSDT", 1, 12, 7_000_000, 100_000, now),
		smartSymbol("FAILUSDT", 1, -12, 4_000_000, 100_000, now),
	}, nil, SmartLocalV2Options{Limit: 30, IncludeExcluded: true})
	if len(result.Candidates) != 1 || result.Candidates[0].Symbol != "PASSUSDT" {
		t.Fatalf("unexpected candidates: %+v", result.Candidates)
	}
	found := false
	for _, item := range result.Excluded {
		if item.Symbol == "FAILUSDT" && item.Reason == "24h 成交额低于阈值" {
			found = true
		}
	}
	if !found {
		t.Fatalf("FAILUSDT missing low-volume exclusion: %+v", result.Excluded)
	}
	if got := result.Meta["min_quote_volume"]; got != float64(5_000_000) {
		t.Fatalf("min_quote_volume=%v want=5000000", got)
	}
}

func TestSmartLocalV2SymmetricChangeScoring(t *testing.T) {
	up := &models.Symbols{
		Symbol: "UPUSDT", Enable: 1, PercentChange: 25,
		Close: "1.08", Open: "1.00", Low: "0.98", High: "1.10",
		QuoteVolume: 100_000_000, TradeCount: 100_000,
		Type: "USDT", UpdateTime: time.Now().UnixMilli(),
		LastClose: "1.0693069307", LastUpdateTime: time.Now().UnixMilli() - 1000,
	}
	down := &models.Symbols{
		Symbol: "DOWNUSDT", Enable: 1, PercentChange: -25,
		Close: "0.92", Open: "1.00", Low: "0.90", High: "1.02",
		QuoteVolume: 100_000_000, TradeCount: 100_000,
		Type: "USDT", UpdateTime: up.UpdateTime,
		LastClose: "0.9292929293", LastUpdateTime: up.LastUpdateTime,
	}
	result := SmartLocalV2FromSymbols([]*models.Symbols{up, down}, nil, SmartLocalV2Options{Limit: 30})
	if len(result.Candidates) != 2 {
		t.Fatalf("candidates=%+v", result.Candidates)
	}
	bySymbol := map[string]PrefilterCandidate{}
	for _, item := range result.Candidates {
		bySymbol[item.Symbol] = item
	}
	if bySymbol["DOWNUSDT"].Score <= 0 || bySymbol["UPUSDT"].Score <= 0 {
		t.Fatalf("unexpected scores: %+v", bySymbol)
	}
	if math.Abs(bySymbol["UPUSDT"].Score-bySymbol["DOWNUSDT"].Score) > 3 {
		t.Fatalf("symmetric up/down structures should score similarly: up=%+v down=%+v", bySymbol["UPUSDT"], bySymbol["DOWNUSDT"])
	}
	if !containsString(bySymbol["DOWNUSDT"].Reasons, "24h 涨跌变化活跃且不过度") {
		t.Fatalf("down candidate missing symmetric change reason: %+v", bySymbol["DOWNUSDT"])
	}
	if !containsString(bySymbol["DOWNUSDT"].Reasons, "本地最近一次价格更新继续向下") {
		t.Fatalf("down candidate should reward downward local momentum: %+v", bySymbol["DOWNUSDT"])
	}
}

func TestSmartLocalV2ExtremeMoveFilterIsSymmetric(t *testing.T) {
	now := time.Now().UnixMilli()
	up := &models.Symbols{
		Symbol: "EXTREMEUPUSDT", Enable: 1, PercentChange: 61,
		Close: "1.50", Open: "1.00", Low: "0.95", High: "1.55",
		QuoteVolume: 100_000_000, TradeCount: 100_000, Type: "USDT", UpdateTime: now,
	}
	down := &models.Symbols{
		Symbol: "EXTREMEDOWNUSDT", Enable: 1, PercentChange: -61,
		Close: "0.50", Open: "1.00", Low: "0.45", High: "1.05",
		QuoteVolume: 100_000_000, TradeCount: 100_000, Type: "USDT", UpdateTime: now,
	}
	result := SmartLocalV2FromSymbols([]*models.Symbols{up, down}, nil, SmartLocalV2Options{Limit: 30, IncludeExcluded: true})
	if len(result.Candidates) != 0 {
		t.Fatalf("extreme moves should be excluded symmetrically: %+v", result.Candidates)
	}
	excluded := map[string]string{}
	for _, item := range result.Excluded {
		excluded[item.Symbol] = item.Reason
	}
	for _, symbol := range []string{"EXTREMEUPUSDT", "EXTREMEDOWNUSDT"} {
		if !strings.Contains(excluded[symbol], "涨跌变化绝对值超过 60%") {
			t.Fatalf("%s exclusion=%q", symbol, excluded[symbol])
		}
	}
}

func TestSmartLocalV2OptInScoreDeltasDoNotLeakIntoGeneric(t *testing.T) {
	now := time.Now().UnixMilli()
	fixtures := []struct {
		name          string
		pct           float64
		tradeCount    float64
		expectedDelta float64
	}{
		{name: "high_trade_count", pct: 10, tradeCount: 220_000, expectedDelta: 8},
		{name: "flat_low_trade_count", pct: 0, tradeCount: 5_000, expectedDelta: -12},
		{name: "medium_trade_count", pct: 10, tradeCount: 100_000, expectedDelta: 4},
		{name: "no_trade_count", pct: 10, tradeCount: 0, expectedDelta: 0},
	}

	var genericBaseline *float64
	for _, tc := range fixtures {
		t.Run(tc.name, func(t *testing.T) {
			// Keep the baseline well below the 100-point clamp so the exact
			// opt-in delta remains observable.
			item := smartSymbol("FIXTUREUSDT", 1, tc.pct, 5_000_000, tc.tradeCount, now)
			item.LastClose = item.Close // neutralize local-momentum score delta

			generic := PrefilterTop30FromSymbols([]*models.Symbols{item}, PrefilterOptions{
				Limit:             5,
				IncludeBenchmarks: true,
				MinQuoteVolume:    5_000_000,
			})
			smart := PrefilterTop30FromSymbols([]*models.Symbols{item}, PrefilterOptions{
				Limit:                  5,
				MaxLimit:               60,
				IncludeBenchmarks:      true,
				MinQuoteVolume:         5_000_000,
				UseTradeCountScore:     true,
				StableSymbolTieBreak:   true,
				SymmetricChangeScoring: true,
			})
			if len(generic.Candidates) != 1 || len(smart.Candidates) != 1 {
				t.Fatalf("unexpected candidates: generic=%+v smart=%+v", generic.Candidates, smart.Candidates)
			}
			genericScore := generic.Candidates[0].Score
			smartScore := smart.Candidates[0].Score
			if genericBaseline == nil {
				baseline := genericScore
				genericBaseline = &baseline
			} else if math.Abs(genericScore-*genericBaseline) > 1e-9 {
				t.Fatalf("generic score drifted across V4-3-only fixtures: got=%v baseline=%v", genericScore, *genericBaseline)
			}
			if delta := smartScore - genericScore; math.Abs(delta-tc.expectedDelta) > 1e-9 {
				t.Fatalf("score delta=%v want=%v; generic=%v smart=%v", delta, tc.expectedDelta, genericScore, smartScore)
			}
			for _, text := range append(append([]string{}, generic.Candidates[0].Reasons...), generic.Candidates[0].Risks...) {
				if strings.Contains(text, "成交笔数") || strings.Contains(text, "涨跌变化") {
					t.Fatalf("V4-3-only reason leaked into generic prefilter: %q", text)
				}
			}
		})
	}
}
