package symbolanalysis

import (
	"fmt"
	"strconv"
	"strings"

	markettypes "go_binance_futures/types"
)

func FormatMarkdown(plan TradingPlanV1) string {
	var out strings.Builder
	out.WriteString("## ")
	out.WriteString(plan.Symbol)
	out.WriteString(" 单币分析\n\n")
	if strings.TrimSpace(plan.AsOf) != "" {
		out.WriteString("> 分析时间：")
		out.WriteString(plan.AsOf)
		out.WriteString("\n\n")
	}
	out.WriteString("### 结论\n")
	out.WriteString("- **方向**：")
	out.WriteString(directionLabel(plan.Direction))
	out.WriteString("\n- **置信度**：")
	out.WriteString(fmt.Sprintf("%.0f%%", plan.Confidence*100))
	out.WriteString("\n- **市场环境**：")
	out.WriteString(marketConditionLabel(plan.MarketCondition))
	out.WriteString("\n\n")
	if strings.TrimSpace(plan.Summary) != "" {
		out.WriteString(plan.Summary)
		out.WriteString("\n\n")
	}
	out.WriteString("### 交易计划\n")
	writeTradingPlan(&out, plan)
	out.WriteString("\n### 风险与失效条件\n")
	writeList(&out, "风险", plan.Risks)
	writeList(&out, "失效条件", plan.InvalidationConditions)
	out.WriteString("\n### 证据\n")
	if len(plan.Evidence) == 0 {
		out.WriteString("- 无\n")
	} else {
		for _, item := range plan.Evidence {
			out.WriteString("- **")
			out.WriteString(strings.TrimSpace(item.Source))
			out.WriteString("**：")
			out.WriteString(strings.TrimSpace(item.Finding))
			out.WriteString("\n")
		}
	}
	if len(plan.DataMissing) > 0 {
		out.WriteString("\n### 数据缺失\n")
		for _, item := range plan.DataMissing {
			out.WriteString("- ")
			out.WriteString(strings.TrimSpace(item))
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String())
}

func writeTradingPlan(out *strings.Builder, plan TradingPlanV1) {
	if len(plan.EntryZones) == 0 && plan.StopLoss == nil && len(plan.TakeProfits) == 0 {
		out.WriteString("- **策略**：当前不建议建立方向仓位\n")
	} else {
		for i, zone := range plan.EntryZones {
			out.WriteString(fmt.Sprintf("- **入场区 %d**：%s - %s\n", i+1, formatPrice(zone.Low), formatPrice(zone.High)))
		}
		if plan.StopLoss != nil {
			out.WriteString("- **止损**：")
			out.WriteString(formatPrice(*plan.StopLoss))
			out.WriteString("\n")
		}
		for i, price := range plan.TakeProfits {
			out.WriteString(fmt.Sprintf("- **止盈 %d**：%s\n", i+1, formatPrice(price)))
		}
	}
	if strings.TrimSpace(plan.LongTrigger) != "" {
		out.WriteString("- **做多触发**：")
		out.WriteString(strings.TrimSpace(plan.LongTrigger))
		out.WriteString("\n")
	}
	if strings.TrimSpace(plan.ShortTrigger) != "" {
		out.WriteString("- **做空触发**：")
		out.WriteString(strings.TrimSpace(plan.ShortTrigger))
		out.WriteString("\n")
	}
}

func writeList(out *strings.Builder, label string, values []string) {
	if len(values) == 0 {
		out.WriteString("- **")
		out.WriteString(label)
		out.WriteString("**：无\n")
		return
	}
	for _, value := range values {
		out.WriteString("- **")
		out.WriteString(label)
		out.WriteString("**：")
		out.WriteString(strings.TrimSpace(value))
		out.WriteString("\n")
	}
}

func directionLabel(direction string) string {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "long":
		return "🟢 做多"
	case "short":
		return "🔴 做空"
	default:
		return "⚪ 观望"
	}
}

func marketConditionLabel(condition *int) string {
	if condition == nil {
		return "未知"
	}
	name := markettypes.MarketConditionName(*condition)
	if name == "" {
		return strconv.Itoa(*condition)
	}
	return fmt.Sprintf("%s（%d）", name, *condition)
}

func formatPrice(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
