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
func (client *fakeClient) Generate(context.Context, llm.Request) (*llm.Response, error) {
	return &llm.Response{Model: "fake", Content: client.response}, nil
}

func TestManagerStartsAndPersistsRuntimeTask(t *testing.T) {
	skills := skill.NewRegistry()
	if err := skills.Register(skill.Definition{SkillName: "test", Prompt: "test", Rounds: 2}); err != nil {
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

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, getErr := manager.Get(context.Background(), started.ID)
		if getErr != nil {
			t.Fatal(getErr)
		}
		if stored.Status == task.StatusSucceeded {
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
