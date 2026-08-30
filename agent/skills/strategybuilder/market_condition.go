package strategybuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go_binance_futures/llm"
)

type marketConditionStrategyRule struct {
	Type   string `json:"type"`
	Code   string `json:"code"`
	Enable bool   `json:"enable"`
}

func RequiresMarketCondition(prompt string) bool {
	normalized := strings.Join(strings.Fields(strings.ToLower(prompt)), "")
	markers := []string{
		"市场趋势", "当前行情", "市场行情", "行情趋势", "市场环境", "市场状态", "大盘趋势", "大盘行情",
		"根据行情", "结合行情", "marketcondition", "markettrend", "marketregime",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
func RequiresMarketConditionForConversation(prompt string, history []llm.Message) bool {
	if RequiresMarketCondition(prompt) {
		return true
	}
	const requirementsMarker = "User requirements:\n"
	for _, message := range history {
		if message.Role != llm.RoleUser {
			continue
		}
		segments := strings.Split(message.Content, requirementsMarker)
		for _, segment := range segments[1:] {
			if end := strings.Index(segment, "\n\nRepair context from the previous attempt."); end >= 0 {
				segment = segment[:end]
			}
			if RequiresMarketCondition(segment) {
				return true
			}
		}
	}
	return false
}

func validateMarketConditionCoverage(raw json.RawMessage) error {
	var payload struct {
		Strategy []marketConditionStrategyRule `json:"strategy"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode MarketCondition strategy coverage: %w", err)
	}
	openingRules := 0
	covered := make([]bool, 12)
	for index, rule := range payload.Strategy {
		if !rule.Enable || (rule.Type != "long" && rule.Type != "short") {
			continue
		}
		openingRules++
		if !strings.Contains(rule.Code, "MarketCondition") {
			return fmt.Errorf("strategy 第 %d 项是启用的开仓规则，必须显式使用 MarketCondition", index+1)
		}
		conditions := marketConditionBranches(rule.Code)
		if len(conditions) == 0 {
			return fmt.Errorf("strategy 第 %d 项必须使用 MarketCondition == \"N\" 的字符串等值条件，可用 &&/|| 组合多个状态", index+1)
		}
		for _, condition := range conditions {
			covered[condition] = true
		}
	}
	if openingRules == 0 {
		return fmt.Errorf("根据市场趋势生成策略时，至少需要一条启用的 long 或 short 开仓规则")
	}

	missing := make([]string, 0, 11)
	for condition := 1; condition <= 11; condition++ {
		if !covered[condition] {
			missing = append(missing, fmt.Sprintf("%d", condition))
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("根据市场趋势生成策略时，开仓规则必须覆盖 MarketCondition 1-11；允许相似状态使用 &&/|| 合并；缺少: %s", strings.Join(missing, ","))
	}
	return nil
}
func marketConditionBranches(code string) []int {
	result := make([]int, 0, 4)
	for condition := 1; condition <= 11; condition++ {
		if marketConditionPattern(condition).MatchString(code) {
			result = append(result, condition)
		}
	}
	return result
}

func marketConditionPattern(condition int) *regexp.Regexp {
	value := regexp.QuoteMeta(fmt.Sprintf("%d", condition))
	pattern := `(?:MarketCondition\s*==\s*["']` + value + `["']|["']` + value + `["']\s*==\s*MarketCondition)`
	return regexp.MustCompile(pattern)
}
