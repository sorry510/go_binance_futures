package tools

import (
	"fmt"
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
