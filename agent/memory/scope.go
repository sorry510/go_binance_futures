package memory

import (
	"encoding/json"
	"fmt"
	"strings"

	"go_binance_futures/agent/skill"
)

func ScopeFromRequest(skillName string, req skill.Request) Scope {
	scope := Scope{User: DefaultUserScope, Skill: strings.TrimSpace(skillName)}
	if value := metadataString(req.Metadata, "user_id"); value != "" {
		scope.User = value
	}
	scope.Symbol = firstNonEmpty(metadataString(req.Metadata, "symbol"), inputString(req.Input, "symbol"))
	scope.Symbol = strings.ToUpper(strings.TrimSpace(scope.Symbol))
	scope.Strategy = firstNonEmpty(metadataString(req.Metadata, "strategy"), metadataString(req.Metadata, "strategy_id"), metadataString(req.Metadata, "strategy_name"), inputString(req.Input, "strategy_id"), inputString(req.Input, "strategy_name"))
	return normalizeScope(scope)
}

func normalizeScope(scope Scope) Scope {
	scope.User = strings.TrimSpace(scope.User)
	scope.Skill = strings.TrimSpace(scope.Skill)
	scope.Symbol = strings.ToUpper(strings.TrimSpace(scope.Symbol))
	scope.Strategy = strings.TrimSpace(scope.Strategy)
	return scope
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	return scalarString(metadata[key])
}

func inputString(raw, key string) string {
	var value map[string]any
	if json.Unmarshal([]byte(raw), &value) != nil {
		return ""
	}
	return scalarString(value[key])
}

func scalarString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strings.TrimSpace(fmt.Sprintf("%.0f", typed))
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
