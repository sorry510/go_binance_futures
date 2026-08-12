package types

const (
	MarketConditionStrongBull        = 1
	MarketConditionBull              = 2
	MarketConditionSideways          = 3
	MarketConditionBear              = 4
	MarketConditionStrongBear        = 5
	MarketConditionBullishDivergence = 6
	MarketConditionBearishDivergence = 7
	MarketConditionBroadRise         = 8
	MarketConditionBroadDecline      = 9
	MarketConditionHighVolatility    = 10
	MarketConditionLowVolatility     = 11
)

var marketConditionNames = map[int]string{
	MarketConditionStrongBull:        "强多头",
	MarketConditionBull:              "偏多头",
	MarketConditionSideways:          "震荡",
	MarketConditionBear:              "偏空头",
	MarketConditionStrongBear:        "强空头",
	MarketConditionBullishDivergence: "多头分化",
	MarketConditionBearishDivergence: "空头分化",
	MarketConditionBroadRise:         "普涨",
	MarketConditionBroadDecline:      "普跌",
	MarketConditionHighVolatility:    "高波动震荡",
	MarketConditionLowVolatility:     "低波动盘整",
}

func IsValidMarketCondition(value int) bool {
	_, exists := marketConditionNames[value]
	return exists
}

func MarketConditionName(value int) string {
	return marketConditionNames[value]
}
