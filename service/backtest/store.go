package backtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"go_binance_futures/models"
	"go_binance_futures/service/historicalmarket"
	strategyservice "go_binance_futures/service/strategy"

	"github.com/beego/beego/v2/client/orm"
)

type Manager struct {
	builder                 DatasetBuilder
	engine                  Engine
	mu                      sync.Mutex
	cancel                  map[string]context.CancelFunc
	prefetchMu              sync.Mutex
	prefetchJobs            map[string]PrefetchSummary
	prefetchActive          string
	marketConditionMu       sync.Mutex
	marketConditionJobs     map[string]MarketConditionBackfillSummary
	marketConditionActive   string
	marketConditionEarliest earliestKlineSource
	deleteMu                sync.Mutex
	deleting                map[string]bool
}

var defaultManagerOnce sync.Once
var defaultManager *Manager

func DefaultManager() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager(DatasetBuilder{Repository: historicalmarket.DefaultRepository()})
		_ = defaultManager.markInterrupted()
		defaultManager.resumeDeletingRuns()
	})
	return defaultManager
}
func NewManager(builder DatasetBuilder) *Manager {
	return &Manager{
		builder: builder, engine: Engine{ResolutionProviderFactory: historicalmarket.DefaultAdaptiveResolutionProvider}, cancel: map[string]context.CancelFunc{},
		prefetchJobs: map[string]PrefetchSummary{}, marketConditionJobs: map[string]MarketConditionBackfillSummary{},
		marketConditionEarliest: historicalmarket.BinanceSource{}, deleting: map[string]bool{},
	}
}

