package manager

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/llm"
)

type Config struct {
	NewClient      func() (llm.Client, error)
	NewClientByID  func(int64) (llm.Client, error)
	Admission      func(skill string) error
	Skills         *skill.Registry
	Tools          *agenttools.Registry
	Store          task.Store
	RuntimeConfig  agentruntime.Config
	CompletionHook func(agentruntime.Request, *task.Task, *agentruntime.Result, error) error
}

type Manager struct {
	cfg     Config
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
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
	if cfg.NewClientByID == nil {
		cfg.NewClientByID = llm.NewFromConfigID
	}
	return &Manager{cfg: cfg, cancels: make(map[string]context.CancelFunc)}, nil
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
	if manager.cfg.Admission != nil {
		if err := manager.cfg.Admission(selectedSkill.Name()); err != nil {
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
	executionSnapshot, err := runner.FreezeExecutionChecked(context.Background(), selectedSkill)
	if err != nil {
		return nil, err
	}
	req.ExecutionSnapshot = &executionSnapshot

	now := time.Now().UTC()
	taskID := task.NewID()
	maxRounds := selectedSkill.MaxRounds()
	if maxRounds <= 0 {
		maxRounds = agentruntime.DefaultConfig().DefaultMaxRounds
	}
	item := &task.Task{ID: taskID, Skill: selectedSkill.Name(), ConversationID: strings.TrimSpace(req.ConversationID), Status: task.StatusQueued, Stage: "queued", Input: req.Input, MaxRounds: maxRounds, Provider: string(client.Provider()), CreatedAt: now, UpdatedAt: now}
	item.ApplyVersionMetadata(executionSnapshot.Version)
	if creator, ok := manager.cfg.Store.(task.CreateStore); ok {
		if err := creator.Create(context.Background(), item); err != nil {
			return nil, err
		}
	} else if err := manager.cfg.Store.Save(context.Background(), item); err != nil {
		return nil, err
	}
	started := *item
	started.Events = append([]task.Event(nil), item.Events...)
	req.TaskID = taskID
	runCtx, cancel := context.WithCancel(context.Background())
	manager.registerCancel(taskID, cancel)
	go func() {
		defer manager.unregisterCancel(taskID)
		defer cancel()
		result, runErr := runner.Run(runCtx, req)
		manager.runCompletionHook(req, taskID, result, runErr)
	}()
	return &started, nil
}

func (manager *Manager) Get(ctx context.Context, taskID string) (*task.Task, error) {
	return manager.cfg.Store.Get(ctx, strings.TrimSpace(taskID))
}

func (manager *Manager) List(ctx context.Context, options task.ListOptions) (task.ListResult, error) {
	return manager.cfg.Store.List(ctx, options)
}

func (manager *Manager) Cancel(ctx context.Context, taskID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	taskID = strings.TrimSpace(taskID)
	manager.mu.Lock()
	cancel := manager.cancels[taskID]
	manager.mu.Unlock()
	if cancel == nil {
		item, err := manager.cfg.Store.Get(ctx, taskID)
		if err != nil {
			return err
		}
		return fmt.Errorf("task %q is not actively running (status=%s)", item.ID, item.Status)
	}
	cancel()
	return nil
}

func (manager *Manager) Resume(ctx context.Context, taskID string) (*task.Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	taskID = strings.TrimSpace(taskID)
	item, err := manager.cfg.Store.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	selectedSkill, ok := manager.cfg.Skills.Get(item.Skill)
	if !ok {
		return nil, fmt.Errorf("skill %q is not registered", item.Skill)
	}
	if manager.cfg.Admission != nil {
		if err := manager.cfg.Admission(selectedSkill.Name()); err != nil {
			return nil, err
		}
	}
	client, err := manager.cfg.NewClientByID(item.ModelConfigID)
	if err != nil {
		return nil, fmt.Errorf("initialize frozen LLM client: %w", err)
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
	req, err := runner.ResumeRequest(ctx, taskID)
	if err != nil {
		return nil, err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	manager.registerCancel(taskID, cancel)
	go func() {
		defer manager.unregisterCancel(taskID)
		defer cancel()
		result, runErr := runner.Resume(runCtx, taskID)
		manager.runCompletionHook(req, taskID, result, runErr)
	}()
	return manager.cfg.Store.Get(ctx, taskID)
}

func (manager *Manager) registerCancel(taskID string, cancel context.CancelFunc) {
	manager.mu.Lock()
	manager.cancels[taskID] = cancel
	manager.mu.Unlock()
}

func (manager *Manager) unregisterCancel(taskID string) {
	manager.mu.Lock()
	delete(manager.cancels, taskID)
	manager.mu.Unlock()
}

func (manager *Manager) runCompletionHook(req agentruntime.Request, taskID string, result *agentruntime.Result, runErr error) {
	if manager.cfg.CompletionHook == nil {
		return
	}
	stored, getErr := manager.cfg.Store.Get(context.Background(), taskID)
	if getErr != nil {
		return
	}
	_ = manager.cfg.CompletionHook(req, stored, result, runErr)
}
