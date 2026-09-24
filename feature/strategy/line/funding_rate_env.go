package line

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"context"

	"go_binance_futures/feature/api/binance"
	"go_binance_futures/models"
	"go_binance_futures/service/binanceapiusage"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
)

const fundingRateEnvLimit = 16

type FundingRateData struct {
	Data []float64 `json:"data"`
	Time []int64   `json:"time"`
}

type fundingRateCacheEntry struct {
	Data      FundingRateData
	ExpiresAt time.Time
}

var fundingRateEnvCache = struct {
	sync.RWMutex
	Items map[string]fundingRateCacheEntry
}{Items: make(map[string]fundingRateCacheEntry)}

func loadFundingRateEnv(symbol string) FundingRateData {
	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	if symbol == "" {
		return FundingRateData{}
	}
	now := time.Now()
	fundingRateEnvCache.RLock()
	cached, ok := fundingRateEnvCache.Items[symbol]
	fundingRateEnvCache.RUnlock()
	if ok && now.Before(cached.ExpiresAt) {
		return cached.Data
	}

	data := loadFundingRateEnvFromDB(symbol, now.UnixMilli())
	expiry := now.Add(30 * time.Minute)
	if !localFundingRateEnvUsable(data, now) {
		remote, err := fetchSettledFundingRateEnv(symbol, now)
		if err != nil {
			logs.Warn("load settled funding history failed for %s: %v", symbol, err)
			expiry = now.Add(5 * time.Minute)
		} else if len(remote.Data) > 0 {
			data = remote
		}
	}

	fundingRateEnvCache.Lock()
	fundingRateEnvCache.Items[symbol] = fundingRateCacheEntry{Data: data, ExpiresAt: expiry}
	fundingRateEnvCache.Unlock()
	return data
}

func localFundingRateEnvUsable(data FundingRateData, now time.Time) bool {
	if len(data.Data) < fundingRateEnvLimit || len(data.Time) == 0 {
		return false
	}
	latest := data.Time[0]
	return latest > 0 && now.Sub(time.UnixMilli(latest)) <= 12*time.Hour
}

func fetchSettledFundingRateEnv(symbol string, now time.Time) (FundingRateData, error) {
	ctx := binanceapiusage.WithSource(context.Background(), "funding_rate_history")
	rows, err := binance.GetFundingRateHistoryContext(ctx, binance.FundingRateParams{
		Symbol:    symbol,
		StartTime: now.Add(-14 * 24 * time.Hour).UnixMilli(),
		EndTime:   now.UnixMilli(),
		Limit:     64,
	})
	if err != nil {
		return FundingRateData{}, err
	}
	result := FundingRateData{
		Data: make([]float64, 0, fundingRateEnvLimit),
		Time: make([]int64, 0, fundingRateEnvLimit),
	}
	for i := len(rows) - 1; i >= 0 && len(result.Data) < fundingRateEnvLimit; i-- {
		row := rows[i]
		if row == nil || row.FundingTime > now.UnixMilli() {
			continue
		}
		rate, parseErr := strconv.ParseFloat(row.FundingRate, 64)
		if parseErr != nil {
			continue
		}
		result.Data = append(result.Data, rate)
		result.Time = append(result.Time, row.FundingTime)
	}
	return result, nil
}

func loadFundingRateEnvFromDB(symbol string, asOf int64) FundingRateData {
	var rows []models.MarketFundingRate
	_, err := orm.NewOrm().QueryTable(new(models.MarketFundingRate)).
		Filter("symbol", symbol).
		Filter("funding_time__lte", asOf).
		OrderBy("-funding_time").
		Limit(fundingRateEnvLimit).
		All(&rows)
	if err != nil {
		return FundingRateData{}
	}
	result := FundingRateData{
		Data: make([]float64, 0, len(rows)),
		Time: make([]int64, 0, len(rows)),
	}
	for _, row := range rows {
		result.Data = append(result.Data, row.FundingRate)
		result.Time = append(result.Time, row.FundingTime)
	}
	return result
}