func (manager *Manager) Start(request StartRequest) (RunSummary, error) {
	request.Symbol = strings.ToUpper(strings.TrimSpace(request.Symbol))
	request.Config = NormalizeRunConfig(request.Config)
	resolutionMode, err := NormalizeResolutionMode(request.ResolutionMode)
	if err != nil {
		return RunSummary{}, err
	}
	request.ResolutionMode = resolutionMode
	engineVersion, resolutionModel := ResolutionMetadata(resolutionMode)
	if request.StrategyTemplateID <= 0 {
		return RunSummary{}, fmt.Errorf("strategy_template_id is required")
	}
	if request.Symbol == "" || !strings.HasSuffix(request.Symbol, "USDT") {
		return RunSummary{}, fmt.Errorf("symbol must be a USDT futures contract")
	}
	if request.StartTime <= 0 || request.EndTime <= request.StartTime {
		return RunSummary{}, fmt.Errorf("valid start_time and end_time are required")
	}
	o := orm.NewOrm()
	template := models.StrategyTemplates{ID: request.StrategyTemplateID}
	if err := o.Read(&template); err != nil {
		return RunSummary{}, fmt.Errorf("load strategy template: %w", err)
	}
	if strategyservice.StrategyUsesMarketCondition(template.Strategy) {
		if err := ValidateMarketConditionCoverage(context.Background(), request.StartTime, request.EndTime); err != nil {
			return RunSummary{}, err
		}
	}
	strategyVersion := strategyservice.StrategySnapshotHash(template.Technology, template.Strategy)
	now := time.Now().UnixMilli()
	runID := newRunID()
	row := models.AgentBacktestRun{RunID: runID, StrategyTemplateID: template.ID, StrategyTemplateName: template.Name, StrategyVersion: strategyVersion, TechnologyJSON: template.Technology, StrategyJSON: template.Strategy, EngineVersion: engineVersion, MarketConditionModel: MarketConditionModel, ResolutionMode: resolutionMode, ResolutionModel: resolutionModel, Symbol: request.Symbol, ExecutionInterval: ReplayInterval, StartTime: request.StartTime, EndTime: request.EndTime, InitialEquity: request.Config.InitialEquity, PositionSizePct: request.Config.PositionSizePct, Leverage: request.Config.Leverage, FeeRate: request.Config.FeeRate, SlippageBps: request.Config.SlippageBps, StopLossPct: request.Config.StopLossPct, TakeProfitPct: request.Config.TakeProfitPct, Status: "queued", Stage: "queued", Progress: 0, CreatedAt: now, UpdatedAt: now}
	if _, err := o.Insert(&row); err != nil {
		return RunSummary{}, fmt.Errorf("create backtest run: %w", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	manager.mu.Lock()
	manager.cancel[runID] = cancel
	manager.mu.Unlock()
	go manager.run(ctx, row, request)
	return summaryFromRow(row), nil
}
func (manager *Manager) run(ctx context.Context, row models.AgentBacktestRun, request StartRequest) {
	defer manager.finishActive(row.RunID)
	runStarted := time.Now()
	manager.updateState(row.RunID, "running", "building_dataset", 5, "", false)
	lastProgress := 5
	reportProgress := func(stage string, base, span, completed, total int) {
		if total <= 0 {
			return
		}
		value := base + completed*span/total
		if value > base+span {
			value = base + span
		}
		if value <= lastProgress {
			return
		}
		lastProgress = value
		manager.updateProgress(row.RunID, stage, value)
	}
	datasetBuildStarted := time.Now()
	dataset, err := manager.builder.BuildWithProgress(ctx, DatasetRequest{Symbol: row.Symbol, ExecutionInterval: row.ExecutionInterval, StartTime: row.StartTime, EndTime: row.EndTime, TechnologyJSON: row.TechnologyJSON, StrategyJSON: row.StrategyJSON}, func(completed, total int) {
		reportProgress("building_dataset", 5, 25, completed, total)
	})
	datasetBuildMs := time.Since(datasetBuildStarted).Milliseconds()
	if err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	datasetSaveStarted := time.Now()
	manifest, err := saveDataset(ctx, dataset)
	datasetSaveMs := time.Since(datasetSaveStarted).Milliseconds()
	if err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	dataset.DatasetID = manifest.DatasetID
	dataset.DatasetSpecHash = manifest.DatasetSpecHash
	lastProgress = 35
	if _, err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", row.RunID).Update(orm.Params{"dataset_id": dataset.DatasetID, "dataset_spec_hash": dataset.DatasetSpecHash, "data_hash": dataset.DataHash, "stage": "running_backtest", "progress": 35, "updated_at": time.Now().UnixMilli()}); err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	runEngine := manager.engine
	lastResolutionActivity := time.Time{}
	lastResolutionStage := ""
	runEngine.ResolutionActivity = func(stage string) {
		if stage == "" {
			stage = "resolving_intrabar_data"
		}
		now := time.Now()
		if stage == lastResolutionStage && now.Sub(lastResolutionActivity) < time.Second {
			return
		}
		lastResolutionStage = stage
		lastResolutionActivity = now
		manager.updateProgress(row.RunID, stage, lastProgress)
	}
	engineStarted := time.Now()
	result, err := runEngine.RunWithResolution(ctx, dataset, StrategySnapshot{TemplateID: row.StrategyTemplateID, TemplateName: row.StrategyTemplateName, TechnologyJSON: row.TechnologyJSON, StrategyJSON: row.StrategyJSON, Version: row.StrategyVersion}, request.Config, row.ResolutionMode, func(completed, total int) {
		reportProgress("running_backtest", 35, 55, completed, total)
	})
	engineMs := time.Since(engineStarted).Milliseconds()
	if err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	lastProgress = 90
	manager.updateProgress(row.RunID, "saving_results", 90)
	saveStarted := time.Now()
	saveTiming, err := saveResultWithProgressTimed(ctx, row.RunID, result, func(completed, total int) {
		reportProgress("saving_results", 90, 9, completed, total)
	})
	saveTotalMs := time.Since(saveStarted).Milliseconds()
	if err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	result.ResolutionStats.Timing = &BacktestTimingStats{
		DatasetBuildMs: datasetBuildMs,
		DatasetSaveMs:  datasetSaveMs,
		EngineMs:       engineMs,
		SaveTradesMs:   saveTiming.TradesMs,
		SaveEventsMs:   saveTiming.EventsMs,
		SaveEquityMs:   saveTiming.EquityMs,
		SaveTotalMs:    saveTotalMs,
		TotalMs:        time.Since(runStarted).Milliseconds(),
	}
	if err := ctx.Err(); err != nil {
		manager.finishError(row.RunID, ctx, err)
		return
	}
	metrics, _ := json.Marshal(result.Metrics)
	resolutionStats, _ := json.Marshal(result.ResolutionStats)
	now := time.Now().UnixMilli()
	_, err = orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", row.RunID).Update(orm.Params{"status": "succeeded", "stage": "completed", "progress": 100, "data_hash": result.DataHash, "engine_version": result.EngineVersion, "resolution_mode": result.ResolutionMode, "resolution_model": result.ResolutionModel, "resolution_stats_json": string(resolutionStats), "metrics_json": string(metrics), "updated_at": now, "completed_at": now})
	if err != nil {
		manager.finishError(row.RunID, ctx, err)
	}
}
func (manager *Manager) finishActive(runID string) {
	manager.mu.Lock()
	if cancel := manager.cancel[runID]; cancel != nil {
		cancel()
	}
	delete(manager.cancel, runID)
	manager.mu.Unlock()
}
func (manager *Manager) updateState(runID, status, stage string, progress int, errorText string, completed bool) {
	params := orm.Params{"status": status, "stage": stage, "progress": progress, "updated_at": time.Now().UnixMilli(), "error": errorText}
	if status == "running" && stage == "building_dataset" {
		params["started_at"] = time.Now().UnixMilli()
	}
	if completed {
		params["completed_at"] = time.Now().UnixMilli()
	}
	_, _ = orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).Update(params)
}
func (manager *Manager) updateProgress(runID, stage string, progress int) {
	_, _ = orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).Update(orm.Params{"stage": stage, "progress": progress, "updated_at": time.Now().UnixMilli()})
}

