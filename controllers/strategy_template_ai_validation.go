package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"go_binance_futures/feature/strategy/line"
	"go_binance_futures/technology"
	"go_binance_futures/types"
	"go_binance_futures/utils"

	"github.com/expr-lang/expr"
)

type strategyTemplateSyntheticMarket struct {
	PercentChange float64
	Close         float64
	Open          float64
	Low           float64
	High          float64
}

func formatStrategyTemplateJSON(value string) string {
	var buffer bytes.Buffer
	if err := json.Indent(&buffer, []byte(value), "", "  "); err != nil {
		return value
	}
	return buffer.String()
}

func validateGeneratedStrategyTemplateJSON(data []byte) error {
	if _, err := parseStrategyTemplateImport(data); err != nil {
		return err
	}
	if err := validateGeneratedTechnologyKeys(data); err != nil {
		return err
	}
	return nil
}

func validateStrategyTemplateRuleExpressions(config technology.TechnologyConfig, rules []strategyTemplateImportRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("strategy 至少需要一条策略规则")
	}
	for index, rule := range rules {
		env := buildSyntheticStrategyTemplateEnv(config, rule.Type)
		program, err := expr.Compile(rule.Code, expr.Env(env), expr.AsBool())
		if err != nil {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 编译失败: %w", index+1, rule.Name, err)
		}
		output, err := expr.Run(program, env)
		if err != nil {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 运行失败: %w", index+1, rule.Name, err)
		}
		if _, ok := output.(bool); !ok {
			return fmt.Errorf("strategy 第 %d 项 %q 的 expr 最终结果必须是布尔值", index+1, rule.Name)
		}
	}
	return nil
}

