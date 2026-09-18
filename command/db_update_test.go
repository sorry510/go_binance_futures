package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go_binance_futures/models"
	"go_binance_futures/utils"

	"github.com/beego/beego/v2/client/orm"
	_ "github.com/mattn/go-sqlite3"
)

func TestSyncDatabaseInitializesAndIsIdempotent(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Dir(cwd)
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	if err := orm.RegisterDriver("sqlite3", orm.DRSqlite); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(t.TempDir(), "sync.db")
	if err := orm.RegisterDataBase("default", "sqlite3", dbPath); err != nil {
		t.Fatal(err)
	}
	orm.RegisterModel(
		new(models.Config),
		new(models.Order),
		new(models.FuturesOrder),
		new(models.StrategyTemplates),
		new(models.TestStrategyResults),
		new(models.Symbols),
		new(models.SpotSymbols),
		new(models.AgentSkill),
		new(models.AgentTask),
		new(models.AgentTradeProposal),
		new(models.AgentTradeExecution),
		new(models.AgentTradeAudit),
		new(models.AgentOpportunity),
		new(models.LLMConfig),
		new(models.AgentMarketEvent), new(models.AgentMarketEventSource), new(models.AgentMarketFact), new(models.AgentMarketSourceStatus), new(models.MarketConditionHistory),
		new(models.MarketDataImportBatch), new(models.MarketKline1s), new(models.MarketKline1m), new(models.MarketKline3m), new(models.MarketKline5m), new(models.MarketKline15m), new(models.MarketKline30m), new(models.MarketKline1h), new(models.MarketKline2h), new(models.MarketKline4h), new(models.MarketKline6h), new(models.MarketKline8h), new(models.MarketKline12h), new(models.MarketKline1d), new(models.MarketKline3d), new(models.MarketKline1w), new(models.MarketKline1mo), new(models.MarketFundingRate), new(models.MarketTrade),
		new(models.AgentBacktestDataset), new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentBacktestEvent), new(models.AgentBacktestEquityPoint), new(models.AgentBacktestEquityChunk), new(models.AgentBacktestEquityPreview),
		new(models.FuturesManagedPosition), new(models.FuturesManagedOrder),
	)

	if err := SyncDatabase(1); err != nil {
		t.Fatal(err)
	}
	config, err := utils.GetSystemConfig()
	if err != nil {
		t.Fatal(err)
	}
	if config.Version != 1 {
		t.Fatalf("expected database version 1, got %d", config.Version)
	}
	o := orm.NewOrm()
	for _, item := range []*models.AgentSkill{
		{Name: "symbol_analysis", DisplayName: "Symbol", Type: "native", Enabled: 1, ChatEnabled: -1},
		{Name: "alert_analysis", DisplayName: "Alert", Type: "native", Enabled: 1, ChatEnabled: -1},
		{Name: "portable_demo", DisplayName: "Portable", Type: "portable", Enabled: 1, ChatEnabled: -1},
	} {
		if _, err := o.Insert(item); err != nil {
			t.Fatal(err)
		}
	}
	if err := SyncDatabase(2); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 2 {
		t.Fatalf("expected database version 2 after chat migration, config=%+v err=%v", config, err)
	}
	for name, want := range map[string]int{"symbol_analysis": 1, "alert_analysis": 0, "portable_demo": 1} {
		var got int
		if err := o.Raw("SELECT chat_enabled FROM agent_skills WHERE name = ?", name).QueryRow(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("chat_enabled for %s = %d, want %d", name, got, want)
		}
	}

	if err := SyncDatabase(3); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 3 {
		t.Fatalf("expected database version 3 after controlled-trade migration, config=%+v err=%v", config, err)
	}
	if config.AgentTradeExecutionEnable != 0 || config.AgentTradeAllowedSymbols != "" {
		t.Fatalf("V2-12 must stay disabled with an empty allowlist after upgrade: %+v", config)
	}
	if config.AgentTradeMaxRiskUSDT != 5 || config.AgentTradeMaxNotionalUSDT != 50 || config.AgentTradeMaxTotalExposureUSDT != 200 || config.AgentTradeMaxLeverage != 3 {
		t.Fatalf("unexpected V2-12 risk defaults: %+v", config)
	}
	for _, table := range []string{"agent_trade_proposals", "agent_trade_executions", "agent_trade_audits"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 3 sync, count=%d err=%v", table, count, err)
		}
	}
	if err := SyncDatabase(4); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 4 {
		t.Fatalf("expected database version 4 after V3-1 migration, config=%+v err=%v", config, err)
	}
	for _, column := range []string{"parent_task_id", "team_run_id", "team_name", "team_role"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM pragma_table_info('agent_tasks') WHERE name=?", column).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected agent_tasks.%s after version 4 sync, count=%d err=%v", column, count, err)
		}
	}
	if err := SyncDatabase(4); err != nil {
		t.Fatalf("second version-4 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(5); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 5 {
		t.Fatalf("expected database version 5 after LLM proxy schema sync, config=%+v err=%v", config, err)
	}
	var proxyColumnCount int
	if err := o.Raw("SELECT COUNT(*) FROM pragma_table_info('llm_configs') WHERE name='proxy_url'").QueryRow(&proxyColumnCount); err != nil || proxyColumnCount != 1 {
		t.Fatalf("expected llm_configs.proxy_url after version 5 sync, count=%d err=%v", proxyColumnCount, err)
	}
	if err := SyncDatabase(5); err != nil {
		t.Fatalf("second version-5 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(6); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 6 {
		t.Fatalf("expected database version 6, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"agent_market_events", "agent_market_event_sources", "agent_market_facts", "agent_market_source_status"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 6 sync, count=%d err=%v", table, count, err)
		}
	}
	if err := SyncDatabase(7); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 7 {
		t.Fatalf("expected database version 7, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"market_data_import_batches", "market_klines_1m", "market_klines_3m", "market_klines_5m", "market_klines_15m", "market_klines_30m", "market_klines_1h", "market_klines_2h", "market_klines_4h", "market_klines_6h", "market_klines_8h", "market_klines_12h", "market_klines_1d", "market_klines_3d", "market_klines_1w", "market_klines_1mo", "market_funding_rates", "agent_backtest_datasets", "agent_backtest_runs", "agent_backtest_trades", "agent_backtest_events", "agent_backtest_equity_points"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 7 sync, count=%d err=%v", table, count, err)
		}
	}
	if err := SyncDatabase(7); err != nil {
		t.Fatalf("second version-7 sync should be idempotent: %v", err)
	}
	legacy := models.StrategyTemplates{
		Name: "legacy-market-env", Technology: "{}",
		Strategy: `[{"name":"open","type":"long","code":"MarketCondition == \"2\" && BasicTrend > 0","fullScreen":true,"enable":true}]`,
	}
	if _, err := o.Insert(&legacy); err != nil {
		t.Fatal(err)
	}
	if err := SyncDatabase(8); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 8 {
		t.Fatalf("expected database version 8, config=%+v err=%v", config, err)
	}
	var historyTableCount int
	if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='market_condition_histories'").QueryRow(&historyTableCount); err != nil || historyTableCount != 1 {
		t.Fatalf("expected market_condition_histories after version 8 sync, count=%d err=%v", historyTableCount, err)
	}
	var migrated string
	if err := o.Raw("SELECT strategy FROM strategy_templates WHERE id = ?", legacy.ID).QueryRow(&migrated); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(migrated, "BasicTrend") || strings.Contains(migrated, "BTCUSDT.") || !strings.Contains(migrated, "MarketCondition") || !strings.Contains(migrated, `"fullScreen":true`) {
		t.Fatalf("version 8 strategy migration is invalid: %s", migrated)
	}
	if err := SyncDatabase(8); err != nil {
		t.Fatalf("second version-8 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(9); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 9 {
		t.Fatalf("expected database version 9, config=%+v err=%v", config, err)
	}
	if err := SyncDatabase(9); err != nil {
		t.Fatalf("second version-9 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(10); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 10 {
		t.Fatalf("expected database version 10, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"futures_managed_positions", "futures_managed_orders"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 10 sync, count=%d err=%v", table, count, err)
		}
	}
	for table, columns := range map[string][]string{
		"agent_backtest_trades":        {"run_id", "sequence"},
		"agent_backtest_events":        {"run_id", "sequence"},
		"agent_backtest_equity_points": {"run_id", "sequence"},
		"market_condition_histories":   {"config_id", "created_at"},
		"order":                        {"side", "updateTime"},
		"test_strategy_results":        {"strategy_template_id", "createTime"},
		"futures_orders":               {"updateTime"},
	} {
		if !sqliteHasIndexColumns(t, o, table, columns) {
			t.Fatalf("expected composite index on %s(%s) after version 10 sync", table, strings.Join(columns, ","))
		}
	}
	if err := SyncDatabase(10); err != nil {
		t.Fatalf("second version-10 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(11); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 11 {
		t.Fatalf("expected database version 11 after V3-4 sparse history schema sync, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"market_klines_1s", "market_trades"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 11 sync, count=%d err=%v", table, count, err)
		}
	}
	if !sqliteHasIndexColumns(t, o, "market_trades", []string{"market", "symbol", "trade_time"}) {
		t.Fatal("expected market_trades(market,symbol,trade_time) index after version 11 sync")
	}
	for table, columns := range map[string][]string{
		"agent_backtest_runs":   {"resolution_mode", "resolution_model", "resolution_stats_json"},
		"agent_backtest_trades": {"entry_resolution", "exit_resolution"},
	} {
		for _, column := range columns {
			var count int
			if err := o.Raw("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?", table, column).QueryRow(&count); err != nil || count != 1 {
				t.Fatalf("expected %s.%s after version 11 sync, count=%d err=%v", table, column, count, err)
			}
		}
	}
	if err := SyncDatabase(11); err != nil {
		t.Fatalf("second version-11 sync should be idempotent: %v", err)
	}

	// V3-4 and V3-5 were developed on separate schema-version lines. The
	// merged branch advances to version 12 so a database that already reached
	// either branch's version 11 is forced through RunSyncdb once more.
	if err := SyncDatabase(12); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 12 {
		t.Fatalf("expected database version 12 after merged V3-4/V3-5 schema sync, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"market_klines_1s", "market_trades", "futures_managed_positions", "futures_managed_orders"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected merged schema table %s after version 12 sync, count=%d err=%v", table, count, err)
		}
	}
	if err := SyncDatabase(12); err != nil {
		t.Fatalf("second version-12 sync should be idempotent: %v", err)
	}

	if err := SyncDatabase(13); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 13 {
		t.Fatalf("expected database version 13 after V3-6 opportunity schema sync, config=%+v err=%v", config, err)
	}
	var opportunityTableCount int
	if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='agent_opportunities'").QueryRow(&opportunityTableCount); err != nil || opportunityTableCount != 1 {
		t.Fatalf("expected agent_opportunities after version 13 sync, count=%d err=%v", opportunityTableCount, err)
	}
	if err := SyncDatabase(13); err != nil {
		t.Fatalf("second version-13 sync should be idempotent: %v", err)
	}

	if err := SyncDatabase(14); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 14 {
		t.Fatalf("expected database version 14 after order close-link schema sync, config=%+v err=%v", config, err)
	}
	for _, column := range []string{"closedPrice", "closeOrderId"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM pragma_table_info('order') WHERE name=?", column).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected order.%s after version 14 sync, count=%d err=%v", column, count, err)
		}
	}
	if !sqliteHasIndexColumns(t, o, "order", []string{"closeOrderId"}) {
		t.Fatal("expected order(closeOrderId) index after version 14 sync")
	}
	if err := SyncDatabase(14); err != nil {
		t.Fatalf("second version-14 sync should be idempotent: %v", err)
	}

	// Simulate a pre-v15 database that still has the three redundant
	// single-column equity indexes created by the previous model tags.
	for _, statement := range []string{
		"CREATE INDEX IF NOT EXISTS agent_backtest_equity_points_run_id ON agent_backtest_equity_points(run_id)",
		"CREATE INDEX IF NOT EXISTS agent_backtest_equity_points_sequence ON agent_backtest_equity_points(sequence)",
		"CREATE INDEX IF NOT EXISTS agent_backtest_equity_points_bar_time ON agent_backtest_equity_points(bar_time)",
	} {
		if _, err := o.Raw(statement).Exec(); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := o.Raw("INSERT INTO agent_backtest_equity_points (run_id,sequence,bar_time,equity,cash,unrealized_pnl,drawdown_pct,position_side) VALUES (?,?,?,?,?,?,?,?)", "v15-preserve", 7, int64(123456789), 101.25, 99.5, 1.75, 2.5, "LONG").Exec(); err != nil {
		t.Fatal(err)
	}

	if err := SyncDatabase(15); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 15 {
		t.Fatalf("expected database version 15 after equity index cleanup, config=%+v err=%v", config, err)
	}
	if !sqliteHasIndexColumns(t, o, "agent_backtest_equity_points", []string{"run_id", "sequence"}) {
		t.Fatal("expected retained agent_backtest_equity_points(run_id,sequence) index after version 15 sync")
	}
	for _, columns := range [][]string{{"run_id"}, {"sequence"}, {"bar_time"}} {
		if sqliteHasIndexColumns(t, o, "agent_backtest_equity_points", columns) {
			t.Fatalf("unexpected redundant equity index on (%s) after version 15 sync", strings.Join(columns, ","))
		}
	}
	var preserved struct {
		Sequence int     `orm:"column(sequence)"`
		BarTime  int64   `orm:"column(bar_time)"`
		Equity   float64 `orm:"column(equity)"`
		Cash     float64 `orm:"column(cash)"`
	}
	if err := o.Raw("SELECT sequence,bar_time,equity,cash FROM agent_backtest_equity_points WHERE run_id=?", "v15-preserve").QueryRow(&preserved); err != nil {
		t.Fatal(err)
	}
	if preserved.Sequence != 7 || preserved.BarTime != 123456789 || preserved.Equity != 101.25 || preserved.Cash != 99.5 {
		t.Fatalf("version 15 index cleanup changed equity data: %+v", preserved)
	}
	if err := SyncDatabase(15); err != nil {
		t.Fatalf("second version-15 sync should be idempotent: %v", err)
	}
	if err := SyncDatabase(16); err != nil {
		t.Fatal(err)
	}
	config, err = utils.GetSystemConfig()
	if err != nil || config.Version != 16 {
		t.Fatalf("expected database version 16 after chunked equity schema sync, config=%+v err=%v", config, err)
	}
	for _, table := range []string{"agent_backtest_equity_chunks", "agent_backtest_equity_preview"} {
		var count int
		if err := o.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table).QueryRow(&count); err != nil || count != 1 {
			t.Fatalf("expected %s after version 16 sync, count=%d err=%v", table, count, err)
		}
	}
	if !sqliteHasIndexColumns(t, o, "agent_backtest_equity_chunks", []string{"run_id", "chunk_index"}) {
		t.Fatal("expected unique agent_backtest_equity_chunks(run_id,chunk_index) access path after version 16 sync")
	}
	if !sqliteHasIndexColumns(t, o, "agent_backtest_equity_preview", []string{"run_id"}) {
		t.Fatal("expected unique agent_backtest_equity_preview(run_id) access path after version 16 sync")
	}
	if err := SyncDatabase(16); err != nil {
		t.Fatalf("second version-16 sync should be idempotent: %v", err)
	}
}

func sqliteHasIndexColumns(t *testing.T, o orm.Ormer, table string, want []string) bool {
	t.Helper()
	var indexes []struct {
		Name string `orm:"column(name)"`
	}
	tableName := strings.ReplaceAll(table, "'", "''")
	if _, err := o.Raw("PRAGMA index_list('" + tableName + "')").QueryRows(&indexes); err != nil {
		t.Fatalf("list indexes for %s: %v", table, err)
	}
	for _, index := range indexes {
		var columns []struct {
			Name string `orm:"column(name)"`
		}
		indexName := strings.ReplaceAll(index.Name, "'", "''")
		if _, err := o.Raw("PRAGMA index_info('" + indexName + "')").QueryRows(&columns); err != nil {
			t.Fatalf("read index %s: %v", index.Name, err)
		}
		if len(columns) != len(want) {
			continue
		}
		match := true
		for i := range want {
			if columns[i].Name != want[i] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
