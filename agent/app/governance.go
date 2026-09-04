package app

import (
	"context"
	"fmt"
	"strings"

	"go_binance_futures/agent/governance"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skillconfig"
	"go_binance_futures/utils"
)

type GovernanceStatus struct {
	Skills        map[string]bool     `json:"skills"`
	Admission     governance.Status   `json:"admission"`
	DefaultBudget agentruntime.Budget `json:"default_budget"`
	TradeEnabled  bool                `json:"trade_enabled"`
}

var defaultSkillStore = skillconfig.Store{}

var defaultLimiter = governance.New(func() governance.Limits {
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		return governance.Limits{PerMinute: 30, PerHour: 300}
	}
	return governance.Limits{PerMinute: cfg.AgentMaxStartsPerMinute, PerHour: cfg.AgentMaxStartsPerHour}
})

func AdmitSkill(skillName string) error {
	skillName = strings.TrimSpace(skillName)
	item, err := defaultSkillStore.GetByName(context.Background(), skillName)
	if err != nil {
		return fmt.Errorf("agent skill %q is not registered in database: %w", skillName, err)
	}
	if item.Enabled != 1 {
		return fmt.Errorf("agent skill %q is disabled", skillName)
	}
	return defaultLimiter.Admit(skillName)
}

func RuntimeBudget(_ string) agentruntime.Budget {
	cfg, err := utils.GetSystemConfig()
	if err != nil {
		return agentruntime.Budget{MaxToolCalls: 12, MaxTotalTokens: 240000}
	}
	budget := agentruntime.Budget{MaxToolCalls: cfg.AgentMaxToolCallsPerTask, MaxTotalTokens: cfg.AgentMaxTokensPerTask}
	if budget.MaxToolCalls <= 0 {
		budget.MaxToolCalls = 12
	}
	if budget.MaxTotalTokens <= 0 {
		budget.MaxTotalTokens = 240000
	}
	return budget
}

func DefaultGovernanceStatus() GovernanceStatus {
	skills := make(map[string]bool)
	if items, err := defaultSkillStore.List(context.Background()); err == nil {
		for _, item := range items {
			skills[item.Name] = item.Enabled == 1
		}
	}
	return GovernanceStatus{
		Skills: skills, Admission: defaultLimiter.Status(),
		DefaultBudget: RuntimeBudget(""), TradeEnabled: false,
	}
}
