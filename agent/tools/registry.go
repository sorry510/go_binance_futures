package tools

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (registry *Registry) Register(tool Tool) error {
	if registry == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if tool == nil || strings.TrimSpace(tool.Name()) == "" {
		return fmt.Errorf("tool name is required")
	}
	name := strings.TrimSpace(tool.Name())
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.tools[name]; exists {
		return fmt.Errorf("tool %q is already registered", name)
	}
	registry.tools[name] = tool
	return nil
}
func (registry *Registry) Get(name string) (Tool, bool) {
	if registry == nil {
		return nil, false
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	tool, exists := registry.tools[strings.TrimSpace(name)]
	return tool, exists
}

func (registry *Registry) Upsert(tool Tool) error {
	if registry == nil {
		return fmt.Errorf("tool registry is nil")
	}
	if tool == nil || strings.TrimSpace(tool.Name()) == "" {
		return fmt.Errorf("tool name is required")
	}
	registry.mu.Lock()
	registry.tools[strings.TrimSpace(tool.Name())] = tool
	registry.mu.Unlock()
	return nil
}

func (registry *Registry) Unregister(name string) {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	delete(registry.tools, strings.TrimSpace(name))
	registry.mu.Unlock()
}

func (registry *Registry) List() []Tool {
	if registry == nil {
		return nil
	}
	registry.mu.RLock()
	items := make([]Tool, 0, len(registry.tools))
	for _, item := range registry.tools {
		items = append(items, item)
	}
	registry.mu.RUnlock()
	sort.Slice(items, func(i, j int) bool { return items[i].Name() < items[j].Name() })
	return items
}
