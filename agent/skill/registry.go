package skill

import (
	"fmt"
	"strings"
	"sync"
)

type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]Skill)}
}

func (registry *Registry) Register(item Skill) error {
	if registry == nil {
		return fmt.Errorf("skill registry is nil")
	}
	if item == nil || strings.TrimSpace(item.Name()) == "" {
		return fmt.Errorf("skill name is required")
	}
	name := strings.TrimSpace(item.Name())
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.skills[name]; exists {
		return fmt.Errorf("skill %q is already registered", name)
	}
	registry.skills[name] = item
	return nil
}
func (registry *Registry) Get(name string) (Skill, bool) {
	if registry == nil {
		return nil, false
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	item, exists := registry.skills[strings.TrimSpace(name)]
	return item, exists
}

func (registry *Registry) Upsert(item Skill) error {
	if registry == nil {
		return fmt.Errorf("skill registry is nil")
	}
	if item == nil || strings.TrimSpace(item.Name()) == "" {
		return fmt.Errorf("skill name is required")
	}
	registry.mu.Lock()
	registry.skills[strings.TrimSpace(item.Name())] = item
	registry.mu.Unlock()
	return nil
}

func (registry *Registry) Unregister(name string) {
	if registry == nil {
		return
	}
	registry.mu.Lock()
	delete(registry.skills, strings.TrimSpace(name))
	registry.mu.Unlock()
}
