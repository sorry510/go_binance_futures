package feature

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	agentapp "go_binance_futures/agent/app"
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
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("encode market regime snapshot: %w", err)
	}
	manager, err := agentapp.DefaultManager()
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("initialize market regime manager: %w", err)
	}
	item, err := manager.Start(agentruntime.Request{Skill: marketregime.Name, Input: string(payload)})
	if err != nil {
		return MarketConditionResult{}, fmt.Errorf("start market regime task: %w", err)
	}
	deadline := time.Now().Add(marketConditionLLMTimeout)
	lastStage := ""
	for {
		current, err := manager.Get(context.Background(), item.ID)
		if err != nil {
			return MarketConditionResult{}, fmt.Errorf("get market regime task: %w", err)
		}
		if current.Stage != lastStage {
			reportMarketRegimeAgentProgress(progressCallback, task.Event{Stage: current.Stage})
			lastStage = current.Stage
		}
		if task.IsTerminalStatus(current.Status) {
			if current.Status != task.StatusSucceeded {
				return MarketConditionResult{}, fmt.Errorf("market regime task %s: %s", current.Status, current.Error)
			}
			var analysis marketregime.Analysis
			if err := json.Unmarshal(current.Result, &analysis); err != nil {
				return MarketConditionResult{}, fmt.Errorf("decode market regime task result: %w", err)
			}
			return MarketConditionResult{
				MarketCondition: analysis.MarketCondition, Name: markettypes.MarketConditionName(analysis.MarketCondition),
				Source: "llm", Confidence: analysis.Confidence, Reason: marketservice.SanitizeReason(analysis.Reason),
			}, nil
		}
		if time.Now().After(deadline) {
			return MarketConditionResult{}, fmt.Errorf("market regime task timeout")
		}
		time.Sleep(300 * time.Millisecond)
	}
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
