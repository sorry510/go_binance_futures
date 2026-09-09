package backtest

import (
	"encoding/json"
	"time"
)

const (
	EngineVersion        = "backtest_engine_v1"
	MarketConditionModel = "backtest_major_regime_v1"
	DefaultWarmupBars    = 200
)

var BenchmarkSymbols = []string{"BTCUSDT", "ETHUSDT", "SOLUSDT", "BNBUSDT"}

type Bar struct {
	Symbol              string  `json:"symbol"`
	Interval            string  `json:"interval"`
	OpenTime            int64   `json:"open_time"`
	CloseTime           int64   `json:"close_time"`
	Open                float64 `json:"open"`
	High                float64 `json:"high"`
	Low                 float64 `json:"low"`
	Close               float64 `json:"close"`
	Volume              float64 `json:"volume"`
	QuoteVolume         float64 `json:"quote_volume"`
	TradeCount          int64   `json:"trade_count"`
	TakerBuyQuoteVolume float64 `json:"taker_buy_quote_volume"`
}

type Funding struct {
	Symbol      string  `json:"symbol"`
	FundingTime int64   `json:"funding_time"`
	FundingRate float64 `json:"funding_rate"`
	MarkPrice   float64 `json:"mark_price"`
}

type Dataset struct {
	DatasetID         string           `json:"dataset_id"`
	DatasetSpecHash   string           `json:"dataset_spec_hash"`
	DataHash          string           `json:"data_hash"`
	Market            string           `json:"market"`
	Symbol            string           `json:"symbol"`
	ExecutionInterval string           `json:"execution_interval"`
	Intervals         []string         `json:"intervals"`
	BenchmarkSymbols  []string         `json:"benchmark_symbols"`
	StartTime         int64            `json:"start_time"`
	EndTime           int64            `json:"end_time"`
	WarmupStartTime   int64            `json:"warmup_start_time"`
	Bars              map[string][]Bar `json:"bars"`
	Funding           []Funding        `json:"funding"`
}

func BarSeriesKey(symbol, interval string) string { return symbol + "|" + interval }

type ProgressCallback func(completed, total int)

type DatasetRequest struct {
	Symbol            string `json:"symbol"`
	ExecutionInterval string `json:"execution_interval"`
	StartTime         int64  `json:"start_time"`
	EndTime           int64  `json:"end_time"`
	TechnologyJSON    string `json:"technology_json"`
}

type Rule struct {
	Name   string `json:"name"`
	Enable bool   `json:"enable"`
	Code   string `json:"code"`
	Type   string `json:"type"`
}

type StrategySnapshot struct {
	TemplateID     int64  `json:"template_id"`
	TemplateName   string `json:"template_name"`
	TechnologyJSON string `json:"technology_json"`
	StrategyJSON   string `json:"strategy_json"`
	Version        string `json:"version"`
}

type RunConfig struct {
	InitialEquity   float64 `json:"initial_equity"`
	PositionSizePct float64 `json:"position_size_pct"`
	Leverage        int     `json:"leverage"`
	FeeRate         float64 `json:"fee_rate"`
	SlippageBps     float64 `json:"slippage_bps"`
	StopLossPct     float64 `json:"stop_loss_pct"`
	TakeProfitPct   float64 `json:"take_profit_pct"`
}

type StartRequest struct {
	StrategyTemplateID int64     `json:"strategy_template_id"`
	Symbol             string    `json:"symbol"`
	ExecutionInterval  string    `json:"execution_interval"`
	StartTime          int64     `json:"start_time"`
	EndTime            int64     `json:"end_time"`
	Config             RunConfig `json:"config"`
}

type Position struct {
	Side             string
	EntryTime        int64
	EntryPrice       float64
	Quantity         float64
	OpenFee          float64
	FundingPnL       float64
	OpenStrategyName string
	OpenStrategyType string
	OpenStrategyHash string
	MarketCondition  int
}

type PendingAction struct {
	Action          string
	Side            string
	StrategyName    string
	StrategyType    string
	StrategyHash    string
	SignalTime      int64
	MarketCondition int
}

type Trade struct {
	Sequence          int     `json:"sequence"`
	Symbol            string  `json:"symbol"`
	Side              string  `json:"side"`
	EntryTime         int64   `json:"entry_time"`
	ExitTime          int64   `json:"exit_time"`
	EntryPrice        float64 `json:"entry_price"`
	ExitPrice         float64 `json:"exit_price"`
	Quantity          float64 `json:"quantity"`
	GrossPnL          float64 `json:"gross_pnl"`
	Fees              float64 `json:"fees"`
	FundingPnL        float64 `json:"funding_pnl"`
	NetPnL            float64 `json:"net_pnl"`
	HoldingMs         int64   `json:"holding_ms"`
	ExitReason        string  `json:"exit_reason"`
	OpenStrategyName  string  `json:"open_strategy_name,omitempty"`
	OpenStrategyType  string  `json:"open_strategy_type,omitempty"`
	OpenStrategyHash  string  `json:"open_strategy_hash,omitempty"`
	CloseStrategyName string  `json:"close_strategy_name,omitempty"`
	CloseStrategyType string  `json:"close_strategy_type,omitempty"`
	CloseStrategyHash string  `json:"close_strategy_hash,omitempty"`
	MarketCondition   int     `json:"market_condition"`
}

