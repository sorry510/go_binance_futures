package line

import (
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	"go_binance_futures/models"

	"github.com/adshao/go-binance/v2/futures"
)

func spotEnvTestCoin(now time.Time) models.SpotSymbols {
	return models.SpotSymbols{
		Symbol: "BTCUSDT", Type: "USDT",
		Close: "105", Open: "101", High: "109", Low: "99", UpdateTime: now.Add(-30 * time.Second).UnixMilli(),
		Technology: `{"ma":[{"name":"ma_5","enable":true,"kline_interval":"5m","period":5}]}`,
	}
}

func spotEnvTestKlines(now time.Time) func(string, string, int) ([]*futures.Kline, error) {
	return func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		if symbol != "BTCUSDT" || interval != "5m" || limit < 150 {
			return nil, errors.New("requested non-Spot data or incorrect indicator warmup")
		}
		rows := make([]*futures.Kline, limit)
		for index := range rows {
			rows[index] = &futures.Kline{
				Open: "100", Close: "105", High: "110", Low: "99",
				OpenTime:  now.Add(time.Duration(index-limit+1) * 5 * time.Minute).UnixMilli(),
				CloseTime: now.Add(time.Duration(index-limit+2)*5*time.Minute).UnixMilli() - 1,
			}
		}
		return rows, nil
	}
}

func TestStage0671VerifiedSpotEnvPreservesDslAndIndicatorData(t *testing.T) {
	now := time.Now()
	coin := spotEnvTestCoin(now)
	env, err := buildVerifiedSpotRuleEnv(coin, now, "", spotEnvTestKlines(now))
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := env["SystemStartTime"]; !exists {
		t.Fatalf("legacy shared SystemStartTime variable is missing")
	}
	if env["NowPrice"] != 105.0 || env["NowSymbolClose"] != 105.0 {
		t.Fatalf("Spot local ticker was not used: %+v", env["NowPrice"])
	}
	ma, ok := env["ma_5"].(ConfigData)
	if !ok || len(ma.Data) == 0 {
		t.Fatalf("Spot MA indicator missing: %#v", env["ma_5"])
	}
	_, ok = env["kline_5m"].(KLinePrice)
	if !ok {
		t.Fatalf("Futures/Spot shared DSL Kline alias missing")
	}
	coin.Close = "0"
	env, err = buildVerifiedSpotRuleEnv(coin, now, "123.25", spotEnvTestKlines(now))
	if err != nil || env["NowPrice"] != 123.25 {
		t.Fatalf("fresh Spot WS price should override missing DB ticker: %v %+v", err, env["NowPrice"])
	}
}

func TestStage0671VerifiedSpotEnvFailsClosedOnBadInputs(t *testing.T) {
	now := time.Now()
	coin := spotEnvTestCoin(now)
	fetch := spotEnvTestKlines(now)
	for _, tc := range []struct {
		name   string
		edit   func(*models.SpotSymbols)
		live   string
		loader func(string, string, int) ([]*futures.Kline, error)
	}{
		{"stale", func(c *models.SpotSymbols) { c.UpdateTime = now.Add(-6 * time.Minute).UnixMilli() }, "", fetch},
		{"future timestamp", func(c *models.SpotSymbols) { c.UpdateTime = now.Add(time.Hour).UnixMilli() }, "", fetch},
		{"stale OHLC despite live WS price", func(c *models.SpotSymbols) { c.UpdateTime = 0 }, "110", fetch},
		{"missing price", func(c *models.SpotSymbols) { c.Close = "0" }, "", fetch},
		{"bad 24h high", func(c *models.SpotSymbols) { c.High = "NaN" }, "", fetch},
		{"bad 24h low", func(c *models.SpotSymbols) { c.Low = "0" }, "", fetch},
		{"bad 24h open", func(c *models.SpotSymbols) { c.Open = "0" }, "", fetch},
		{"inverted 24h range", func(c *models.SpotSymbols) { c.High = "1" }, "", fetch},
		{"invalid live price", func(c *models.SpotSymbols) {}, "NaN", fetch},
		{"unsupported quote", func(c *models.SpotSymbols) { c.Symbol = "BTCUSD" }, "", fetch},
		{"missing indicator", func(c *models.SpotSymbols) {}, "", func(_, _ string, _ int) ([]*futures.Kline, error) { return nil, errors.New("Spot market unavailable") }},
		{"reserved name", func(c *models.SpotSymbols) {
			c.Technology = `{"ma":[{"name":"ROI","enable":true,"kline_interval":"5m","period":5}]}`
		}, "", fetch},
		{"bad JSON", func(c *models.SpotSymbols) { c.Technology = "garbage" }, "", fetch},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := coin
			tc.edit(&bad)
			env, err := buildVerifiedSpotRuleEnv(bad, now, tc.live, tc.loader)
			if err == nil || env != nil {
				t.Fatalf("invalid Spot market data returned tradeable environment: err=%v env=%v", err, env != nil)
			}
		})
	}
}

