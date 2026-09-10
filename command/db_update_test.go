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
		new(models.StrategyTemplates),
		new(models.Symbols),
		new(models.SpotSymbols),
		new(models.AgentSkill),
		new(models.AgentTask),
		new(models.AgentTradeProposal),
		new(models.AgentTradeExecution),
		new(models.AgentTradeAudit),
		new(models.LLMConfig),
		new(models.AgentMarketEvent), new(models.AgentMarketEventSource), new(models.AgentMarketFact), new(models.AgentMarketSourceStatus), new(models.MarketConditionHistory),
		new(models.MarketDataImportBatch), new(models.MarketKline1m), new(models.MarketKline3m), new(models.MarketKline5m), new(models.MarketKline15m), new(models.MarketKline30m), new(models.MarketKline1h), new(models.MarketKline2h), new(models.MarketKline4h), new(models.MarketKline6h), new(models.MarketKline8h), new(models.MarketKline12h), new(models.MarketKline1d), new(models.MarketKline3d), new(models.MarketKline1w), new(models.MarketKline1mo), new(models.MarketFundingRate),
		new(models.AgentBacktestDataset), new(models.AgentBacktestRun), new(models.AgentBacktestTrade), new(models.AgentBacktestEvent), new(models.AgentBacktestEquityPoint),
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
}
