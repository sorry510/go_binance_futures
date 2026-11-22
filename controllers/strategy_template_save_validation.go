package controllers

import (
	"encoding/json"
	"fmt"
	"strings"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/technology"
)

// The UI creates a name-only draft, then saves indicators and rules in
// separate requests. Validate every supplied component immediately without
// preventing drafts or disabled, unfinished rule rows from being edited.
// Once rules are enabled, both the indicator contract and expressions must
// compile and execute against the shared validated synthetic environment.
func validateEditableStrategyTemplate(template models.StrategyTemplates) error {
	if strings.TrimSpace(template.Name) == "" {
		return fmt.Errorf("模板名称不能为空")
	}

	var config technology.TechnologyConfig
	rawTechnology := strings.TrimSpace(template.Technology)
	if rawTechnology != "" && rawTechnology != "null" {
		if !strings.HasPrefix(rawTechnology, "{") {
			return fmt.Errorf("technology 必须是技术指标配置对象")
		}
		if err := json.Unmarshal([]byte(rawTechnology), &config); err != nil {
			return fmt.Errorf("technology JSON 无效: %w", err)
		}
		if err := line.ValidateTechnologyConfig(config); err != nil {
			return fmt.Errorf("technology 配置错误: %w", err)
		}
	}

	rawStrategy := strings.TrimSpace(template.Strategy)
	if rawStrategy == "" || rawStrategy == "null" {
		return nil
	}
	if !strings.HasPrefix(rawStrategy, "[") {
		return fmt.Errorf("strategy 必须是策略规则数组")
	}
	var rules []strategyTemplateImportRule
	if err := json.Unmarshal([]byte(rawStrategy), &rules); err != nil {
		return fmt.Errorf("strategy JSON 无效: %w", err)
	}
	enabled := make([]strategyTemplateImportRule, 0, len(rules))
	usedNames := make(map[string]bool)
	for index, rule := range rules {
		if rule.Enable == nil || !*rule.Enable {
			// A disabled rule is an editable draft, not a trading signal.
			continue
		}
		if strings.TrimSpace(rule.Name) == "" {
			return fmt.Errorf("strategy 第 %d 项已启用但 name 为空", index+1)
		}
		if usedNames[rule.Name] {
			return fmt.Errorf("strategy 第 %d 项的 name %q 重复", index+1, rule.Name)
		}
		usedNames[rule.Name] = true
		switch rule.Type {
		case "long", "short", "close_long", "close_short":
		default:
			return fmt.Errorf("strategy 第 %d 项的 type %q 无效", index+1, rule.Type)
		}
		if strings.TrimSpace(rule.Code) == "" {
			return fmt.Errorf("strategy 第 %d 项 %q 的 code 不能为空", index+1, rule.Name)
		}
		enabled = append(enabled, rule)
	}
	if len(enabled) == 0 {
		return nil
	}
	return validateStrategyTemplateRuleExpressions(config, enabled)
}
