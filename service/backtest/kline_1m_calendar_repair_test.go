package backtest

import (
	"context"
	"strings"
	"testing"
	"time"

	"go_binance_futures/service/historicalmarket"
)

type calendarRepairTestSource struct {
	bars []historicalmarket.Kline
}

func (s *calendarRepairTestSource) EarliestKline(_ context.Context, _, _, _ string) (historicalmarket.Kline, error) {
	return s.bars[0], nil
}
func (s *calendarRepairTestSource) Klines(_ context.Context, _, _, _ string, start, end int64) ([]historicalmarket.Kline, error) {
	out := make([]historicalmarket.Kline, 0, len(s.bars))
	for _, row := range s.bars {
		if row.OpenTime >= start && row.CloseTime <= end {
			out = append(out, row)
		}
	}
	return out, nil
}
func (s *calendarRepairTestSource) Funding(context.Context, string, string, int64, int64) ([]historicalmarket.FundingRate, error) {
	return nil, nil
}

func TestKline1mRepairWithoutStrategyTemplate(t *testing.T) {
	setupBacktestStoreTest(t)
	now := time.Now().UTC().Truncate(time.Minute)
	first := now.Add(-8 * time.Minute)
	if first.Month() != now.Month() {
		t.Skip("test near UTC month boundary")
	}
	bars := make([]historicalmarket.Kline, 8)
	for i := range bars {
		at := first.Add(time.Duration(i) * time.Minute)
		bars[i] = historicalmarket.Kline{
			Market:    historicalmarket.MarketFuturesUSDT,
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  at.UnixMilli(),
			CloseTime: at.Add(time.Minute - time.Millisecond).UnixMilli(),
			Open:      100, High: 101, Low: 99, Close: 100,
		}
	}
	repo := historicalmarket.NewRepository(&calendarRepairTestSource{bars: bars})
	manager := NewManager(DatasetBuilder{Repository: repo})
	month := now.Format("2006-01")
	future := now.AddDate(0, 1, 0).Format("2006-01")
	if _, err := manager.StartKline1mRepair(Kline1mRepairRequest{Symbol: "DOGEUSDT", From: future, To: future}); err == nil {
		t.Fatal("future month must be rejected")
	}
	job, err := manager.StartKline1mRepair(Kline1mRepairRequest{Symbol: "btcusdt", From: month, To: month})
	if err != nil {
		t.Fatal(err)
	}
	if job.Symbol != "BTCUSDT" || len(job.Intervals) != 1 || job.Intervals[0] != "1m" || job.StrategyTemplateID != 0 {
		t.Fatalf("repair should be independent of strategy templates: %+v", job)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		item, err := manager.GetPrefetch(job.JobID)
		if err != nil {
			t.Fatal(err)
		}
		if item.Status == "succeeded" {
			break
		}
		if item.Status == "failed" {
			t.Fatalf("1m repair failed: %s", item.Error)
		}
		if time.Now().After(deadline) {
			t.Fatalf("1m repair did not complete: %+v", item)
		}
		time.Sleep(25 * time.Millisecond)
	}
	completed, count, err := repo.KlineRangeComplete(context.Background(),
		historicalmarket.MarketFuturesUSDT, "BTCUSDT", "1m", first.UnixMilli(), now.Add(-time.Millisecond).UnixMilli())
	if err != nil || !completed || count != len(bars) {
		t.Fatalf("repaired data incomplete: complete=%v count=%d err=%v", completed, count, err)
	}
}

func TestKline1mRepairRejectsBadInputs(t *testing.T) {
	manager := NewManager(DatasetBuilder{})
	future := time.Now().UTC().AddDate(0, 1, 0).Format("2006-01")
	for _, request := range []Kline1mRepairRequest{
		{Symbol: "BTCUSDT", From: "", To: ""},
		{Symbol: "BTCUSDT", From: future, To: future},
		{Symbol: "BTC/USDT", From: "2025-01", To: "2025-02"},
		{Symbol: "BTCUSDT", From: "2025-02", To: "2025-01"},
		{Symbol: "ETH", From: "2025-01", To: "2025-02"},
	} {
		if _, err := manager.StartKline1mRepair(request); err == nil || strings.TrimSpace(err.Error()) == "" {
			t.Fatalf("unexpectedly accepted invalid repair request: %+v", request)
		}
	}
}
