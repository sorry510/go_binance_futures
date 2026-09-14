package opportunity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

type StartTaskFunc func(agentruntime.Request) (*task.Task, error)
type GetTaskFunc func(context.Context, string) (*task.Task, error)

type Trigger struct {
	Symbol     string `json:"symbol"`
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Prompt     string `json:"prompt,omitempty"`
}

type ProcessResult struct {
	Opportunity models.AgentOpportunity `json:"opportunity"`
	Created     bool                    `json:"created"`
	Suppressed  bool                    `json:"suppressed"`
	Reason      string                  `json:"reason,omitempty"`
}

type OpportunityCallback func(context.Context, models.AgentOpportunity) error

type PipelineConfig struct {
	Service       Service
	StartTask     StartTaskFunc
	GetTask       GetTaskFunc
	QueueSize     int
	Workers       int
	Cooldown      time.Duration
	TTL           time.Duration
	PollInterval  time.Duration
	TaskTimeout   time.Duration
	OnOpportunity OpportunityCallback
}

type PipelineStats struct {
	Received           uint64 `json:"received"`
	Dropped            uint64 `json:"dropped"`
	SuppressedReplay   uint64 `json:"suppressed_replay"`
	SuppressedCooldown uint64 `json:"suppressed_cooldown"`
	TasksStarted       uint64 `json:"tasks_started"`
	Succeeded          uint64 `json:"succeeded"`
	Failed             uint64 `json:"failed"`
	DataMissing        uint64 `json:"data_missing"`
	Neutral            uint64 `json:"neutral"`
}

type Pipeline struct {
	cfg         PipelineConfig
	queue       chan Trigger
	startOnce   sync.Once
	admissionMu sync.Mutex

	received           atomic.Uint64
	dropped            atomic.Uint64
	suppressedReplay   atomic.Uint64
	suppressedCooldown atomic.Uint64
	tasksStarted       atomic.Uint64
	succeeded          atomic.Uint64
	failed             atomic.Uint64
	dataMissing        atomic.Uint64
	neutral            atomic.Uint64
}

func NewPipeline(cfg PipelineConfig) (*Pipeline, error) {
	if cfg.StartTask == nil || cfg.GetTask == nil {
		return nil, fmt.Errorf("opportunity pipeline requires task dependencies")
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 128
	}
	if cfg.Workers <= 0 {
		cfg.Workers = 2
	}
	if cfg.Cooldown <= 0 {
		cfg.Cooldown = 30 * time.Minute
	}
	if cfg.TTL <= 0 {
		cfg.TTL = time.Hour
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.TaskTimeout <= 0 {
		cfg.TaskTimeout = 2 * time.Minute
	}
	return &Pipeline{cfg: cfg, queue: make(chan Trigger, cfg.QueueSize)}, nil
}

func (p *Pipeline) Start(ctx context.Context) {
	if p == nil {
		return
	}
	p.startOnce.Do(func() {
		for i := 0; i < p.cfg.Workers; i++ {
			go p.worker(ctx)
		}
	})
}

func (p *Pipeline) Emit(trigger Trigger) bool {
	if p == nil {
		return false
	}
	p.received.Add(1)
	select {
	case p.queue <- trigger:
		return true
	default:
		p.dropped.Add(1)
		return false
	}
}

func (p *Pipeline) Stats() PipelineStats {
	if p == nil {
		return PipelineStats{}
	}
	return PipelineStats{
		Received: p.received.Load(), Dropped: p.dropped.Load(),
		SuppressedReplay: p.suppressedReplay.Load(), SuppressedCooldown: p.suppressedCooldown.Load(),
		TasksStarted: p.tasksStarted.Load(), Succeeded: p.succeeded.Load(), Failed: p.failed.Load(),
		DataMissing: p.dataMissing.Load(), Neutral: p.neutral.Load(),
	}
}

func (p *Pipeline) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trigger := <-p.queue:
			if _, err := p.Process(ctx, trigger); err != nil {
				logs.Warning("opportunity pipeline trigger failed:", err)
			}
		}
	}
}

