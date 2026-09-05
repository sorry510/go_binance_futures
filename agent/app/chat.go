package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"go_binance_futures/agent/contextengine"
	"go_binance_futures/agent/conversation"
	agentruntime "go_binance_futures/agent/runtime"
	"go_binance_futures/agent/skill"
	"go_binance_futures/agent/skillconfig"
	"go_binance_futures/agent/skills/symbolanalysis"
	"go_binance_futures/agent/task"
	"go_binance_futures/llm"
)

var defaultConversationStore = conversation.NewORMStore()

type ChatSkill struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Version     string `json:"version"`
}

func ConversationHistory(ctx context.Context, conversationID, currentTaskID string) ([]contextengine.ContextBlock, error) {
	return defaultConversationStore.SuccessfulHistory(ctx, conversationID, currentTaskID, 30)
}
func ChatSkills(ctx context.Context) ([]ChatSkill, error) {
	manager, err := DefaultManager()
	if err != nil {
		return nil, err
	}
	configs, err := (skillconfig.Store{}).List(ctx)
	if err != nil {
		return nil, err
	}
	configByName := make(map[string]struct {
		display, description, kind string
		enabled                    bool
	}, len(configs))
	for _, item := range configs {
		configByName[item.Name] = struct {
			display, description, kind string
			enabled                    bool
		}{item.DisplayName, item.Description, item.Type, item.Enabled == 1}
	}
	result := []ChatSkill{}
	for _, runtimeSkill := range manager.Skills() {
		adapter, ok := runtimeSkill.(skill.ChatAdapter)
		cfg, exists := configByName[runtimeSkill.Name()]
		if !ok || !adapter.ChatEnabled() || !exists || !cfg.enabled {
			continue
		}
		version := skill.ResolveVersionInfo(runtimeSkill, runtimeSkill.SystemPrompt())
		result = append(result, ChatSkill{Name: runtimeSkill.Name(), DisplayName: cfg.display, Description: cfg.description, Type: cfg.kind, Version: version.SkillVersion})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func StartChatMessage(ctx context.Context, conversationID, skillName, content string) (*task.Task, error) {
	conversationID = strings.TrimSpace(conversationID)
	skillName = strings.TrimSpace(skillName)
	content = strings.TrimSpace(content)
	if conversationID == "" || skillName == "" || content == "" {
		return nil, fmt.Errorf("conversation_id, skill and content are required")
	}
	conv, err := defaultConversationStore.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if conv.Skill != conversation.ChatSkill || conv.Status != conversation.StatusActive {
		return nil, fmt.Errorf("conversation %q is not an active chat conversation", conversationID)
	}
	manager, err := DefaultManager()
	if err != nil {
		return nil, err
	}
	tasks, err := manager.List(ctx, task.ListOptions{ConversationID: conversationID, Page: 1, Limit: 100})
	if err != nil {
		return nil, err
	}
	for _, item := range tasks.List {
		if task.IsRunningStatus(item.Status) {
			return nil, fmt.Errorf("conversation already has a running task %s", item.ID)
		}
	}
	var selected skill.Skill
	for _, candidate := range manager.Skills() {
		if candidate.Name() == skillName {
			selected = candidate
			break
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("skill %q is not registered in runtime", skillName)
	}
	adapter, ok := selected.(skill.ChatAdapter)
	if !ok || !adapter.ChatEnabled() {
		return nil, fmt.Errorf("skill %q does not support chat", skillName)
	}
	previousInputs := make([]string, 0, 8)
	for _, previousTask := range tasks.List {
		if previousTask.Skill == skillName && previousTask.Status == task.StatusSucceeded && strings.TrimSpace(previousTask.Input) != "" {
			previousInputs = append(previousInputs, previousTask.Input)
			if len(previousInputs) >= 8 {
				break
			}
		}
	}
	var input string
	if contextual, supportsContext := selected.(skill.ChatContextAdapter); supportsContext {
		input, err = contextual.BuildChatInputWithContext(ctx, content, previousInputs)
	} else {
		input, err = adapter.BuildChatInput(ctx, content)
	}
	if err != nil {
		return nil, err
	}
	item, err := manager.Start(agentruntime.Request{Skill: skillName, Input: input, ConversationID: conversationID})
	if err != nil {
		return nil, err
	}
	if err := defaultConversationStore.AppendOnce(ctx, conversationID, item.ID, skillName, llmMessageUser(content)); err != nil {
		_ = manager.Cancel(context.Background(), item.ID)
		return nil, fmt.Errorf("persist chat user message: %w", err)
	}
	if err := defaultConversationStore.SetTitleFromFirstMessage(ctx, conversationID, content); err != nil {
		return nil, err
	}
	return item, nil
}

func llmMessageUser(content string) llm.Message {
	return llm.Message{Role: llm.RoleUser, Content: content}
}

func ChatConversationStore() *conversation.ORMStore { return defaultConversationStore }

func chatAssistantText(result *agentruntime.Result, item *task.Task) string {
	summary := ""
	raw := []byte(nil)
	if result != nil {
		summary = strings.TrimSpace(result.Summary)
		raw = append(raw, result.Raw...)
	} else if item != nil {
		raw = append(raw, item.Result...)
	}
	if len(raw) == 0 {
		return summary
	}
	if item != nil && item.Skill == symbolanalysis.Name {
		var plan symbolanalysis.TradingPlanV1
		if json.Unmarshal(raw, &plan) == nil && strings.TrimSpace(plan.Symbol) != "" {
			return symbolanalysis.FormatMarkdown(plan)
		}
	}
	var text string
	var stringValue string
	if json.Unmarshal(raw, &stringValue) == nil {
		text = strings.TrimSpace(stringValue)
	} else if json.Valid(raw) {
		var compact bytes.Buffer
		if err := json.Compact(&compact, raw); err == nil {
			text = compact.String()
		} else {
			text = strings.TrimSpace(string(raw))
		}
	} else {
		text = strings.TrimSpace(string(raw))
	}
	if summary != "" && summary != text {
		text = summary + "\n\n" + text
	}
	const maxAssistantBytes = 32 * 1024
	if len(text) > maxAssistantBytes {
		text = text[:maxAssistantBytes] + "\n…"
	}
	return strings.TrimSpace(text)
}

func persistChatCompletion(item *task.Task, result *agentruntime.Result) error {
	if item == nil || strings.TrimSpace(item.ConversationID) == "" || item.Status != task.StatusSucceeded {
		return nil
	}
	conv, err := defaultConversationStore.Get(context.Background(), item.ConversationID)
	if err != nil {
		return err
	}
	if conv.Skill != conversation.ChatSkill {
		return nil
	}
	content := chatAssistantText(result, item)
	if content == "" {
		content = "任务已完成。"
	}
	return defaultConversationStore.AppendOnce(context.Background(), item.ConversationID, item.ID, item.Skill, llm.Message{Role: llm.RoleAssistant, Content: content})
}
