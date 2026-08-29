package feature

import (
	"context"
	"go_binance_futures/llm"
	"go_binance_futures/models"
	marketservice "go_binance_futures/service/market"
	markettypes "go_binance_futures/types"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const marketConditionLLMTimeout = 2 * time.Minute

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
	symbols, err := marketservice.LoadRegimeSymbols(context.Background())
	if err != nil {
		return result, err
	}
	reportMarketConditionProgress(progressCallback, 30, "calculating_fallback")
	fallbackCondition, err := marketservice.CalculateAlgorithmCondition(symbols)
	if err != nil {
		return result, err
	}

	result = newMarketConditionResult(fallbackCondition, "algorithm")
	result.Reason = "使用本地行情加权算法分析"

	if isLLMConfigured() {
		reportMarketConditionProgress(progressCallback, 40, "preparing_llm")
		snapshot := marketservice.BuildRegimeSnapshot(symbols)
		aiResult, aiErr := analyzeMarketConditionWithRuntime(snapshot, progressCallback)
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
		if err := marketservice.SaveMarketCondition(context.Background(), systemConfig.ID, result.MarketCondition); err != nil {
			return result, err
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

func reportMarketConditionProgress(callback MarketConditionProgressCallback, progress int, stage string) {
	if callback == nil {
		return
	}
	callback(MarketConditionProgress{Progress: progress, Stage: stage})
}

func isLLMConfigured() bool {
	_, err := llm.LoadConfig()
	return err == nil
}

func newMarketConditionResult(condition int, source string) MarketConditionResult {
	return MarketConditionResult{
		MarketCondition: condition,
		Name:            markettypes.MarketConditionName(condition),
		Source:          source,
	}
}

func sanitizeMarketConditionReason(reason string) string {
	return marketservice.SanitizeReason(reason)
}
