package scheduler

import (
	"context"
	"sync"
	"testing"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
)

type fakeManager struct {
	mu    sync.Mutex
	items map[string]*task.Task
	count int
}

func (manager *fakeManager) Start(req agentruntime.Request) (*task.Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.count++
	id := "task-test"
	item := &task.Task{ID: id, Skill: req.Skill, Status: task.StatusRunning, Stage: "running", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	manager.items[id] = item
	copy := *item
	return &copy, nil
}

func (manager *fakeManager) Get(ctx context.Context, id string) (*task.Task, error) {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	copy := *manager.items[id]
	return &copy, nil
}

func (manager *fakeManager) finish() {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	item := manager.items["task-test"]
	now := time.Now()
	item.Status = task.StatusSucceeded
	item.Stage = "completed"
	item.CompletedAt = &now
}

func TestSchedulerSkipIfRunning(t *testing.T) {
	manager := &fakeManager{items: map[string]*task.Task{}}
	scheduler, err := New(manager, []Job{{
		Name: "market_regime", Skill: "market_regime", Interval: func() time.Duration { return time.Hour },
		ConcurrencyPolicy: SkipIfRunning, BuildInput: func(context.Context) (string, error) { return `{}`, nil },
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Trigger(context.Background(), "market_regime"); err != nil {
		t.Fatal(err)
	}
	if err := scheduler.Trigger(context.Background(), "market_regime"); err != nil {
		t.Fatal(err)
	}
	status := scheduler.Status()[0]
	if manager.count != 1 || status.SkipCount != 1 || !status.Running {
		t.Fatalf("unexpected running status: manager=%d status=%+v", manager.count, status)
	}
	manager.finish()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !scheduler.Status()[0].Running {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("scheduler job did not leave running state")
}

func TestSchedulerRebasesNextRunWhenIntervalChanges(t *testing.T) {
	manager := &fakeManager{items: map[string]*task.Task{}}
	interval := time.Hour
	scheduler, err := New(manager, []Job{{
		Name: "market_regime", Skill: "market_regime", Interval: func() time.Duration { return interval },
		BuildInput: func(context.Context) (string, error) { return `{}`, nil },
	}})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	scheduler.tick(context.Background(), now)
	first := scheduler.Status()[0].NextRunAt
	interval = 5 * time.Minute
	later := now.Add(time.Second)
	scheduler.tick(context.Background(), later)
	second := scheduler.Status()[0].NextRunAt
	if second >= first {
		t.Fatalf("next run was not rebased: first=%d second=%d", first, second)
	}
	want := later.Add(5 * time.Minute).UnixMilli()
	if delta := second - want; delta < -10 || delta > 10 {
		t.Fatalf("next run=%d want approximately %d", second, want)
	}
}
