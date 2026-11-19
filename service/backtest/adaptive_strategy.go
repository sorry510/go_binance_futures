package backtest

import "math"

// ROICandidate detects possible ROI-gate crossings from replay OHLC.
// Its bounds must have the SAME price basis as grossROI(): currently replay
// last/trade price, not the live loop's preferred Binance mark-price WS.
// Never reuse these bounds to prune mark-price-based ROI without mark OHLC.
type ROICandidate struct {
	Required       bool    `json:"required"`
	MinROI         float64 `json:"min_roi"`
	MaxROI         float64 `json:"max_roi"`
	StopPossible   bool    `json:"stop_possible"`
	ProfitPossible bool    `json:"profit_possible"`
}

func DetectROICandidate(position *Position, bar Bar, config RunConfig) ROICandidate {
	if position == nil || bar.High <= 0 || bar.Low <= 0 || bar.High < bar.Low {
		return ROICandidate{}
	}
	lowROI := grossROI(position, bar.Low, config.Leverage)
	highROI := grossROI(position, bar.High, config.Leverage)
	result := ROICandidate{MinROI: math.Min(lowROI, highROI), MaxROI: math.Max(lowROI, highROI)}
	if config.StopLossPct > 0 && result.MinROI <= -config.StopLossPct {
		result.StopPossible = true
	}
	if config.TakeProfitPct > 0 && result.MaxROI >= config.TakeProfitPct {
		result.ProfitPossible = true
	}
	result.Required = result.StopPossible || result.ProfitPossible
	return result
}
