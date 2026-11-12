package line

import (
	"errors"
	"strings"
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/service/spotledger"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/adshao/go-binance/v2/futures"
)

func spotRuleTestCoin() models.SpotSymbols {
	return models.SpotSymbols{
		Symbol: "BTCUSDT", Type: "USDT", Profit: "8", Loss: "6",
		Strategy: `[
			{"name":"disabled","type":"long","enable":false,"code":"true"},
			{"name":"not yet","type":"long","enable":true,"code":"NowPrice > 200"},
			{"name":"entry","type":"long","enable":true,"code":"NowPrice > 90 and len(Positions) == 0"},
			{"name":"close","type":"close_long","enable":true,"code":"ROI >= 8 and Position.Amount == 0.8 and Position.EntryPrice == 100 and Position.UnrealizedProfit >= 8"}
		]`,
	}
}

func spotRuleTestSnapshot() spotledger.Snapshot {
	return spotledger.Snapshot{
		FullyPriced: true,
		Position: models.SpotManagedPosition{
			ID: 1, Symbol: "BTCUSDT", Status: "OPEN", BaseAsset: "BTC",
			QuoteAsset: "USDT", RemainingQty: "0.800000000000000000",
			AvgCostQuote: "100.000000000000000000", CreatedAt: 1000,
		},
	}
}

func spotRuleTestEvaluator(price float64) SpotRuleEvaluator {
	return SpotRuleEvaluator{
		LoadEnvironment: func(coin models.SpotSymbols) (map[string]interface{}, error) {
			return map[string]interface{}{
				"NowPrice": price, "NowSymbolClose": price, "NowTime": int64(123),
			}, nil
		},
	}
}

func TestStage0671SpotEntryLongOnlyPriorityAndHash(t *testing.T) {
	coin := spotRuleTestCoin()
	signal, err := spotRuleTestEvaluator(110).EvaluateEntry(coin)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "BUY" || signal.Reason != "long" || signal.RuleName != "entry" ||
		signal.RuleHash != strategyservice.RuleHash("NowPrice > 90 and len(Positions) == 0") {
		t.Fatalf("incorrect Spot-only LONG result: %+v", signal)
	}
	signal, err = spotRuleTestEvaluator(50).EvaluateEntry(coin)
	if err != nil || signal.Action != "" {
		t.Fatalf("false LONG opened a position: %+v %v", signal, err)
	}
	// A configured SELL rule must never generate BUY from the entry stage.
	coin.Strategy = `[{"name":"only exit","type":"close_long","enable":true,"code":"true"}]`
	signal, err = spotRuleTestEvaluator(110).EvaluateEntry(coin)
	if err != nil || signal.Action != "" {
		t.Fatalf("close_long generated a BUY: %+v %v", signal, err)
	}
	coin.Strategy = `[{"name":"bad","type":"short","enable":true,"code":"true"}]`
	if signal, err = spotRuleTestEvaluator(110).EvaluateEntry(coin); err == nil || signal.Action != "" {
		t.Fatalf("Futures SHORT was accepted: %+v %v", signal, err)
	}
}

func TestStage0671SpotExitUsesUnleveragedROIAndVerifiedPosition(t *testing.T) {
	coin := spotRuleTestCoin()
	snapshot := spotRuleTestSnapshot()
	e := spotRuleTestEvaluator(110)
	signal, err := e.evaluateExitSnapshot(coin, snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if signal.Action != "SELL" || signal.Reason != "take_profit" || signal.RuleName != "close" ||
		signal.ROI != "10" {
		t.Fatalf("Spot +10%% exit does not match unleveraged cost: %+v", signal)
	}
	e = spotRuleTestEvaluator(107)
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" || signal.ROI != "7" {
		t.Fatalf("close_long must not bypass configured +8%% threshold: %+v %v", signal, err)
	}
	coin.Strategy = `[{"name":"false close","type":"close_long","enable":true,"code":"false"}]`
	e = spotRuleTestEvaluator(110)
	e.LoadKlines = func(_, _ string, _ int) ([]*futures.Kline, error) {
		t.Fatal("fallback read Spot Kline despite configured close_long")
		return nil, nil
	}
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" {
		t.Fatalf("false configured close rule must suppress fallback: %+v %v", signal, err)
	}
}

func TestStage0671KnownThirdAssetFeeDoesNotBlockSafeExit(t *testing.T) {
	coin := spotRuleTestCoin()
	snapshot := spotRuleTestSnapshot()
	snapshot.FullyPriced = false
	snapshot.UnpricedFees = map[string]string{"BNB": "0.0001"}
	signal, err := spotRuleTestEvaluator(110).evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "SELL" || signal.ROI != "10" {
		t.Fatalf("known, unpriced BNB commission must not trap valid Spot inventory: %+v %v", signal, err)
	}
}

