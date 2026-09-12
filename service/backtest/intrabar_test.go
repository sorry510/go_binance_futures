package backtest

import (
	"context"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"
)

type intrabarFixtureProvider struct {
	seconds     []historicalmarket.Kline
	trades      []historicalmarket.PublicDataTrade
	secondCalls int
	tradeCalls  int
}

func (provider *intrabarFixtureProvider) MinuteBars(context.Context, string, string, string, int64, int64) ([]historicalmarket.Kline, historicalmarket.ResolutionEvidence, error) {
	return nil, historicalmarket.ResolutionEvidence{}, nil
}
func (provider *intrabarFixtureProvider) SecondBars(context.Context, string, string, int64, int64) ([]historicalmarket.Kline, historicalmarket.ResolutionEvidence, error) {
	provider.secondCalls++
	return append([]historicalmarket.Kline(nil), provider.seconds...), historicalmarket.ResolutionEvidence{Resolution: "1s", EvidenceHash: "seconds"}, nil
}
func (provider *intrabarFixtureProvider) Trades(context.Context, string, string, int64, int64) ([]historicalmarket.PublicDataTrade, historicalmarket.ResolutionEvidence, error) {
	provider.tradeCalls++
	return append([]historicalmarket.PublicDataTrade(nil), provider.trades...), historicalmarket.ResolutionEvidence{Resolution: "trades", EvidenceHash: "trades"}, nil
}
func (*intrabarFixtureProvider) Stats() historicalmarket.ResolutionStats {
	return historicalmarket.ResolutionStats{}
}
func (*intrabarFixtureProvider) Close() error { return nil }

func intrabarMinute() historicalmarket.Kline {
	start := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC).UnixMilli()
	return historicalmarket.Kline{Market: historicalmarket.MarketFuturesUSDT, Symbol: "BTCUSDT", Interval: "1m", OpenTime: start, CloseTime: start + 59_999, Open: 100, High: 110, Low: 90, Close: 100}
}

func TestIntrabarResolverSingleKnownEventStaysAtOneMinute(t *testing.T) {
	minute := intrabarMinute()
	event := PriceEvent{ID: "tp", Type: "take_profit", TriggerPrice: 108, Direction: PriceDirectionGTE, KnownAt: minute.OpenTime - 1, Priority: 10}
	decision, err := (IntrabarResolver{}).Resolve(context.Background(), minute, []PriceEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Resolved || decision.Event.ID != "tp" || decision.Resolution != "1m" || decision.HitPrice != 108 {
		t.Fatalf("single event should resolve at 1m: %+v", decision)
	}
}

func TestIntrabarResolverUsesSecondsForMinuteConflict(t *testing.T) {
	minute := intrabarMinute()
	provider := &intrabarFixtureProvider{seconds: []historicalmarket.Kline{
		{Market: minute.Market, Symbol: minute.Symbol, Interval: "1s", OpenTime: minute.OpenTime, CloseTime: minute.OpenTime + 999, Open: 100, High: 101, Low: 91, Close: 92},
		{Market: minute.Market, Symbol: minute.Symbol, Interval: "1s", OpenTime: minute.OpenTime + 1000, CloseTime: minute.OpenTime + 1999, Open: 92, High: 109, Low: 92, Close: 108},
	}}
	events := []PriceEvent{
		{ID: "tp", Type: "take_profit", TriggerPrice: 108, Direction: PriceDirectionGTE, KnownAt: minute.OpenTime - 1, Priority: 10},
		{ID: "sl", Type: "stop_loss", TriggerPrice: 92, Direction: PriceDirectionLTE, KnownAt: minute.OpenTime - 1, Priority: 10},
	}
	decision, err := (IntrabarResolver{Provider: provider}).Resolve(context.Background(), minute, events)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Resolved || decision.Event.ID != "sl" || decision.Resolution != "1s" {
		t.Fatalf("second sequence did not resolve first event: %+v", decision)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 0 {
		t.Fatalf("unexpected resolution depth: second=%d trades=%d", provider.secondCalls, provider.tradeCalls)
	}
}

func TestIntrabarResolverUsesTradesForSameSecondConflict(t *testing.T) {
	minute := intrabarMinute()
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: minute.Market, Symbol: minute.Symbol, Interval: "1s", OpenTime: minute.OpenTime, CloseTime: minute.OpenTime + 999, Open: 100, High: 109, Low: 91, Close: 100}},
		trades: []historicalmarket.PublicDataTrade{
			{TradeID: 1, TradeTime: minute.OpenTime + 100, Price: 100},
			{TradeID: 2, TradeTime: minute.OpenTime + 300, Price: 91},
			{TradeID: 3, TradeTime: minute.OpenTime + 500, Price: 109},
		},
	}
	events := []PriceEvent{
		{ID: "tp", Type: "take_profit", TriggerPrice: 108, Direction: PriceDirectionGTE, KnownAt: minute.OpenTime - 1, Priority: 10},
		{ID: "sl", Type: "stop_loss", TriggerPrice: 92, Direction: PriceDirectionLTE, KnownAt: minute.OpenTime - 1, Priority: 10},
	}
	decision, err := (IntrabarResolver{Provider: provider}).Resolve(context.Background(), minute, events)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Resolved || decision.Event.ID != "sl" || decision.Resolution != "trades" || decision.HitPrice != 91 || decision.HitTime != minute.OpenTime+300 {
		t.Fatalf("trade path did not resolve first crossing: %+v", decision)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 1 || decision.TradeEvidence == nil {
		t.Fatalf("trade drill-down was not recorded: %+v", decision)
	}
}

