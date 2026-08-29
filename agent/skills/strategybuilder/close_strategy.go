package strategybuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type closeStrategyIndicator struct {
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
}

type closeStrategyRule struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Code   string `json:"code"`
	Enable bool   `json:"enable"`
}

func validateCloseStrategyQuality(raw json.RawMessage) error {
	var payload struct {
		Technology map[string][]closeStrategyIndicator `json:"technology"`
		Strategy   []closeStrategyRule                 `json:"strategy"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("decode close strategy quality: %w", err)
	}
	indicatorNames := make([]string, 0)
	for _, indicators := range payload.Technology {
		for _, indicator := range indicators {
			if indicator.Enable && strings.TrimSpace(indicator.Name) != "" {
				indicatorNames = append(indicatorNames, indicator.Name)
			}
		}
	}

	for index, rule := range payload.Strategy {
		if !rule.Enable || (rule.Type != "close_long" && rule.Type != "close_short") {
			continue
		}
		if !hasCloseMarketConfirmation(rule.Code, indicatorNames) {
			return fmt.Errorf("strategy 第 %d 项 %q 的平仓逻辑过于简单：不能只依赖 ROI/持仓盈亏或时间阈值，必须加入指标、K 线结构或市场趋势确认", index+1, rule.Name)
		}
		if pnlOnlyBranch := closePnlOnlyOrBranch(rule.Code, indicatorNames); pnlOnlyBranch != "" {
			return fmt.Errorf("strategy 第 %d 项 %q 存在可独立触发的盈亏分支 %q；ROI/未实现盈亏只能与市场确认条件共同触发平仓", index+1, rule.Name, pnlOnlyBranch)
		}
	}
	return nil
}

func hasCloseMarketConfirmation(code string, indicatorNames []string) bool {
	for _, name := range indicatorNames {
		if containsExprIdentifier(code, name) {
			return true
		}
	}
	for _, name := range []string{
		"MarketCondition", "BasicTrend", "NowSymbolPercentChange", "NowSymbolClose", "NowSymbolOpen", "NowSymbolLow", "NowSymbolHigh",
		"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT",
	} {
		if containsExprIdentifier(code, name) {
			return true
		}
	}
	return strings.Contains(code, "kline_")
}
func closePnlOnlyOrBranch(code string, indicatorNames []string) string {
	for _, branch := range strings.Split(code, "||") {
		if !containsClosePnlReference(branch) {
			continue
		}
		if !hasCloseMarketConfirmation(branch, indicatorNames) {
			return strings.TrimSpace(branch)
		}
	}
	return ""
}

func containsClosePnlReference(code string) bool {
	return containsExprIdentifier(code, "ROI") || strings.Contains(code, "UnrealizedProfit")
}

func containsExprIdentifier(code, name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}
	pattern := `(^|[^A-Za-z0-9_])` + regexp.QuoteMeta(name) + `([^A-Za-z0-9_]|$)`
	return regexp.MustCompile(pattern).MatchString(code)
}
