package strategy

import (
	"math"
	"sort"
	"strconv"
	"strings"
)

type TestResultStatsGroup struct {
	Key              string  `json:"key"`
	TemplateID       int64   `json:"template_id,omitempty"`
	TemplateName     string  `json:"template_name,omitempty"`
	StrategyName     string  `json:"strategy_name,omitempty"`
	StrategyType     string  `json:"strategy_type,omitempty"`
	VersionHash      string  `json:"version_hash,omitempty"`
	Total            int     `json:"total"`
	Open             int     `json:"open"`
	Closed           int     `json:"closed"`
	Wins             int     `json:"wins"`
	Losses           int     `json:"losses"`
	Breakeven        int     `json:"breakeven"`
	WinRate          float64 `json:"win_rate"`
	GrossProfit      float64 `json:"gross_profit"`
	NetProfit        float64 `json:"net_profit"`
	Fees             float64 `json:"fees"`
	AverageNetProfit float64 `json:"average_net_profit"`
	LongTrades       int     `json:"long_trades"`
	ShortTrades      int     `json:"short_trades"`
}

type TestResultReviewStats struct {
	Total            int                    `json:"total"`
	Open             int                    `json:"open"`
	Closed           int                    `json:"closed"`
	Wins             int                    `json:"wins"`
	Losses           int                    `json:"losses"`
	Breakeven        int                    `json:"breakeven"`
	WinRate          float64                `json:"win_rate"`
	GrossProfit      float64                `json:"gross_profit"`
	NetProfit        float64                `json:"net_profit"`
	Fees             float64                `json:"fees"`
	AverageNetProfit float64                `json:"average_net_profit"`
	LongTrades       int                    `json:"long_trades"`
	ShortTrades      int                    `json:"short_trades"`
	ByTemplate       []TestResultStatsGroup `json:"by_template"`
	ByOpenStrategy   []TestResultStatsGroup `json:"by_open_strategy"`
	ByCloseStrategy  []TestResultStatsGroup `json:"by_close_strategy"`
}

func CalculateTestResultReviewStats(rows []TestResult) TestResultReviewStats {
	stats := TestResultReviewStats{
		ByTemplate:      []TestResultStatsGroup{},
		ByOpenStrategy:  []TestResultStatsGroup{},
		ByCloseStrategy: []TestResultStatsGroup{},
	}
	templateGroups := map[string]*TestResultStatsGroup{}
	openGroups := map[string]*TestResultStatsGroup{}
	closeGroups := map[string]*TestResultStatsGroup{}

	for index := range rows {
		row := &rows[index]
		stats.Total++
		addSideCounts(&stats.LongTrades, &stats.ShortTrades, row.PositionSide)

		closed, metrics := realizedMetrics(*row)
		if closed {
			stats.Closed++
			applyRealizedSummary(&stats.Wins, &stats.Losses, &stats.Breakeven, &stats.GrossProfit, &stats.NetProfit, &stats.Fees, metrics)
		} else {
			stats.Open++
		}

		templateGroup := ensureTemplateStatsGroup(templateGroups, *row)
		applyGroupRow(templateGroup, *row, closed, metrics)

		openGroup := ensureRuleStatsGroup(openGroups, "open", *row)
		if openGroup != nil {
			applyGroupRow(openGroup, *row, closed, metrics)
		}
		if closed {
			closeGroup := ensureRuleStatsGroup(closeGroups, "close", *row)
			if closeGroup != nil {
				applyGroupRow(closeGroup, *row, true, metrics)
			}
		}
	}

	finalizeReviewSummary(&stats)
	stats.ByTemplate = finalizeStatsGroups(templateGroups)
	stats.ByOpenStrategy = finalizeStatsGroups(openGroups)
	stats.ByCloseStrategy = finalizeStatsGroups(closeGroups)
	return stats
}

func realizedMetrics(row TestResult) (bool, TestTradeProfit) {
	exitPrice, errExit := strconv.ParseFloat(strings.TrimSpace(row.ClosePrice), 64)
	entryPrice, errEntry := strconv.ParseFloat(strings.TrimSpace(row.Price), 64)
	amount, errAmount := strconv.ParseFloat(strings.TrimSpace(row.PositionAmt), 64)
	if errExit != nil || errEntry != nil || errAmount != nil || exitPrice <= 0 || entryPrice <= 0 || amount == 0 {
		return false, TestTradeProfit{}
	}
	return true, CalculateTestTradeProfit(entryPrice, exitPrice, amount, row.Leverage, row.OpenFeeRate, row.CloseFeeRate)
}

