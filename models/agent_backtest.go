package models

// AgentBacktestDataset is a reusable historical query specification. It does
// not own OHLCV rows; every run resolves the latest canonical market data.
type AgentBacktestDataset struct {
	ID                   int64  `orm:"column(id);auto" json:"id"`
	DatasetID            string `orm:"column(dataset_id);size(64);unique" json:"dataset_id"`
	DatasetSpecHash      string `orm:"column(dataset_spec_hash);size(64);unique" json:"dataset_spec_hash"`
	Symbol               string `orm:"column(symbol);size(32);index" json:"symbol"`
	ExecutionInterval    string `orm:"column(execution_interval);size(16);index" json:"execution_interval"`
	IntervalsJSON        string `orm:"column(intervals_json);type(text)" json:"-"`
	BenchmarkSymbolsJSON string `orm:"column(benchmark_symbols_json);type(text)" json:"-"`
	StartTime            int64  `orm:"column(start_time);index" json:"start_time"`
	EndTime              int64  `orm:"column(end_time);index" json:"end_time"`
	WarmupStartTime      int64  `orm:"column(warmup_start_time);index" json:"warmup_start_time"`
	Market               string `orm:"column(market);size(32);index" json:"market"`
	CreatedAt            int64  `orm:"column(created_at);index" json:"created_at"`
	UpdatedAt            int64  `orm:"column(updated_at);index" json:"updated_at"`
}

func (*AgentBacktestDataset) TableName() string { return "agent_backtest_datasets" }

// AgentBacktestRun snapshots strategy and execution parameters for reproducibility.
type AgentBacktestRun struct {
	ID                   int64   `orm:"column(id);auto" json:"id"`
	RunID                string  `orm:"column(run_id);size(64);unique" json:"run_id"`
	DatasetID            string  `orm:"column(dataset_id);size(64);index" json:"dataset_id"`
	DatasetSpecHash      string  `orm:"column(dataset_spec_hash);size(64);index" json:"dataset_spec_hash"`
	DataHash             string  `orm:"column(data_hash);size(64);index" json:"data_hash"`
	StrategyTemplateID   int64   `orm:"column(strategy_template_id);index" json:"strategy_template_id"`
	StrategyTemplateName string  `orm:"column(strategy_template_name);size(128);null" json:"strategy_template_name"`
	StrategyVersion      string  `orm:"column(strategy_version);size(64);index" json:"strategy_version"`
	TechnologyJSON       string  `orm:"column(technology_json);type(text)" json:"-"`
	StrategyJSON         string  `orm:"column(strategy_json);type(text)" json:"-"`
	EngineVersion        string  `orm:"column(engine_version);size(64);index" json:"engine_version"`
	MarketConditionModel string  `orm:"column(market_condition_model);size(64)" json:"market_condition_model"`
	ResolutionMode       string  `orm:"column(resolution_mode);size(32);default(standard_1m);index" json:"resolution_mode"`
	ResolutionModel      string  `orm:"column(resolution_model);size(64);default(standard_1m_v1)" json:"resolution_model"`
	ResolutionStatsJSON  string  `orm:"column(resolution_stats_json);type(text);null" json:"-"`
	Symbol               string  `orm:"column(symbol);size(32);index" json:"symbol"`
	ExecutionInterval    string  `orm:"column(execution_interval);size(16)" json:"execution_interval"`
	StartTime            int64   `orm:"column(start_time);index" json:"start_time"`
	EndTime              int64   `orm:"column(end_time);index" json:"end_time"`
	InitialEquity        float64 `orm:"column(initial_equity);digits(30);decimals(8)" json:"initial_equity"`
	PositionSizePct      float64 `orm:"column(position_size_pct);digits(12);decimals(6)" json:"position_size_pct"`
	Leverage             int     `orm:"column(leverage)" json:"leverage"`
	FeeRate              float64 `orm:"column(fee_rate);digits(16);decimals(10)" json:"fee_rate"`
	SlippageBps          float64 `orm:"column(slippage_bps);digits(16);decimals(6)" json:"slippage_bps"`
	StopLossPct          float64 `orm:"column(stop_loss_pct);digits(16);decimals(6)" json:"stop_loss_pct"`
	TakeProfitPct        float64 `orm:"column(take_profit_pct);digits(16);decimals(6)" json:"take_profit_pct"`
	Status               string  `orm:"column(status);size(32);index" json:"status"`
	Stage                string  `orm:"column(stage);size(64);index" json:"stage"`
	Progress             int     `orm:"column(progress)" json:"progress"`
	MetricsJSON          string  `orm:"column(metrics_json);type(text);null" json:"-"`
	Error                string  `orm:"column(error);type(text);null" json:"error,omitempty"`
	CreatedAt            int64   `orm:"column(created_at);index" json:"created_at"`
	StartedAt            int64   `orm:"column(started_at);index" json:"started_at,omitempty"`
	UpdatedAt            int64   `orm:"column(updated_at);index" json:"updated_at"`
	CompletedAt          int64   `orm:"column(completed_at);index" json:"completed_at,omitempty"`
}

