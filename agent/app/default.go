package app

import (
	"sync"
	"time"

	agentmanager "go_binance_futures/agent/manager"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	agenttools "go_binance_futures/agent/tools"
	domaintools "go_binance_futures/agent/tools/domain"
)

var defaultManagerOnce sync.Once
var defaultManager *agentmanager.Manager
var defaultManagerErr error

func DefaultManager() (*agentmanager.Manager, error) {
	defaultManagerOnce.Do(func() {
		skills := skill.NewRegistry()
		for _, definition := range []skill.Skill{symbolanalysis.New(), alertanalysis.New()} {
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
		defaultManager, defaultManagerErr = agentmanager.New(agentmanager.Config{
			Skills:         skills,
			Tools:          tools,
			CompletionHook: persistTaskCompletion,
			RuntimeConfig: agentruntime.Config{
				Timeout:            3 * time.Minute,
				DefaultMaxRounds:   8,
				MaxContextBytes:    256 * 1024,
				MaxToolResultBytes: 256 * 1024,
				MaxToolCalls:       12,
				Retry:              agentruntime.RetryPolicy{MaxAttempts: 2, Delay: time.Second},
			},
		})
	})
	return defaultManager, defaultManagerErr
}
