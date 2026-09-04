package app

import (
	"context"
	"sync"
	"time"

	agentmanager "go_binance_futures/agent/manager"
	"go_binance_futures/agent/observability"
	"go_binance_futures/agent/permission"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	marketregime "go_binance_futures/agent/skills/marketregime"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	domaintools "go_binance_futures/agent/tools/domain"

	"github.com/beego/beego/v2/core/logs"
)

var defaultManagerOnce sync.Once
var defaultManager *agentmanager.Manager
var defaultManagerErr error

func DefaultManager() (*agentmanager.Manager, error) {
	defaultManagerOnce.Do(func() {
		skills := skill.NewRegistry()
		for _, definition := range []skill.Skill{symbolanalysis.New(), alertanalysis.New(), marketregime.New()} {
			if err := skills.Register(definition); err != nil {
				defaultManagerErr = err
				return
			}
		}
		tools := agenttools.NewRegistry()
		if err := domaintools.RegisterReadOnly(tools, domaintools.DefaultDependencies()); err != nil {
			defaultManagerErr = err
			return
		}
		if err := initializeDefaultMCP(tools); err != nil {
			defaultManagerErr = err
			return
		}
		store := task.NewORMStore()
		if interrupted, err := store.MarkInterrupted(context.Background(), time.Now().UTC()); err != nil {
			defaultManagerErr = err
			return
		} else if interrupted > 0 {
			logs.Warning("marked interrupted agent tasks after restart:", interrupted)
		}
		defaultManager, defaultManagerErr = agentmanager.New(agentmanager.Config{
			Skills:         skills,
			Admission:      AdmitSkill,
			Store:          store,
			Tools:          tools,
			CompletionHook: persistTaskCompletion,
			RuntimeConfig: agentruntime.Config{
				Timeout:                 10 * time.Minute,
				Policy:                  permission.AllowWritesFor(nil),
				BudgetProvider:          RuntimeBudget,
				ToolAllowlistProvider:   MCPToolAllowlist,
				ContextResourceProvider: MCPContextResources,
				Observer:                observability.Default(),
				DefaultMaxRounds:        8,
				MaxContextBytes:         256 * 1024,
				MaxToolResultBytes:      256 * 1024,
				Retry:                   agentruntime.RetryPolicy{MaxAttempts: 2, Delay: time.Second},
			},
		})
	})
	return defaultManager, defaultManagerErr
}
