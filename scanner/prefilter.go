package scanner

import (
	"context"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"go_binance_futures/models"

	"github.com/beego/beego/v2/client/orm"
)

const (
	defaultTop30Limit      = 30
	defaultMinQuoteVolume  = 10_000_000
	defaultMaxCenterOffset = 0.15
)

type PrefilterOptions struct {
	Limit          int     `json:"limit"`
	MinQuoteVolume float64 `json:"min_quote_volume"`
	MaxDataAgeMs   int64   `json:"max_data_age_ms"`
}

type PrefilterResult struct {
	GeneratedAt int64                  `json:"generated_at"`
	Source      string                 `json:"source"`
	Candidates  []PrefilterCandidate   `json:"candidates"`
	Excluded    []PrefilterExclusion   `json:"excluded"`
	DataMissing []string               `json:"data_missing"`
	Meta        map[string]interface{} `json:"meta"`
}

type PrefilterCandidate struct {
	Rank               int      `json:"rank"`
	Symbol             string   `json:"symbol"`
	Score              float64  `json:"score"`
	Grade              string   `json:"grade"`
	Price              float64  `json:"price"`
	PercentChange24h   float64  `json:"percent_change_24h"`
	QuoteVolume24h     float64  `json:"quote_volume_24h"`
	TradeCount24h      float64  `json:"trade_count_24h"`
	High24h            float64  `json:"high_24h"`
	Low24h             float64  `json:"low_24h"`
	Open24h            float64  `json:"open_24h"`
	CenterOffsetPct    float64  `json:"center_offset_pct"`
	UpperWickRatio     float64  `json:"upper_wick_ratio"`
	RetraceFromHighPct float64  `json:"retrace_from_high_pct"`
	LocalMomentumPct   float64  `json:"local_momentum_pct"`
	NotChase           bool     `json:"not_chase"`
	Reasons            []string `json:"reasons"`
	Risks              []string `json:"risks"`
	Missing            []string `json:"missing"`
	LastUpdateTime     int64    `json:"last_update_time"`
}

type PrefilterExclusion struct {
	Symbol string `json:"symbol"`
	Reason string `json:"reason"`
}

type symbolMetrics struct {
	price            float64
	open             float64
	low              float64
	high             float64
	centerOffsetPct  float64
	upperWickRatio   float64
	retraceFromHigh  float64
	localMomentumPct float64
	hasLocalMomentum bool
}

func ScanTop30(ctx context.Context, opts PrefilterOptions) (*PrefilterResult, error) {
	var symbols []*models.Symbols
	_, err := orm.NewOrm().QueryTable(new(models.Symbols)).All(&symbols,
		"Symbol", "PercentChange", "Close", "Open", "Low", "High",
		"QuoteVolume", "TradeCount", "Type", "LastClose", "LastUpdateTime", "UpdateTime",
	)
	if err != nil {
		return nil, err
	}

	result := PrefilterTop30FromSymbols(symbols, opts)
	return &result, nil
}

func PrefilterTop30FromSymbols(symbols []*models.Symbols, opts PrefilterOptions) PrefilterResult {
	limit := opts.Limit
	if limit <= 0 || limit > defaultTop30Limit {
		limit = defaultTop30Limit
	}
	minQuoteVolume := opts.MinQuoteVolume
	if minQuoteVolume <= 0 {
		minQuoteVolume = defaultMinQuoteVolume
	}

	now := time.Now().UnixMilli()
	result := PrefilterResult{
		GeneratedAt: now,
		Source:      "local_db.symbols",
		DataMissing: []string{
			"15m/1h K线转强状态未在 symbols 表本地持久化",
			"盘口点差未在本地 depth/order_book 表持久化",
			"Binance 官方风险公告未在本地公告表持久化",
		},
		Meta: map[string]interface{}{
			"limit":            limit,
			"min_quote_volume": minQuoteVolume,
			"rest_api_used":    false,
		},
	}

	for _, item := range symbols {
		if item == nil {
			continue
		}
		metrics, ok, reason := localSymbolMetrics(item)
		if !ok {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: reason})
			continue
		}
		if strings.TrimSpace(item.Type) != "USDT" || !strings.HasSuffix(item.Symbol, "USDT") {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "非 USDT 本位合约"})
			continue
		}
		if item.Symbol == "BTCUSDT" || item.Symbol == "ETHUSDT" {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "BTC/ETH 仅用于市场环境，不作为山寨候选"})
			continue
		}
		if item.QuoteVolume < minQuoteVolume {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "24h 成交额低于阈值"})
			continue
		}
		if opts.MaxDataAgeMs > 0 && item.UpdateTime > 0 && now-item.UpdateTime > opts.MaxDataAgeMs {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "本地行情数据过旧"})
			continue
		}
		if item.PercentChange > 60 {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "24h 涨幅超过 60%，默认剔除"})
			continue
		}
		if item.PercentChange > 40 && (metrics.upperWickRatio >= 0.35 || metrics.retraceFromHigh >= 18) {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "24h 涨幅超过 40% 且长上影/冲高回落明显"})
			continue
		}
		if metrics.centerOffsetPct > defaultMaxCenterOffset*100 {
			result.Excluded = append(result.Excluded, PrefilterExclusion{Symbol: item.Symbol, Reason: "价格远离 24h 中枢"})
			continue
		}

		candidate := buildCandidate(item, metrics, minQuoteVolume)
		result.Candidates = append(result.Candidates, candidate)
	}

	sort.SliceStable(result.Candidates, func(i, j int) bool {
		if result.Candidates[i].Score == result.Candidates[j].Score {
			return result.Candidates[i].QuoteVolume24h > result.Candidates[j].QuoteVolume24h
		}
		return result.Candidates[i].Score > result.Candidates[j].Score
	})
	if len(result.Candidates) > limit {
		result.Candidates = result.Candidates[:limit]
	}
	for i := range result.Candidates {
		result.Candidates[i].Rank = i + 1
	}

	return result
}

