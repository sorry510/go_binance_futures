package backtest

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/historicalmarket"
	markettypes "go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
)

const (
	marketConditionHourMillis int64 = int64(time.Hour / time.Millisecond)
)

type earliestKlineSource interface {
	EarliestKline(context.Context, string, string, string) (historicalmarket.Kline, error)
}

type MarketConditionBackfillSummary struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	Stage        string `json:"stage"`
	Progress     int    `json:"progress"`
	BTCStartTime int64  `json:"btc_start_time,omitempty"`
	ETHStartTime int64  `json:"eth_start_time,omitempty"`
	StartTime    int64  `json:"start_time,omitempty"`
	EndTime      int64  `json:"end_time,omitempty"`
	BTCRows      int    `json:"btc_rows"`
	ETHRows      int    `json:"eth_rows"`
	InferredRows int    `json:"inferred_rows"`
	InsertedRows int    `json:"inserted_rows"`
	SkippedRows  int    `json:"skipped_rows"`
	Error        string `json:"error,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	CompletedAt  int64  `json:"completed_at,omitempty"`
}

func (manager *Manager) StartMarketConditionBackfill() (MarketConditionBackfillSummary, error) {
	now := time.Now().UnixMilli()
	manager.marketConditionMu.Lock()
	defer manager.marketConditionMu.Unlock()
	if manager.marketConditionActive != "" {
		active := manager.marketConditionJobs[manager.marketConditionActive]
		if active.Status == "queued" || active.Status == "running" {
			return active, nil
		}
		manager.marketConditionActive = ""
	}
	job := MarketConditionBackfillSummary{
		JobID: "market_condition_" + newRunID()[4:], Status: "queued", Stage: "queued",
		Progress: 0, CreatedAt: now, UpdatedAt: now,
	}
	manager.marketConditionJobs[job.JobID] = job
	manager.marketConditionActive = job.JobID
	go manager.runMarketConditionBackfill(job.JobID)
	return job, nil
}

func (manager *Manager) GetMarketConditionBackfill(jobID string) (MarketConditionBackfillSummary, error) {
	manager.marketConditionMu.Lock()
	defer manager.marketConditionMu.Unlock()
	job, ok := manager.marketConditionJobs[strings.TrimSpace(jobID)]
	if !ok {
		return MarketConditionBackfillSummary{}, fmt.Errorf("MarketCondition backfill job %q not found", jobID)
	}
	return job, nil
}

func (manager *Manager) runMarketConditionBackfill(jobID string) {
	fail := func(err error) {
		manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
			job.Status, job.Stage, job.Progress, job.Error = "failed", "failed", 100, err.Error()
			job.CompletedAt = time.Now().UnixMilli()
		})
	}
	manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
		job.Status, job.Stage, job.Progress = "running", "discovering_listing", 2
	})
	ctx := context.Background()
	source := manager.marketConditionEarliest
	if source == nil {
		source = historicalmarket.BinanceSource{}
	}
	latestClosedHourEnd := time.Now().UTC().Truncate(time.Hour).Add(-time.Millisecond).UnixMilli()
	btcFirst, err := source.EarliestKline(ctx, historicalmarket.MarketFuturesUSDT, "BTCUSDT", "1h")
	if err != nil {
		fail(fmt.Errorf("discover BTCUSDT listing: %w", err))
		return
	}
	ethFirst, err := source.EarliestKline(ctx, historicalmarket.MarketFuturesUSDT, "ETHUSDT", "1h")
	if err != nil {
		fail(fmt.Errorf("discover ETHUSDT listing: %w", err))
		return
	}
	manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
		job.BTCStartTime, job.ETHStartTime = btcFirst.OpenTime, ethFirst.OpenTime
		job.Stage, job.Progress = "loading_btc", 10
	})
	repo := manager.builder.Repository
	if repo == nil {
		repo = historicalmarket.DefaultRepository()
	}
	btcRows, err := repo.LoadKlines(ctx, historicalmarket.MarketFuturesUSDT, "BTCUSDT", "1h", btcFirst.OpenTime, latestClosedHourEnd)
	if err != nil {
		fail(fmt.Errorf("load BTCUSDT 1h history: %w", err))
		return
	}
	manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
		job.BTCRows, job.Stage, job.Progress = len(btcRows), "loading_eth", 40
	})
	ethRows, err := repo.LoadKlines(ctx, historicalmarket.MarketFuturesUSDT, "ETHUSDT", "1h", ethFirst.OpenTime, latestClosedHourEnd)
	if err != nil {
		fail(fmt.Errorf("load ETHUSDT 1h history: %w", err))
		return
	}
	manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
		job.ETHRows, job.Stage, job.Progress = len(ethRows), "inferring", 70
	})
	candidates := inferHistoricalMarketConditions(btcRows, ethRows)
	if len(candidates) == 0 {
		fail(fmt.Errorf("BTCUSDT and ETHUSDT have no aligned closed 1h history"))
		return
	}
	config, err := utils.GetSystemConfig()
	if err != nil {
		fail(fmt.Errorf("load system config: %w", err))
		return
	}
	startTime, endTime := candidates[0].CreatedAt, candidates[len(candidates)-1].CreatedAt
	var existing []models.MarketConditionHistory
	_, err = orm.NewOrm().QueryTable(new(models.MarketConditionHistory)).
		Filter("config_id", config.ID).Filter("created_at__gte", hourBucketStart(startTime)).
		Filter("created_at__lte", endTime).All(&existing)
	if err != nil {
		fail(fmt.Errorf("load existing MarketCondition history: %w", err))
		return
	}
	occupied := make(map[int64]struct{}, len(existing))
	for _, row := range existing {
		occupied[hourBucketStart(row.CreatedAt)] = struct{}{}
	}
	pending := make([]models.MarketConditionHistory, 0, len(candidates))
	skipped := 0
	for _, row := range candidates {
		bucket := hourBucketStart(row.CreatedAt)
		if _, exists := occupied[bucket]; exists {
			skipped++
			continue
		}
		row.ConfigID = config.ID
		pending = append(pending, row)
		occupied[bucket] = struct{}{}
	}
	inserted := 0
	for start := 0; start < len(pending); start += 500 {
		end := start + 500
		if end > len(pending) {
			end = len(pending)
		}
		chunk := pending[start:end]
		if _, err := orm.NewOrm().InsertMulti(500, &chunk); err != nil {
			fail(fmt.Errorf("insert inferred MarketCondition history: %w", err))
			return
		}
		inserted += len(chunk)
		progress := 75 + inserted*24/maxInt(len(pending), 1)
		manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
			job.Stage, job.Progress, job.InsertedRows, job.SkippedRows = "saving", progress, inserted, skipped
		})
	}
	manager.updateMarketConditionBackfill(jobID, func(job *MarketConditionBackfillSummary) {
		job.Status, job.Stage, job.Progress = "succeeded", "completed", 100
		job.StartTime, job.EndTime = startTime, endTime
		job.InferredRows, job.InsertedRows, job.SkippedRows = len(candidates), inserted, skipped
		job.CompletedAt = time.Now().UnixMilli()
	})
}

func (manager *Manager) updateMarketConditionBackfill(jobID string, update func(*MarketConditionBackfillSummary)) {
	manager.marketConditionMu.Lock()
	defer manager.marketConditionMu.Unlock()
	job, ok := manager.marketConditionJobs[jobID]
	if !ok {
		return
	}
	update(&job)
	job.UpdatedAt = time.Now().UnixMilli()
	manager.marketConditionJobs[jobID] = job
	if job.Status == "succeeded" || job.Status == "failed" {
		if manager.marketConditionActive == jobID {
			manager.marketConditionActive = ""
		}
	}
}

func inferHistoricalMarketConditions(btcRows, ethRows []historicalmarket.Kline) []models.MarketConditionHistory {
	btcIndex := make(map[int64]int, len(btcRows))
	for i, row := range btcRows {
		btcIndex[row.CloseTime] = i
	}
	result := make([]models.MarketConditionHistory, 0, len(ethRows))
	for ethIndex, eth := range ethRows {
		btcPos, ok := btcIndex[eth.CloseTime]
		if !ok {
			continue
		}
		btcChange, btcRange := rollingKlineStats(btcRows, btcPos, 24)
		ethChange, ethRange := rollingKlineStats(ethRows, ethIndex, 24)
		condition := inferBTCETHMarketCondition(btcChange, ethChange, (btcRange+ethRange)/2)
		result = append(result, models.MarketConditionHistory{MarketCondition: condition, CreatedAt: eth.CloseTime})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt < result[j].CreatedAt })
	return result
}

func rollingKlineStats(rows []historicalmarket.Kline, index, window int) (changePct, averageRangePct float64) {
	if index < 0 || index >= len(rows) {
		return 0, 0
	}
	start := index - window + 1
	if start < 0 {
		start = 0
	}
	firstOpen := rows[start].Open
	if firstOpen > 0 {
		changePct = (rows[index].Close - firstOpen) / firstOpen * 100
	}
	count := 0
	for i := start; i <= index; i++ {
		if rows[i].Open <= 0 {
			continue
		}
		averageRangePct += (rows[i].High - rows[i].Low) / rows[i].Open * 100
		count++
	}
	if count > 0 {
		averageRangePct /= float64(count)
	}
	return changePct, averageRangePct
}

// inferBTCETHMarketCondition is an explicit approximation for historical replay.
// It uses 24h BTC/ETH direction/relationship plus average 1h range and never
// claims to reproduce the live all-market/LLM regime classifier exactly.
func inferBTCETHMarketCondition(btcChange, ethChange, averageHourlyRange float64) int {
	weighted := btcChange*0.6 + ethChange*0.4
	switch {
	case btcChange >= 0.5 && ethChange <= -0.2:
		return markettypes.MarketConditionBullishDivergence
	case btcChange <= -0.5 && ethChange >= 0.2:
		return markettypes.MarketConditionBearishDivergence
	case weighted >= 5 && btcChange > 0 && ethChange > 0:
		return markettypes.MarketConditionStrongBull
	case weighted <= -5 && btcChange < 0 && ethChange < 0:
		return markettypes.MarketConditionStrongBear
	case btcChange >= 0.5 && ethChange >= 0.5:
		return markettypes.MarketConditionBroadRise
	case btcChange <= -0.5 && ethChange <= -0.5:
		return markettypes.MarketConditionBroadDecline
	case math.Abs(weighted) < 0.5 && averageHourlyRange >= 2:
		return markettypes.MarketConditionHighVolatility
	case math.Abs(weighted) < 0.5 && averageHourlyRange <= 0.35:
		return markettypes.MarketConditionLowVolatility
	case weighted >= 1:
		return markettypes.MarketConditionBull
	case weighted <= -1:
		return markettypes.MarketConditionBear
	default:
		return markettypes.MarketConditionSideways
	}
}

func hourBucketStart(value int64) int64 { return value - value%marketConditionHourMillis }
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
