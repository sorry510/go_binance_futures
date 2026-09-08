package app

import (
	"context"
	"sort"

	"go_binance_futures/agent/skillconfig"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	generalchat "go_binance_futures/agent/skills/generalchat"
	marketregime "go_binance_futures/agent/skills/marketregime"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
	symbolteam "go_binance_futures/agent/skills/symbolteam"
	workflowSkills "go_binance_futures/agent/skills/workflows"
)

type SkillImplementation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	ChatDefault int    `json:"chat_default"`
}

var skillCatalog = map[string]SkillImplementation{
	generalchat.Name:                             {Name: generalchat.Name, DisplayName: "通用对话", Description: "未选择业务 Skill 时使用的内部连续对话能力；无 Tool，使用当前 Conversation 历史上下文。", Type: "native", ChatDefault: 1},
	symbolanalysis.Name:                          {Name: symbolanalysis.Name, DisplayName: "单币分析", Description: "分析指定 USDT 永续合约并输出结构化 TradingPlan。", Type: "native", ChatDefault: 1},
	"symbol_analysis_team":                       {Name: "symbol_analysis_team", DisplayName: "多智能体单币分析", Description: "Technical + Flow 并行分析共享行情上下文，再由 Supervisor 汇总 Typed Result。", Type: "team", ChatDefault: 1},
	symbolteam.TechnicalSkillName:                {Name: symbolteam.TechnicalSkillName, DisplayName: "技术分析 Agent", Description: "Multi-Agent Team 内部角色：基于共享上下文分析趋势、结构、波动与关键价位。", Type: "native", ChatDefault: 0},
	symbolteam.FlowSkillName:                     {Name: symbolteam.FlowSkillName, DisplayName: "资金流分析 Agent", Description: "Multi-Agent Team 内部角色：基于共享上下文分析 Funding、OI、Taker、Depth 与强平。", Type: "native", ChatDefault: 0},
	symbolteam.NewsSkillName:                     {Name: symbolteam.NewsSkillName, DisplayName: "新闻事件分析 Agent", Description: "Multi-Agent Team 内部角色：仅消费统一 Market Intelligence，分析公告、Alpha、新闻与本地 Signal 的时效和影响。", Type: "native", ChatDefault: 0},
	symbolteam.SupervisorSkillName:               {Name: symbolteam.SupervisorSkillName, DisplayName: "Team Supervisor", Description: "Multi-Agent Team 内部角色：仅汇总 Typed Child Result，不增加 Tool 权限。", Type: "native", ChatDefault: 0},
	alertanalysis.Name:                           {Name: alertanalysis.Name, DisplayName: "事件报警分析", Description: "对 Signal Engine 预筛选异常进行 AI 二次确认。", Type: "native", ChatDefault: 0},
	marketregime.Name:                            {Name: marketregime.Name, DisplayName: "市场趋势分析", Description: "根据确定性市场快照识别 Market Regime。", Type: "native", ChatDefault: 0},
	strategybuilder.Name:                         {Name: strategybuilder.Name, DisplayName: "策略生成", Description: "多轮生成和修复策略模板 JSON。", Type: "native", ChatDefault: 0},
	workflowSkills.MarketScanName:                {Name: workflowSkills.MarketScanName, DisplayName: "市场机会扫描", Description: "确定性 Scanner 初筛后由 Agent 排序少量候选，输出 Opportunity Set。", Type: "native", ChatDefault: 1},
	workflowSkills.StrategyReviewName:            {Name: workflowSkills.StrategyReviewName, DisplayName: "策略复盘", Description: "结合模板、手续费后测试结果和 MarketCondition 输出修改建议，不修改正式模板。", Type: "native", ChatDefault: 1},
	workflowSkills.StrategyExperimentProposeName: {Name: workflowSkills.StrategyExperimentProposeName, DisplayName: "策略实验提议", Description: "V2-11 strategy_experiment 内部步骤：生成待验证候选。", Type: "native", ChatDefault: 0},
	workflowSkills.StrategyExperimentSummaryName: {Name: workflowSkills.StrategyExperimentSummaryName, DisplayName: "策略实验归纳", Description: "V2-11 strategy_experiment 内部步骤：归纳确定性测试结果。", Type: "native", ChatDefault: 0},
	workflowSkills.AlertTriageName:               {Name: workflowSkills.AlertTriageName, DisplayName: "报警事件归并", Description: "对确定性聚合的同期 Signal 进行 Incident 归并和通知建议。", Type: "native", ChatDefault: 0},
	workflowSkills.DailyMarketBriefName:          {Name: workflowSkills.DailyMarketBriefName, DisplayName: "每日市场摘要", Description: "聚合 MarketCondition、Scanner 和重要 Signal，输出固定 Schema 摘要。", Type: "native", ChatDefault: 1},
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
			Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Type: item.Type,
			Enabled: 1, ChatEnabled: item.ChatDefault,
		})
	}
	return (skillconfig.Store{}).EnsureDefaults(context.Background(), defaults)
}