func localSymbolMetrics(item *models.Symbols) (symbolMetrics, bool, string) {
	price, ok := parsePositive(item.Close)
	if !ok {
		return symbolMetrics{}, false, "缺少有效当前价格"
	}
	open, ok := parsePositive(item.Open)
	if !ok {
		return symbolMetrics{}, false, "缺少有效 24h open"
	}
	low, ok := parsePositive(item.Low)
	if !ok {
		return symbolMetrics{}, false, "缺少有效 24h low"
	}
	high, ok := parsePositive(item.High)
	if !ok {
		return symbolMetrics{}, false, "缺少有效 24h high"
	}
	if high <= low {
		return symbolMetrics{}, false, "24h high/low 异常"
	}

	center := (high + low) / 2
	upperWick := high - math.Max(open, price)
	if upperWick < 0 {
		upperWick = 0
	}
	m := symbolMetrics{
		price:            price,
		open:             open,
		low:              low,
		high:             high,
		centerOffsetPct:  math.Abs(price-center) / price * 100,
		upperWickRatio:   upperWick / (high - low),
		retraceFromHigh:  (high - price) / price * 100,
		localMomentumPct: 0,
	}
	lastClose, hasLastClose := parsePositive(item.LastClose)
	if hasLastClose && item.UpdateTime > item.LastUpdateTime {
		m.localMomentumPct = (price - lastClose) / lastClose * 100
		m.hasLocalMomentum = true
	}
	return m, true, ""
}

func buildCandidate(item *models.Symbols, metrics symbolMetrics, minQuoteVolume float64) PrefilterCandidate {
	score := 35.0
	reasons := make([]string, 0, 5)
	risks := make([]string, 0, 4)
	missing := []string{
		"15m/1h K线",
		"盘口点差",
		"Binance 风险公告",
	}

	if item.QuoteVolume >= minQuoteVolume {
		score += clamp(math.Log10(item.QuoteVolume/minQuoteVolume+1)*18, 0, 24)
		reasons = append(reasons, "24h 成交额满足流动性阈值")
	}
	if item.PercentChange >= -5 && item.PercentChange <= 35 {
		score += 20
		reasons = append(reasons, "24h 涨幅处于初筛理想区间")
	} else if item.PercentChange > 35 {
		score += 5
		risks = append(risks, "24h 涨幅偏高，需要等待回踩确认")
	} else {
		score += 4
		risks = append(risks, "24h 表现偏弱")
	}

	centerScore := 18 - metrics.centerOffsetPct
	score += clamp(centerScore, 0, 18)
	if metrics.centerOffsetPct <= 8 {
		reasons = append(reasons, "价格未明显远离 24h 中枢")
	} else {
		risks = append(risks, "价格距离 24h 中枢偏远")
	}

	wickScore := 14 - metrics.upperWickRatio*28
	score += clamp(wickScore, -8, 14)
	if metrics.upperWickRatio <= 0.25 && metrics.retraceFromHigh <= 8 {
		reasons = append(reasons, "未出现明显长上影或冲高回落")
	} else {
		risks = append(risks, "存在上影线或冲高回落风险")
	}

	if metrics.hasLocalMomentum {
		if metrics.localMomentumPct > 0 {
			score += clamp(metrics.localMomentumPct*3, 0, 8)
			reasons = append(reasons, "本地最近一次价格更新呈转强")
		} else {
			score -= clamp(math.Abs(metrics.localMomentumPct)*2, 0, 8)
			risks = append(risks, "本地最近一次价格更新偏弱")
		}
	} else {
		missing = append(missing, "本地 lastClose 动量")
	}

	notChase := item.PercentChange > 35 || metrics.upperWickRatio > 0.30 || metrics.retraceFromHigh > 12
	if notChase {
		risks = append(risks, "当前不适合追高，只适合作为观察候选")
	}

	score = clamp(score, 0, 100)
	return PrefilterCandidate{
		Symbol:             item.Symbol,
		Score:              round(score, 2),
		Grade:              grade(score),
		Price:              metrics.price,
		PercentChange24h:   item.PercentChange,
		QuoteVolume24h:     item.QuoteVolume,
		TradeCount24h:      item.TradeCount,
		High24h:            metrics.high,
		Low24h:             metrics.low,
		Open24h:            metrics.open,
		CenterOffsetPct:    round(metrics.centerOffsetPct, 4),
		UpperWickRatio:     round(metrics.upperWickRatio, 4),
		RetraceFromHighPct: round(metrics.retraceFromHigh, 4),
		LocalMomentumPct:   round(metrics.localMomentumPct, 4),
		NotChase:           notChase,
		Reasons:            reasons,
		Risks:              uniqueStrings(risks),
		Missing:            uniqueStrings(missing),
		LastUpdateTime:     item.UpdateTime,
	}
}

func parsePositive(raw string) (float64, bool) {
	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0, false
	}
	return v, true
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func round(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}

func grade(score float64) string {
	switch {
	case score >= 85:
		return "A+"
	case score >= 75:
		return "A"
	case score >= 65:
		return "B"
	case score >= 55:
		return "C"
	default:
		return "D"
	}
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		result = append(result, item)
	}
	return result
}
