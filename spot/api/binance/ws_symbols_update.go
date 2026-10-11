package binance

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"go_binance_futures/service/mysqlretry"

	spotbinance "github.com/adshao/go-binance/v2"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/config"
	"github.com/beego/beego/v2/core/logs"
)

const (
	spotWSFlushInterval = time.Second
	spotWSBatchSize     = 200
)

var (
	spotWSFlushOnce sync.Once
	spotWSLatestMu  sync.Mutex
	spotWSLatest    = make(map[string]spotbinance.WsMarketStatEvent)
)

func spotQuoteType(symbol string) string {
	switch {
	case strings.HasSuffix(symbol, "USDT"):
		return "USDT"
	case strings.HasSuffix(symbol, "USDC"):
		return "USDC"
	case strings.HasSuffix(symbol, "FDUSD"):
		return "FDUSD"
	default:
		return ""
	}
}

func enqueueSpotWSTickers(event spotbinance.WsAllMarketsStatEvent) {
	spotWSLatestMu.Lock()
	defer spotWSLatestMu.Unlock()
	for _, item := range event {
		if item == nil || item.Symbol == "" || item.LastPrice == "" || spotQuoteType(item.Symbol) == "" {
			continue
		}
		previous, found := spotWSLatest[item.Symbol]
		if !found || previous.Time <= item.Time {
			spotWSLatest[item.Symbol] = *item
		}
	}
}

func startSpotWSFlushTask() {
	spotWSFlushOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(spotWSFlushInterval)
			defer ticker.Stop()
			for range ticker.C {
				flushSpotWSTickers()
			}
		}()
	})
}

func flushSpotWSTickers() {
	spotWSLatestMu.Lock()
	if len(spotWSLatest) == 0 {
		spotWSLatestMu.Unlock()
		return
	}
	rows := make([]spotbinance.WsMarketStatEvent, 0, len(spotWSLatest))
	for symbol, row := range spotWSLatest {
		rows = append(rows, row)
		delete(spotWSLatest, symbol)
	}
	spotWSLatestMu.Unlock()

	sort.Slice(rows, func(i, j int) bool { return rows[i].Symbol < rows[j].Symbol })
	o := orm.NewOrm()
	for start := 0; start < len(rows); start += spotWSBatchSize {
		end := start + spotWSBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		query, args := buildSpotWSUpsertSQL(rows[start:end])
		if query == "" {
			continue
		}
		if err := mysqlretry.Do(context.Background(), func() error {
			_, err := o.Raw(query, args...).Exec()
			return err
		}); err != nil {
			logs.Error("spot ws price database sync failed:", err)
			spotWSLatestMu.Lock()
			for _, row := range rows[start:] {
				newer, exists := spotWSLatest[row.Symbol]
				if !exists || newer.Time < row.Time {
					spotWSLatest[row.Symbol] = row
				}
			}
			spotWSLatestMu.Unlock()
			return
		}
	}
}

func buildSpotWSUpsertSQL(rows []spotbinance.WsMarketStatEvent) (string, []interface{}) {
	parts := make([]string, 0, len(rows))
	args := make([]interface{}, 0, len(rows)*20)
	for _, ticker := range rows {
		quoteType := spotQuoteType(ticker.Symbol)
		if quoteType == "" || ticker.LastPrice == "" {
			continue
		}
		parts = append(parts, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		args = append(args,
			ticker.Symbol,
			ticker.PriceChangePercent,
			ticker.LastPrice,
			ticker.OpenPrice,
			ticker.LowPrice,
			ticker.HighPrice,
			ticker.Time,
			ticker.BaseVolume,
			ticker.QuoteVolume,
			ticker.CloseQty,
			ticker.Count,
			ticker.LastPrice,
			ticker.Time,
			0, // New spot pairs are disabled by default.
			"10", "10", "10", quoteType, "{}", "[]",
		)
	}
	if len(parts) == 0 {
		return "", nil
	}
	columns := "(`symbol`, `percentChange`, `close`, `open`, `low`, `high`, `updateTime`, `baseVolume`, `quoteVolume`, `closeQty`, `tradeCount`, `lastClose`, `lastUpdateTime`, `enable`, `usdt`, `profit`, `loss`, `type`, `technology`, `strategy`)"
	driver, _ := config.String("database::driver")
	updates := []string{"percentChange", "close", "open", "low", "high", "updateTime", "baseVolume", "quoteVolume"}
	var assignment []string
	if driver == "mysql" {
		assignment = append(assignment,
			"`lastClose` = `close`",
			"`lastUpdateTime` = `updateTime`",
		)
		for _, field := range updates {
			assignment = append(assignment, fmt.Sprintf("`%s` = VALUES(`%s`)", field, field))
		}
		return fmt.Sprintf("INSERT INTO `spot_symbols` %s VALUES %s ON DUPLICATE KEY UPDATE %s",
			columns, strings.Join(parts, ", "), strings.Join(assignment, ", ")), args
	}
	assignment = append(assignment,
		"`lastClose` = `spot_symbols`.`close`",
		"`lastUpdateTime` = `spot_symbols`.`updateTime`",
	)
	for _, field := range updates {
		assignment = append(assignment, fmt.Sprintf("`%s` = excluded.`%s`", field, field))
	}
	return fmt.Sprintf("INSERT INTO `spot_symbols` %s VALUES %s ON CONFLICT(`symbol`) DO UPDATE SET %s",
		columns, strings.Join(parts, ", "), strings.Join(assignment, ", ")), args
}