func (manager *Manager) finishError(runID string, ctx context.Context, cause error) {
	if ctx.Err() != nil {
		manager.updateState(runID, "cancelled", "cancelled", 100, "backtest cancelled", true)
		return
	}
	manager.updateState(runID, "failed", "failed", 100, cause.Error(), true)
}
func (manager *Manager) Cancel(runID string) error {
	var row models.AgentBacktestRun
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).One(&row); err != nil {
		return err
	}
	if row.Status != "queued" && row.Status != "running" {
		return fmt.Errorf("backtest run is already %s", row.Status)
	}
	manager.mu.Lock()
	cancel := manager.cancel[runID]
	manager.mu.Unlock()
	if cancel == nil {
		return fmt.Errorf("backtest run is not active in this process")
	}
	cancel()
	manager.updateState(runID, "cancelled", "cancelled", 100, "backtest cancelled", true)
	return nil
}

func (manager *Manager) StartDelete(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("run_id is required")
	}
	var row models.AgentBacktestRun
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).One(&row); err != nil {
		if err == orm.ErrNoRows {
			return fmt.Errorf("backtest run %q not found", runID)
		}
		return err
	}
	if row.Status == "queued" || row.Status == "running" {
		return fmt.Errorf("active backtest run cannot be deleted; cancel it first")
	}
	manager.mu.Lock()
	_, active := manager.cancel[runID]
	manager.mu.Unlock()
	if active {
		return fmt.Errorf("backtest run is still active; wait for cancellation to finish before deleting")
	}
	if row.Status != "deleting" {
		now := time.Now().UnixMilli()
		startProgress := 0
		if row.Status == "delete_failed" && row.Progress > 0 {
			startProgress = row.Progress
		}
		if _, err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).Update(orm.Params{
			"status": "deleting", "stage": "deleting_equity", "progress": startProgress, "error": "", "updated_at": now,
		}); err != nil {
			return fmt.Errorf("mark backtest deleting: %w", err)
		}
		row.Status = "deleting"
		row.Stage = "deleting_equity"
		row.Progress = startProgress
	}
	manager.startDeleteWorker(row)
	return nil
}

// Delete keeps a synchronous path for internal callers and tests. HTTP deletion
// uses StartDelete so a large history never holds the request open.
func (manager *Manager) Delete(runID string) error {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return fmt.Errorf("run_id is required")
	}
	var row models.AgentBacktestRun
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).One(&row); err != nil {
		return err
	}
	if row.Status == "queued" || row.Status == "running" {
		return fmt.Errorf("active backtest run cannot be deleted; cancel it first")
	}
	return manager.deleteRunData(row, nil)
}

func (manager *Manager) startDeleteWorker(row models.AgentBacktestRun) {
	manager.deleteMu.Lock()
	if manager.deleting[row.RunID] {
		manager.deleteMu.Unlock()
		return
	}
	manager.deleting[row.RunID] = true
	manager.deleteMu.Unlock()
	go func() {
		defer func() {
			manager.deleteMu.Lock()
			delete(manager.deleting, row.RunID)
			manager.deleteMu.Unlock()
		}()
		currentProgress := row.Progress
		if err := manager.deleteRunData(row, func(stage string, progress int) {
			if progress < currentProgress {
				progress = currentProgress
			}
			currentProgress = progress
			manager.updateProgress(row.RunID, stage, progress)
		}); err != nil {
			manager.updateState(row.RunID, "delete_failed", "delete_failed", currentProgress, err.Error(), false)
		}
	}()
}

func (manager *Manager) resumeDeletingRuns() {
	var rows []models.AgentBacktestRun
	if _, err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("status", "deleting").All(&rows); err != nil {
		return
	}
	for _, row := range rows {
		manager.startDeleteWorker(row)
	}
}

type backtestDeleteTable struct {
	name  string
	table string
	stage string
}

