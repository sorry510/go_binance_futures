package opportunity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	workflowSkills "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/task"
	"go_binance_futures/models"
	"go_binance_futures/notify"
	signalservice "go_binance_futures/service/signal"
)

type ConfigProvider func() models.Config

var defaultMu sync.RWMutex
var defaultPipeline *Pipeline
var defaultProvider ConfigProvider

func StartDefault(ctx context.Context, provider ConfigProvider, start StartTaskFunc, get GetTaskFunc) error {
	if provider == nil {
		return fmt.Errorf("opportunity config provider is required")
	}
	pipeline, err := NewPipeline(PipelineConfig{
		Service: DefaultService(), StartTask: start, GetTask: get,
		QueueSize: 256, Workers: 2, Cooldown: 30 * time.Minute, TTL: time.Hour,
		OnOpportunity: func(callbackCtx context.Context, row models.AgentOpportunity) error {
			cfg := provider()
			threshold := cfg.AgentOpportunityMinConfidence
			if threshold <= 0 {
				threshold = 0.7
			}
			if row.AnalysisStatus != AnalysisSucceeded || row.Direction == DirectionNeutral || row.Confidence < threshold {
				return nil
			}
			_, err := notify.GetNotifyChannel().AgentAlert(notify.AgentAlertParams{
				Title: "notification.agent_opportunity_title", Module: "agent_opportunity_watch",
				EventID: row.OpportunityID, SignalID: row.SourceID, TaskID: row.AnalysisTaskID,
				EventTime: row.CreatedAt, Symbol: row.Symbol, SignalType: "opportunity", Severity: "medium",
				Summary:       row.Summary,
				MarketContext: fmt.Sprintf("direction=%s · confidence=%.0f%% · market_condition=%d", row.Direction, row.Confidence*100, row.MarketCondition),
				ConfirmedBy:   []string{row.SourceType}, Source: "Opportunity Watch",
			})
			return err
		},
	})
	if err != nil {
		return err
	}
	defaultMu.Lock()
	defaultProvider = provider
	defaultPipeline = pipeline
	defaultMu.Unlock()
	pipeline.Start(ctx)
	return nil
}

func currentDefault() (*Pipeline, ConfigProvider) {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultPipeline, defaultProvider
}

func DefaultStats() PipelineStats {
	pipeline, _ := currentDefault()
	if pipeline == nil {
		return PipelineStats{}
	}
	return pipeline.Stats()
}

func DefaultEmit(trigger Trigger) bool {
	pipeline, provider := currentDefault()
	if pipeline == nil || provider == nil || provider().AgentOpportunityWatchEnable != 1 {
		return false
	}
	return pipeline.Emit(trigger)
}

func DefaultEmitSignal(value signalservice.Signal) bool {
	switch value.Type {
	case signalservice.TypeFastMove, signalservice.TypeLiquidationSpike:
	default:
		return false
	}
	sourceID := strings.TrimSpace(value.SignalID)
	if sourceID == "" {
		return false
	}
	prompt := fmt.Sprintf("Opportunity Watch trigger: %s %s severity=%s window=%s. Analyze the symbol using current market data; do not assume the signal itself proves a trade direction.", value.Type, value.Symbol, value.Severity, value.Window)
	return DefaultEmit(Trigger{Symbol: value.Symbol, SourceType: string(value.Type), SourceID: sourceID, Prompt: prompt})
}

func DefaultEmitMarketScanTask(item *task.Task) int {
	pipeline, provider := currentDefault()
	if pipeline == nil || provider == nil || provider().AgentOpportunityWatchEnable != 1 || item == nil || item.Status != task.StatusSucceeded {
		return 0
	}
	var result workflowSkills.OpportunitySetV1
	if err := jsonUnmarshalTaskResult(item, &result); err != nil {
		return 0
	}
	threshold := provider().AgentOpportunityMinConfidence
	if threshold <= 0 {
		threshold = 0.7
	}
	emitted := 0
	for _, candidate := range result.Opportunities {
		direction := strings.ToLower(strings.TrimSpace(candidate.Direction))
		if candidate.Confidence < threshold || direction == "avoid" {
			continue
		}
		prompt := fmt.Sprintf("Scheduled market_scan candidate rank=%d direction=%s confidence=%.2f thesis=%s. Perform a fresh symbol_analysis before treating this as a trade opportunity.", candidate.Rank, direction, candidate.Confidence, strings.TrimSpace(candidate.Thesis))
		if pipeline.Emit(Trigger{Symbol: candidate.Symbol, SourceType: "market_scan", SourceID: item.ID + ":" + strings.ToUpper(strings.TrimSpace(candidate.Symbol)), Prompt: prompt}) {
			emitted++
		}
	}
	return emitted
}

func jsonUnmarshalTaskResult(item *task.Task, out any) error {
	if item == nil || len(item.Result) == 0 {
		return fmt.Errorf("task result is empty")
	}
	return json.Unmarshal(item.Result, out)
}
