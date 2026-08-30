package scheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/task"
)

type ConcurrencyPolicy string

const SkipIfRunning ConcurrencyPolicy = "skip_if_running"

type Manager interface {
	Start(agentruntime.Request) (*task.Task, error)
	Get(context.Context, string) (*task.Task, error)
}

type Job struct {
	Name              string
	Skill             string
	Enabled           func() bool
	Interval          func() time.Duration
	Timeout           time.Duration
	ConcurrencyPolicy ConcurrencyPolicy
	BuildInput        func(context.Context) (string, error)
	OnComplete        func(context.Context, *task.Task)
	OnError           func(context.Context, error)
}

type JobStatus struct {
	Name              string            `json:"name"`
	Skill             string            `json:"skill"`
	Enabled           bool              `json:"enabled"`
	IntervalSeconds   int64             `json:"interval_seconds"`
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrency_policy"`
	Running           bool              `json:"running"`
	LastTaskID        string            `json:"last_task_id,omitempty"`
	LastStatus        string            `json:"last_status,omitempty"`
	LastError         string            `json:"last_error,omitempty"`
	RunCount          uint64            `json:"run_count"`
	SkipCount         uint64            `json:"skip_count"`
	LastRunAt         int64             `json:"last_run_at,omitempty"`
	NextRunAt         int64             `json:"next_run_at,omitempty"`
}

type jobState struct {
	JobStatus
	next     time.Time
	interval time.Duration
}

type Scheduler struct {
	manager Manager
	jobs    map[string]Job
	states  map[string]*jobState
	mu      sync.Mutex
	once    sync.Once
}

func New(manager Manager, jobs []Job) (*Scheduler, error) {
	if manager == nil {
		return nil, fmt.Errorf("scheduler manager is required")
	}
	result := &Scheduler{manager: manager, jobs: map[string]Job{}, states: map[string]*jobState{}}
	for _, job := range jobs {
		job.Name = strings.TrimSpace(job.Name)
		job.Skill = strings.TrimSpace(job.Skill)
		if job.Name == "" || job.Skill == "" || job.BuildInput == nil || job.Interval == nil {
			return nil, fmt.Errorf("scheduler job requires name, skill, interval and input builder")
		}
		if _, exists := result.jobs[job.Name]; exists {
			return nil, fmt.Errorf("scheduler job %q already exists", job.Name)
		}
		if job.Timeout <= 0 {
			job.Timeout = 5 * time.Minute
		}
		if job.ConcurrencyPolicy == "" {
			job.ConcurrencyPolicy = SkipIfRunning
		}
		result.jobs[job.Name] = job
		result.states[job.Name] = &jobState{JobStatus: JobStatus{Name: job.Name, Skill: job.Skill, ConcurrencyPolicy: job.ConcurrencyPolicy}}
	}
	return result, nil
}

func (scheduler *Scheduler) Start(ctx context.Context) {
	if scheduler == nil {
		return
	}
	scheduler.once.Do(func() {
		now := time.Now()
		scheduler.mu.Lock()
		for name, job := range scheduler.jobs {
			interval := normalizedInterval(job.Interval())
			scheduler.states[name].interval = interval
			scheduler.states[name].next = now.Add(interval)
		}
		scheduler.mu.Unlock()
		go scheduler.loop(ctx)
	})
}

func (scheduler *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			scheduler.tick(ctx, now)
		}
	}
}

func (scheduler *Scheduler) tick(ctx context.Context, now time.Time) {
	for name, job := range scheduler.jobs {
		enabled := job.Enabled == nil || job.Enabled()
		interval := normalizedInterval(job.Interval())
		scheduler.mu.Lock()
		state := scheduler.states[name]
		state.Enabled = enabled
		state.IntervalSeconds = int64(interval / time.Second)
		if state.next.IsZero() || (state.interval > 0 && state.interval != interval) {
			state.next = now.Add(interval)
		}
		state.interval = interval
		state.NextRunAt = state.next.UnixMilli()
		due := enabled && !now.Before(state.next)
		if due {
			state.next = now.Add(interval)
			state.NextRunAt = state.next.UnixMilli()
		}
		scheduler.mu.Unlock()
		if due {
			scheduler.trigger(ctx, name, job)
		}
	}
}

