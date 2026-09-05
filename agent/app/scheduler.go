package app

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go_binance_futures/agent/scheduler"
	"go_binance_futures/agent/security"
	marketregime "go_binance_futures/agent/skills/marketregime"
	workflowSkills "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/agent/task"
	"go_binance_futures/models"
	marketservice "go_binance_futures/service/market"
	workflowservice "go_binance_futures/service/workflow"

	"github.com/beego/beego/v2/core/logs"
)

type SchedulerConfigProvider func() models.Config

var defaultSchedulerOnce sync.Once
var defaultScheduler *scheduler.Scheduler
var defaultSchedulerErr error

func StartDefaultScheduler(ctx context.Context, provider SchedulerConfigProvider) error {
	defaultSchedulerOnce.Do(func() {
		if provider == nil {
			defaultSchedulerErr = fmt.Errorf("agent scheduler config provider is required")
			return
		}
		manager, err := DefaultManager()
		if err != nil {
			defaultSchedulerErr = err
			return
		}
		defaultScheduler, err = scheduler.New(manager, []scheduler.Job{
			{
				Name:  "market_regime",
				Skill: marketregime.Name,
				Enabled: func() bool {
					cfg := provider()
					return cfg.AgentMarketRegimeScheduleEnable == 1 && cfg.MarketConditionIsAuto == 1
				},
				Interval: func() time.Duration {
					minutes := provider().AgentMarketRegimeIntervalMin
					if minutes <= 0 {
						minutes = 60
					}
					return time.Duration(minutes) * time.Minute
				},
				Timeout:           5 * time.Minute,
				ConcurrencyPolicy: scheduler.SkipIfRunning,
				BuildInput: func(buildCtx context.Context) (string, error) {
					symbols, err := marketservice.LoadRegimeSymbols(buildCtx)
					if err != nil {
						return "", err
					}
					payload, err := json.Marshal(marketservice.BuildRegimeSnapshot(symbols))
					if err != nil {
						return "", err
					}
					return string(payload), nil
				},
				OnComplete: func(callbackCtx context.Context, item *task.Task) {
					if item != nil && item.Status != task.StatusSucceeded {
						applyMarketRegimeFallback(callbackCtx, provider, item.Error)
					}
				},
				OnError: func(callbackCtx context.Context, err error) {
					applyMarketRegimeFallback(callbackCtx, provider, err.Error())
				},
			},
			{
				Name:    "daily_market_brief",
				Skill:   workflowSkills.DailyMarketBriefName,
				Enabled: func() bool { return provider().AgentDailyMarketBriefScheduleEnable == 1 },
				Interval: func() time.Duration {
					minutes := provider().AgentDailyMarketBriefIntervalMin
					if minutes <= 0 {
						minutes = 1440
					}
					return time.Duration(minutes) * time.Minute
				},
				Timeout:           8 * time.Minute,
				ConcurrencyPolicy: scheduler.SkipIfRunning,
				BuildInput: func(buildCtx context.Context) (string, error) {
					return workflowservice.BuildDailyMarketBriefInput(buildCtx, 24)
				},
			},
		})
		if err != nil {
			defaultSchedulerErr = err
			return
		}
		defaultScheduler.Start(ctx)
	})
	return defaultSchedulerErr
}

func DefaultSchedulerStatus() []scheduler.JobStatus {
	if defaultScheduler == nil {
		return nil
	}
	return defaultScheduler.Status()
}

func TriggerDefaultSchedulerJob(ctx context.Context, name string) error {
	if defaultScheduler == nil {
		return fmt.Errorf("agent scheduler is not started")
	}
	return defaultScheduler.Trigger(ctx, name)
}

func applyMarketRegimeFallback(ctx context.Context, provider SchedulerConfigProvider, reason string) {
	cfg := provider()
	if cfg.MarketConditionIsAuto != 1 {
		return
	}
	symbols, err := marketservice.LoadRegimeSymbols(ctx)
	if err != nil {
		logs.Error("market regime scheduler fallback load symbols:", err)
		return
	}
	condition, err := marketservice.CalculateAlgorithmCondition(symbols)
	if err != nil {
		logs.Error("market regime scheduler fallback calculate:", err)
		return
	}
	if err := marketservice.SaveMarketCondition(ctx, cfg.ID, condition); err != nil {
		logs.Error("market regime scheduler fallback save:", err)
		return
	}
	logs.Warning("market regime scheduler used algorithm fallback: condition=%d reason=%s", condition, security.RedactText(reason))
}
