package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type ListOptions struct {
	Skill          string
	Status         string
	ConversationID string
	Page           int
	Limit          int
}

type ListResult struct {
	Page  int     `json:"page"`
	Limit int     `json:"limit"`
	Total int64   `json:"total"`
	List  []*Task `json:"list"`
}

type Store interface {
	Save(ctx context.Context, item *Task) error
	Get(ctx context.Context, id string) (*Task, error)
	List(ctx context.Context, options ListOptions) (ListResult, error)
	MarkInterrupted(ctx context.Context, at time.Time) (int64, error)
}

type MemoryStore struct {
	mu    sync.RWMutex
	items map[string]*Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{items: make(map[string]*Task)}
}

func (store *MemoryStore) Save(ctx context.Context, item *Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if store == nil || item == nil || item.ID == "" {
		return fmt.Errorf("task store requires a task id")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	store.items[item.ID] = clone(item)
	return nil
}

func (store *MemoryStore) Get(ctx context.Context, id string) (*Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if store == nil {
		return nil, fmt.Errorf("task store is nil")
	}
	store.mu.RLock()
	defer store.mu.RUnlock()
	item := store.items[id]
	if item == nil {
		return nil, fmt.Errorf("task %q not found", id)
	}
	return clone(item), nil
}

func (store *MemoryStore) List(ctx context.Context, options ListOptions) (ListResult, error) {
	if err := ctx.Err(); err != nil {
		return ListResult{}, err
	}
	page, limit := normalizeListOptions(options)
	store.mu.RLock()
	items := make([]*Task, 0, len(store.items))
	for _, item := range store.items {
		if options.Skill != "" && item.Skill != strings.TrimSpace(options.Skill) {
			continue
		}
		if options.Status != "" && string(item.Status) != strings.TrimSpace(options.Status) {
			continue
		}
		if options.ConversationID != "" && item.ConversationID != strings.TrimSpace(options.ConversationID) {
			continue
		}
		items = append(items, clone(item))
	}
	store.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	total := len(items)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	return ListResult{Page: page, Limit: limit, Total: int64(total), List: items[start:end]}, nil
}

func (store *MemoryStore) MarkInterrupted(ctx context.Context, at time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	var count int64
	for _, item := range store.items {
		if !IsRunningStatus(item.Status) {
			continue
		}
		markInterrupted(item, at)
		count++
	}
	return count, nil
}

func normalizeListOptions(options ListOptions) (int, int) {
	page, limit := options.Page, options.Limit
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func markInterrupted(item *Task, at time.Time) {
	at = at.UTC()
	item.Status = StatusInterrupted
	item.Stage = "interrupted"
	item.Error = "process restarted before task completed"
	item.UpdatedAt = at
	item.CompletedAt = &at
	item.Events = append(item.Events, Event{
		TaskID: item.ID, Skill: item.Skill, Stage: "interrupted", Progress: item.Progress,
		Round: item.Round, Status: string(StatusInterrupted), Message: item.Error, Time: at,
	})
}

func clone(item *Task) *Task {
	copyItem := *item
	copyItem.Result = append([]byte(nil), item.Result...)
	copyItem.Events = append([]Event(nil), item.Events...)
	return &copyItem
}
