package manager

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

type Config struct {
	NewClient      func() (llm.Client, error)
	Skills         *skill.Registry
	Tools          *agenttools.Registry
	Store          task.Store
	RuntimeConfig  agentruntime.Config
	CompletionHook func(agentruntime.Request, *task.Task, *agentruntime.Result, error) error
}

type Manager struct {
	cfg Config
}

func New(cfg Config) (*Manager, error) {
	if cfg.Skills == nil {
		return nil, fmt.Errorf("agent manager requires a skill registry")
	}
	if cfg.Tools == nil {
		cfg.Tools = agenttools.NewRegistry()
	}
	if cfg.Store == nil {
		cfg.Store = task.NewMemoryStore()
	}
	if cfg.NewClient == nil {
		cfg.NewClient = llm.NewFromConfig
	}
	return &Manager{cfg: cfg}, nil
}
func (manager *Manager) Start(req agentruntime.Request) (*task.Task, error) {
	req.Skill = strings.TrimSpace(req.Skill)
	selectedSkill, ok := manager.cfg.Skills.Get(req.Skill)
	if !ok {
		return nil, fmt.Errorf("skill %q is not registered", req.Skill)
	}
	skillRequest := skill.Request{Input: req.Input, ConversationID: req.ConversationID, Metadata: req.Metadata}
	if inputValidator, ok := selectedSkill.(skill.InputValidator); ok {
		if err := inputValidator.ValidateInput(skillRequest); err != nil {
			return nil, err
		}
	}
	client, err := manager.cfg.NewClient()
	if err != nil {
		return nil, fmt.Errorf("initialize LLM client: %w", err)
	}
	runtimeConfig := manager.cfg.RuntimeConfig
	runtimeConfig.Client = client
	runtimeConfig.Skills = manager.cfg.Skills
	runtimeConfig.Tools = manager.cfg.Tools
	runtimeConfig.Tasks = manager.cfg.Store
	runner, err := agentruntime.NewRunner(runtimeConfig)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	taskID := task.NewID()
	maxRounds := selectedSkill.MaxRounds()
	if maxRounds <= 0 {
		maxRounds = agentruntime.DefaultConfig().DefaultMaxRounds
	}
	item := &task.Task{ID: taskID, Skill: selectedSkill.Name(), Status: task.StatusQueued, Stage: "queued", Input: req.Input, MaxRounds: maxRounds, Provider: string(client.Provider()), CreatedAt: now, UpdatedAt: now}
	if err := manager.cfg.Store.Save(context.Background(), item); err != nil {
		return nil, err
	}
	req.TaskID = taskID
	go func() {
		result, runErr := runner.Run(context.Background(), req)
		if manager.cfg.CompletionHook == nil {
			return
		}
		stored, getErr := manager.cfg.Store.Get(context.Background(), taskID)
		if getErr != nil {
			return
		}
		_ = manager.cfg.CompletionHook(req, stored, result, runErr)
	}()
	return manager.cfg.Store.Get(context.Background(), taskID)
}

func (manager *Manager) Get(ctx context.Context, taskID string) (*task.Task, error) {
	return manager.cfg.Store.Get(ctx, strings.TrimSpace(taskID))
}