func ensureTemplateStatsGroup(groups map[string]*TestResultStatsGroup, row TestResult) *TestResultStatsGroup {
	templateID := row.StrategyTemplateID
	name := strings.TrimSpace(row.StrategyTemplateName)
	hash := strings.TrimSpace(row.StrategySnapshotHash)
	if hash == "" {
		hash = StrategySnapshotHash(row.Technology, row.Strategy)
	}
	key := "custom|" + hash
	if templateID > 0 {
		key = "template|" + strconv.FormatInt(templateID, 10) + "|" + hash
	} else if name != "" {
		key = "template-name|" + name + "|" + hash
	}
	group := groups[key]
	if group == nil {
		group = &TestResultStatsGroup{Key: key, TemplateID: templateID, TemplateName: name, VersionHash: hash}
		groups[key] = group
	}
	return group
}

func ensureRuleStatsGroup(groups map[string]*TestResultStatsGroup, prefix string, row TestResult) *TestResultStatsGroup {
	var name, ruleType, hash, code string
	if prefix == "open" {
		name, ruleType, hash, code = row.OpenStrategyName, row.OpenStrategyType, row.OpenStrategyHash, row.OpenStrategy
	} else {
		name, ruleType, hash, code = row.CloseStrategyName, row.CloseStrategyType, row.CloseStrategyHash, row.CloseStrategy
	}
	name = strings.TrimSpace(name)
	ruleType = strings.TrimSpace(ruleType)
	hash = strings.TrimSpace(hash)
	if hash == "" {
		hash = RuleHash(code)
	}
	if name == "" && ruleType == "" && hash == "" {
		return nil
	}
	key := prefix + "|" + name + "|" + ruleType + "|" + hash
	group := groups[key]
	if group == nil {
		group = &TestResultStatsGroup{Key: key, StrategyName: name, StrategyType: ruleType, VersionHash: hash}
		groups[key] = group
	}
	return group
}

func applyGroupRow(group *TestResultStatsGroup, row TestResult, closed bool, metrics TestTradeProfit) {
	group.Total++
	addSideCounts(&group.LongTrades, &group.ShortTrades, row.PositionSide)
	if !closed {
		group.Open++
		return
	}
	group.Closed++
	applyRealizedSummary(&group.Wins, &group.Losses, &group.Breakeven, &group.GrossProfit, &group.NetProfit, &group.Fees, metrics)
}

func applyRealizedSummary(wins, losses, breakeven *int, grossProfit, netProfit, fees *float64, metrics TestTradeProfit) {
	*grossProfit += metrics.GrossProfit
	*netProfit += metrics.NetProfit
	*fees += metrics.TotalFee
	const epsilon = 0.000000001
	if metrics.NetProfit > epsilon {
		*wins++
	} else if metrics.NetProfit < -epsilon {
		*losses++
	} else {
		*breakeven++
	}
}

func addSideCounts(longTrades, shortTrades *int, side string) {
	if strings.EqualFold(strings.TrimSpace(side), "LONG") {
		*longTrades++
	} else if strings.EqualFold(strings.TrimSpace(side), "SHORT") {
		*shortTrades++
	}
}

func finalizeReviewSummary(stats *TestResultReviewStats) {
	if stats.Closed > 0 {
		stats.WinRate = roundStats(float64(stats.Wins) / float64(stats.Closed) * 100)
		stats.AverageNetProfit = roundStats(stats.NetProfit / float64(stats.Closed))
	}
	stats.GrossProfit = roundStats(stats.GrossProfit)
	stats.NetProfit = roundStats(stats.NetProfit)
	stats.Fees = roundStats(stats.Fees)
}

func finalizeStatsGroups(groups map[string]*TestResultStatsGroup) []TestResultStatsGroup {
	result := make([]TestResultStatsGroup, 0, len(groups))
	for _, group := range groups {
		if group.Closed > 0 {
			group.WinRate = roundStats(float64(group.Wins) / float64(group.Closed) * 100)
			group.AverageNetProfit = roundStats(group.NetProfit / float64(group.Closed))
		}
		group.GrossProfit = roundStats(group.GrossProfit)
		group.NetProfit = roundStats(group.NetProfit)
		group.Fees = roundStats(group.Fees)
		result = append(result, *group)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Closed != result[j].Closed {
			return result[i].Closed > result[j].Closed
		}
		if result[i].Total != result[j].Total {
			return result[i].Total > result[j].Total
		}
		leftName := result[i].TemplateName + result[i].StrategyName
		rightName := result[j].TemplateName + result[j].StrategyName
		if leftName != rightName {
			return leftName < rightName
		}
		return result[i].VersionHash < result[j].VersionHash
	})
	return result
}

func roundStats(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Round(value*1_000_000) / 1_000_000
}