var backtestDeleteTables = []backtestDeleteTable{
	{name: "equity points", table: "agent_backtest_equity_points", stage: "deleting_equity"},
	{name: "events", table: "agent_backtest_events", stage: "deleting_events"},
	{name: "trades", table: "agent_backtest_trades", stage: "deleting_trades"},
}

func (manager *Manager) deleteRunData(row models.AgentBacktestRun, progress func(stage string, progress int)) error {
	o := orm.NewOrm()
	total := int64(0)
	counts := make(map[string]int64, len(backtestDeleteTables))
	for _, item := range backtestDeleteTables {
		var count int64
		if err := o.Raw("SELECT COUNT(*) FROM "+item.table+" WHERE run_id = ?", row.RunID).QueryRow(&count); err != nil {
			return fmt.Errorf("count backtest %s: %w", item.name, err)
		}
		counts[item.table] = count
		total += count
	}
	deletedTotal := int64(0)
	report := func(stage string) {
		if progress == nil {
			return
		}
		value := 95
		if total > 0 {
			value = 5 + int(deletedTotal*90/total)
			if value > 95 {
				value = 95
			}
		}
		progress(stage, value)
	}
	for _, item := range backtestDeleteTables {
		report(item.stage)
		if counts[item.table] == 0 {
			continue
		}
		if err := deleteBacktestRowsInBatches(item.table, row.RunID, func(deleted int64) {
			deletedTotal += deleted
			report(item.stage)
		}); err != nil {
			return fmt.Errorf("delete backtest %s: %w", item.name, err)
		}
	}
	if progress != nil {
		progress("finalizing_delete", 99)
	}
	tx, err := o.Begin()
	if err != nil {
		return fmt.Errorf("begin delete backtest transaction: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	result, err := tx.Raw("DELETE FROM agent_backtest_runs WHERE run_id = ?", row.RunID).Exec()
	if err != nil {
		return rollback(fmt.Errorf("delete backtest run: %w", err))
	}
	deleted, err := result.RowsAffected()
	if err != nil {
		return rollback(fmt.Errorf("read deleted backtest run count: %w", err))
	}
	if deleted == 0 {
		return rollback(fmt.Errorf("backtest run %q not found", row.RunID))
	}
	if strings.TrimSpace(row.DatasetID) != "" {
		refs, err := tx.QueryTable(new(models.AgentBacktestRun)).Filter("dataset_id", row.DatasetID).Count()
		if err != nil {
			return rollback(fmt.Errorf("count backtest dataset references: %w", err))
		}
		if refs == 0 {
			if _, err := tx.QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_id", row.DatasetID).Delete(); err != nil {
				return rollback(fmt.Errorf("delete unused backtest dataset manifest: %w", err))
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete backtest transaction: %w", err)
	}
	return nil
}

const (
	backtestDeleteBatchSizeMySQL   = 50000
	backtestDeleteBatchSizeGeneric = 20000
)

func deleteBacktestRowsInBatches(table, runID string, progress func(deleted int64)) error {
	batchSize := backtestDeleteBatchSizeGeneric
	query := ""
	if backtestDeleteMySQL() {
		batchSize = backtestDeleteBatchSizeMySQL
		// MySQL supports ORDER BY/LIMIT directly on DELETE. This avoids the nested
		// ID subquery previously executed for every batch and is substantially
		// faster for multi-million-row 1m equity histories.
		query = fmt.Sprintf("DELETE FROM %s WHERE run_id = ? LIMIT %d", table, batchSize)
	} else {
		query = fmt.Sprintf("DELETE FROM %s WHERE id IN (SELECT id FROM (SELECT id FROM %s WHERE run_id = ? ORDER BY id LIMIT %d) AS delete_batch)", table, table, batchSize)
	}
	for {
		result, err := orm.NewOrm().Raw(query, runID).Exec()
		if err != nil {
			return err
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if progress != nil && deleted > 0 {
			progress(deleted)
		}
		if deleted < int64(batchSize) {
			return nil
		}
	}
}

func backtestDeleteMySQL() bool {
	if db, err := orm.GetDB("default"); err == nil {
		driverType := strings.ToLower(fmt.Sprintf("%T", db.Driver()))
		if strings.Contains(driverType, "mysql") {
			return true
		}
		if strings.Contains(driverType, "sqlite") || strings.Contains(driverType, "pq") || strings.Contains(driverType, "postgres") {
			return false
		}
	}
	return false
}

func (manager *Manager) markInterrupted() error {
	now := time.Now().UnixMilli()
	_, err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("status__in", "queued", "running").Update(orm.Params{"status": "interrupted", "stage": "interrupted", "progress": 100, "error": "process restarted before backtest completed", "updated_at": now, "completed_at": now})
	return err
}

func (manager *Manager) List(page, limit int) ([]RunSummary, int64, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	q := orm.NewOrm().QueryTable(new(models.AgentBacktestRun))
	total, err := q.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.AgentBacktestRun
	if _, err = q.OrderBy("-created_at").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return nil, 0, err
	}
	out := make([]RunSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, summaryFromRow(row))
	}
	return out, total, nil
}
func (manager *Manager) Get(runID string) (RunDetail, error) {
	var row models.AgentBacktestRun
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestRun)).Filter("run_id", runID).One(&row); err != nil {
		return RunDetail{}, err
	}
	detail := RunDetail{RunSummary: summaryFromRow(row), Trades: []Trade{}, Events: []AuditEvent{}, Equity: []EquityPoint{}}
	if row.DatasetID != "" {
		m, err := loadDatasetManifest(row.DatasetID)
		if err == nil {
			detail.Dataset = &m
		}
	}
	return detail, nil
}
func (manager *Manager) Trades(runID string, limit int) ([]Trade, error) {
	items, _, err := manager.TradesPage(runID, 1, limit)
	return items, err
}