func (*AgentBacktestRun) TableName() string { return "agent_backtest_runs" }

// AgentBacktestTrade is one fully closed deterministic simulated position.
type AgentBacktestTrade struct {
	ID                int64   `orm:"column(id);auto" json:"id"`
	RunID             string  `orm:"column(run_id);size(64);index" json:"run_id"`
	Sequence          int     `orm:"column(sequence);index" json:"sequence"`
	Symbol            string  `orm:"column(symbol);size(32);index" json:"symbol"`
	Side              string  `orm:"column(side);size(16);index" json:"side"`
	EntryTime         int64   `orm:"column(entry_time);index" json:"entry_time"`
	ExitTime          int64   `orm:"column(exit_time);index" json:"exit_time"`
	EntryPrice        float64 `orm:"column(entry_price);digits(30);decimals(12)" json:"entry_price"`
	ExitPrice         float64 `orm:"column(exit_price);digits(30);decimals(12)" json:"exit_price"`
	Quantity          float64 `orm:"column(quantity);digits(30);decimals(12)" json:"quantity"`
	GrossPnL          float64 `orm:"column(gross_pnl);digits(30);decimals(8)" json:"gross_pnl"`
	Fees              float64 `orm:"column(fees);digits(30);decimals(8)" json:"fees"`
	FundingPnL        float64 `orm:"column(funding_pnl);digits(30);decimals(8)" json:"funding_pnl"`
	NetPnL            float64 `orm:"column(net_pnl);digits(30);decimals(8)" json:"net_pnl"`
	HoldingMs         int64   `orm:"column(holding_ms)" json:"holding_ms"`
	ExitReason        string  `orm:"column(exit_reason);size(32);index" json:"exit_reason"`
	OpenStrategyName  string  `orm:"column(open_strategy_name);size(128);null" json:"open_strategy_name,omitempty"`
	OpenStrategyType  string  `orm:"column(open_strategy_type);size(32);null" json:"open_strategy_type,omitempty"`
	OpenStrategyHash  string  `orm:"column(open_strategy_hash);size(64);null" json:"open_strategy_hash,omitempty"`
	CloseStrategyName string  `orm:"column(close_strategy_name);size(128);null" json:"close_strategy_name,omitempty"`
	CloseStrategyType string  `orm:"column(close_strategy_type);size(32);null" json:"close_strategy_type,omitempty"`
	CloseStrategyHash string  `orm:"column(close_strategy_hash);size(64);null" json:"close_strategy_hash,omitempty"`
	MarketCondition   int     `orm:"column(market_condition);index" json:"market_condition"`
	EntryResolution   string  `orm:"column(entry_resolution);size(16);default(1m)" json:"entry_resolution"`
	ExitResolution    string  `orm:"column(exit_resolution);size(16);default(1m)" json:"exit_resolution"`
}

func (*AgentBacktestTrade) TableName() string { return "agent_backtest_trades" }
func (*AgentBacktestTrade) TableIndex() [][]string {
	return [][]string{{"RunID", "Sequence"}}
}