func (p *Pipeline) Process(ctx context.Context, trigger Trigger) (ProcessResult, error) {
	if p == nil {
		return ProcessResult{}, fmt.Errorf("opportunity pipeline is nil")
	}
	trigger.Symbol = strings.ToUpper(strings.TrimSpace(trigger.Symbol))
	trigger.SourceType = strings.ToLower(strings.TrimSpace(trigger.SourceType))
	trigger.SourceID = strings.TrimSpace(trigger.SourceID)
	trigger.Prompt = strings.TrimSpace(trigger.Prompt)
	if trigger.Symbol == "" || !strings.HasSuffix(trigger.Symbol, "USDT") || trigger.SourceType == "" || trigger.SourceID == "" {
		return ProcessResult{}, fmt.Errorf("invalid opportunity trigger")
	}

	row, admission, admitted, err := p.admit(ctx, trigger)
	if err != nil {
		return ProcessResult{}, err
	}
	if !admitted {
		return admission, nil
	}

	input, err := json.Marshal(symbolanalysis.Input{Symbol: trigger.Symbol, Prompt: trigger.Prompt})
	if err != nil {
		return ProcessResult{}, p.failOpportunity(ctx, row, "", err)
	}
	item, err := p.cfg.StartTask(agentruntime.Request{
		Skill: symbolanalysis.Name, Input: string(input),
		Metadata: map[string]any{
			"opportunity_id": row.OpportunityID,
			"source_type":    trigger.SourceType,
			"source_id":      trigger.SourceID,
		},
	})
	if err != nil {
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, "", err)
	}
	if item == nil || strings.TrimSpace(item.ID) == "" {
		err := fmt.Errorf("symbol_analysis task start returned no task id")
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, "", err)
	}
	p.tasksStarted.Add(1)
	row.AnalysisTaskID = item.ID
	row.UpdatedAt = p.cfg.Service.now().UnixMilli()
	if err := p.cfg.Service.Store.Save(ctx, &row); err != nil {
		return ProcessResult{}, err
	}

	completed, err := p.waitTask(ctx, item.ID)
	if err != nil {
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, item.ID, err)
	}
	if completed.Status != task.StatusSucceeded {
		reason := strings.TrimSpace(completed.Error)
		if reason == "" {
			reason = "symbol_analysis task did not succeed"
		}
		err := fmt.Errorf("%s", reason)
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, item.ID, err)
	}

	var plan symbolanalysis.TradingPlanV1
	if err := json.Unmarshal(completed.Result, &plan); err != nil {
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, item.ID, fmt.Errorf("decode TradingPlanV1: %w", err))
	}
	if strings.ToUpper(strings.TrimSpace(plan.Symbol)) != trigger.Symbol {
		return ProcessResult{Opportunity: row, Created: true}, p.failOpportunity(ctx, row, item.ID, fmt.Errorf("analysis symbol mismatch: got %s want %s", plan.Symbol, trigger.Symbol))
	}
	analysisStatus := AnalysisSucceeded
	if len(plan.DataMissing) > 0 {
		analysisStatus = AnalysisDataMissing
	}
	marketCondition := 0
	if plan.MarketCondition != nil {
		marketCondition = *plan.MarketCondition
	}
	updated, err := p.cfg.Service.UpdateAnalysis(ctx, row.OpportunityID, AnalysisUpdate{
		Direction: plan.Direction, TaskID: item.ID, Summary: plan.Summary,
		Confidence: plan.Confidence, MarketCondition: marketCondition, Status: analysisStatus,
	})
	if err != nil {
		return ProcessResult{}, err
	}
	if analysisStatus == AnalysisDataMissing {
		p.dataMissing.Add(1)
	} else {
		p.succeeded.Add(1)
	}
	if updated.Direction == DirectionNeutral {
		p.neutral.Add(1)
	}
	if p.cfg.OnOpportunity != nil {
		if callbackErr := p.cfg.OnOpportunity(ctx, updated); callbackErr != nil {
			logs.Warning("opportunity callback failed:", callbackErr)
		}
	}
	return ProcessResult{Opportunity: updated, Created: true}, nil
}

func (p *Pipeline) admit(ctx context.Context, trigger Trigger) (models.AgentOpportunity, ProcessResult, bool, error) {
	// Cooldown is a check-then-create decision. Serialize this short section so
	// multiple workers cannot admit two same-symbol/source-type triggers that
	// arrive at the same instant. Multi-instance coordination is intentionally
	// out of scope for this single-user deployment.
	p.admissionMu.Lock()
	defer p.admissionMu.Unlock()

	if existing, err := p.cfg.Service.Store.FindBySource(ctx, trigger.SourceType, trigger.SourceID); err == nil {
		p.suppressedReplay.Add(1)
		return models.AgentOpportunity{}, ProcessResult{Opportunity: existing, Suppressed: true, Reason: "source_replay"}, false, nil
	} else if err != orm.ErrNoRows {
		return models.AgentOpportunity{}, ProcessResult{}, false, err
	}

	if p.cfg.Cooldown > 0 {
		since := p.cfg.Service.now().Add(-p.cfg.Cooldown)
		hit, recent, err := p.cfg.Service.InCooldown(ctx, trigger.Symbol, trigger.SourceType, since)
		if err != nil {
			return models.AgentOpportunity{}, ProcessResult{}, false, err
		}
		if hit {
			p.suppressedCooldown.Add(1)
			return models.AgentOpportunity{}, ProcessResult{Opportunity: recent, Suppressed: true, Reason: "cooldown"}, false, nil
		}
	}

	expiresAt := p.cfg.Service.now().Add(p.cfg.TTL).UnixMilli()
	row, created, err := p.cfg.Service.Create(ctx, CreateInput{
		Symbol: trigger.Symbol, Direction: DirectionNeutral,
		SourceType: trigger.SourceType, SourceID: trigger.SourceID,
		AnalysisStatus: AnalysisPending, ExpiresAt: expiresAt,
	})
	if err != nil {
		return models.AgentOpportunity{}, ProcessResult{}, false, err
	}
	if !created {
		p.suppressedReplay.Add(1)
		return models.AgentOpportunity{}, ProcessResult{Opportunity: row, Suppressed: true, Reason: "source_replay"}, false, nil
	}
	return row, ProcessResult{}, true, nil
}

func (p *Pipeline) failOpportunity(ctx context.Context, row models.AgentOpportunity, taskID string, cause error) error {
	p.failed.Add(1)
	_, saveErr := p.cfg.Service.UpdateAnalysis(ctx, row.OpportunityID, AnalysisUpdate{
		Direction: DirectionNeutral, TaskID: taskID, Summary: row.Summary,
		Confidence: 0, MarketCondition: row.MarketCondition, Status: AnalysisFailed, Error: cause.Error(),
	})
	if saveErr != nil {
		return fmt.Errorf("%v; persist opportunity failure: %w", cause, saveErr)
	}
	return cause
}

func (p *Pipeline) waitTask(ctx context.Context, taskID string) (*task.Task, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, p.cfg.TaskTimeout)
	defer cancel()
	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()
	for {
		item, err := p.cfg.GetTask(timeoutCtx, taskID)
		if err != nil {
			return nil, err
		}
		if task.IsTerminalStatus(item.Status) {
			return item, nil
		}
		select {
		case <-timeoutCtx.Done():
			return nil, timeoutCtx.Err()
		case <-ticker.C:
		}
	}
}
