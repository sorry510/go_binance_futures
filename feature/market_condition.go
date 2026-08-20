package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"go_binance_futures/llm"
	"go_binance_futures/models"
	markettypes "go_binance_futures/types"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const (
	marketConditionLLMTimeout = 2 * time.Minute
	marketConditionListLimit  = 8
)

var marketConditionUpdateMu sync.Mutex

type MarketConditionResult struct {
	MarketCondition int     `json:"marketCondition"`
	Name            string  `json:"name"`
	Source          string  `json:"source"`
	Confidence      float64 `json:"confidence,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

type MarketConditionProgress struct {
	Progress int    `json:"progress"`
	Stage    string `json:"stage"`
}

type MarketConditionProgressCallback func(MarketConditionProgress)

type marketConditionSnapshot struct {
	AsOf                      string                  `json:"as_of"`
	SymbolCount               int                     `json:"symbol_count"`
	AdvancingCount            int                     `json:"advancing_count"`
	DecliningCount            int                     `json:"declining_count"`
	UnchangedCount            int                     `json:"unchanged_count"`
	AdvancingRatio            float64                 `json:"advancing_ratio"`
	DecliningRatio            float64                 `json:"declining_ratio"`
	AverageChange             float64                 `json:"average_change_pct"`
	MedianChange              float64                 `json:"median_change_pct"`
	ChangeStdDev              float64                 `json:"change_std_dev"`
	AverageRange              float64                 `json:"average_range_pct"`
	MajorWeightedChange       float64                 `json:"major_weighted_change_pct"`
	QuoteVolumeWeightedChange float64                 `json:"quote_volume_weighted_change_pct"`
	MajorSymbols              []marketConditionSymbol `json:"major_symbols"`
	TopGainers                []marketConditionSymbol `json:"top_gainers"`
	TopLosers                 []marketConditionSymbol `json:"top_losers"`
}

type marketConditionSymbol struct {
	Symbol        string  `json:"symbol"`
	PercentChange float64 `json:"percent_change_pct"`
	Range         float64 `json:"range_pct,omitempty"`
	QuoteVolume   float64 `json:"quote_volume,omitempty"`
}

type marketConditionLLMResponse struct {
	MarketCondition int     `json:"market_condition"`
	Confidence      float64 `json:"confidence"`
	Reason          string  `json:"reason"`
}

// UpdateMarketCondition updates the current market condition with LLM analysis when available.
func UpdateMarketCondition(systemConfig *models.Config) (MarketConditionResult, error) {
	return UpdateMarketConditionWithProgress(systemConfig, nil)
}

// UpdateMarketConditionWithProgress reports coarse-grained stages without exposing model output.
func UpdateMarketConditionWithProgress(systemConfig *models.Config, progressCallback MarketConditionProgressCallback) (MarketConditionResult, error) {
	reportMarketConditionProgress(progressCallback, 5, "waiting")
	marketConditionUpdateMu.Lock()
	defer marketConditionUpdateMu.Unlock()

	result := newMarketConditionResult(systemConfig.MarketCondition, "manual")
	if systemConfig.MarketConditionIsAuto == 0 {
		result.Reason = "当前为手动模式"
		reportMarketConditionProgress(progressCallback, 100, "completed")
		return result, nil
	}

	reportMarketConditionProgress(progressCallback, 15, "loading_market_data")
	symbols, err := loadMarketConditionSymbols()
	if err != nil {
		return result, err
	}
	reportMarketConditionProgress(progressCallback, 30, "calculating_fallback")
	fallbackCondition, err := calculateLegacyMarketCondition(symbols)
	if err != nil {
		return result, err
	}

	result = newMarketConditionResult(fallbackCondition, "algorithm")
	result.Reason = "使用本地行情加权算法分析"

	if isLLMConfigured() {
		reportMarketConditionProgress(progressCallback, 40, "preparing_llm")
		snapshot := buildMarketConditionSnapshot(symbols)
		aiResult, aiErr := analyzeMarketConditionWithLLM(snapshot, progressCallback)
		if aiErr != nil {
			logs.Warning("market condition LLM analysis unavailable, use algorithm fallback:", aiErr)
			result.Reason = "LLM 分析不可用，已使用本地行情加权算法"
			reportMarketConditionProgress(progressCallback, 85, "using_fallback")
		} else {
			result = aiResult
		}
	} else {
		reportMarketConditionProgress(progressCallback, 85, "using_fallback")
	}

	reportMarketConditionProgress(progressCallback, 92, "saving")
	if result.MarketCondition != systemConfig.MarketCondition {
		if _, err := orm.NewOrm().QueryTable("config").Filter("id", systemConfig.ID).Update(orm.Params{
			"market_condition": result.MarketCondition,
		}); err != nil {
			return result, fmt.Errorf("update market condition: %w", err)
		}
		systemConfig.MarketCondition = result.MarketCondition
	}

	logs.Info("market condition updated: condition=%d name=%s source=%s confidence=%.2f reason=%s",
		result.MarketCondition,
		result.Name,
		result.Source,
		result.Confidence,
		sanitizeMarketConditionReason(result.Reason),
	)
	reportMarketConditionProgress(progressCallback, 100, "completed")
	return result, nil
}

func loadMarketConditionSymbols() ([]models.Symbols, error) {
	var symbols []models.Symbols
	_, err := orm.NewOrm().QueryTable("symbols").Filter("type", "USDT").All(&symbols)
	if err != nil {
		return nil, fmt.Errorf("load market condition symbols: %w", err)
	}
	if len(symbols) <= 4 {
		return nil, fmt.Errorf("market condition requires more than four USDT symbols")
	}
	return symbols, nil
}

func calculateLegacyMarketCondition(symbols []models.Symbols) (int, error) {
	otherLengths := len(symbols) - 4
	if otherLengths <= 0 {
		return 0, fmt.Errorf("market condition requires more than four symbols")
	}

	otherWeight := 0.35 / float64(otherLengths)
	var weightedSum float64
	for _, symbol := range symbols {
		switch symbol.Symbol {
		case "BTCUSDT":
			weightedSum += symbol.PercentChange * 0.35
		case "ETHUSDT":
			weightedSum += symbol.PercentChange * 0.2
		case "SOLUSDT", "BNBUSDT":
			weightedSum += symbol.PercentChange * 0.05
		default:
			weightedSum += symbol.PercentChange * otherWeight
		}
	}

	m := math.Tanh(weightedSum)
	switch {
	case m >= 0.45:
		return markettypes.MarketConditionStrongBull, nil
	case m >= 0.2:
		return markettypes.MarketConditionBull, nil
	case m >= -0.2:
		return markettypes.MarketConditionSideways, nil
	case m >= -0.45:
		return markettypes.MarketConditionBear, nil
	default:
		return markettypes.MarketConditionStrongBear, nil
	}
}

func buildMarketConditionSnapshot(symbols []models.Symbols) marketConditionSnapshot {
	items := make([]marketConditionSymbol, 0, len(symbols))
	changes := make([]float64, 0, len(symbols))
	majorWeights := map[string]float64{
		"BTCUSDT": 0.5,
		"ETHUSDT": 0.3,
		"SOLUSDT": 0.1,
		"BNBUSDT": 0.1,
	}

	var advancingCount int
	var decliningCount int
	var unchangedCount int
	var rangeSum float64
	var rangeCount int
	var majorWeightedChange float64
	var majorWeightSum float64
	var volumeWeightedChange float64
	var quoteVolumeSum float64

	for _, symbol := range symbols {
		rangePercent := symbolRangePercent(symbol)
		item := marketConditionSymbol{
			Symbol:        symbol.Symbol,
			PercentChange: symbol.PercentChange,
			Range:         rangePercent,
			QuoteVolume:   symbol.QuoteVolume,
		}
		items = append(items, item)
		changes = append(changes, symbol.PercentChange)

		switch {
		case symbol.PercentChange > 0:
			advancingCount++
		case symbol.PercentChange < 0:
			decliningCount++
		default:
			unchangedCount++
		}
		if rangePercent > 0 {
			rangeSum += rangePercent
			rangeCount++
		}
		if weight, exists := majorWeights[symbol.Symbol]; exists {
			majorWeightedChange += symbol.PercentChange * weight
			majorWeightSum += weight
		}
		if symbol.QuoteVolume > 0 {
			volumeWeightedChange += symbol.PercentChange * symbol.QuoteVolume
			quoteVolumeSum += symbol.QuoteVolume
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].PercentChange > items[j].PercentChange
	})
	sortedChanges := append([]float64(nil), changes...)
	sort.Float64s(sortedChanges)
	averageChange := averageFloat64(changes)

	snapshot := marketConditionSnapshot{
		AsOf:           time.Now().UTC().Format(time.RFC3339),
		SymbolCount:    len(symbols),
		AdvancingCount: advancingCount,
		DecliningCount: decliningCount,
		UnchangedCount: unchangedCount,
		AdvancingRatio: float64(advancingCount) / float64(len(symbols)),
		DecliningRatio: float64(decliningCount) / float64(len(symbols)),
		AverageChange:  averageChange,
		MedianChange:   medianFloat64(sortedChanges),
		ChangeStdDev:   standardDeviation(changes, averageChange),
		MajorSymbols:   filterMajorSymbols(items, majorWeights),
		TopGainers:     copyMarketConditionSymbols(items, 0, marketConditionListLimit),
		TopLosers:      copyMarketConditionSymbols(items, len(items)-marketConditionListLimit, len(items)),
	}
	if rangeCount > 0 {
		snapshot.AverageRange = rangeSum / float64(rangeCount)
	}
	if majorWeightSum > 0 {
		snapshot.MajorWeightedChange = majorWeightedChange / majorWeightSum
	}
	if quoteVolumeSum > 0 {
		snapshot.QuoteVolumeWeightedChange = volumeWeightedChange / quoteVolumeSum
	}
	return snapshot
}

func analyzeMarketConditionWithLLM(snapshot marketConditionSnapshot, progressCallback MarketConditionProgressCallback) (MarketConditionResult, error) {
	client, err := llm.NewFromConfig()
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("initialize LLM client: %w", err)
	}

	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("encode market snapshot: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), marketConditionLLMTimeout)
	defer cancel()
	reportMarketConditionProgress(progressCallback, 55, "calling_llm")
	response, err := client.Generate(ctx, llm.Request{
		System: "You are a cryptocurrency futures market-regime classifier. Analyze only the supplied snapshot. Return exactly one JSON object without Markdown, trading advice, positions, leverage, or orders.",
		Messages: []llm.Message{
			{
				Role:    llm.RoleUser,
				Content: buildMarketConditionPrompt(string(snapshotJSON)),
			},
		},
	})
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("generate market condition analysis: %w", err)
	}

	reportMarketConditionProgress(progressCallback, 80, "validating_llm")
	analysis, err := parseMarketConditionLLMResponse(response.Content)
	if err != nil {
		return MarketConditionResult{}, err
	}
	return MarketConditionResult{
		MarketCondition: analysis.MarketCondition,
		Name:            markettypes.MarketConditionName(analysis.MarketCondition),
		Source:          "llm",
		Confidence:      analysis.Confidence,
		Reason:          sanitizeMarketConditionReason(analysis.Reason),
	}, nil
}

func reportMarketConditionProgress(callback MarketConditionProgressCallback, progress int, stage string) {
	if callback == nil {
		return
	}
	callback(MarketConditionProgress{Progress: progress, Stage: stage})
}

func buildMarketConditionPrompt(snapshotJSON string) string {
	return `Classify the market into exactly one condition:
1 Strong bull: majors and the broad market rise strongly together.
2 Bull: directional bias is positive but not broad or strong enough for 1 or 8.
3 Sideways: no clear direction and no special regime below dominates.
4 Bear: directional bias is negative but not broad or strong enough for 5 or 9.
5 Strong bear: majors and the broad market fall strongly together.
6 Bullish divergence: majors are weak or negative while broad-market breadth is resilient or positive.
7 Bearish divergence: majors are positive while broad-market breadth is weak or negative.
8 Broad rise: advancing breadth is exceptionally high, while major direction is not strong enough for 1.
9 Broad decline: declining breadth is exceptionally high, while major direction is not strong enough for 5.
10 High-volatility sideways: dispersion or intraday range is high without a reliable direction.
11 Low-volatility consolidation: direction, dispersion, and intraday range are all subdued.

Prefer 6-11 only when the special regime is clearly supported; otherwise use 1-5.
Return this schema exactly:
{"market_condition":1,"confidence":0.0,"reason":"Chinese explanation within 200 characters"}
confidence must be between 0 and 1. Do not include any other keys.

Snapshot:
` + snapshotJSON
}

func parseMarketConditionLLMResponse(content string) (marketConditionLLMResponse, error) {
	trimmed := strings.TrimSpace(content)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return marketConditionLLMResponse{}, fmt.Errorf("LLM market condition response is not a JSON object")
	}

	var response marketConditionLLMResponse
	if err := json.Unmarshal([]byte(trimmed[start:end+1]), &response); err != nil {
		return marketConditionLLMResponse{}, fmt.Errorf("decode LLM market condition response: %w", err)
	}
	if !markettypes.IsValidMarketCondition(response.MarketCondition) {
		return marketConditionLLMResponse{}, fmt.Errorf("LLM returned unsupported market condition %d", response.MarketCondition)
	}
	if response.Confidence < 0 || response.Confidence > 1 {
		return marketConditionLLMResponse{}, fmt.Errorf("LLM returned invalid confidence %.4f", response.Confidence)
	}
	if strings.TrimSpace(response.Reason) == "" {
		return marketConditionLLMResponse{}, fmt.Errorf("LLM returned an empty market condition reason")
	}
	return response, nil
}

func isLLMConfigured() bool {
	cfg, err := llm.LoadConfig()
	return err == nil && strings.TrimSpace(cfg.APIKey) != ""
}

func newMarketConditionResult(condition int, source string) MarketConditionResult {
	return MarketConditionResult{
		MarketCondition: condition,
		Name:            markettypes.MarketConditionName(condition),
		Source:          source,
	}
}

func symbolRangePercent(symbol models.Symbols) float64 {
	openPrice, openErr := strconv.ParseFloat(symbol.Open, 64)
	highPrice, highErr := strconv.ParseFloat(symbol.High, 64)
	lowPrice, lowErr := strconv.ParseFloat(symbol.Low, 64)
	if openErr != nil || highErr != nil || lowErr != nil || openPrice <= 0 || highPrice < lowPrice {
		return 0
	}
	return (highPrice - lowPrice) / openPrice * 100
}

func filterMajorSymbols(items []marketConditionSymbol, weights map[string]float64) []marketConditionSymbol {
	result := make([]marketConditionSymbol, 0, len(weights))
	for _, item := range items {
		if _, exists := weights[item.Symbol]; exists {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Symbol < result[j].Symbol
	})
	return result
}

func copyMarketConditionSymbols(items []marketConditionSymbol, start int, end int) []marketConditionSymbol {
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	if start >= end {
		return []marketConditionSymbol{}
	}
	return append([]marketConditionSymbol(nil), items[start:end]...)
}

func averageFloat64(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}

func medianFloat64(sortedValues []float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	middle := len(sortedValues) / 2
	if len(sortedValues)%2 == 0 {
		return (sortedValues[middle-1] + sortedValues[middle]) / 2
	}
	return sortedValues[middle]
}

func standardDeviation(values []float64, average float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var squaredDifferenceSum float64
	for _, value := range values {
		difference := value - average
		squaredDifferenceSum += difference * difference
	}
	return math.Sqrt(squaredDifferenceSum / float64(len(values)))
}

func sanitizeMarketConditionReason(reason string) string {
	reason = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(reason, "\r", " "), "\n", " "))
	const maxReasonRunes = 200
	runes := []rune(reason)
	if len(runes) > maxReasonRunes {
		return string(runes[:maxReasonRunes])
	}
	return reason
}
