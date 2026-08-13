package webnotification

import (
	"regexp"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

var htmlTagPattern = regexp.MustCompile(`<[^>]+>`)

func Publish(module, content string) (*models.Notification, error) {
	plainContent := normalizeContent(content)
	notification := &models.Notification{
		Title:      extractTitle(plainContent),
		Content:    plainContent,
		Module:     strings.TrimSpace(module),
		Level:      "info",
		CreateTime: time.Now().UnixMilli(),
	}
	if notification.Module == "" {
		notification.Module = "system"
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
