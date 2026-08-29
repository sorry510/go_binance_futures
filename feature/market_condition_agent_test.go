package feature

import (
	"context"
	"strings"
	"testing"

	"go_binance_futures/llm"
	marketservice "go_binance_futures/service/market"
)

type marketRegimeFakeClient struct {
	responses []string
	requests  []llm.Request
}

func (client *marketRegimeFakeClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (client *marketRegimeFakeClient) Generate(_ context.Context, request llm.Request) (*llm.Response, error) {
	client.requests = append(client.requests, request)
	index := len(client.requests) - 1
	if index >= len(client.responses) {
		return &llm.Response{Content: `{"action":"error","error":"unexpected call"}`}, nil
	}
	return &llm.Response{Model: "fake", Content: client.responses[index]}, nil
}

func testRegimeSnapshot() marketservice.RegimeSnapshot {
	return marketservice.RegimeSnapshot{
		SymbolCount: 20, AdvancingCount: 14, DecliningCount: 6,
		AdvancingRatio: 0.7, DecliningRatio: 0.3,
		MajorWeightedChange: 1.2, AverageChange: 0.8,
	}
}

func TestMarketRegimeRuntimeReturnsCompatibleResult(t *testing.T) {
	client := &marketRegimeFakeClient{responses: []string{
		`{"action":"final","summary":"偏多","result":{"market_condition":2,"confidence":0.86,"reason":"主流币与市场广度同步偏强"}}`,
	}}
	result, err := analyzeMarketConditionWithRuntimeClient(testRegimeSnapshot(), client, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.MarketCondition != 2 || result.Name != "偏多头" || result.Source != "llm" || result.Confidence != 0.86 {
		t.Fatalf("unexpected result: %+v", result)
	}
}
func TestMarketRegimeRuntimeRepairsInvalidFinal(t *testing.T) {
	client := &marketRegimeFakeClient{responses: []string{
		`{"action":"final","result":{"market_condition":99,"confidence":0.5,"reason":"bad"}}`,
		`{"action":"final","result":{"market_condition":3,"confidence":0.72,"reason":"方向不明确，市场震荡"}}`,
	}}
	result, err := analyzeMarketConditionWithRuntimeClient(testRegimeSnapshot(), client, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.MarketCondition != 3 || len(client.requests) != 2 {
		t.Fatalf("unexpected repaired result=%+v calls=%d", result, len(client.requests))
	}
	second := client.requests[1]
	if len(second.Messages) == 0 || !strings.Contains(second.Messages[len(second.Messages)-1].Content, "AGENT_FEEDBACK") {
		t.Fatalf("repair feedback missing: %+v", second.Messages)
	}
}

func TestMarketRegimeRuntimeFailsAfterInvalidRounds(t *testing.T) {
	client := &marketRegimeFakeClient{responses: []string{
		`{"action":"final","result":{"market_condition":99,"confidence":0.5,"reason":"bad"}}`,
		`{"action":"final","result":{"market_condition":99,"confidence":0.5,"reason":"bad"}}`,
	}}
	_, err := analyzeMarketConditionWithRuntimeClient(testRegimeSnapshot(), client, nil)
	if err == nil || !strings.Contains(err.Error(), "maximum 2 rounds") {
		t.Fatalf("unexpected error: %v", err)
	}
}
