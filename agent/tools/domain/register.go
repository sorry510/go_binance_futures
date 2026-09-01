package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"go_binance_futures/agent/permission"
	agenttools "go_binance_futures/agent/tools"
	"go_binance_futures/scanner"
	liquidationservice "go_binance_futures/service/liquidation"
	marketservice "go_binance_futures/service/market"
	strategyservice "go_binance_futures/service/strategy"
	symbolservice "go_binance_futures/service/symbol"
	symbolanalysisservice "go_binance_futures/service/symbolanalysis"
)

type Dependencies struct {
	GetSymbol                  func(context.Context, string) (any, error)
	ListSymbols                func(context.Context, symbolservice.ListOptions) (symbolservice.ListResult, error)
	GetKlines                  func(context.Context, string, string, int) (any, error)
	GetFundingRate             func(context.Context, string) (any, error)
	GetMarketCondition         func(context.Context) (any, error)
	ListLiquidations           func(context.Context, liquidationservice.ListOptions) (liquidationservice.ListResult, error)
	ScanSymbols                func(context.Context, scanner.PrefilterOptions) (*scanner.PrefilterResult, error)
	ListTestResults            func(context.Context, strategyservice.TestResultsOptions) (strategyservice.TestResultsResult, error)
	GetStrategyTemplate        func(context.Context, strategyservice.TemplateQuery) (any, error)
	BuildSymbolAnalysisContext func(context.Context, string) (symbolanalysisservice.Context, error)
}

func DefaultDependencies() Dependencies {
	symbols := symbolservice.Service{}
	market := marketservice.Service{}
	liquidations := liquidationservice.Service{}
	strategies := strategyservice.Service{}
	return Dependencies{
		GetSymbol:   func(ctx context.Context, value string) (any, error) { return symbols.Snapshot(ctx, value) },
		ListSymbols: symbols.List,
		GetKlines: func(ctx context.Context, symbol, interval string, limit int) (any, error) {
			return market.Klines(ctx, symbol, interval, limit)
		},
		GetFundingRate:     func(ctx context.Context, symbol string) (any, error) { return market.FundingRate(ctx, symbol) },
		GetMarketCondition: func(ctx context.Context) (any, error) { return market.MarketCondition(ctx) },
		ListLiquidations:   liquidations.List,
		ScanSymbols:        scanner.ScanTop30,
		ListTestResults:    strategies.ListTestResults,
		GetStrategyTemplate: func(ctx context.Context, query strategyservice.TemplateQuery) (any, error) {
			return strategies.GetTemplate(ctx, query)
		},
		BuildSymbolAnalysisContext: func(ctx context.Context, symbol string) (symbolanalysisservice.Context, error) {
			return symbolanalysisservice.Build(ctx, symbol, symbolanalysisservice.DefaultDependencies())
		},
	}
}

func RegisterReadOnly(registry *agenttools.Registry, deps Dependencies) error {
	if registry == nil {
		return fmt.Errorf("tool registry is nil")
	}
	definitions := []agenttools.Tool{
		newFeaturesTool(deps), newSymbolSnapshotTool(deps), newKlinesTool(deps), newFundingRateTool(deps), newLiquidationsTool(deps),
		newMarketConditionTool(deps), newSymbolAnalysisContextTool(deps), newScanSymbolsTool(deps), newTestResultsTool(deps), newStrategyTemplateTool(deps),
	}
	for _, tool := range definitions {
		if err := registry.Register(tool); err != nil {
			return err
		}
	}
	return nil
}

func metadata(inputSchema string, timeout time.Duration, maxBytes int) agenttools.Metadata {
	return agenttools.Metadata{InputSchema: json.RawMessage(inputSchema), OutputSchema: json.RawMessage(`{"type":["object","array"]}`), Timeout: timeout, MaxResultBytes: maxBytes, Idempotent: true}
}

func newFeaturesTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Sort       string `json:"sort"`
		SymbolType string `json:"symbol_type"`
		Symbol     string `json:"symbol"`
		Enable     string `json:"enable"`
		MarginType string `json:"margin_type"`
		Pin        string `json:"pin"`
		Page       int    `json:"page"`
		Limit      int    `json:"limit"`
	}
	return agenttools.Func{ToolName: "get_features", ToolDescription: "查询合约列表、行情快照和交易配置，兼容策略生成数据查询", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","properties":{"sort":{"type":"string"},"symbol_type":{"type":"string"},"symbol":{"type":"string"},"enable":{"type":"string"},"margin_type":{"type":"string"},"pin":{"type":"string"},"page":{"type":"integer"},"limit":{"type":"integer","maximum":20}}}`, 5*time.Second, 128<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.ListSymbols == nil {
				return nil, fmt.Errorf("symbol service is unavailable")
			}
			return deps.ListSymbols(ctx, symbolservice.ListOptions{Sort: in.Sort, SymbolType: in.SymbolType, Symbol: in.Symbol, Enable: in.Enable, MarginType: in.MarginType, Pin: in.Pin, Page: in.Page, Limit: in.Limit, DefaultLimit: 20, MaxLimit: 20})
		}}
}

func newSymbolSnapshotTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol string `json:"symbol"`
	}
	return agenttools.Func{ToolName: "get_symbol_snapshot", ToolDescription: "查询单个合约本地最新行情和交易配置快照", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","required":["symbol"],"properties":{"symbol":{"type":"string"}}}`, 5*time.Second, 32<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.GetSymbol == nil {
				return nil, fmt.Errorf("symbol service is unavailable")
			}
			return deps.GetSymbol(ctx, in.Symbol)
		}}
}

func newKlinesTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol   string `json:"symbol"`
		Interval string `json:"interval"`
		Limit    int    `json:"limit"`
	}
	return agenttools.Func{ToolName: "get_klines", ToolDescription: "获取 Binance U 本位合约 K 线，结果按最新到最旧排列", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","required":["symbol","interval"],"properties":{"symbol":{"type":"string"},"interval":{"type":"string"},"limit":{"type":"integer","minimum":1,"maximum":1000}}}`, 20*time.Second, 256<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.GetKlines == nil {
				return nil, fmt.Errorf("market service is unavailable")
			}
			return deps.GetKlines(ctx, in.Symbol, in.Interval, in.Limit)
		}}
}

func newFundingRateTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol string `json:"symbol"`
	}
	return agenttools.Func{ToolName: "get_funding_rate", ToolDescription: "获取合约当前标记价格和资金费率", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","required":["symbol"],"properties":{"symbol":{"type":"string"}}}`, 15*time.Second, 64<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.GetFundingRate == nil {
				return nil, fmt.Errorf("market service is unavailable")
			}
			return deps.GetFundingRate(ctx, in.Symbol)
		}}
}

func newMarketConditionTool(deps Dependencies) agenttools.Tool {
	return agenttools.Func{ToolName: "get_market_condition", ToolDescription: "获取当前市场趋势：读取 Phase 3A Market Regime 最新保存到 systemConfig.MarketCondition 的结果，不触发重新分析", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","additionalProperties":false}`, 5*time.Second, 8<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in struct{}
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.GetMarketCondition == nil {
				return nil, fmt.Errorf("market service is unavailable")
			}
			return deps.GetMarketCondition(ctx)
		}}
}

func newSymbolAnalysisContextTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol string `json:"symbol"`
	}
	return agenttools.Func{ToolName: "get_symbol_analysis_context", ToolDescription: "聚合单个 USDT 永续合约分析上下文：行情快照、MarketCondition、多周期 K 线特征、Funding、OI、Taker、Depth 与最近强平；失败项写入 data_missing", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","required":["symbol"],"additionalProperties":false,"properties":{"symbol":{"type":"string"}}}`, 40*time.Second, 96<<10),
		RestoreCheckpointFunc: func(raw json.RawMessage) (any, error) {
			var value symbolanalysisservice.Context
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, err
			}
			return value, nil
		},
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.BuildSymbolAnalysisContext == nil {
				return nil, fmt.Errorf("symbol analysis context service is unavailable")
			}
			return deps.BuildSymbolAnalysisContext(ctx, in.Symbol)
		}}
}

func newLiquidationsTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol      string  `json:"symbol"`
		Side        string  `json:"side"`
		StartTime   int64   `json:"start_time"`
		EndTime     int64   `json:"end_time"`
		MinNotional float64 `json:"min_notional"`
		Page        int     `json:"page"`
		Limit       int     `json:"limit"`
	}
	return agenttools.Func{ToolName: "get_liquidations", ToolDescription: "查询本地采集的合约强平订单", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","properties":{"symbol":{"type":"string"},"side":{"type":"string"},"start_time":{"type":"integer"},"end_time":{"type":"integer"},"min_notional":{"type":"number"},"page":{"type":"integer"},"limit":{"type":"integer"}}}`, 5*time.Second, 128<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.ListLiquidations == nil {
				return nil, fmt.Errorf("liquidation service is unavailable")
			}
			return deps.ListLiquidations(ctx, liquidationservice.ListOptions{Symbol: in.Symbol, Side: in.Side, StartTime: in.StartTime, EndTime: in.EndTime, MinNotional: in.MinNotional, Page: in.Page, Limit: in.Limit, DefaultLimit: 100, MaxLimit: 500})
		}}
}

func newScanSymbolsTool(deps Dependencies) agenttools.Tool {
	return agenttools.Func{ToolName: "scan_symbols", ToolDescription: "使用本地确定性预筛选器扫描高流动性合约候选", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","properties":{"limit":{"type":"integer"},"min_quote_volume":{"type":"number"},"max_data_age_ms":{"type":"integer"}}}`, 8*time.Second, 128<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in scanner.PrefilterOptions
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.ScanSymbols == nil {
				return nil, fmt.Errorf("scanner service is unavailable")
			}
			return deps.ScanSymbols(ctx, in)
		}}
}

func newTestResultsTool(deps Dependencies) agenttools.Tool {
	type input struct {
		Symbol       string `json:"symbol"`
		PositionSide string `json:"position_side"`
		StartTime    string `json:"start_time"`
		EndTime      string `json:"end_time"`
		Type         string `json:"type"`
		Page         int    `json:"page"`
		Limit        int    `json:"limit"`
	}
	return agenttools.Func{ToolName: "get_test_strategy_results", ToolDescription: "查询模拟策略测试结果及当前浮动收益汇总", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","properties":{"symbol":{"type":"string"},"position_side":{"type":"string"},"start_time":{"type":"string"},"end_time":{"type":"string"},"type":{"type":"string"},"page":{"type":"integer"},"limit":{"type":"integer"}}}`, 8*time.Second, 192<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if deps.ListTestResults == nil {
				return nil, fmt.Errorf("strategy service is unavailable")
			}
			return deps.ListTestResults(ctx, strategyservice.TestResultsOptions{Symbol: in.Symbol, PositionSide: in.PositionSide, StartTime: in.StartTime, EndTime: in.EndTime, Type: in.Type, Page: in.Page, Limit: in.Limit, DefaultLimit: 100, MaxLimit: 100})
		}}
}

func newStrategyTemplateTool(deps Dependencies) agenttools.Tool {
	type input struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	return agenttools.Func{ToolName: "get_strategy_template", ToolDescription: "按 ID 或精确名称查询一个自定义策略模板", ToolRisk: permission.RiskRead,
		ToolMetadata: metadata(`{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}}}`, 5*time.Second, 128<<10),
		ExecuteFunc: func(ctx context.Context, raw json.RawMessage) (any, error) {
			var in input
			if err := strictDecode(raw, &in); err != nil {
				return nil, err
			}
			if in.ID <= 0 && strings.TrimSpace(in.Name) == "" {
				return nil, fmt.Errorf("id or name is required")
			}
			if deps.GetStrategyTemplate == nil {
				return nil, fmt.Errorf("strategy service is unavailable")
			}
			return deps.GetStrategyTemplate(ctx, strategyservice.TemplateQuery{ID: in.ID, Name: in.Name})
		}}
}

func strictDecode(raw json.RawMessage, target any) error {
	if len(raw) == 0 || string(raw) == "null" {
		raw = json.RawMessage(`{}`)
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("invalid tool arguments: multiple JSON values")
		}
		return fmt.Errorf("invalid tool arguments: %w", err)
	}
	return nil
}
