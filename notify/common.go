package notify

import (
	"time"

	"github.com/beego/beego/v2/core/config"
)

func nowTime() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func getStatusColor(status string) string {
	var color string
	switch status {
		case "success":
			color = "#008000"
		case "fail":
			color = "#FF0000"
		default:
			color = "#008000"
	}
	return color
}

func GetNotifyChannel() (pusher Pusher) {
	var notification_channel, _ = config.String("notification::channel") // 通知方式
	switch (notification_channel) {
		case "dingding":
			pusher = DingDing{}
		case "slack":
			pusher = Slack{}
		default:
			pusher = DingDing{}
	}
	return pusher
}
