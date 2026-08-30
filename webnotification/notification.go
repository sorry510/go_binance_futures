package webnotification

import (
	"regexp"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

type PublishOptions struct {
	Level             string
	EventType         string
	EventID           string
	SignalID          string
	TaskID            string
	Symbol            string
	LiquidationSide   string
	AggregateNotional float64
	OrderCount        int
	WindowStart       int64
	WindowEnd         int64
}

func Publish(module, content string) (*models.Notification, error) {
	return PublishWithOptions(module, content, PublishOptions{})
}

func PublishWithOptions(module, content string, options PublishOptions) (*models.Notification, error) {
	plainContent := normalizeContent(content)
	notification := &models.Notification{
		Title:             extractTitle(plainContent),
		Content:           plainContent,
		Module:            strings.TrimSpace(module),
		Level:             strings.TrimSpace(options.Level),
		EventType:         strings.TrimSpace(options.EventType),
		EventID:           strings.TrimSpace(options.EventID),
		SignalID:          strings.TrimSpace(options.SignalID),
		TaskID:            strings.TrimSpace(options.TaskID),
		Symbol:            strings.ToUpper(strings.TrimSpace(options.Symbol)),
		LiquidationSide:   strings.ToLower(strings.TrimSpace(options.LiquidationSide)),
		AggregateNotional: options.AggregateNotional,
		OrderCount:        options.OrderCount,
		WindowStart:       options.WindowStart,
		WindowEnd:         options.WindowEnd,
		CreateTime:        time.Now().UnixMilli(),
	}
	if notification.Module == "" {
		notification.Module = "system"
	}
	if notification.Level == "" {
		notification.Level = "info"
	}

	id, err := orm.NewOrm().Insert(notification)
	if err != nil {
		return nil, err
	}
	notification.ID = id
	if err := Broadcast("notification", notification); err != nil {
		return notification, err
	}
	return notification, nil
}

func normalizeContent(content string) string {
	content = htmlTagPattern.ReplaceAllString(content, "")
	replacer := strings.NewReplacer("**", "", "####", "", "###", "", "##", "", ">", "")
	lines := strings.Split(replacer.Replace(content), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)
		if line == "" || lowerLine == "author" || strings.HasPrefix(lowerLine, "author ") {
			continue
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func extractTitle(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if title := strings.TrimSpace(line); title != "" {
			runes := []rune(title)
			if len(runes) > 255 {
				return string(runes[:255])
			}
			return title
		}
	}
	return "系统通知"
}