func (scheduler *Scheduler) Trigger(ctx context.Context, name string) error {
	job, ok := scheduler.jobs[strings.TrimSpace(name)]
	if !ok {
		return fmt.Errorf("scheduler job %q not found", name)
	}
	if job.Enabled != nil && !job.Enabled() {
		return fmt.Errorf("scheduler job %q is disabled", name)
	}
	return scheduler.trigger(ctx, job.Name, job)
}

func (scheduler *Scheduler) trigger(ctx context.Context, name string, job Job) error {
	scheduler.mu.Lock()
	state := scheduler.states[name]
	if job.ConcurrencyPolicy == SkipIfRunning && state.Running {
		state.SkipCount++
		scheduler.mu.Unlock()
		return nil
	}
	state.Running = true
	state.LastRunAt = time.Now().UnixMilli()
	state.RunCount++
	state.LastError = ""
	scheduler.mu.Unlock()

	input, err := job.BuildInput(ctx)
	if err != nil {
		scheduler.finishError(name, job, ctx, err)
		return err
	}
	item, err := scheduler.manager.Start(agentruntime.Request{Skill: job.Skill, Input: input, Metadata: map[string]any{"scheduler_job": name}})
	if err != nil {
		scheduler.finishError(name, job, ctx, err)
		return err
	}
	scheduler.mu.Lock()
	state.LastTaskID = item.ID
	state.LastStatus = string(item.Status)
	scheduler.mu.Unlock()
	go scheduler.wait(context.Background(), name, job, item.ID)
	return nil
}

func (scheduler *Scheduler) wait(ctx context.Context, name string, job Job, taskID string) {
	waitCtx, cancel := context.WithTimeout(ctx, job.Timeout)
	defer cancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		item, err := scheduler.manager.Get(waitCtx, taskID)
		if err != nil {
			scheduler.finishError(name, job, waitCtx, err)
			return
		}
		if task.IsTerminalStatus(item.Status) {
			scheduler.mu.Lock()
			state := scheduler.states[name]
			state.Running = false
			state.LastStatus = string(item.Status)
			if item.Error != "" {
				state.LastError = item.Error
			}
			scheduler.mu.Unlock()
			if job.OnComplete != nil {
				job.OnComplete(context.Background(), item)
			}
			return
		}
		select {
		case <-waitCtx.Done():
			scheduler.finishError(name, job, context.Background(), waitCtx.Err())
			return
		case <-ticker.C:
		}
	}
}

func (scheduler *Scheduler) finishError(name string, job Job, ctx context.Context, err error) {
	scheduler.mu.Lock()
	state := scheduler.states[name]
	state.Running = false
	state.LastStatus = "error"
	state.LastError = err.Error()
	scheduler.mu.Unlock()
	if job.OnError != nil {
		job.OnError(ctx, err)
	}
}

func (scheduler *Scheduler) Status() []JobStatus {
	if scheduler == nil {
		return nil
	}
	scheduler.mu.Lock()
	defer scheduler.mu.Unlock()
	result := make([]JobStatus, 0, len(scheduler.states))
	for name, state := range scheduler.states {
		copy := state.JobStatus
		job := scheduler.jobs[name]
		copy.Enabled = job.Enabled == nil || job.Enabled()
		copy.IntervalSeconds = int64(normalizedInterval(job.Interval()) / time.Second)
		if !state.next.IsZero() {
			copy.NextRunAt = state.next.UnixMilli()
		}
		result = append(result, copy)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func normalizedInterval(value time.Duration) time.Duration {
	if value < time.Second {
		return time.Second
	}
	return value
}