func (manager *Manager) TradesPage(runID string, page, limit int) ([]Trade, int64, error) {
	page, limit = normalizePage(page, limit, 20, 5000)
	query := orm.NewOrm().QueryTable(new(models.AgentBacktestTrade)).Filter("run_id", runID)
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.AgentBacktestTrade
	if _, err := query.OrderBy("sequence").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return nil, 0, err
	}
	out := make([]Trade, 0, len(rows))
	for _, r := range rows {
		out = append(out, Trade{Sequence: r.Sequence, Symbol: r.Symbol, Side: r.Side, EntryTime: r.EntryTime, ExitTime: r.ExitTime, EntryPrice: r.EntryPrice, ExitPrice: r.ExitPrice, Quantity: r.Quantity, GrossPnL: r.GrossPnL, Fees: r.Fees, FundingPnL: r.FundingPnL, NetPnL: r.NetPnL, HoldingMs: r.HoldingMs, ExitReason: r.ExitReason, OpenStrategyName: r.OpenStrategyName, OpenStrategyType: r.OpenStrategyType, OpenStrategyHash: r.OpenStrategyHash, CloseStrategyName: r.CloseStrategyName, CloseStrategyType: r.CloseStrategyType, CloseStrategyHash: r.CloseStrategyHash, MarketCondition: r.MarketCondition, EntryResolution: r.EntryResolution, ExitResolution: r.ExitResolution})
	}
	return out, total, nil
}

func (manager *Manager) Events(runID string, limit int) ([]AuditEvent, error) {
	items, _, err := manager.EventsPage(runID, 1, limit)
	return items, err
}

func (manager *Manager) EventsPage(runID string, page, limit int) ([]AuditEvent, int64, error) {
	page, limit = normalizePage(page, limit, 50, 10000)
	query := orm.NewOrm().QueryTable(new(models.AgentBacktestEvent)).Filter("run_id", runID)
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.AgentBacktestEvent
	if _, err := query.OrderBy("sequence").Limit(limit, (page-1)*limit).All(&rows); err != nil {
		return nil, 0, err
	}
	out := make([]AuditEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, AuditEvent{Sequence: r.Sequence, EventTime: r.EventTime, Type: r.Type, Action: r.Action, Side: r.Side, Price: r.Price, Quantity: r.Quantity, Data: json.RawMessage(r.DataJSON)})
	}
	return out, total, nil
}

func normalizePage(page, limit, defaultLimit, maxLimit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return page, limit
}

const maxEquityChartPoints = 20000

