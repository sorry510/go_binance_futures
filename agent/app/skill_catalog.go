package app

import (
	"context"
	"sort"

	"go_binance_futures/agent/skillconfig"
	alertanalysis "go_binance_futures/agent/skills/alertanalysis"
	marketregime "go_binance_futures/agent/skills/marketregime"
	strategybuilder "go_binance_futures/agent/skills/strategybuilder"
	symbolanalysis "go_binance_futures/agent/skills/symbolanalysis"
)

type SkillImplementation struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

var skillCatalog = map[string]SkillImplementation{
	symbolanalysis.Name:  {Name: symbolanalysis.Name, DisplayName: "单币分析", Description: "分析指定 USDT 永续合约并输出结构化 TradingPlan。"},
	alertanalysis.Name:   {Name: alertanalysis.Name, DisplayName: "事件报警分析", Description: "对 Signal Engine 预筛选异常进行 AI 二次确认。"},
	marketregime.Name:    {Name: marketregime.Name, DisplayName: "市场趋势分析", Description: "根据确定性市场快照识别 Market Regime。"},
	strategybuilder.Name: {Name: strategybuilder.Name, DisplayName: "策略生成", Description: "多轮生成和修复策略模板 JSON。"},
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
			Enabled: 1,
		})
	}
	return (skillconfig.Store{}).EnsureDefaults(context.Background(), defaults)
}
