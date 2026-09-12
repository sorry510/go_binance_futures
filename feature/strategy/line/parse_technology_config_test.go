package line

import (
	"encoding/json"
	"strings"
	"testing"

	"go_binance_futures/technology"
)

func TestHasEnabledCloseStrategy(t *testing.T) {
	var strategies technology.StrategyConfig
	if err := json.Unmarshal([]byte(`[
		{"name":"long close","enable":true,"code":"true","type":"close_long"},
		{"name":"short close","enable":false,"code":"true","type":"close_short"}
	]`), &strategies); err != nil {
		t.Fatal(err)
	}
	if !HasEnabledCloseStrategy(strategies, "LONG") {
		t.Fatal("enabled close_long must protect LONG from fallback close logic")
	}
	if HasEnabledCloseStrategy(strategies, "SHORT") {
		t.Fatal("disabled close_short must not suppress fallback close logic")
	}
}

func TestTechnologyKlineLimitUsesIndicatorWarmup(t *testing.T) {
	config := technology.TechnologyConfig{
		EMA: []technology.IndicatorConfig{{Name: "ema200", Enable: true, KlineInterval: "1h", Period: 200}},
		ADX: []technology.IndicatorConfig{{Name: "adx100", Enable: true, KlineInterval: "4h", Period: 100}},
	}
	// EMA200 needs 200 + 31 reserve = 231; ADX100 needs 200 + 31 reserve = 231.
	if got := technologyKlineLimit(config); got != 231 {
		t.Fatalf("technologyKlineLimit=%d, want 231", got)
	}
	if err := ValidateTechnologyConfig(config); err != nil {
		t.Fatalf("periods above the old 150-bar limit should now be valid: %v", err)
	}
}

func TestTechnologyKlineLimitKeepsCompatibilityBaseline(t *testing.T) {
	config := technology.TechnologyConfig{
		RSI:  []technology.IndicatorConfig{{Name: "rsi14", Enable: true, KlineInterval: "1h", Period: 14}},
		MACD: []technology.IndicatorConfig{{Name: "macd", Enable: true, KlineInterval: "1h", FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9}},
	}
	// Mathematical requirements are below 150, but existing strategies keep the old
	// 150-bar warmup baseline so indicator values do not shift after this change.
	if got := technologyKlineLimit(config); got != defaultStrategyKlineLimit {
		t.Fatalf("technologyKlineLimit=%d, want compatibility baseline %d", got, defaultStrategyKlineLimit)
	}
}

func TestShortHistoryCanStillCalculateWhenIndicatorMinimumIsMet(t *testing.T) {
	prices := make([]float64, 20)
	for i := range prices {
		prices[i] = 100 + float64(i)
	}
	values, err := CalculateRSI(prices, 14)
	if err != nil {
		t.Fatalf("20 available bars should be enough for RSI14 even though the request limit is 150: %v", err)
	}
	if len(values) != 6 {
		t.Fatalf("RSI output length=%d, want 6", len(values))
	}
}

func TestValidateTechnologyConfigRejectsBeyondBinanceLimit(t *testing.T) {
	config := technology.TechnologyConfig{
		EMA: []technology.IndicatorConfig{{Name: "ema1470", Enable: true, KlineInterval: "1h", Period: 1470}},
	}
	err := ValidateTechnologyConfig(config)
	if err == nil {
		t.Fatal("expected validation error when warmup plus output reserve exceeds Binance 1500-bar limit")
	}
	if !strings.Contains(err.Error(), "maximum is 1500") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}