func (manager *Manager) Equity(runID string, limit int) ([]EquityPoint, error) {
	if limit <= 0 || limit > maxEquityChartPoints {
		limit = maxEquityChartPoints
	}
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return nil, fmt.Errorf("run id is required")
	}
	o := orm.NewOrm()
	total, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).Count()
	if err != nil {
		return nil, err
	}
	if total == 0 {
		return []EquityPoint{}, nil
	}
	var rows []models.AgentBacktestEquityPoint
	if total <= int64(limit) {
		if _, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).OrderBy("sequence").All(&rows); err != nil {
			return nil, err
		}
		return equityPointsFromRows(rows), nil
	}
	var bounds []models.AgentBacktestEquityPoint
	if _, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).OrderBy("sequence").Limit(1).All(&bounds); err != nil {
		return nil, err
	}
	if len(bounds) == 0 {
		return []EquityPoint{}, nil
	}
	minSequence := bounds[0].Sequence
	bounds = bounds[:0]
	if _, err := o.QueryTable(new(models.AgentBacktestEquityPoint)).Filter("run_id", runID).OrderBy("-sequence").Limit(1).All(&bounds); err != nil {
		return nil, err
	}
	maxSequence := bounds[0].Sequence
	targets := equitySampleSequences(minSequence, maxSequence, limit)
	rows = make([]models.AgentBacktestEquityPoint, 0, len(targets))
	const queryBatch = 400
	for start := 0; start < len(targets); start += queryBatch {
		end := start + queryBatch
		if end > len(targets) {
			end = len(targets)
		}
		placeholders := make([]string, end-start)
		args := make([]interface{}, 0, 1+end-start)
		args = append(args, runID)
		for i, sequence := range targets[start:end] {
			placeholders[i] = "?"
			args = append(args, sequence)
		}
		query := "SELECT id,run_id,sequence,bar_time,equity,cash,unrealized_pnl,drawdown_pct,COALESCE(position_side, '') AS position_side FROM agent_backtest_equity_points WHERE run_id=? AND sequence IN (" + strings.Join(placeholders, ",") + ") ORDER BY sequence"
		var batchRows []models.AgentBacktestEquityPoint
		if _, err := o.Raw(query, args...).QueryRows(&batchRows); err != nil {
			return nil, err
		}
		rows = append(rows, batchRows...)
	}
	return equityPointsFromRows(rows), nil
}

func equitySampleSequences(minSequence, maxSequence, limit int) []int {
	if limit <= 0 || maxSequence < minSequence {
		return nil
	}
	count := maxSequence - minSequence + 1
	if count <= limit {
		result := make([]int, 0, count)
		for sequence := minSequence; sequence <= maxSequence; sequence++ {
			result = append(result, sequence)
		}
		return result
	}
	if limit == 1 {
		return []int{minSequence}
	}
	result := make([]int, 0, limit)
	span := int64(maxSequence - minSequence)
	denominator := int64(limit - 1)
	last := minSequence - 1
	for i := 0; i < limit; i++ {
		sequence := minSequence + int(span*int64(i)/denominator)
		if sequence == last {
			continue
		}
		result = append(result, sequence)
		last = sequence
	}
	if result[len(result)-1] != maxSequence {
		result[len(result)-1] = maxSequence
	}
	return result
}

func equityPointsFromRows(rows []models.AgentBacktestEquityPoint) []EquityPoint {
	out := make([]EquityPoint, 0, len(rows))
	for _, r := range rows {
		out = append(out, EquityPoint{Sequence: r.Sequence, BarTime: r.BarTime, Equity: r.Equity, Cash: r.Cash, UnrealizedPnL: r.UnrealizedPnL, DrawdownPct: r.DrawdownPct, PositionSide: r.PositionSide})
	}
	return out
}

func saveDataset(ctx context.Context, d Dataset) (DatasetManifest, error) {
	if err := ctx.Err(); err != nil {
		return DatasetManifest{}, err
	}
	o := orm.NewOrm()
	var existing models.AgentBacktestDataset
	if err := o.QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_spec_hash", d.DatasetSpecHash).One(&existing); err == nil {
		return manifestFromRow(existing), nil
	} else if err != orm.ErrNoRows {
		return DatasetManifest{}, err
	}
	intervals, _ := json.Marshal(d.Intervals)
	now := time.Now().UnixMilli()
	row := models.AgentBacktestDataset{DatasetID: d.DatasetID, DatasetSpecHash: d.DatasetSpecHash, Symbol: d.Symbol, ExecutionInterval: d.ExecutionInterval, IntervalsJSON: string(intervals), BenchmarkSymbolsJSON: "[]", StartTime: d.StartTime, EndTime: d.EndTime, WarmupStartTime: d.WarmupStartTime, Market: d.Market, CreatedAt: now, UpdatedAt: now}
	if _, err := o.Insert(&row); err != nil {
		if reread := o.QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_spec_hash", d.DatasetSpecHash).One(&existing); reread == nil {
			return manifestFromRow(existing), nil
		}
		return DatasetManifest{}, err
	}
	return manifestFromRow(row), nil
}
func loadDatasetManifest(id string) (DatasetManifest, error) {
	var row models.AgentBacktestDataset
	if err := orm.NewOrm().QueryTable(new(models.AgentBacktestDataset)).Filter("dataset_id", id).One(&row); err != nil {
		return DatasetManifest{}, err
	}
	return manifestFromRow(row), nil
}
func manifestFromRow(row models.AgentBacktestDataset) DatasetManifest {
	var intervals []string
	_ = json.Unmarshal([]byte(row.IntervalsJSON), &intervals)
	return DatasetManifest{DatasetID: row.DatasetID, DatasetSpecHash: row.DatasetSpecHash, Market: row.Market, Symbol: row.Symbol, ExecutionInterval: row.ExecutionInterval, Intervals: intervals, StartTime: row.StartTime, EndTime: row.EndTime, WarmupStartTime: row.WarmupStartTime, CreatedAt: row.CreatedAt}
}
func saveResult(ctx context.Context, runID string, result Result) error {
	return saveResultWithProgress(ctx, runID, result, nil)
}

