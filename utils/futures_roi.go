package utils

import "math"

// FuturesLeveragedROI matches the gross ROI formula used by the live futures loop.
// Fees and funding are intentionally excluded because Binance UnrealizedProfit does
// not include them in the live close-trigger calculation.
func FuturesLeveragedROI(unrealizedProfit, positionQtyAbs, markPrice float64, leverage int64) float64 {
	if positionQtyAbs <= 0 || markPrice <= 0 || leverage <= 0 {
		return 0
	}
	return unrealizedProfit / (math.Abs(positionQtyAbs) * markPrice) * float64(leverage) * 100
}