func TestStage0671SpotDefaultCloseMatchesFuturesPriorityButUsesSpotKlines(t *testing.T) {
	coin := spotRuleTestCoin()
	coin.Strategy = `[{"name":"entry","type":"long","enable":true,"code":"NowPrice > 0"}]`
	snapshot := spotRuleTestSnapshot()
	e := spotRuleTestEvaluator(110)
	spotKlineCalls := 0
	e.LoadKlines = func(symbol, interval string, limit int) ([]*futures.Kline, error) {
		spotKlineCalls++
		if symbol != "BTCUSDT" || interval != "5m" || limit != 2 {
			t.Fatalf("unexpected non-Spot fallback: %s %s %d", symbol, interval, limit)
		}
		return []*futures.Kline{{Close: "100"}, {Close: "101"}}, nil
	}
	signal, err := e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "SELL" || signal.Reason != "take_profit" ||
		signal.RuleHash != strategyservice.RuleHash(spotSimpleCloseRuleIdentity) ||
		signal.RuleName != spotSimpleCloseRuleIdentity || spotKlineCalls != 1 {
		t.Fatalf("Spot simple close fallback: %+v %v calls=%d", signal, err, spotKlineCalls)
	}
	// Profit below configured threshold means not even reading the fallback.
	e = spotRuleTestEvaluator(105)
	e.LoadKlines = func(_, _ string, _ int) ([]*futures.Kline, error) {
		t.Fatal("fallback called without ROI threshold")
		return nil, nil
	}
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" {
		t.Fatalf("premature fallback exit: %+v %v", signal, err)
	}
	// Futures simpleCloseStrategy suppresses fallback inside +/-3%% ROI,
	// even if a custom profit threshold is configured smaller.
	coin.Profit = "1"
	e = spotRuleTestEvaluator(102)
	e.LoadKlines = func(_, _ string, _ int) ([]*futures.Kline, error) {
		t.Fatal("simpleCloseStrategy neutral 3% band was bypassed")
		return nil, nil
	}
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" {
		t.Fatalf("neutral ROI must not SELL: %+v %v", signal, err)
	}
	coin.Profit = "8"
	e = spotRuleTestEvaluator(90)
	e.LoadKlines = func(_, _ string, _ int) ([]*futures.Kline, error) {
		return nil, errors.New("Spot Kline unavailable")
	}
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err == nil || signal.Action != "" {
		t.Fatalf("missing market data triggered a SELL: %+v %v", signal, err)
	}
	e.LoadKlines = func(_, _ string, _ int) ([]*futures.Kline, error) {
		return []*futures.Kline{{Close: "101"}, {Close: "100"}}, nil
	}
	signal, err = e.evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" {
		t.Fatalf("fallback trend gate reversed: %+v %v", signal, err)
	}
}

func TestStage0671SpotFailClosedOnInvalidDataOrRule(t *testing.T) {
	coin := spotRuleTestCoin()
	snapshot := spotRuleTestSnapshot()
	for _, tc := range []struct {
		name string
		edit func(*spotledger.Snapshot)
	}{
		{"not owned Spot position", func(s *spotledger.Snapshot) { s.Position.Symbol = "WBTCUSDT" }},
		{"not OPEN", func(s *spotledger.Snapshot) { s.Position.Status = "CLOSED" }},
		{"no Fill inventory", func(s *spotledger.Snapshot) { s.Position.RemainingQty = "0" }},
		{"bad average cost", func(s *spotledger.Snapshot) { s.Position.AvgCostQuote = "0" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := snapshot
			tc.edit(&changed)
			signal, err := spotRuleTestEvaluator(110).evaluateExitSnapshot(coin, changed)
			if err == nil || signal.Action != "" {
				t.Fatalf("unverified Spot position generated SELL: %+v %v", signal, err)
			}
		})
	}
	for _, price := range []float64{0, -1} {
		signal, err := spotRuleTestEvaluator(price).EvaluateEntry(coin)
		if err == nil || signal.Action != "" {
			t.Fatalf("invalid Spot market price generated BUY: %+v %v", signal, err)
		}
	}
	coin.Strategy = `[
		{"name":"bad","type":"long","enable":true,"code":"unknown_ma.Data[0] > 1"},
		{"name":"later","type":"long","enable":true,"code":"true"}
	]`
	signal, err := spotRuleTestEvaluator(110).EvaluateEntry(coin)
	if err == nil || signal.Action != "" || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("unknown indicator silently skipped to later BUY: %+v %v", signal, err)
	}
	coin.Strategy = `[{"name":"syntax","type":"long","enable":true,"code":"NowPrice >"}]`
	signal, err = spotRuleTestEvaluator(110).EvaluateEntry(coin)
	if err == nil || signal.Action != "" {
		t.Fatalf("malformed Spot rule triggered BUY: %+v %v", signal, err)
	}
}

// Review F1: the pre-existing default 100% ROI thresholds do not mean
// close_long alone is an unconditional sell. Preserve Futures semantics
// rather than silently changing the user's configured strategy.
func TestReview0671DefaultROIThresholdIsExplicitPermissionGate(t *testing.T) {
	coin := spotRuleTestCoin()
	coin.Profit = "100"
	coin.Loss = "100"
	coin.Strategy = `[{"name":"allow exit","type":"close_long","enable":true,"code":"true"}]`
	snapshot := spotRuleTestSnapshot()
	signal, err := spotRuleTestEvaluator(110).evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "" || signal.ROI != "10" {
		t.Fatalf("100%% threshold unexpectedly sold at +10%%: %+v %v", signal, err)
	}
	signal, err = spotRuleTestEvaluator(201).evaluateExitSnapshot(coin, snapshot)
	if err != nil || signal.Action != "SELL" || signal.Reason != "take_profit" {
		t.Fatalf("explicit 100%% threshold failed at +101%%: %+v %v", signal, err)
	}
}
