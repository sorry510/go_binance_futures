package notify

import (
	"fmt"
	"strings"

	"go_binance_futures/webnotification"
)

func agentAlertContent(params AgentAlertParams) string {
	source := strings.TrimSpace(params.Source)
	if source == "" {
		source = "AI"
	}
	if params.Fallback {
		source += " FALLBACK"
	}
	confirmed := strings.Join(params.ConfirmedBy, "；")
	if confirmed == "" {
		confirmed = "无"
	}
	risks := strings.Join(params.Risks, "；")
	if risks == "" {
		risks = "无"
	}
	return fmt.Sprintf(`
## %s
#### Symbol：%s
#### Signal：%s
#### Severity：%s
#### Source：%s
#### Summary：%s
#### Market Context：%s
#### Confirmed By：%s
#### Risks：%s
#### Signal ID：%s
#### Task ID：%s
`, params.Title, params.Symbol, params.SignalType, params.Severity, source,
		params.Summary, params.MarketContext, confirmed, risks, params.SignalID, params.TaskID)
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
