package symbolanalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go_binance_futures/models"
	liquidationservice "go_binance_futures/service/liquidation"
	marketservice "go_binance_futures/service/market"

	"github.com/adshao/go-binance/v2/futures"
)

func TestBuildAggregatesContext(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	deps := testDependencies(now)
	result, err := Build(context.Background(), "ongusdt", deps)
	if err != nil {
		t.Fatal(err)
	}
	if result.Symbol != "ONGUSDT" || result.Snapshot.Price != 103 || result.MarketCondition == nil || result.MarketCondition.MarketCondition != 2 {
		t.Fatalf("unexpected base context: %+v", result)
	}
	if len(result.Klines) != 4 || len(result.DataMissing) != 0 {
		t.Fatalf("unexpected kline/missing data: %+v", result)
	}
	if result.OpenInterest == nil || result.OpenInterest.ChangePct <= 0 || result.Taker == nil || result.Depth == nil || result.Liquidations == nil {
		t.Fatalf("recommended data missing: %+v", result)
	}
	if result.Liquidations.LongNotional != 100 || result.Liquidations.ShortNotional != 50 {
		t.Fatalf("unexpected liquidation summary: %+v", result.Liquidations)
	}
}
func TestBuildRecordsPartialFailures(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	deps := testDependencies(now)
	originalKlines := deps.GetKlines
	deps.GetKlines = func(ctx context.Context, symbol, interval string, limit int) ([]*futures.Kline, error) {
		if interval == "15m" {
			return nil, fmt.Errorf("kline unavailable")
		}
		return originalKlines(ctx, symbol, interval, limit)
	}
	deps.GetFundingRate = func(context.Context, string) ([]*futures.PremiumIndex, error) {
		return nil, fmt.Errorf("funding unavailable")
	}
	deps.GetOpenInterestStatistics = func(context.Context, string, string, int) ([]*futures.OpenInterestStatistic, error) {
		return nil, fmt.Errorf("oi history unavailable")
	}
	deps.GetTakerLongShortRatio = func(context.Context, string, string, int) ([]*futures.TakerLongShortRatio, error) {
		return nil, fmt.Errorf("taker unavailable")
	}
	deps.GetDepth = func(context.Context, string, int) (*futures.DepthResponse, error) {
		return nil, fmt.Errorf("depth unavailable")
	}

	result, err := Build(context.Background(), "ONGUSDT", deps)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"kline_15m", "funding_rate", "open_interest_change", "taker_ratio", "depth"} {
		if !contains(result.DataMissing, expected) {
			t.Fatalf("missing %q in data_missing: %v", expected, result.DataMissing)
		}
	}
	if contains(result.DataMissing, "open_interest") || result.OpenInterest == nil {
		t.Fatalf("current OI should remain available: %+v", result)
	}
}

func TestBuildRejectsNonUSDTContract(t *testing.T) {
	if _, err := Build(context.Background(), "BTCUSDC", testDependencies(time.Now())); err == nil {
		t.Fatal("expected non-USDT symbol to fail")
	}
}
func testDependencies(now time.Time) Dependencies {
	return Dependencies{
		GetSymbol: func(context.Context, string) (models.Symbols, error) {
			return models.Symbols{Symbol: "ONGUSDT", Close: "103", Open: "100", High: "110", Low: "95", PercentChange: 3, QuoteVolume: 1_000_000, TradeCount: 12000, UpdateTime: now.Add(-time.Second).UnixMilli()}, nil
		},
		GetMarketCondition: func(context.Context) (marketservice.Condition, error) {
			return marketservice.Condition{MarketCondition: 2, Name: "偏多头", Auto: true}, nil
		},
		GetKlines: func(context.Context, string, string, int) ([]*futures.Kline, error) {
			return testKlines(), nil
		},
		GetFundingRate: func(context.Context, string) ([]*futures.PremiumIndex, error) {
			return []*futures.PremiumIndex{{Symbol: "ONGUSDT", MarkPrice: "103.1", LastFundingRate: "0.0001", NextFundingTime: now.Add(time.Hour).UnixMilli()}}, nil
		},
		GetOpenInterest: func(context.Context, string) (*futures.OpenInterest, error) {
			return &futures.OpenInterest{Symbol: "ONGUSDT", OpenInterest: "1000", Time: now.UnixMilli()}, nil
		},
		GetOpenInterestStatistics: func(context.Context, string, string, int) ([]*futures.OpenInterestStatistic, error) {
			return []*futures.OpenInterestStatistic{{SumOpenInterest: "900", Timestamp: now.Add(-25 * time.Minute).UnixMilli()}, {SumOpenInterest: "1000", Timestamp: now.UnixMilli()}}, nil
		},
		GetTakerLongShortRatio: func(context.Context, string, string, int) ([]*futures.TakerLongShortRatio, error) {
			return []*futures.TakerLongShortRatio{{BuySellRatio: "1.5", BuyVol: "600", SellVol: "400", Timestamp: uint64(now.UnixMilli())}}, nil
		},
		GetDepth: func(context.Context, string, int) (*futures.DepthResponse, error) {
			return &futures.DepthResponse{Bids: []futures.Bid{{Price: "103", Quantity: "10"}}, Asks: []futures.Ask{{Price: "104", Quantity: "5"}}}, nil
		},
		ListLiquidations: func(context.Context, liquidationservice.ListOptions) (liquidationservice.ListResult, error) {
			return liquidationservice.ListResult{List: []models.FuturesLiquidationOrder{{Symbol: "ONGUSDT", Side: "SELL", Notional: 100}, {Symbol: "ONGUSDT", Side: "BUY", Notional: 50}}}, nil
		},
		Now: func() time.Time { return now },
	}
}

func testKlines() []*futures.Kline {
	values := []float64{104, 103, 102, 101, 100}
	result := make([]*futures.Kline, 0, len(values))
	for index, value := range values {
		result = append(result, &futures.Kline{
			OpenTime: int64(5-index) * 60_000,
			Open:     fmt.Sprintf("%.1f", value-0.5), High: fmt.Sprintf("%.1f", value+1), Low: fmt.Sprintf("%.1f", value-1), Close: fmt.Sprintf("%.1f", value),
			QuoteAssetVolume: "1000", TakerBuyQuoteAssetVolume: "600",
		})
	}
	return result
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestBuildIncludesRecentSuccessfulAnalyses(t *testing.T) {
	now := time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)
	deps := testDependencies(now)
	deps.ListHistory = func(context.Context, HistoryListOptions) (HistoryListResult, error) {
		return HistoryListResult{List: []HistoryItem{{
			SymbolAnalysisHistory: models.SymbolAnalysisHistory{
				TaskID: "old-task", Symbol: "ONGUSDT", Status: "succeeded",
				Direction: "long", Confidence: 0.7, MarketCondition: 2,
				AnalysisPrice: 100, Summary: "此前偏多", CreatedAt: now.Add(-time.Hour).UnixMilli(),
			},
			Result:       json.RawMessage(`{"version":"trading_plan_v1","direction":"long"}`),
			CurrentPrice: 103, PriceChangePct: 3,
		}}}, nil
	}
	result, err := Build(context.Background(), "ONGUSDT", deps)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.PreviousAnalyses) != 1 || result.PreviousAnalyses[0].TaskID != "old-task" {
		t.Fatalf("unexpected previous analyses: %+v", result.PreviousAnalyses)
	}
	if result.PreviousAnalyses[0].PriceChangePct != 3 || len(result.PreviousAnalyses[0].Plan) == 0 {
		t.Fatalf("previous analysis comparison missing: %+v", result.PreviousAnalyses[0])
	}
}