const (
	backtestEventInsertBatchSizeMySQL       = 3000
	backtestEquityInsertBatchSizeMySQL      = 6000
	backtestTradeInsertBatchSizeMySQL       = 2000
	backtestResultInsertBatchSizeGeneric    = 500
	backtestEquityTransactionRowsMySQL      = 30000
	backtestEquityTransactionBatchesGeneric = 10
)

func backtestResultInsertBatchSizes() (trade, event, equity int) {
	if backtestDeleteMySQL() {
		return backtestTradeInsertBatchSizeMySQL, backtestEventInsertBatchSizeMySQL, backtestEquityInsertBatchSizeMySQL
	}
	return backtestResultInsertBatchSizeGeneric, backtestResultInsertBatchSizeGeneric, backtestResultInsertBatchSizeGeneric
}

func backtestEquityTransactionRows(equityBatchSize int) int {
	if backtestDeleteMySQL() {
		return backtestEquityTransactionRowsMySQL
	}
	return equityBatchSize * backtestEquityTransactionBatchesGeneric
}

type resultSaveTiming struct {
	TradesMs int64
	EventsMs int64
	EquityMs int64
}

func saveResultWithProgress(ctx context.Context, runID string, result Result, progress ProgressCallback) error {
	_, err := saveResultWithProgressTimed(ctx, runID, result, progress)
	return err
}

func saveResultWithProgressTimed(ctx context.Context, runID string, result Result, progress ProgressCallback) (resultSaveTiming, error) {
	var timing resultSaveTiming
	if err := ctx.Err(); err != nil {
		return timing, err
	}
	o := orm.NewOrm()
	tradeBatchSize, eventBatchSize, equityBatchSize := backtestResultInsertBatchSizes()
	totalRows := len(result.Trades) + len(result.Events) + len(result.Equity)
	completedRows := 0
	report := func() {
		if progress != nil {
			if totalRows == 0 {
				progress(1, 1)
				return
			}
			progress(completedRows, totalRows)
		}
	}
	report()

	tradeStarted := time.Now()
	tradeRows := make([]models.AgentBacktestTrade, 0, len(result.Trades))
	for _, t := range result.Trades {
		tradeRows = append(tradeRows, models.AgentBacktestTrade{RunID: runID, Sequence: t.Sequence, Symbol: t.Symbol, Side: t.Side, EntryTime: t.EntryTime, ExitTime: t.ExitTime, EntryPrice: t.EntryPrice, ExitPrice: t.ExitPrice, Quantity: t.Quantity, GrossPnL: t.GrossPnL, Fees: t.Fees, FundingPnL: t.FundingPnL, NetPnL: t.NetPnL, HoldingMs: t.HoldingMs, ExitReason: t.ExitReason, OpenStrategyName: t.OpenStrategyName, OpenStrategyType: t.OpenStrategyType, OpenStrategyHash: t.OpenStrategyHash, CloseStrategyName: t.CloseStrategyName, CloseStrategyType: t.CloseStrategyType, CloseStrategyHash: t.CloseStrategyHash, MarketCondition: t.MarketCondition, EntryResolution: t.EntryResolution, ExitResolution: t.ExitResolution})
	}
	if len(tradeRows) > 0 {
		for start := 0; start < len(tradeRows); start += tradeBatchSize {
			if err := ctx.Err(); err != nil {
				return timing, err
			}
			end := start + tradeBatchSize
			if end > len(tradeRows) {
				end = len(tradeRows)
			}
			chunk := tradeRows[start:end]
			if _, err := o.InsertMulti(tradeBatchSize, &chunk); err != nil {
				return timing, err
			}
			completedRows += len(chunk)
			report()
		}
	}
	timing.TradesMs = time.Since(tradeStarted).Milliseconds()

	eventStarted := time.Now()
	for start := 0; start < len(result.Events); start += eventBatchSize {
		if err := ctx.Err(); err != nil {
			return timing, err
		}
		end := start + eventBatchSize
		if end > len(result.Events) {
			end = len(result.Events)
		}
		eventRows := make([]models.AgentBacktestEvent, 0, end-start)
		for _, e := range result.Events[start:end] {
			eventRows = append(eventRows, models.AgentBacktestEvent{RunID: runID, Sequence: e.Sequence, EventTime: e.EventTime, Type: e.Type, Action: e.Action, Side: e.Side, Price: e.Price, Quantity: e.Quantity, DataJSON: string(e.Data)})
		}
		if _, err := o.InsertMulti(eventBatchSize, &eventRows); err != nil {
			return timing, err
		}
		completedRows += len(eventRows)
		report()
	}
	timing.EventsMs = time.Since(eventStarted).Milliseconds()

	equityStarted := time.Now()
	transactionRows := backtestEquityTransactionRows(equityBatchSize)
	for groupStart := 0; groupStart < len(result.Equity); groupStart += transactionRows {
		if err := ctx.Err(); err != nil {
			return timing, err
		}
		groupEnd := groupStart + transactionRows
		if groupEnd > len(result.Equity) {
			groupEnd = len(result.Equity)
		}
		tx, err := o.BeginWithCtx(ctx)
		if err != nil {
			return timing, err
		}
		groupRows := 0
		for start := groupStart; start < groupEnd; start += equityBatchSize {
			if err := ctx.Err(); err != nil {
				_ = tx.Rollback()
				return timing, err
			}
			end := start + equityBatchSize
			if end > groupEnd {
				end = groupEnd
			}
			eqRows := make([]models.AgentBacktestEquityPoint, 0, end-start)
			for _, e := range result.Equity[start:end] {
				eqRows = append(eqRows, models.AgentBacktestEquityPoint{RunID: runID, Sequence: e.Sequence, BarTime: e.BarTime, Equity: e.Equity, Cash: e.Cash, UnrealizedPnL: e.UnrealizedPnL, DrawdownPct: e.DrawdownPct, PositionSide: e.PositionSide})
			}
			if _, err := tx.InsertMultiWithCtx(ctx, equityBatchSize, &eqRows); err != nil {
				_ = tx.Rollback()
				return timing, err
			}
			groupRows += len(eqRows)
		}
		if err := tx.Commit(); err != nil {
			_ = tx.Rollback()
			return timing, err
		}
		completedRows += groupRows
		report()
	}
	timing.EquityMs = time.Since(equityStarted).Milliseconds()
	if totalRows == 0 {
		report()
	}
	return timing, nil
}

