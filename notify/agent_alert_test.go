package notify

import (
	"strings"
	"testing"
)

func TestRenderAgentAlertContentLocalizesLabelsAndHighlightsImportantValues(t *testing.T) {
	translate := testAgentAlertTranslator(map[string]string{
		"notification.agent_alert_title":     "AI 异常行情分析",
		"notification.none":                  "无",
		"notification.separator":             "；",
		"notification.fallback":              "回退模式",
		"notification.label.event_time":      "事件时间",
		"notification.label.symbol":          "交易对",
		"notification.label.signal":          "信号类型",
		"notification.label.severity":        "严重程度",
		"notification.label.source":          "来源",
		"notification.label.summary":         "摘要",
		"notification.label.market_context":  "市场背景",
		"notification.label.confirmed_by":    "确认依据",
		"notification.label.risks":           "风险",
		"notification.label.signal_id":       "信号 ID",
		"notification.label.task_id":         "任务 ID",
		"notification.signal_type.fast_move": "快速行情波动",
		"notification.severity.critical":     "严重",
	})
	params := AgentAlertParams{
		Title:         "notification.agent_alert_title",
		Symbol:        "BTCUSDT",
		SignalType:    "fast_move",
		Severity:      "critical",
		Summary:       "token、tool、skill 预算异常",
		MarketContext: "BTCUSDT 价格快速上涨",
		ConfirmedBy:   []string{"成交量", "OI"},
		Risks:         []string{"可能快速反转"},
		SignalID:      "sig-1",
		TaskID:        "task-1",
		EventTime:     1700000000000,
		Source:        "AI",
	}

	content := renderAgentAlertContent(params, agentAlertDingTalk, translate)
	for _, expected := range []string{
		"## AI 异常行情分析",
		`#### **事件时间**：<font color="#008000">` + agentAlertEventTime(1700000000000) + `</font>`,
		`#### **交易对**：<font color="#008000">BTCUSDT</font>`,
		`#### **信号类型**：<font color="#008000">快速行情波动</font>`,
		`#### **严重程度**：<font color="#FF0000">严重</font>`,
		`#### **风险**：<font color="#FF0000">可能快速反转</font>`,
		`#### **摘要**：<font color="#008000">token、tool、skill 预算异常</font>`,
		`#### **确认依据**：<font color="#008000">成交量；OI</font>`,
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
	for _, unexpected := range []string{"#### Symbol：", "#### Signal：", "#### Severity：", "#### Market Context："} {
		if strings.Contains(content, unexpected) {
			t.Fatalf("content retains English label %q:\n%s", unexpected, content)
		}
	}
}

func TestRenderAgentAlertContentUsesSlackColorIndicators(t *testing.T) {
	translate := testAgentAlertTranslator(map[string]string{
		"notification.rule_alert_title":      "Abnormal Market Rule Alert",
		"notification.none":                  "none",
		"notification.separator":             "; ",
		"notification.fallback":              "fallback mode",
		"notification.label.event_time":      "Event Time",
		"notification.label.symbol":          "Symbol",
		"notification.label.signal":          "Signal Type",
		"notification.label.severity":        "Severity",
		"notification.label.source":          "Source",
		"notification.label.summary":         "Summary",
		"notification.label.market_context":  "Market Context",
		"notification.label.confirmed_by":    "Confirmed By",
		"notification.label.risks":           "Risks",
		"notification.label.signal_id":       "Signal ID",
		"notification.label.task_id":         "Task ID",
		"notification.source.rule":           "Rule Engine",
		"notification.signal_type.fast_move": "Fast Market Move",
		"notification.severity.high":         "High",
	})
	params := AgentAlertParams{
		Title: "notification.rule_alert_title", Symbol: "ETHUSDT", SignalType: "fast_move",
		Severity: "high", Summary: "fast move", MarketContext: "volatile", Source: "RULE", Fallback: true,
		Risks: []string{"reversal"},
	}

	content := renderAgentAlertContent(params, agentAlertSlack, translate)
	for _, expected := range []string{
		">*Symbol*：🟢 ETHUSDT",
		">*Severity*：🔴 High",
		">*Source*：🔴 Rule Engine · fallback mode",
		">*Risks*：🔴 reversal",
	} {
		if !strings.Contains(content, expected) {
			t.Fatalf("content missing %q:\n%s", expected, content)
		}
	}
	if strings.Contains(content, "<font") {
		t.Fatalf("Slack content must not contain unsupported HTML: %s", content)
	}
}

func testAgentAlertTranslator(values map[string]string) agentAlertTranslator {
	return func(key string) string {
		if value, ok := values[key]; ok {
			return value
		}
		return key
	}
}
