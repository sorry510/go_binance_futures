package binance

import (
	"go_binance_futures/models"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const marketNoticeLogKeepDays = 10

func CleanupOldMarketNoticeLogs(keepDays int) (int64, error) {
	if keepDays <= 0 {
		keepDays = marketNoticeLogKeepDays
	}

	cutoff := time.Now().AddDate(0, 0, -keepDays).UnixMilli()
	num, err := orm.NewOrm().QueryTable(new(models.FuturesMarketNoticeLog)).Filter("createTime__lt", cutoff).Delete()
	if err != nil {
		return 0, err
	}
	return num, nil
}

func StartMarketNoticeLogCleanupTask() {
	logs.Info("market notice log cleanup task start, keep days:", marketNoticeLogKeepDays)

	deleted, err := CleanupOldMarketNoticeLogs(marketNoticeLogKeepDays)
	if err != nil {
		logs.Error("market notice log cleanup task init error:", err)
	} else {
		logs.Info("market notice log cleanup task init done, deleted:", deleted)
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		deleted, err := CleanupOldMarketNoticeLogs(marketNoticeLogKeepDays)
		if err != nil {
			logs.Error("market notice log cleanup task error:", err)
			continue
		}
		logs.Info("market notice log cleanup task done, deleted:", deleted)
	}
}
