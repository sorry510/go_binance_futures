package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/task"
)

type Service struct{ Store *ORMStore }

func (service Service) store() *ORMStore {
	if service.Store != nil {
		return service.Store
	}
	return NewORMStore()
}

func (service Service) Context(ctx context.Context, skillName string, req skill.Request) ([]contextengine.ContextBlock, error) {
	scope := ScopeFromRequest(skillName, req)
	return service.store().ContextBlocks(ctx, QueryScope{User: scope.User, Skill: scope.Skill, Symbol: scope.Symbol, Strategy: scope.Strategy, Limit: 20})
}

func (service Service) PersistTaskSummary(ctx context.Context, reqSkill string, input string, metadata map[string]any, item *task.Task, summary string) (*Memory, error) {
	if item == nil || item.Status != task.StatusSucceeded {
		return nil, nil
	}
	if err := ValidateAutomaticWrite(TypeTaskSummary); err != nil {
		return nil, err
	}
	existing, err := service.store().List(ctx, ListOptions{Type: string(TypeTaskSummary), SourceTaskID: item.ID, IncludeExpired: true, Page: 1, Limit: 1})
	if err != nil {
		return nil, err
	}
	if len(existing.List) > 0 {
		copy := existing.List[0]
		return &copy, nil
	}
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = extractSummary(item.Result)
	}
	if summary == "" {
		return nil, nil
	}
	req := skill.Request{Input: input, Metadata: metadata}
	scope := ScopeFromRequest(reqSkill, req)
	content := strings.TrimSpace(summary)
	if len(content) > 4000 {
		content = content[:4000]
	}
	created, err := service.store().Create(ctx, CreateInput{Type: TypeTaskSummary, Scope: scope, SourceTaskID: item.ID, Confidence: 1, Status: StatusActive, Content: content, ExpiresAt: time.Now().UTC().Add(DefaultTaskSummaryTTL).UnixMilli()})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func extractSummary(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var object map[string]any
	if json.Unmarshal(raw, &object) == nil {
		for _, key := range []string{"summary", "reason", "analysis"} {
			if value, ok := object[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func ValidateAutomaticWrite(memoryType Type) error {
	switch memoryType {
	case TypeTaskSummary, TypeLesson:
		return nil
	case TypeStrategyFact, TypeMarketHypothesis:
		return fmt.Errorf("%s cannot be permanently saved automatically; create a candidate or require user approval", memoryType)
	case TypeUserPreference:
		return fmt.Errorf("user_preference requires explicit user management")
	default:
		return fmt.Errorf("unsupported automatic memory type %q", memoryType)
	}
}