func TestIntrabarResolverTradeTieUsesPriority(t *testing.T) {
	minute := intrabarMinute()
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: minute.Market, Symbol: minute.Symbol, Interval: "1s", OpenTime: minute.OpenTime, CloseTime: minute.OpenTime + 999, Open: 100, High: 100, Low: 100, Close: 100}},
		trades:  []historicalmarket.PublicDataTrade{{TradeID: 7, TradeTime: minute.OpenTime + 100, Price: 100}},
	}
	events := []PriceEvent{
		{ID: "gte", Type: "upper", TriggerPrice: 100, Direction: PriceDirectionGTE, KnownAt: minute.OpenTime - 1, Priority: 20},
		{ID: "lte", Type: "lower", TriggerPrice: 100, Direction: PriceDirectionLTE, KnownAt: minute.OpenTime - 1, Priority: 5},
	}
	decision, err := (IntrabarResolver{Provider: provider}).Resolve(context.Background(), minute, events)
	if err != nil {
		t.Fatal(err)
	}
	if !decision.Resolved || decision.Event.ID != "lte" || decision.Resolution != "trades" {
		t.Fatalf("deterministic priority tie-break failed: %+v", decision)
	}
}

func TestIntrabarResolverDoesNotUseTriggerBeforeKnownAt(t *testing.T) {
	minute := intrabarMinute()
	knownAt := minute.OpenTime + 500
	provider := &intrabarFixtureProvider{
		seconds: []historicalmarket.Kline{{Market: minute.Market, Symbol: minute.Symbol, Interval: "1s", OpenTime: minute.OpenTime, CloseTime: minute.OpenTime + 999, Open: 100, High: 109, Low: 100, Close: 101}},
		trades: []historicalmarket.PublicDataTrade{
			{TradeID: 1, TradeTime: minute.OpenTime + 100, Price: 109},
			{TradeID: 2, TradeTime: minute.OpenTime + 600, Price: 101},
		},
	}
	event := PriceEvent{ID: "late", Type: "late_stop", TriggerPrice: 108, Direction: PriceDirectionGTE, KnownAt: knownAt, Priority: 1}
	decision, err := (IntrabarResolver{Provider: provider}).Resolve(context.Background(), minute, []PriceEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Resolved {
		t.Fatalf("price touch before known_at leaked into decision: %+v", decision)
	}
	if provider.secondCalls != 1 || provider.tradeCalls != 1 {
		t.Fatalf("mid-second known_at must use trades: second=%d trades=%d", provider.secondCalls, provider.tradeCalls)
	}
}

func TestIntrabarResolverIgnoresFutureEvent(t *testing.T) {
	minute := intrabarMinute()
	event := PriceEvent{ID: "future", Type: "future", TriggerPrice: 108, Direction: PriceDirectionGTE, KnownAt: minute.CloseTime + 1, Priority: 1}
	decision, err := (IntrabarResolver{}).Resolve(context.Background(), minute, []PriceEvent{event})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Resolved || len(decision.Candidates) != 0 {
		t.Fatalf("future event participated in past minute: %+v", decision)
	}
}

func TestMergeResolutionEvidenceHashIsStableAndEvidenceSensitive(t *testing.T) {
	base := "base-data-hash"
	a := historicalmarket.ResolutionEvidence{EvidenceHash: "aaa"}
	b := historicalmarket.ResolutionEvidence{EvidenceHash: "bbb"}
	first := MergeResolutionEvidenceHash(base, a, b)
	second := MergeResolutionEvidenceHash(base, b, a, a)
	if first != second {
		t.Fatalf("evidence order or duplicates changed data hash: %s != %s", first, second)
	}
	if first == base {
		t.Fatal("high-resolution evidence must extend the canonical data hash")
	}
	changed := MergeResolutionEvidenceHash(base, a, historicalmarket.ResolutionEvidence{EvidenceHash: "ccc"})
	if changed == first {
		t.Fatal("changed archive evidence must change the final data hash")
	}
	if got := MergeResolutionEvidenceHash(base); got != base {
		t.Fatalf("no evidence must preserve V3-3 data hash: %s", got)
	}
}
