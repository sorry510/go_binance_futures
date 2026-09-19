package app

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go_binance_futures/agent/conversation"
	generalchat "go_binance_futures/agent/skills/generalchat"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

const (
	ChatSkillModeAuto     = "auto"
	ChatSkillModeExplicit = "explicit"
)

type ChatMessageOptions struct {
	SkillMode     string
	Skill         string
	Content       string
	Symbol        string
	ModelConfigID *int64
}

func ConversationChatSkillNames(ctx context.Context, conversationID string) ([]string, error) {
	return defaultConversationStore.ChatSkillNames(ctx, conversationID)
}
func AttachConversationChatSkill(ctx context.Context, conversationID, skillName string) error {
	skillName = strings.TrimSpace(skillName)
	available, err := ChatSkills(ctx)
	if err != nil {
		return err
	}
	found := false
	for _, item := range available {
		if item.Name == skillName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("skill %q is not available for chat", skillName)
	}
	return defaultConversationStore.AttachChatSkill(ctx, conversationID, skillName)
}

func RemoveConversationChatSkill(ctx context.Context, conversationID, skillName string) error {
	return defaultConversationStore.RemoveChatSkill(ctx, conversationID, skillName)
}

func UpdateConversationChatModel(ctx context.Context, conversationID string, modelConfigID int64) error {
	if err := validateChatModelConfig(ctx, modelConfigID); err != nil {
		return err
	}
	return defaultConversationStore.SetModelConfigID(ctx, conversationID, modelConfigID)
}
func StartChatMessageWithOptions(ctx context.Context, conversationID string, options ChatMessageOptions) (*task.Task, error) {
	conversationID = strings.TrimSpace(conversationID)
	conv, err := defaultConversationStore.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Skill != conversation.ChatSkill || conv.Status != conversation.StatusActive {
		return nil, fmt.Errorf("conversation %q is not an active chat conversation", conversationID)
	}

	mode := strings.ToLower(strings.TrimSpace(options.SkillMode))
	if mode == "" {
		if strings.TrimSpace(options.Skill) != "" {
			mode = ChatSkillModeExplicit
		} else {
			mode = ChatSkillModeAuto
		}
	}
	attached, err := defaultConversationStore.ChatSkillNames(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	selectedSkill, err := resolveChatSkill(ctx, mode, options.Skill, options.Content, options.Symbol, attached)
	if err != nil {
		return nil, err
	}
	modelConfigID := conv.ModelConfigID
	if options.ModelConfigID != nil {
		modelConfigID = *options.ModelConfigID
	}
	if err := validateChatModelConfig(ctx, modelConfigID); err != nil {
		return nil, err
	}
	return startChatMessage(ctx, conversationID, selectedSkill, options.Content, options.Symbol, modelConfigID)
}

func validateChatModelConfig(ctx context.Context, modelConfigID int64) error {
	if modelConfigID == 0 {
		return nil
	}
	if modelConfigID < 0 {
		return fmt.Errorf("model_config_id must be >= 0")
	}
	row, err := (llm.Store{}).Get(ctx, modelConfigID)
	if err != nil {
		return fmt.Errorf("load chat model config %d: %w", modelConfigID, err)
	}
	if row.Enabled != 1 && row.RouterCandidate != 1 {
		return fmt.Errorf("LLM model config %d is not enabled for routing", modelConfigID)
	}
	return nil
}
func resolveChatSkill(ctx context.Context, mode, requestedSkill, content, symbol string, attached []string) (string, error) {
	available, err := ChatSkills(ctx)
	if err != nil {
		return "", err
	}
	catalog := make(map[string]ChatSkill, len(available))
	for _, item := range available {
		catalog[item.Name] = item
	}
	attachedSet := make(map[string]bool, len(attached)+1)
	for _, name := range attached {
		attachedSet[strings.TrimSpace(name)] = true
	}
	attachedSet[generalchat.Name] = true

	switch mode {
	case ChatSkillModeExplicit:
		name := strings.TrimSpace(requestedSkill)
		if name == "" {
			return "", fmt.Errorf("skill is required in explicit mode")
		}
		if !attachedSet[name] {
			return "", fmt.Errorf("skill %q is not attached to this conversation", name)
		}
		if _, ok := catalog[name]; !ok {
			return "", fmt.Errorf("skill %q is not available for chat", name)
		}
		return name, nil
	case ChatSkillModeAuto:
		return autoSelectChatSkill(chatRoutingText(content, symbol), attachedSet, catalog), nil
	default:
		return "", fmt.Errorf("unsupported skill_mode %q", mode)
	}
}
func chatRoutingText(content, symbol string) string {
	text := strings.TrimSpace(content)
	if selectedSymbol := strings.ToUpper(strings.TrimSpace(symbol)); selectedSymbol != "" {
		if text != "" {
			text += " "
		}
		text += selectedSymbol
	}
	return text
}

func autoSelectChatSkill(content string, attached map[string]bool, catalog map[string]ChatSkill) string {
	text := strings.ToLower(strings.TrimSpace(content))
	bestName, bestScore := generalchat.Name, 0
	names := make([]string, 0, len(attached))
	for name := range attached {
		if name != generalchat.Name {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	for _, name := range names {
		item, ok := catalog[name]
		if !ok {
			continue
		}
		score := chatSkillScore(text, item)
		if score > bestScore {
			bestName, bestScore = name, score
		}
	}
	if bestScore < 6 {
		return generalchat.Name
	}
	return bestName
}
func chatSkillScore(text string, item ChatSkill) int {
	name := strings.ToLower(strings.TrimSpace(item.Name))
	display := strings.ToLower(strings.TrimSpace(item.DisplayName))
	score := 0
	if name != "" && (strings.Contains(text, "@"+name) || strings.Contains(text, "/"+name)) {
		return 100
	}
	if name != "" && strings.Contains(text, name) {
		score += 12
	}
	if len([]rune(display)) >= 2 && strings.Contains(text, display) {
		score += 10
	}
	for _, keyword := range chatSkillKeywords(name) {
		if strings.Contains(text, keyword) {
			score += 3
		}
	}
	return score
}

func chatSkillKeywords(name string) []string {
	switch name {
	case "symbol_analysis":
		return []string{"usdt", "分析", "走势", "技术", "支撑", "阻力", "多空", "止损", "止盈"}
	case "symbol_analysis_team":
		return []string{"团队分析", "综合分析", "多智能体", "资金流", "新闻影响"}
	case "market_scan":
		return []string{"扫描", "选币", "市场机会", "market scan", "候选币"}
	case "strategy_builder":
		return []string{"写策略", "策略构建", "strategy builder", "策略json"}
	case "alert_analysis":
		return []string{"报警", "预警", "告警", "alert"}
	case "market_regime":
		return []string{"市场环境", "市场状态", "regime", "风险偏好"}
	default:
		return nil
	}
}