func validateGeneratedTechnologyKeys(data []byte) error {
	var payload struct {
		Technology map[string]json.RawMessage `json:"technology"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	required := []string{"ma", "ema", "macd", "adx", "mfi", "obv", "cci", "roc", "kdj", "rsi", "kc", "boll", "donchian", "atr", "supertrend"}
	for _, key := range required {
		value, exists := payload.Technology[key]
		if !exists {
			return fmt.Errorf("technology 缺少数组字段 %s", key)
		}
		var items []json.RawMessage
		if err := json.Unmarshal(value, &items); err != nil {
			return fmt.Errorf("technology.%s 必须是数组", key)
		}
	}
	return nil
}

func buildSyntheticStrategyTemplateEnv(config technology.TechnologyConfig, ruleType string) map[string]interface{} {
	series := syntheticStrategyTemplateSeries()
	market := strategyTemplateSyntheticMarket{PercentChange: 1.0, Close: 101.0, Open: 100.0, Low: 99.0, High: 102.0}
	env := map[string]interface{}{
		"SystemStartTime":        int64(1_700_000_000_000),
		"MarketCondition":        "3",
		"NowTime":                int64(1_700_000_060_000),
		"NowPrice":               101.0,
		"NowSymbolPercentChange": 1.0,
		"NowSymbolClose":         101.0,
		"NowSymbolOpen":          100.0,
		"NowSymbolLow":           99.0,
		"NowSymbolHigh":          102.0,
		"BasicTrend":             0.5,
		"KdjSimple":              line.KdjSimple,
		"IsAsc":                  utils.IsAsc,
		"IsDesc":                 utils.IsDesc,
		"BTCUSDT":                market,
		"ETHUSDT":                market,
		"SOLUSDT":                market,
		"BNBUSDT":                market,
	}
	if ruleType == "long" || ruleType == "short" {
		env["Positions"] = []types.FuturesPosition{}
	} else {
		env["ROI"] = 8.0
		env["Position"] = types.FuturesPositionCode{
			Symbol: "BTCUSDT", Side: "LONG", Amount: 0.01, Leverage: 3,
			EntryPrice: 100.0, MarkPrice: 101.0, UnrealizedProfit: 1.0,
			CreateTime: 1_700_000_000_000, SourceType: "local",
		}
	}
	intervals := make(map[string]struct{})
	addItems := func(kind string, items []technology.IndicatorConfig) {
		for _, item := range items {
			if !item.Enable {
				continue
			}
			intervals[item.KlineInterval] = struct{}{}
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
	addItems("ma", config.MA)
	addItems("ema", config.EMA)
	addItems("macd", config.MACD)
	addItems("adx", config.ADX)
	addItems("mfi", config.MFI)
	addItems("obv", config.OBV)
	addItems("cci", config.CCI)
	addItems("roc", config.ROC)
	addItems("kdj", config.KDJ)
	addItems("rsi", config.RSI)
	addItems("kc", config.KC)
	addItems("boll", config.BOLL)
	addItems("donchian", config.Donchian)
	addItems("atr", config.ATR)
	addItems("supertrend", config.Supertrend)
	for interval := range intervals {
		env["kline_"+interval] = line.KLinePrice{High: series, Low: series, Close: series, Open: series, Amount: series, Qps: series}
	}
	return env
}

func syntheticStrategyTemplateSeries() []float64 {
	values := make([]float64, 150)
	for index := range values {
		values[index] = 100 + float64(150-index)/10
	}
	return values
}

func buildStrategyTemplateAIRepairGuidance(errorMessage, candidateJSON string) string {
	unknownName := extractStrategyTemplateAIUnknownName(errorMessage)
	if unknownName == "" {
		return ""
	}

	type indicatorName struct {
		Name string `json:"name"`
	}
	var candidate struct {
		Technology map[string][]indicatorName `json:"technology"`
	}
	_ = json.Unmarshal([]byte(candidateJSON), &candidate)
	fieldsByFamily := map[string][]string{
		"ma":         {"Data"},
		"ema":        {"Data"},
		"macd":       {"DIF", "DEA", "Histogram"},
		"adx":        {"ADX", "PlusDI", "MinusDI"},
		"mfi":        {"Data"},
		"obv":        {"Data"},
		"cci":        {"Data"},
		"roc":        {"Data"},
		"kdj":        {"K", "D", "J"},
		"rsi":        {"Data"},
		"kc":         {"High", "Mid", "Low"},
		"boll":       {"High", "Mid", "Low"},
		"donchian":   {"High", "Mid", "Low"},
		"atr":        {"Data"},
		"supertrend": {"Data", "Trend"},
	}
	for family, indicators := range candidate.Technology {
		for _, indicator := range indicators {
			for _, field := range fieldsByFamily[family] {
				flattenedName := indicator.Name + "_" + field
				if unknownName == flattenedName {
					return fmt.Sprintf("Replace every %s[...] with %s.%s[...]. %s is the configured indicator object and %s is its field; flattened indicator variables do not exist.", unknownName, indicator.Name, field, indicator.Name, field)
				}
			}
		}
	}

	knownFields := []string{"Data", "DIF", "DEA", "Histogram", "ADX", "PlusDI", "MinusDI", "K", "D", "J", "High", "Mid", "Low", "Trend", "Close", "Open", "Amount", "Qps"}
	for _, field := range knownFields {
		suffix := "_" + field
		if strings.HasSuffix(unknownName, suffix) {
			objectName := strings.TrimSuffix(unknownName, suffix)
			return fmt.Sprintf("%s is not a runtime variable. Use object field syntax %s.%s[...] instead of the flattened form %s[...], and fix every occurrence in all strategy rules.", unknownName, objectName, field, unknownName)
		}
	}
	return "Use only configured indicator object names, kline_INTERVAL objects, and documented global variables; do not invent flattened variable names."
}

func extractStrategyTemplateAIUnknownName(errorMessage string) string {
	const marker = "unknown name "
	position := strings.Index(errorMessage, marker)
	if position < 0 {
		return ""
	}
	value := errorMessage[position+len(marker):]
	if end := strings.IndexAny(value, " \t\r\n("); end >= 0 {
		value = value[:end]
	}
	return strings.TrimSpace(value)
}
