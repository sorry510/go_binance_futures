package permission

import "fmt"

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
	MaxRisk RiskLevel
	Denied  map[string]bool
}

func AllowReadOnly() StaticPolicy {
	return StaticPolicy{MaxRisk: RiskRead}
}

func AllowWrites() StaticPolicy {
	return StaticPolicy{MaxRisk: RiskWrite}
}

func (policy StaticPolicy) Allow(skillName, toolName string, risk RiskLevel) error {
	if policy.Denied != nil && policy.Denied[toolName] {
		return fmt.Errorf("tool %q is denied by permission policy", toolName)
	}
	if riskRank(risk) > riskRank(policy.MaxRisk) {
		return fmt.Errorf("tool %q risk %q exceeds allowed risk %q", toolName, risk, policy.MaxRisk)
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
