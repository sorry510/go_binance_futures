package task

import (
	"context"
	"fmt"
	"sync"
)

type Store interface {
	Save(ctx context.Context, item *Task) error
	Get(ctx context.Context, id string) (*Task, error)
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

func clone(item *Task) *Task {
	copyItem := *item
	copyItem.Result = append([]byte(nil), item.Result...)
	copyItem.Events = append([]Event(nil), item.Events...)
	return &copyItem
}