type AuditEvent struct {
	Sequence  int             `json:"sequence"`
	EventTime int64           `json:"event_time"`
	Type      string          `json:"type"`
	Action    string          `json:"action"`
	Side      string          `json:"side,omitempty"`
	Price     float64         `json:"price,omitempty"`
	Quantity  float64         `json:"quantity,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type EquityPoint struct {
	Sequence      int     `json:"sequence"`
	BarTime       int64   `json:"bar_time"`
	Equity        float64 `json:"equity"`
	Cash          float64 `json:"cash"`
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	DrawdownPct   float64 `json:"drawdown_pct"`
	PositionSide  string  `json:"position_side,omitempty"`
}

type GroupMetrics struct {
	Key              string  `json:"key"`
	TradeCount       int     `json:"trade_count"`
	NetPnL           float64 `json:"net_pnl"`
	WinRate          float64 `json:"win_rate"`
	ProfitFactor     float64 `json:"profit_factor"`
	Fees             float64 `json:"fees"`
	Funding          float64 `json:"funding"`
	AverageHoldingMs int64   `json:"average_holding_ms"`
}

type Metrics struct {
	NetPnL            float64        `json:"net_pnl"`
	ReturnPct         float64        `json:"return_pct"`
	MaxDrawdownPct    float64        `json:"max_drawdown_pct"`
	WinRate           float64        `json:"win_rate"`
	ProfitFactor      float64        `json:"profit_factor"`
	Sharpe            float64        `json:"sharpe"`
	Sortino           float64        `json:"sortino"`
	TradeCount        int            `json:"trade_count"`
	Fees              float64        `json:"fees"`
	Funding           float64        `json:"funding"`
	AverageHoldingMs  int64          `json:"average_holding_ms"`
	BySide            []GroupMetrics `json:"by_side"`
	ByMarketCondition []GroupMetrics `json:"by_market_condition"`
}

type Result struct {
	DatasetID            string        `json:"dataset_id"`
	DatasetSpecHash      string        `json:"dataset_spec_hash"`
	DataHash             string        `json:"data_hash"`
	StrategyVersion      string        `json:"strategy_version"`
	EngineVersion        string        `json:"engine_version"`
	MarketConditionModel string        `json:"market_condition_model"`
	Metrics              Metrics       `json:"metrics"`
	Trades               []Trade       `json:"trades"`
	Events               []AuditEvent  `json:"events"`
	Equity               []EquityPoint `json:"equity"`
}

type RunSummary struct {
	RunID                string    `json:"run_id"`
	DatasetID            string    `json:"dataset_id"`
	DatasetSpecHash      string    `json:"dataset_spec_hash"`
	DataHash             string    `json:"data_hash"`
	StrategyTemplateID   int64     `json:"strategy_template_id"`
	StrategyTemplateName string    `json:"strategy_template_name"`
	StrategyVersion      string    `json:"strategy_version"`
	EngineVersion        string    `json:"engine_version"`
	MarketConditionModel string    `json:"market_condition_model"`
	Symbol               string    `json:"symbol"`
	ExecutionInterval    string    `json:"execution_interval"`
	StartTime            int64     `json:"start_time"`
	EndTime              int64     `json:"end_time"`
	Config               RunConfig `json:"config"`
	Status               string    `json:"status"`
	Stage                string    `json:"stage"`
	Progress             int       `json:"progress"`
	Metrics              *Metrics  `json:"metrics,omitempty"`
	Error                string    `json:"error,omitempty"`
	CreatedAt            int64     `json:"created_at"`
	StartedAt            int64     `json:"started_at,omitempty"`
	UpdatedAt            int64     `json:"updated_at"`
	CompletedAt          int64     `json:"completed_at,omitempty"`
}

type RunDetail struct {
	RunSummary
	Dataset *DatasetManifest `json:"dataset,omitempty"`
	Trades  []Trade          `json:"trades"`
	Events  []AuditEvent     `json:"events"`
	Equity  []EquityPoint    `json:"equity"`
}

type DatasetManifest struct {
	DatasetID         string   `json:"dataset_id"`
	DatasetSpecHash   string   `json:"dataset_spec_hash"`
	Market            string   `json:"market"`
	Symbol            string   `json:"symbol"`
	ExecutionInterval string   `json:"execution_interval"`
	Intervals         []string `json:"intervals"`
	BenchmarkSymbols  []string `json:"benchmark_symbols"`
	StartTime         int64    `json:"start_time"`
	EndTime           int64    `json:"end_time"`
	WarmupStartTime   int64    `json:"warmup_start_time"`
	CreatedAt         int64    `json:"created_at"`
}

func DefaultRunConfig() RunConfig {
	return RunConfig{InitialEquity: 1000, PositionSizePct: 1, Leverage: 1, FeeRate: 0.0005, SlippageBps: 5}
}

func NormalizeRunConfig(config RunConfig) RunConfig {
	defaults := DefaultRunConfig()
	if config == (RunConfig{}) {
		return defaults
	}
	if config.InitialEquity <= 0 {
		config.InitialEquity = defaults.InitialEquity
	}
	if config.PositionSizePct <= 0 {
		config.PositionSizePct = defaults.PositionSizePct
	}
	if config.PositionSizePct > 1 {
		config.PositionSizePct = 1
	}
	if config.Leverage <= 0 {
		config.Leverage = defaults.Leverage
	}
	if config.Leverage > 125 {
		config.Leverage = 125
	}
	if config.FeeRate < 0 {
		config.FeeRate = 0
	}
	if config.SlippageBps < 0 {
		config.SlippageBps = 0
	}
	if config.StopLossPct < 0 {
		config.StopLossPct = 0
	}
	if config.TakeProfitPct < 0 {
		config.TakeProfitPct = 0
	}
	return config
}

func intervalAnnualPeriods(interval time.Duration) float64 {
	if interval <= 0 {
		return 0
	}
	return (365 * 24 * float64(time.Hour)) / float64(interval)
}
