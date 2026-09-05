package app

import (
	"context"
	"sort"

	"go_binance_futures/agent/skillconfig"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	marketregime "go_binance_futures/agent/skills/marketregime"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	workflowSkills "go_binance_futures/agent/skills/workflows"
)

type SkillImplementation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	ChatDefault int    `json:"chat_default"`
}

var skillCatalog = map[string]SkillImplementation{
	symbolanalysis.Name:                          {Name: symbolanalysis.Name, DisplayName: "单币分析", Description: "分析指定 USDT 永续合约并输出结构化 TradingPlan。", ChatDefault: 1},
	alertanalysis.Name:                           {Name: alertanalysis.Name, DisplayName: "事件报警分析", Description: "对 Signal Engine 预筛选异常进行 AI 二次确认。", ChatDefault: 0},
	marketregime.Name:                            {Name: marketregime.Name, DisplayName: "市场趋势分析", Description: "根据确定性市场快照识别 Market Regime。", ChatDefault: 0},
	strategybuilder.Name:                         {Name: strategybuilder.Name, DisplayName: "策略生成", Description: "多轮生成和修复策略模板 JSON。", ChatDefault: 0},
	workflowSkills.MarketScanName:                {Name: workflowSkills.MarketScanName, DisplayName: "市场机会扫描", Description: "确定性 Scanner 初筛后由 Agent 排序少量候选，输出 Opportunity Set。", ChatDefault: 1},
	workflowSkills.StrategyReviewName:            {Name: workflowSkills.StrategyReviewName, DisplayName: "策略复盘", Description: "结合模板、手续费后测试结果和 MarketCondition 输出修改建议，不修改正式模板。", ChatDefault: 1},
	workflowSkills.StrategyExperimentProposeName: {Name: workflowSkills.StrategyExperimentProposeName, DisplayName: "策略实验提议", Description: "V2-11 strategy_experiment 内部步骤：生成待验证候选。", ChatDefault: 0},
	workflowSkills.StrategyExperimentSummaryName: {Name: workflowSkills.StrategyExperimentSummaryName, DisplayName: "策略实验归纳", Description: "V2-11 strategy_experiment 内部步骤：归纳确定性测试结果。", ChatDefault: 0},
	workflowSkills.AlertTriageName:               {Name: workflowSkills.AlertTriageName, DisplayName: "报警事件归并", Description: "对确定性聚合的同期 Signal 进行 Incident 归并和通知建议。", ChatDefault: 0},
	workflowSkills.DailyMarketBriefName:          {Name: workflowSkills.DailyMarketBriefName, DisplayName: "每日市场摘要", Description: "聚合 MarketCondition、Scanner 和重要 Signal，输出固定 Schema 摘要。", ChatDefault: 1},
}

func AvailableSkillImplementations() []SkillImplementation {
	items := make([]SkillImplementation, 0, len(skillCatalog))
	for _, item := range skillCatalog {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func SkillImplementationByName(name string) (SkillImplementation, bool) {
	item, ok := skillCatalog[name]
	return item, ok
}
func EnsureDefaultSkillConfigs() error {
	defaults := make([]skillconfig.CreateInput, 0, len(skillCatalog))
	for _, item := range AvailableSkillImplementations() {
		defaults = append(defaults, skillconfig.CreateInput{
			Name: item.Name, DisplayName: item.DisplayName, Description: item.Description,
			Enabled: 1, ChatEnabled: item.ChatDefault,
		})
	}
	return (skillconfig.Store{}).EnsureDefaults(context.Background(), defaults)
}
