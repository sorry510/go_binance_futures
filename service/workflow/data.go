package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/expr-lang/expr"
	workflowSkill "go_binance_futures/agent/skills/workflows"
	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/models"
	"go_binance_futures/scanner"
	marketservice "go_binance_futures/service/market"
	strategyservice "go_binance_futures/service/strategy"
	"go_binance_futures/technology"
	markettypes "go_binance_futures/types"
	"go_binance_futures/utils"
)

func marketCondition(ctx context.Context) (*int, error) {
	c, err := (marketservice.Service{}).MarketCondition(ctx)
	if err != nil {
		return nil, err
	}
	v := c.MarketCondition
	return &v, nil
}
func buildMarketScanInput(ctx context.Context, analyze int) (workflowSkill.MarketScanInput, error) {
	if analyze <= 0 {
		analyze = 8
	}
	if analyze > 10 {
		analyze = 10
	}
	scan, err := scanner.ScanTop30(ctx, scanner.PrefilterOptions{Limit: 30, MaxDataAgeMs: int64(10 * time.Minute / time.Millisecond)})
	if err != nil {
		return workflowSkill.MarketScanInput{}, err
	}
	c, mcErr := marketCondition(ctx)
	missing := append([]string(nil), scan.DataMissing...)
	if mcErr != nil {
		missing = append(missing, "market_condition: "+mcErr.Error())
		c = nil
	}
	candidates := scan.Candidates
	if len(candidates) > analyze {
		candidates = candidates[:analyze]
	}
	return workflowSkill.MarketScanInput{Version: "market_scan_input_v1", GeneratedAt: scan.GeneratedAt, MarketCondition: c, Candidates: candidates, DataMissing: missing}, nil
}
func loadTemplate(ctx context.Context, id int64, name string) (models.StrategyTemplates, error) {
	return (strategyservice.Service{}).GetTemplate(ctx, strategyservice.TemplateQuery{ID: id, Name: name})
}
func templateSnapshot(t models.StrategyTemplates) workflowSkill.TemplateSnapshot {
	return workflowSkill.TemplateSnapshot{ID: t.ID, Name: t.Name, Technology: t.Technology, Strategy: t.Strategy, UpdatedAt: t.UpdateTime}
}
func buildStrategyStats(ctx context.Context, t models.StrategyTemplates, days int) (workflowSkill.StrategyStats, error) {
	if days <= 0 {
		days = 30
	}
	if days > 365 {
		days = 365
	}
	end := time.Now().UTC().UnixMilli()
	start := end - int64(time.Duration(days)*24*time.Hour/time.Millisecond)
	q := orm.NewOrm().QueryTable(new(models.TestStrategyResults)).Filter("createTime__gte", start).Filter("createTime__lte", end)
	if strings.TrimSpace(t.Strategy) != "" {
		q = q.Filter("strategy", t.Strategy)
	}
	if strings.TrimSpace(t.Technology) != "" {
		q = q.Filter("technology", t.Technology)
	}
	var rows []models.TestStrategyResults
	_, err := q.OrderBy("-createTime").Limit(2000).All(&rows)
	if err != nil {
		return workflowSkill.StrategyStats{}, err
	}
	s := workflowSkill.StrategyStats{Total: len(rows), WindowStart: start, WindowEnd: end}
	for _, r := range rows {
		if strings.EqualFold(r.PositionSide, "LONG") {
			s.LongTrades++
		} else if strings.EqualFold(r.PositionSide, "SHORT") {
			s.ShortTrades++
		}
		entry, e1 := strconv.ParseFloat(r.Price, 64)
		exit, e2 := strconv.ParseFloat(r.ClosePrice, 64)
		amt, e3 := strconv.ParseFloat(r.PositionAmt, 64)
		if e1 != nil || e2 != nil || e3 != nil || exit <= 0 || amt == 0 {
			continue
		}
		m := strategyservice.CalculateTestTradeProfit(entry, exit, amt, r.Leverage, r.OpenFeeRate, r.CloseFeeRate)
		s.Closed++
		s.GrossProfit += m.GrossProfit
		s.NetProfit += m.NetProfit
		s.Fees += m.TotalFee
		if m.NetProfit > 0 {
			s.Wins++
		} else {
			s.Losses++
		}
	}
	if s.Closed > 0 {
		s.WinRate = float64(s.Wins) / float64(s.Closed)
		s.AverageNet = s.NetProfit / float64(s.Closed)
	}
	return s, nil
}
func buildStrategyReviewInput(ctx context.Context, id int64, name string, days int) (workflowSkill.StrategyReviewInput, error) {
	t, err := loadTemplate(ctx, id, name)
	if err != nil {
		return workflowSkill.StrategyReviewInput{}, err
	}
	stats, err := buildStrategyStats(ctx, t, days)
	if err != nil {
		return workflowSkill.StrategyReviewInput{}, err
	}
	mc, mcErr := marketCondition(ctx)
	missing := []string{}
	if stats.Closed == 0 {
		missing = append(missing, "no closed matching test results in review window")
	}
	if mcErr != nil {
		missing = append(missing, "market_condition unavailable")
		mc = nil
	}
	return workflowSkill.StrategyReviewInput{Version: "strategy_review_input_v1", Template: templateSnapshot(t), Stats: stats, MarketCondition: mc, DataMissing: missing}, nil
}
func buildProposalInput(ctx context.Context, id int64, name, goal string) (workflowSkill.StrategyExperimentProposalInput, error) {
	t, err := loadTemplate(ctx, id, name)
	if err != nil {
		return workflowSkill.StrategyExperimentProposalInput{}, err
	}
	mc, _ := marketCondition(ctx)
	return workflowSkill.StrategyExperimentProposalInput{Version: "strategy_experiment_proposal_input_v1", Template: templateSnapshot(t), Goal: strings.TrimSpace(goal), MarketCondition: mc}, nil
}
func testProposal(p workflowSkill.StrategyExperimentProposalV1) workflowSkill.ExperimentTestReport {
	report := workflowSkill.ExperimentTestReport{Version: "strategy_experiment_test_v1", Errors: []string{}}
	var tech technology.TechnologyConfig
	if err := json.Unmarshal([]byte(p.TechnologyJSON), &tech); err != nil {
		report.Errors = append(report.Errors, "technology_json: "+err.Error())
		return report
	}
	var rules technology.StrategyConfig
	if err := json.Unmarshal([]byte(p.StrategyJSON), &rules); err != nil {
		report.Errors = append(report.Errors, "strategy_json: "+err.Error())
		return report
	}
	report.RuleCount = len(rules)
	env := syntheticEnv(tech)
	for _, r := range rules {
		if !r.Enable {
			continue
		}
		report.EnabledRuleCount++
		program, err := expr.Compile(r.Code, expr.Env(env), expr.AsBool())
		if err != nil {
			report.Errors = append(report.Errors, fmt.Sprintf("%s compile: %v", r.Name, err))
			continue
		}
		report.CompiledRules++
		for _, scenario := range []float64{-8, 0, 8} {
			env["NowSymbolPercentChange"] = scenario
			env["BasicTrend"] = scenario / 2
			report.ScenarioRuns++
			if _, err := expr.Run(program, env); err == nil {
				report.ScenarioPasses++
			} else {
				report.Errors = append(report.Errors, fmt.Sprintf("%s run: %v", r.Name, err))
				break
			}
		}
	}
	report.Valid = report.EnabledRuleCount > 0 && report.CompiledRules == report.EnabledRuleCount && len(report.Errors) == 0
	return report
}
func syntheticEnv(tech technology.TechnologyConfig) map[string]any {
	series := make([]float64, 150)
	for i := range series {
		series[i] = 100 + float64(150-i)/10
	}
	market := struct{ PercentChange, Close, Open, Low, High float64 }{1, 101, 100, 99, 102}
	env := map[string]any{
		"SystemStartTime": int64(1_700_000_000_000), "MarketCondition": "3", "NowTime": int64(1_700_000_060_000),
		"NowPrice": 101.0, "NowSymbolPercentChange": 1.0, "NowSymbolClose": 101.0, "NowSymbolOpen": 100.0, "NowSymbolLow": 99.0, "NowSymbolHigh": 102.0,
		"BasicTrend": 0.5, "ROI": 5.0, "KdjSimple": line.KdjSimple, "IsAsc": utils.IsAsc, "IsDesc": utils.IsDesc,
		"BTCUSDT": market, "ETHUSDT": market, "SOLUSDT": market, "BNBUSDT": market,
		"Positions": []markettypes.FuturesPosition{}, "Position": markettypes.FuturesPositionCode{Symbol: "BTCUSDT", Side: "LONG", Amount: 0.01, Leverage: 3, EntryPrice: 100, MarkPrice: 101, UnrealizedProfit: 1, CreateTime: 1_700_000_000_000, SourceType: "local"},
	}
	intervals := map[string]bool{}
	add := func(kind string, items []technology.IndicatorConfig) {
		for _, item := range items {
			if !item.Enable || strings.TrimSpace(item.Name) == "" {
				continue
			}
			intervals[item.KlineInterval] = true
			switch kind {
			case "macd":
				env[item.Name] = line.MACDConfigData{KlineInterval: item.KlineInterval, FastPeriod: item.FastPeriod, SlowPeriod: item.SlowPeriod, SignalPeriod: item.SignalPeriod, DIF: series, DEA: series, Histogram: series}
			case "adx":
				env[item.Name] = line.ADXConfigData{KlineInterval: item.KlineInterval, Period: item.Period, ADX: series, PlusDI: series, MinusDI: series}
			case "kdj":
				env[item.Name] = line.KDJConfigData{KlineInterval: item.KlineInterval, Period: item.Period, KPeriod: item.KPeriod, DPeriod: item.DPeriod, K: series, D: series, J: series}
			case "supertrend":
				env[item.Name] = line.SupertrendConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, Data: series, Trend: series}
			case "obv":
				env[item.Name] = line.OBVConfigData{KlineInterval: item.KlineInterval, Data: series}
			case "kc", "boll", "donchian":
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Multiplier: item.Multiplier, StdDevMultiplier: item.StdDevMultiplier, High: series, Mid: series, Low: series}
			default:
				env[item.Name] = line.ConfigData{KlineInterval: item.KlineInterval, Period: item.Period, Data: series}
			}
		}
	}
	add("ma", tech.MA)
	add("ema", tech.EMA)
	add("macd", tech.MACD)
	add("rsi", tech.RSI)
	add("kc", tech.KC)
	add("boll", tech.BOLL)
	add("atr", tech.ATR)
	add("adx", tech.ADX)
	add("mfi", tech.MFI)
	add("obv", tech.OBV)
	add("cci", tech.CCI)
	add("roc", tech.ROC)
	add("kdj", tech.KDJ)
	add("supertrend", tech.Supertrend)
	add("donchian", tech.Donchian)
	for interval := range intervals {
		env["kline_"+interval] = line.KLinePrice{High: series, Low: series, Close: series, Open: series, Amount: series, Qps: series}
	}
	return env
}
func loadAlertTraceRows(ctx context.Context, start, end int64, limit int) ([]models.AgentAlertPipelineTrace, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := orm.NewOrm().QueryTable(new(models.AgentAlertPipelineTrace)).Filter("created_at__gte", start).Filter("created_at__lte", end)
	total, err := q.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.AgentAlertPipelineTrace
	_, err = q.OrderBy("-created_at").Limit(limit).All(&rows)
	return rows, total, err
}
func buildAlertTriageInput(ctx context.Context, minutes, maxSignals int) (workflowSkill.AlertTriageInput, error) {
	if minutes <= 0 {
		minutes = 15
	}
	if minutes > 120 {
		minutes = 120
	}
	if maxSignals <= 0 {
		maxSignals = 100
	}
	end := time.Now().UTC().UnixMilli()
	start := end - int64(time.Duration(minutes)*time.Minute/time.Millisecond)
	rows, _, err := loadAlertTraceRows(ctx, start, end, min(maxSignals, 100))
	if err != nil {
		return workflowSkill.AlertTriageInput{}, err
	}
	bySymbol := map[string][]workflowSkill.IncidentSignal{}
	for _, x := range rows {
		bySymbol[x.Symbol] = append(bySymbol[x.Symbol], workflowSkill.IncidentSignal{SignalID: x.SignalID, Symbol: x.Symbol, Type: x.Type, Severity: x.Severity, CreatedAt: x.CreatedAt})
	}
	keys := make([]string, 0, len(bySymbol))
	for k := range bySymbol {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	candidates := []workflowSkill.IncidentCandidate{}
	for _, symbol := range keys {
		signals := bySymbol[symbol]
		if len(signals) == 0 {
			continue
		}
		sort.Slice(signals, func(i, j int) bool { return signals[i].CreatedAt < signals[j].CreatedAt })
		candidates = append(candidates, workflowSkill.IncidentCandidate{CandidateID: fmt.Sprintf("inc_%s_%d", strings.ToLower(symbol), signals[0].CreatedAt), WindowStart: signals[0].CreatedAt, WindowEnd: signals[len(signals)-1].CreatedAt, Symbols: []string{symbol}, Signals: signals})
	}
	return workflowSkill.AlertTriageInput{Version: "alert_triage_input_v1", WindowStart: start, WindowEnd: end, Candidates: candidates}, nil
}
func buildSignalSummary(ctx context.Context, hours int) (workflowSkill.SignalSummary, error) {
	if hours <= 0 {
		hours = 24
	}
	end := time.Now().UTC().UnixMilli()
	start := end - int64(time.Duration(hours)*time.Hour/time.Millisecond)
	rows, total, err := loadAlertTraceRows(ctx, start, end, 100)
	if err != nil {
		return workflowSkill.SignalSummary{}, err
	}
	s := workflowSkill.SignalSummary{Total: int(total), ByType: map[string]int{}, BySeverity: map[string]int{}, Symbols: []string{}}
	seen := map[string]bool{}
	for _, x := range rows {
		s.ByType[x.Type]++
		s.BySeverity[x.Severity]++
		if !seen[x.Symbol] {
			seen[x.Symbol] = true
			s.Symbols = append(s.Symbols, x.Symbol)
		}
	}
	sort.Strings(s.Symbols)
	return s, nil
}
func buildDailyBriefInput(ctx context.Context, hours int) (workflowSkill.DailyMarketBriefInput, error) {
	scan, err := buildMarketScanInput(ctx, 8)
	if err != nil {
		return workflowSkill.DailyMarketBriefInput{}, err
	}
	signals, sigErr := buildSignalSummary(ctx, hours)
	missing := append([]string(nil), scan.DataMissing...)
	if sigErr != nil {
		missing = append(missing, "signal summary unavailable: "+sigErr.Error())
		signals = workflowSkill.SignalSummary{ByType: map[string]int{}, BySeverity: map[string]int{}, Symbols: []string{}}
	}
	return workflowSkill.DailyMarketBriefInput{Version: "daily_market_brief_input_v1", AsOf: time.Now().UTC().Format(time.RFC3339), MarketCondition: scan.MarketCondition, Candidates: scan.Candidates, Signals: signals, DataMissing: missing}, nil
}
func finite(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func resolveStrategyTemplateFromChat(ctx context.Context, prompt string) (int64, string, bool, error) {
	if err := ctx.Err(); err != nil {
		return 0, "", false, err
	}
	var templates []models.StrategyTemplates
	if _, err := orm.NewOrm().QueryTable(new(models.StrategyTemplates)).OrderBy("id").All(&templates); err != nil {
		return 0, "", false, err
	}
	normalized := strings.ToLower(strings.TrimSpace(prompt))
	best := -1
	for i := range templates {
		name := strings.TrimSpace(templates[i].Name)
		if name == "" {
			continue
		}
		if strings.Contains(normalized, strings.ToLower(name)) {
			if best < 0 || len(name) > len(strings.TrimSpace(templates[best].Name)) {
				best = i
			}
		}
	}
	if best >= 0 {
		return templates[best].ID, "", true, nil
	}
	for _, prefix := range []string{"id=", "id:", "id ", "#", "模板id=", "模板id:", "模板id "} {
		index := strings.Index(normalized, prefix)
		if index < 0 {
			continue
		}
		rest := strings.TrimSpace(normalized[index+len(prefix):])
		digits := strings.Builder{}
		for _, r := range rest {
			if r < '0' || r > '9' {
				break
			}
			digits.WriteRune(r)
		}
		if digits.Len() == 0 {
			continue
		}
		id, err := strconv.ParseInt(digits.String(), 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		for _, item := range templates {
			if item.ID == id {
				return id, "", true, nil
			}
		}
	}
	return 0, "", false, nil
}
