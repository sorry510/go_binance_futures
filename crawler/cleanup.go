package crawler

import (
	"fmt"
	"go_binance_futures/models"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

func StartNewsCleanupTask() {
	if !getNewsCleanupEnable() {
		logs.Info("news cleanup task disabled")
		return
	}

	keepDays := getNewsKeepDays()
	if keepDays <= 0 {
		keepDays = 7
	}

	scheduleTime := getNewsCleanupTime()
	logs.Info("news cleanup task start, keep days:", keepDays, "schedule:", scheduleTime)

	for {
		wait := durationUntilNext(scheduleTime)
		logs.Info("news cleanup task next run after:", wait)
		timer := time.NewTimer(wait)
		<-timer.C

		deleted, err := cleanOldNews(keepDays)
		if err != nil {
			logs.Error("news cleanup task error:", err)
			continue
		}
		logs.Info("news cleanup task done, deleted:", deleted)
	}
}

func cleanOldNews(keepDays int) (int64, error) {
	o := orm.NewOrm()
	cutoff := time.Now().AddDate(0, 0, -keepDays).UnixMilli()

	num, err := o.QueryTable(new(models.News)).Filter("published_at__lt", cutoff).Delete()
	if err != nil {
		return 0, err
	}
	return num, nil
}

func getNewsCleanupEnable() bool {
	enable, err := config.Int("crawler::news_cleanup_enable")
	if err != nil {
		return true
	}
	return enable == 1
}

func getNewsKeepDays() int {
	keepDays, err := config.Int("crawler::news_keep_days")
	if err != nil {
		return 7
	}
	return keepDays
}

func getNewsCleanupTime() string {
	val, err := config.String("crawler::news_cleanup_time")
	if err != nil {
		return "03:00"
	}
	val = strings.TrimSpace(val)
	if val == "" {
		return "03:00"
	}
	if _, err = time.Parse("15:04", val); err != nil {
		logs.Warn("invalid crawler::news_cleanup_time, fallback to 03:00, value:", val)
		return "03:00"
	}
	return val
}

func durationUntilNext(scheduleTime string) time.Duration {
	now := time.Now()
	parts := strings.Split(scheduleTime, ":")
	hour, errHour := strconv.Atoi(parts[0])
	min, errMin := strconv.Atoi(parts[1])
	if errHour != nil || errMin != nil {
		logs.Warn("parse schedule time failed, fallback to 03:00")
		hour = 3
		min = 0
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	d := next.Sub(now)
	if d <= 0 {
		return time.Hour
	}
	return d
}

func RunNewsCleanupNow() error {
	keepDays := getNewsKeepDays()
	if keepDays <= 0 {
		keepDays = 7
	}
	num, err := cleanOldNews(keepDays)
	if err != nil {
		return err
	}
	logs.Info(fmt.Sprintf("news cleanup manual run done, deleted: %d", num))
	return nil
}