// AgentBacktestEvent is the deterministic audit stream: signal -> order -> fill -> position.
type AgentBacktestEvent struct {
	ID        int64   `orm:"column(id);auto" json:"id"`
	RunID     string  `orm:"column(run_id);size(64);index" json:"run_id"`
	Sequence  int     `orm:"column(sequence);index" json:"sequence"`
	EventTime int64   `orm:"column(event_time);index" json:"event_time"`
	Type      string  `orm:"column(event_type);size(32);index" json:"type"`
	Action    string  `orm:"column(action);size(64);index" json:"action"`
	Side      string  `orm:"column(side);size(16);null" json:"side,omitempty"`
	Price     float64 `orm:"column(price);digits(30);decimals(12)" json:"price,omitempty"`
	Quantity  float64 `orm:"column(quantity);digits(30);decimals(12)" json:"quantity,omitempty"`
	DataJSON  string  `orm:"column(data_json);type(text);null" json:"-"`
}

func (*AgentBacktestEvent) TableName() string { return "agent_backtest_events" }
func (*AgentBacktestEvent) TableIndex() [][]string {
	return [][]string{{"RunID", "Sequence"}}
}

// AgentBacktestEquityPoint is the legacy row-per-minute equity storage retained for old-run reads and deletion.
type AgentBacktestEquityPoint struct {
	ID            int64   `orm:"column(id);auto" json:"id"`
	RunID         string  `orm:"column(run_id);size(64)" json:"run_id"`
	Sequence      int     `orm:"column(sequence)" json:"sequence"`
	BarTime       int64   `orm:"column(bar_time)" json:"bar_time"`
	Equity        float64 `orm:"column(equity);digits(30);decimals(8)" json:"equity"`
	Cash          float64 `orm:"column(cash);digits(30);decimals(8)" json:"cash"`
	UnrealizedPnL float64 `orm:"column(unrealized_pnl);digits(30);decimals(8)" json:"unrealized_pnl"`
	DrawdownPct   float64 `orm:"column(drawdown_pct);digits(16);decimals(8)" json:"drawdown_pct"`
	PositionSide  string  `orm:"column(position_side);size(16);null" json:"position_side,omitempty"`
}

func (*AgentBacktestEquityPoint) TableName() string { return "agent_backtest_equity_points" }
func (*AgentBacktestEquityPoint) TableIndex() [][]string {
	return [][]string{{"RunID", "Sequence"}}
}

// AgentBacktestEquityChunk stores the complete lossless equity curve in compressed chunks.
type AgentBacktestEquityChunk struct {
	ID            int64  `orm:"column(id);auto" json:"id"`
	RunID         string `orm:"column(run_id);size(64)" json:"run_id"`
	ChunkIndex    int    `orm:"column(chunk_index)" json:"chunk_index"`
	StartSequence int    `orm:"column(start_sequence)" json:"start_sequence"`
	EndSequence   int    `orm:"column(end_sequence)" json:"end_sequence"`
	StartTime     int64  `orm:"column(start_time)" json:"start_time"`
	EndTime       int64  `orm:"column(end_time)" json:"end_time"`
	PointCount    int    `orm:"column(point_count)" json:"point_count"`
	Encoding      string `orm:"column(encoding);size(32)" json:"encoding"`
	Compression   string `orm:"column(compression);size(16)" json:"compression"`
	Checksum      int64  `orm:"column(checksum)" json:"checksum"`
	Payload       string `orm:"column(payload);type(text)" json:"-"`
	CreatedAt     int64  `orm:"column(created_at)" json:"created_at"`
}

func (*AgentBacktestEquityChunk) TableName() string { return "agent_backtest_equity_chunks" }
func (*AgentBacktestEquityChunk) TableUnique() [][]string {
	return [][]string{{"RunID", "ChunkIndex"}}
}

// AgentBacktestEquityPreview stores the <=20k sampled points used by the detail chart.
type AgentBacktestEquityPreview struct {
	ID          int64  `orm:"column(id);auto" json:"id"`
	RunID       string `orm:"column(run_id);size(64);unique" json:"run_id"`
	PointCount  int    `orm:"column(point_count)" json:"point_count"`
	Encoding    string `orm:"column(encoding);size(32)" json:"encoding"`
	Compression string `orm:"column(compression);size(16)" json:"compression"`
	Checksum    int64  `orm:"column(checksum)" json:"checksum"`
	Payload     string `orm:"column(payload);type(text)" json:"-"`
	CreatedAt   int64  `orm:"column(created_at)" json:"created_at"`
}

func (*AgentBacktestEquityPreview) TableName() string { return "agent_backtest_equity_preview" }