func TestStage0671SpotStrategyNoFuturesDataNoLiveOrder(t *testing.T) {
	now := time.Now()
	coin := spotEnvTestCoin(now)
	coin.Strategy = `[{"name":"spot MA entry","type":"long","enable":true,"code":"NowPrice > 100 and ma_5.Data[0] > 0 and kline_5m.Close[0] > 0"}]`
	e := SpotRuleEvaluator{
		LoadEnvironment: func(c models.SpotSymbols) (map[string]interface{}, error) {
			return buildVerifiedSpotRuleEnv(c, now, "", spotEnvTestKlines(now))
		},
	}
	signal, err := e.EvaluateEntry(coin)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "BUY" || !strings.Contains(signal.RuleName, "MA") {
		t.Fatalf("Spot indicator did not feed same expr DSL: %+v", signal)
	}
	// A false Spot price cannot be converted into an entry through a stale
	// Futures record, even if some other market happens to have prices.
	coin.UpdateTime = now.Add(-10 * time.Minute).UnixMilli()
	signal, err = e.EvaluateEntry(coin)
	if err == nil || signal.Action != "" {
		t.Fatalf("stale Spot ticker fallback silently opened: %+v %v", signal, err)
	}
}

// Review F5: use each interval's actual duration, not a double-interval
// tolerance. A 60-day-old 1M candle must not authorize live Spot signals.
func TestReview0671SpotKlineFreshnessByInterval(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		interval string
		fresh    time.Duration
		stale    time.Duration
	}{
		{"1d", 20 * time.Hour, 26 * time.Hour},
		{"1w", 6 * 24 * time.Hour, 8 * 24 * time.Hour},
		{"1M", 20 * 24 * time.Hour, 60 * 24 * time.Hour},
	} {
		t.Run(tc.interval, func(t *testing.T) {
			makeRow := func(age time.Duration) []*futures.Kline {
				open := now.Add(-age)
				return []*futures.Kline{{
					Open: "100", High: "110", Low: "90", Close: "105",
					OpenTime:  open.UnixMilli(),
					CloseTime: open.Add(spotKlineInterval(tc.interval)).UnixMilli() - 1,
				}}
			}
			if err := verifySpotKlineRows(makeRow(tc.fresh), "BTCUSDT", tc.interval, now); err != nil {
				t.Fatalf("fresh Spot candle was rejected: %v", err)
			}
			if err := verifySpotKlineRows(makeRow(tc.stale), "BTCUSDT", tc.interval, now); err == nil {
				t.Fatalf("stale %s candle accepted after %s", tc.interval, tc.stale)
			}
		})
	}
	rows := []*futures.Kline{{
		Open: "100", High: "101", Low: "99", Close: "100",
		OpenTime: now.Add(-time.Minute).UnixMilli(), CloseTime: 1,
	}}
	if err := verifySpotKlineRows(rows, "BTCUSDT", "1m", now); err == nil {
		t.Fatal("Kline CloseTime before OpenTime accepted")
	}
}

// Review F6: MA5 needs five source bars, even if an implementation
// accidentally returned one output point. Partial/malformed indicator
// vectors cannot substitute for a complete tradable signal.
func TestReview0671SpotIndicatorSampleCompleteness(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value interface{}
		valid bool
	}{
		{"MA output", ConfigData{Data: []float64{1}}, true},
		{"empty MA", ConfigData{Data: []float64{}}, false},
		{"partial bands", ConfigData{High: []float64{1}}, false},
		{"full bands", ConfigData{High: []float64{1}, Mid: []float64{2}, Low: []float64{3}}, true},
		{"partial MACD", MACDConfigData{DIF: []float64{1}}, false},
		{"full MACD", MACDConfigData{DIF: []float64{1}, DEA: []float64{1}, Histogram: []float64{1}}, true},
		{"partial KDJ", KDJConfigData{K: []float64{1}, D: []float64{1}}, false},
		{"invalid NaN", ConfigData{Data: []float64{math.NaN()}}, false},
		{"mismatched vectors", ADXConfigData{ADX: []float64{1}, PlusDI: []float64{1, 2}, MinusDI: []float64{3}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasSpotIndicatorSamples(tc.value); got != tc.valid {
				t.Fatalf("%s sample validity=%t expected=%t", tc.name, got, tc.valid)
			}
		})
	}
	now := time.Now()
	coin := spotEnvTestCoin(now) // MA5
	onlyFourBars := func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		full, err := spotEnvTestKlines(now)(symbol, interval, limit)
		if err != nil {
			return nil, err
		}
		return full[:4], nil
	}
	env, err := buildVerifiedSpotRuleEnv(coin, now, "", onlyFourBars)
	if err == nil || env != nil {
		t.Fatalf("MA5 accepted four Spot Klines: env=%v error=%v", env != nil, err)
	}
}
