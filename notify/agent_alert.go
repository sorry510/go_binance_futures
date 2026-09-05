package notify

import (
	"fmt"
	"strings"
	"time"

	"go_binance_futures/lang"
	"go_binance_futures/webnotification"
)

const (
	agentAlertGreen = "#008000"
	agentAlertRed   = "#FF0000"
)

type agentAlertFormat int

const (
	agentAlertDingTalk agentAlertFormat = iota
	agentAlertSlack
)

type agentAlertLine struct {
	labelKey  string
	content   string
	important bool
}

type agentAlertTranslator func(string) string

func agentAlertContent(params AgentAlertParams) string {
	return agentAlertDingTalkContent(params)
}

func agentAlertDingTalkContent(params AgentAlertParams) string {
	return renderAgentAlertContent(params, agentAlertDingTalk, lang.Lang)
}

func agentAlertSlackContent(params AgentAlertParams) string {
	return renderAgentAlertContent(params, agentAlertSlack, lang.Lang)
}

func renderAgentAlertContent(params AgentAlertParams, format agentAlertFormat, translate agentAlertTranslator) string {
	source := strings.TrimSpace(params.Source)
	if source == "" {
		source = "AI"
	}
	if strings.EqualFold(source, "RULE") {
		source = translateAgentAlertValue(translate, "notification.source", source)
	}
	if params.Fallback {
		source += " · " + translate("notification.fallback")
	}
	confirmed := strings.Join(params.ConfirmedBy, translate("notification.separator"))
	if confirmed == "" {
		confirmed = translate("notification.none")
	}
	risks := strings.Join(params.Risks, translate("notification.separator"))
	if risks == "" {
		risks = translate("notification.none")
	}
	severity := strings.ToLower(strings.TrimSpace(params.Severity))
	title := strings.TrimSpace(params.Title)
	if title == "" {
		title = "notification.agent_alert_title"
		if params.Fallback {
			title = "notification.rule_alert_title"
		}
	}
	title = translateAgentAlertValue(translate, "", title)
	lines := []agentAlertLine{
		{labelKey: "notification.label.event_time", content: agentAlertEventTime(params.EventTime)},
		{labelKey: "notification.label.symbol", content: params.Symbol},
		{labelKey: "notification.label.signal", content: translateAgentAlertValue(translate, "notification.signal_type", params.SignalType)},
		{labelKey: "notification.label.severity", content: translateAgentAlertValue(translate, "notification.severity", severity), important: severity == "high" || severity == "critical"},
		{labelKey: "notification.label.source", content: source, important: params.Fallback},
		{labelKey: "notification.label.summary", content: params.Summary},
		{labelKey: "notification.label.market_context", content: params.MarketContext},
		{labelKey: "notification.label.confirmed_by", content: confirmed},
		{labelKey: "notification.label.risks", content: risks, important: len(params.Risks) > 0},
		{labelKey: "notification.label.signal_id", content: params.SignalID},
		{labelKey: "notification.label.task_id", content: params.TaskID},
	}
	var content strings.Builder
	content.WriteString("\n## ")
	content.WriteString(title)
	content.WriteByte('\n')
	for _, line := range lines {
		content.WriteString(renderAgentAlertLine(translate(line.labelKey), line.content, line.important, format))
		content.WriteByte('\n')
	}
	return content.String()
}

func agentAlertEventTime(value int64) string {
	if value <= 0 {
		return nowTime()
	}
	return time.UnixMilli(value).In(time.Local).Format("2006-01-02 15:04:05")
}

func renderAgentAlertLine(label, content string, important bool, format agentAlertFormat) string {
	if format == agentAlertSlack {
		indicator := "🟢"
		if important {
			indicator = "🔴"
		}
		return fmt.Sprintf(">*%s*：%s %s", label, indicator, content)
	}
	color := agentAlertGreen
	if important {
		color = agentAlertRed
	}
	return fmt.Sprintf("#### **%s**：<font color=\"%s\">%s</font>", label, color, content)
}

func translateAgentAlertValue(translate agentAlertTranslator, prefix, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	key := value
	if prefix != "" {
		normalized := strings.ToLower(value)
		normalized = strings.NewReplacer("-", "_", " ", "_").Replace(normalized)
		key = prefix + "." + normalized
	}
	translated := translate(key)
	if translated == "" || translated == key {
		return value
	}
	return translated
}

func agentAlertPublishOptions(params AgentAlertParams) webnotification.PublishOptions {
	return webnotification.PublishOptions{
		Level:     agentAlertLevel(params.Severity),
		EventType: "agent_alert_" + strings.TrimSpace(params.SignalType),
		EventID:   strings.TrimSpace(params.EventID),
		SignalID:  strings.TrimSpace(params.SignalID),
		TaskID:    strings.TrimSpace(params.TaskID),
		Symbol:    strings.ToUpper(strings.TrimSpace(params.Symbol)),
	}
}

func agentAlertLevel(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "error"
	case "high":
		return "warning"
	case "medium":
		return "warning"
	default:
		return "info"
	}
}

func agentAlertModule(params AgentAlertParams) string {
	if value := strings.TrimSpace(params.Module); value != "" {
		return value
	}
	return "futures_market_listen"
}
