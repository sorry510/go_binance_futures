package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	marketregime "go_binance_futures/agent/skills/marketregime"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
	marketservice "go_binance_futures/service/market"
	markettypes "go_binance_futures/types"
)

var newMarketRegimeLLMClient = llm.NewFromConfig

func analyzeMarketConditionWithRuntime(snapshot marketservice.RegimeSnapshot, progressCallback MarketConditionProgressCallback) (MarketConditionResult, error) {
	client, err := newMarketRegimeLLMClient()
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("initialize market regime LLM client: %w", err)
	}
	return analyzeMarketConditionWithRuntimeClient(snapshot, client, progressCallback)
}
func analyzeMarketConditionWithRuntimeClient(snapshot marketservice.RegimeSnapshot, client llm.Client, progressCallback MarketConditionProgressCallback) (MarketConditionResult, error) {
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("encode market regime snapshot: %w", err)
	}
	skills := skill.NewRegistry()
	if err := skills.Register(marketregime.New()); err != nil {
		return MarketConditionResult{}, fmt.Errorf("register market regime skill: %w", err)
	}
	runner, err := agentruntime.NewRunner(agentruntime.Config{
		Client:           client,
		Skills:           skills,
		Timeout:          marketConditionLLMTimeout,
		DefaultMaxRounds: 2,
		Retry:            agentruntime.RetryPolicy{MaxAttempts: 2, Delay: time.Second},
		EventHook: func(event task.Event) {
			reportMarketRegimeAgentProgress(progressCallback, event)
		},
	})
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("initialize market regime runtime: %w", err)
	}
	result, err := runner.Run(context.Background(), agentruntime.Request{Skill: marketregime.Name, Input: string(payload)})
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("run market regime agent: %w", err)
	}
	analysis, ok := result.Value.(marketregime.Analysis)
	if !ok {
		return MarketConditionResult{}, fmt.Errorf("market regime runtime returned unexpected result type %T", result.Value)
	}
	return MarketConditionResult{
		MarketCondition: analysis.MarketCondition,
		Name:            markettypes.MarketConditionName(analysis.MarketCondition),
		Source:          "llm",
		Confidence:      analysis.Confidence,
		Reason:          marketservice.SanitizeReason(analysis.Reason),
	}, nil
}
func reportMarketRegimeAgentProgress(callback MarketConditionProgressCallback, event task.Event) {
	if callback == nil {
		return
	}
	switch event.Stage {
	case "building_input":
		reportMarketConditionProgress(callback, 45, "preparing_agent")
	case "waiting_llm":
		reportMarketConditionProgress(callback, 55, "calling_llm")
	case "validating":
		reportMarketConditionProgress(callback, 80, "validating_llm")
	case "repairing_decision", "repairing_final":
		reportMarketConditionProgress(callback, 82, "repairing_llm")
	}
}
