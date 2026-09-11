package backtest

import "math"

// ROICandidate describes whether the 1m price range proves that the current
// live-equivalent ROI gate may have been crossed before the minute closed.
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
