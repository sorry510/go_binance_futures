package strategybuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type readableStrategyRule struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Enable bool   `json:"enable"`
}

var letStatementPattern = regexp.MustCompile(`(?m)^\s*let\s+[A-Za-z_][A-Za-z0-9_]*\s*=`)

func validateStrategyCodeReadability(raw json.RawMessage) error {
	var payload struct {
		Strategy []readableStrategyRule `json:"strategy"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode strategy readability: %w", err)
	}
	for index, rule := range payload.Strategy {
		if !rule.Enable || !isComplexStrategyCode(rule.Code) {
			continue
		}
		if !strings.Contains(rule.Code, "\n") {
			return fmt.Errorf("strategy 第 %d 项 %q 的 code 过于复杂，不能全部写在一行；请用多行 let 变量拆解条件后再组合最终布尔表达式", index+1, rule.Name)
		}
		if len(letStatementPattern.FindAllStringIndex(rule.Code, -1)) < 2 {
			return fmt.Errorf("strategy 第 %d 项 %q 的 code 过于复杂；请至少使用两个有意义的 let 局部变量拆解行情、趋势、动量或结构条件", index+1, rule.Name)
		}
	}
	return nil
}

func isComplexStrategyCode(code string) bool {
	trimmed := strings.TrimSpace(code)
	logicalOperators := strings.Count(trimmed, "&&") + strings.Count(trimmed, "||")
	return len(trimmed) >= 180 || logicalOperators >= 6
}
