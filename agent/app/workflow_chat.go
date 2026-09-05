package app

import (
	"context"
	"fmt"
	"strings"

	workflowSkills "go_binance_futures/agent/skills/workflows"
	workflowservice "go_binance_futures/service/workflow"
)

type workflowChatSkill struct {
	*workflowSkills.Definition
	buildChatInput func(context.Context, string) (string, error)
}

func (skill *workflowChatSkill) ChatEnabled() bool { return true }

func (skill *workflowChatSkill) BuildChatInput(ctx context.Context, content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("chat content is required")
	}
	if skill.buildChatInput == nil {
		return "", fmt.Errorf("skill %q does not have a chat input adapter", skill.Name())
	}
	return skill.buildChatInput(ctx, content)
}

func newWorkflowChatSkill(definition *workflowSkills.Definition) *workflowChatSkill {
	adapter := &workflowChatSkill{Definition: definition}
	switch definition.Name() {
	case workflowSkills.MarketScanName:
		adapter.buildChatInput = workflowservice.BuildMarketScanChatInput
	case workflowSkills.StrategyReviewName:
		adapter.buildChatInput = workflowservice.BuildStrategyReviewChatInput
	case workflowSkills.DailyMarketBriefName:
		adapter.buildChatInput = workflowservice.BuildDailyMarketBriefChatInput
	}
	return adapter
}
