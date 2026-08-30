package permission

import (
	"fmt"
	"strings"
)

type RiskLevel string

const (
	RiskRead  RiskLevel = "read"
	RiskWrite RiskLevel = "write"
	RiskTrade RiskLevel = "trade"
)

type Policy interface {
	Allow(skillName, toolName string, risk RiskLevel) error
}

type StaticPolicy struct {
	MaxRisk      RiskLevel
	Denied       map[string]bool
	WriteAllowed map[string]map[string]bool
	TradeEnabled bool
}

func AllowReadOnly() StaticPolicy {
	return StaticPolicy{MaxRisk: RiskRead}
}

func AllowWrites() StaticPolicy {
	return StaticPolicy{MaxRisk: RiskWrite, WriteAllowed: map[string]map[string]bool{}}
}
func AllowWritesFor(rules map[string][]string) StaticPolicy {
	allowed := make(map[string]map[string]bool, len(rules))
	for skillName, tools := range rules {
		skillName = strings.TrimSpace(skillName)
		if skillName == "" {
			continue
		}
		allowed[skillName] = map[string]bool{}
		for _, toolName := range tools {
			toolName = strings.TrimSpace(toolName)
			if toolName != "" {
				allowed[skillName][toolName] = true
			}
		}
	}
	return StaticPolicy{MaxRisk: RiskWrite, WriteAllowed: allowed}
}

func (policy StaticPolicy) Allow(skillName, toolName string, risk RiskLevel) error {
	skillName = strings.TrimSpace(skillName)
	toolName = strings.TrimSpace(toolName)
	if policy.Denied != nil && policy.Denied[toolName] {
		return fmt.Errorf("tool %q is denied by permission policy", toolName)
	}
	if risk == RiskTrade && !policy.TradeEnabled {
		return fmt.Errorf("trade tool %q is globally disabled", toolName)
	}
	if riskRank(risk) > riskRank(policy.MaxRisk) {
		return fmt.Errorf("tool %q risk %q exceeds allowed risk %q", toolName, risk, policy.MaxRisk)
	}
	if risk == RiskWrite {
		tools := policy.WriteAllowed[skillName]
		if tools == nil || !tools[toolName] {
			return fmt.Errorf("write tool %q is not whitelisted for skill %q", toolName, skillName)
		}
	}
	return nil
}

func riskRank(risk RiskLevel) int {
	switch risk {
	case RiskRead:
		return 1
	case RiskWrite:
		return 2
	case RiskTrade:
		return 3
	default:
		return 100
	}
}
