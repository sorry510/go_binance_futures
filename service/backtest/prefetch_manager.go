package backtest

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/models"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/beego/beego/v2/client/orm"
)

type PrefetchSummary struct {
	JobID                string   `json:"job_id"`
	Status               string   `json:"status"`
	Stage                string   `json:"stage"`
	Progress             int      `json:"progress"`
	StrategyTemplateID   int64    `json:"strategy_template_id"`
	StrategyTemplateName string   `json:"strategy_template_name"`
	Symbol               string   `json:"symbol"`
	ReplayInterval       string   `json:"replay_interval"`
	Intervals            []string `json:"intervals"`
	StartTime            int64    `json:"start_time"`
	EndTime              int64    `json:"end_time"`
	WarmupStartTime      int64    `json:"warmup_start_time"`
	RemoteCalls          int      `json:"remote_calls"`
	RemoteRows           int      `json:"remote_rows"`
	Error                string   `json:"error,omitempty"`
	CreatedAt            int64    `json:"created_at"`
	UpdatedAt            int64    `json:"updated_at"`
	CompletedAt          int64    `json:"completed_at,omitempty"`
}

func (manager *Manager) StartPrefetch(request PrefetchRequest) (PrefetchSummary, error) {
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	if request.StrategyTemplateID <= 0 {
		return PrefetchSummary{}, fmt.Errorf("strategy_template_id is required")
	}
	if request.Symbol == "" || !strings.HasSuffix(request.Symbol, "USDT") {
		return PrefetchSummary{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if request.StartTime <= 0 || request.EndTime <= request.StartTime {
		return PrefetchSummary{}, fmt.Errorf("valid start_time and end_time are required")
	}
	var template models.StrategyTemplates
	if err := orm.NewOrm().QueryTable(new(models.StrategyTemplates)).Filter("id", request.StrategyTemplateID).One(&template); err != nil {
		return PrefetchSummary{}, fmt.Errorf("load strategy template: %w", err)
	}
	if strategyservice.StrategyUsesMarketCondition(template.Strategy) {
		if err := ValidateMarketConditionCoverage(context.Background(), request.StartTime, request.EndTime); err != nil {
			return PrefetchSummary{}, err
		}
	}
	plan, err := manager.builder.PrefetchPlan(DatasetRequest{
		Symbol: request.Symbol, ExecutionInterval: ReplayInterval,
		StartTime: request.StartTime, EndTime: request.EndTime, TechnologyJSON: template.Technology, StrategyJSON: template.Strategy,
	})
	if err != nil {
		return PrefetchSummary{}, err
	}
	now := time.Now().UnixMilli()
	job := PrefetchSummary{
		JobID: "prefetch_" + newRunID()[4:], Status: "queued", Stage: "queued", Progress: 0,
		StrategyTemplateID: template.ID, StrategyTemplateName: template.Name,
		Symbol: plan.Symbol, ReplayInterval: plan.ReplayInterval,
		Intervals: append([]string(nil), plan.Intervals...),
		StartTime: plan.StartTime, EndTime: plan.EndTime, WarmupStartTime: plan.WarmupStartTime,
		CreatedAt: now, UpdatedAt: now,
	}
	manager.prefetchMu.Lock()
	if manager.prefetchActive != "" {
		active := manager.prefetchJobs[manager.prefetchActive]
		if active.Status == "queued" || active.Status == "running" {
			if active.StrategyTemplateID == template.ID && active.Symbol == plan.Symbol && active.StartTime == plan.StartTime && active.EndTime == plan.EndTime {
				manager.prefetchMu.Unlock()
				return clonePrefetchSummary(active), nil
			}
			manager.prefetchMu.Unlock()
			return PrefetchSummary{}, fmt.Errorf("another historical data prefetch is already running")
		}
		manager.prefetchActive = ""
	}
	manager.prefetchJobs[job.JobID] = job
	manager.prefetchActive = job.JobID
	manager.prefetchMu.Unlock()
	go manager.runPrefetch(job.JobID, template.Technology, template.Strategy)
	return clonePrefetchSummary(job), nil
}

func (manager *Manager) GetPrefetch(jobID string) (PrefetchSummary, error) {
	manager.prefetchMu.Lock()
	defer manager.prefetchMu.Unlock()
	job, ok := manager.prefetchJobs[strings.TrimSpace(jobID)]
	if !ok {
		return PrefetchSummary{}, fmt.Errorf("historical data prefetch job %q not found", jobID)
	}
	return clonePrefetchSummary(job), nil
}
func (manager *Manager) runPrefetch(jobID, technologyJSON, strategyJSON string) {
	manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
		job.Status = "running"
		job.Stage = "fetching"
		job.Progress = 1
	})
	job, err := manager.GetPrefetch(jobID)
	if err != nil {
		return
	}
	result, err := manager.builder.Prefetch(context.Background(), DatasetRequest{
		Symbol: job.Symbol, ExecutionInterval: ReplayInterval,
		StartTime: job.StartTime, EndTime: job.EndTime, TechnologyJSON: technologyJSON, StrategyJSON: strategyJSON,
	}, func(completed, total int) {
		progress := 1
		if total > 0 {
			progress = 1 + completed*98/total
		}
		manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
			job.Stage = "fetching"
			job.Progress = progress
		})
	})
	if err != nil {
		manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
			job.Status = "failed"
			job.Stage = "failed"
			job.Progress = 100
			job.Error = err.Error()
			job.CompletedAt = time.Now().UnixMilli()
		})
		return
	}
	manager.updatePrefetch(jobID, func(job *PrefetchSummary) {
		job.Status = "succeeded"
		job.Stage = "completed"
		job.Progress = 100
		job.RemoteCalls = result.RemoteCalls
		job.RemoteRows = result.RemoteRows
		job.CompletedAt = time.Now().UnixMilli()
	})
}

func (manager *Manager) updatePrefetch(jobID string, update func(*PrefetchSummary)) {
	manager.prefetchMu.Lock()
	defer manager.prefetchMu.Unlock()
	job, ok := manager.prefetchJobs[jobID]
	if !ok {
		return
	}
	update(&job)
	job.UpdatedAt = time.Now().UnixMilli()
	manager.prefetchJobs[jobID] = job
	if job.Status == "succeeded" || job.Status == "failed" {
		if manager.prefetchActive == jobID {
			manager.prefetchActive = ""
		}
	}
}

func clonePrefetchSummary(job PrefetchSummary) PrefetchSummary {
	job.Intervals = append([]string(nil), job.Intervals...)
	return job
}
