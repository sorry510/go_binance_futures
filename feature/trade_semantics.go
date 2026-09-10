package feature

import (
	"strconv"

	"go_binance_futures/utils"
)

const disabledTradeROIThreshold = utils.DisabledFuturesROIThreshold

// resolveTradeROIThresholds keeps the custom-strategy Profit/Loss semantics
// identical between live and paper trading. A literal "0" keeps the live loop
// default of 1,000,000%; all other strings preserve the live ParseFloat behavior.
func resolveTradeROIThresholds(profitText, lossText string) (profit, loss float64) {
	profit, loss = disabledTradeROIThreshold, disabledTradeROIThreshold
	if profitText != "0" {
		profit, _ = strconv.ParseFloat(profitText, 64)
	}
	if lossText != "0" {
		loss, _ = strconv.ParseFloat(lossText, 64)
	}
	return profit, loss
}
