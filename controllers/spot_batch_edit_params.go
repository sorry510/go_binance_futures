package controllers

type SpotBatchEditParams struct {
	Usdt               string `json:"usdt"`
	Profit             string `json:"profit"`
	Loss               string `json:"loss"`
	StrategyType       string `json:"strategyType"`
	StrategyTemplateId int64  `json:"strategyTemplateId"`
}