func summaryFromRow(row models.AgentBacktestRun) RunSummary {
	var metrics *Metrics
	if strings.TrimSpace(row.MetricsJSON) != "" {
		var m Metrics
		if json.Unmarshal([]byte(row.MetricsJSON), &m) == nil {
			metrics = &m
		}
	}
	var resolutionStats ResolutionStats
	if strings.TrimSpace(row.ResolutionStatsJSON) != "" {
		_ = json.Unmarshal([]byte(row.ResolutionStatsJSON), &resolutionStats)
	}
	mode := row.ResolutionMode
	if mode == "" {
		mode = ResolutionModeStandard
	}
	model := row.ResolutionModel
	if model == "" {
		model = StandardResolutionModel
	}
	return RunSummary{RunID: row.RunID, DatasetID: row.DatasetID, DatasetSpecHash: row.DatasetSpecHash, DataHash: row.DataHash, StrategyTemplateID: row.StrategyTemplateID, StrategyTemplateName: row.StrategyTemplateName, StrategyVersion: row.StrategyVersion, EngineVersion: row.EngineVersion, MarketConditionModel: row.MarketConditionModel, ResolutionMode: mode, ResolutionModel: model, ResolutionStats: resolutionStats, Symbol: row.Symbol, ExecutionInterval: row.ExecutionInterval, StartTime: row.StartTime, EndTime: row.EndTime, Config: RunConfig{InitialEquity: row.InitialEquity, PositionSizePct: row.PositionSizePct, Leverage: row.Leverage, FeeRate: row.FeeRate, SlippageBps: row.SlippageBps, StopLossPct: row.StopLossPct, TakeProfitPct: row.TakeProfitPct}, Status: row.Status, Stage: row.Stage, Progress: row.Progress, Metrics: metrics, Error: row.Error, CreatedAt: row.CreatedAt, StartedAt: row.StartedAt, UpdatedAt: row.UpdatedAt, CompletedAt: row.CompletedAt}
}
func newRunID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err == nil {
		return "bt_" + hex.EncodeToString(b)
	}
	return fmt.Sprintf("bt_%d", time.Now().UnixNano())
}
