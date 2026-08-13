package webnotification

import (
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const (
	notificationKeepDays        = 30
	notificationCleanupInterval = 6 * time.Hour
)

func CleanupNotificationsBefore(cutoff int64) (int64, error) {
	return orm.NewOrm().QueryTable(new(models.Notification)).
		Filter("create_time__lt", cutoff).
		Delete()
}

func CleanupOldNotifications() (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -notificationKeepDays).UnixMilli()
	return CleanupNotificationsBefore(cutoff)
}

func StartNotificationCleanupTask() {
	cleanupNotifications()

	ticker := time.NewTicker(notificationCleanupInterval)
	defer ticker.Stop()
	for range ticker.C {
		cleanupNotifications()
	}
}

func cleanupNotifications() {
	deleted, err := CleanupOldNotifications()
	if err != nil {
		logs.Error("notification cleanup task error:", err)
		return
	}
	logs.Debug("notification cleanup task done, deleted:", deleted)
}
