package manager

import (
	"context"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

type fakeClient struct {
	response string
}

func (*fakeClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (*fakeClient) ConfigID() int64        { return 42 }
func (client *fakeClient) Generate(context.Context, llm.Request) (*llm.Response, error) {
	return &llm.Response{Model: "fake", Content: client.response}, nil
}

func TestManagerStartsAndPersistsRuntimeTask(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Prompt: "test", Rounds: 2, Version: skill.VersionInfo{SkillVersion: "1.2.3", PromptVersion: "2.0.0", InputContractVersion: "input_v1", OutputContractVersion: "output_v1", Source: skill.DefaultSource, SourceVersion: "builtin-v1"}}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	manager, err := New(Config{
		Skills: skills,
		Store:  store,
		NewClient: func() (llm.Client, error) {
			return &fakeClient{response: `{"action":"final","summary":"done","result":{"ok":true}}`}, nil
		},
		RuntimeConfig: agentruntime.Config{Timeout: time.Second, Retry: agentruntime.RetryPolicy{MaxAttempts: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := manager.Start(agentruntime.Request{Skill: "test", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if started.ID == "" || started.Status != task.StatusQueued {
		t.Fatalf("unexpected started task: %+v", started)
	}
	if started.RuntimeVersion != agentruntime.CurrentVersion || started.SkillVersion != "1.2.3" || started.PromptVersion != "2.0.0" || started.PromptHash != skill.HashPrompt("test") || started.ModelConfigID != 42 || len(started.SkillPackageHash) != 64 || len(started.ToolCatalogHash) != 64 {
		t.Fatalf("task version identity was not frozen at start: %+v", started.VersionMetadata())
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, getErr := manager.Get(context.Background(), started.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if stored.Status == task.StatusSucceeded {
			if stored.VersionMetadata() != started.VersionMetadata() {
				t.Fatalf("task version identity changed while running: start=%+v stored=%+v", started.VersionMetadata(), stored.VersionMetadata())
			}
			if stored.ID != started.ID || stored.Progress != 100 || string(stored.Result) != `{"ok":true}` {
				t.Fatalf("unexpected completed task: %+v", stored)
			}
			return
		}
		if stored.Status == task.StatusFailed {
			t.Fatalf("runtime task failed: %+v", stored)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("runtime task did not complete")
}

type createFastPathStore struct {
	*task.MemoryStore
	createCalls int
	getCalls    int
}

func (store *createFastPathStore) Create(ctx context.Context, item *task.Task) error {
	store.createCalls++
	return store.MemoryStore.Create(ctx, item)
}

func (store *createFastPathStore) Get(ctx context.Context, id string) (*task.Task, error) {
	store.getCalls++
	return store.MemoryStore.Get(ctx, id)
}

func TestManagerStartUsesCreateFastPathWithoutReadBack(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "fast", Prompt: "fast", Rounds: 1}); err != nil {
		t.Fatal(err)
	}
	blocking := &managerBlockingClient{started: make(chan struct{}, 1)}
	store := &createFastPathStore{MemoryStore: task.NewMemoryStore()}
	manager, err := New(Config{
		Skills: skills, Store: store,
		NewClient:     func() (llm.Client, error) { return blocking, nil },
		RuntimeConfig: agentruntime.Config{Timeout: time.Second, Retry: agentruntime.RetryPolicy{MaxAttempts: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := manager.Start(agentruntime.Request{Skill: "fast", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != task.StatusQueued || store.createCalls != 1 || store.getCalls != 0 {
		t.Fatalf("unexpected start fast path: task=%+v create=%d get=%d", started, store.createCalls, store.getCalls)
	}
	if err := manager.Cancel(context.Background(), started.ID); err != nil {
		t.Fatal(err)
	}
}

func TestManagerCallsCompletionHook(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Rounds: 1}); err != nil {
		t.Fatal(err)
	}
	called := make(chan string, 1)
	manager, err := New(Config{
		Skills: skills,
		NewClient: func() (llm.Client, error) {
			return &fakeClient{response: `{"action":"final","result":{"ok":true}}`}, nil
		},
		RuntimeConfig: agentruntime.Config{Timeout: time.Second},
		CompletionHook: func(req agentruntime.Request, item *task.Task, result *agentruntime.Result, runErr error) error {
			if runErr != nil || result == nil || item.Status != task.StatusSucceeded {
				t.Errorf("unexpected completion state: item=%+v result=%+v err=%v", item, result, runErr)
			}
			called <- req.TaskID
			return nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := manager.Start(agentruntime.Request{Skill: "test", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case taskID := <-called:
		if taskID != started.ID {
			t.Fatalf("hook task id %s, want %s", taskID, started.ID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("completion hook was not called")
	}
}

type managerBlockingClient struct {
	started chan struct{}
}

func (*managerBlockingClient) Provider() llm.Provider { return llm.ProviderOpenAICompatible }
func (*managerBlockingClient) ConfigID() int64        { return 42 }
func (client *managerBlockingClient) Generate(ctx context.Context, _ llm.Request) (*llm.Response, error) {
	select {
	case client.started <- struct{}{}:
	default:
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestManagerCancelAndResumeUsesFrozenModelConfig(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "resume", Prompt: "resume", Rounds: 3}); err != nil {
		t.Fatal(err)
	}
	store := task.NewMemoryStore()
	blocking := &managerBlockingClient{started: make(chan struct{}, 1)}
	var requestedConfigID int64
	manager, err := New(Config{
		Skills: skills,
		Store:  store,
		NewClient: func() (llm.Client, error) {
			return blocking, nil
		},
		NewClientByID: func(id int64) (llm.Client, error) {
			requestedConfigID = id
			return &fakeClient{response: `{"action":"final","result":{"ok":true}}`}, nil
		},
		RuntimeConfig: agentruntime.Config{Timeout: time.Second, Retry: agentruntime.RetryPolicy{MaxAttempts: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	started, err := manager.Start(agentruntime.Request{Skill: "resume", Input: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	<-blocking.started
	if err := manager.Cancel(context.Background(), started.ID); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		item, getErr := store.Get(context.Background(), started.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if item.Status == task.StatusCancelled {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	cancelled, err := store.Get(context.Background(), started.ID)
	if err != nil || cancelled.Status != task.StatusCancelled {
		t.Fatalf("task was not cancelled: %+v err=%v", cancelled, err)
	}
	if _, err := manager.Resume(context.Background(), started.ID); err != nil {
		t.Fatal(err)
	}
	if requestedConfigID != 42 {
		t.Fatalf("resume model config id = %d, want 42", requestedConfigID)
	}
	deadline = time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		item, getErr := store.Get(context.Background(), started.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if item.Status == task.StatusSucceeded {
			if item.ResumeCount != 1 {
				t.Fatalf("resume count = %d, want 1", item.ResumeCount)
			}
			return
		}
		if item.Status == task.StatusFailed {
			t.Fatalf("resumed task failed: %+v", item)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("resumed task did not complete")
}
