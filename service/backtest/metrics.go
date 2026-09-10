package backtest

import (
	"math"
	"sort"
	"time"
)

func calculateMetrics(trades []Trade, equity []EquityPoint, initialEquity float64, interval time.Duration) Metrics {
	metrics := Metrics{BySide: []GroupMetrics{}}
	if len(equity) > 0 {
		metrics.NetPnL = equity[len(equity)-1].Equity - initialEquity
		if initialEquity > 0 {
			metrics.ReturnPct = metrics.NetPnL / initialEquity * 100
		}
	}
	for _, point := range equity {
		if point.DrawdownPct > metrics.MaxDrawdownPct {
			metrics.MaxDrawdownPct = point.DrawdownPct
		}
	}
	metrics.TradeCount = len(trades)
	wins, lossesAbs, winsPnL, holding := 0, 0.0, 0.0, int64(0)
	bySide := map[string][]Trade{}
	for _, trade := range trades {
		metrics.Fees += trade.Fees
		metrics.Funding += trade.FundingPnL
		holding += trade.HoldingMs
		if trade.NetPnL > 0 {
			wins++
			winsPnL += trade.NetPnL
		} else if trade.NetPnL < 0 {
			lossesAbs += -trade.NetPnL
		}
		bySide[trade.Side] = append(bySide[trade.Side], trade)
	}
	if len(trades) > 0 {
		metrics.WinRate = float64(wins) / float64(len(trades))
		metrics.AverageHoldingMs = holding / int64(len(trades))
	}
	if lossesAbs > 0 {
		metrics.ProfitFactor = winsPnL / lossesAbs
	} else if winsPnL > 0 {
		metrics.ProfitFactor = winsPnL
	}
	metrics.Sharpe, metrics.Sortino = riskRatios(equity, interval)
	for key, items := range bySide {
		metrics.BySide = append(metrics.BySide, groupMetrics(key, items))
	}
	sort.Slice(metrics.BySide, func(i, j int) bool { return metrics.BySide[i].Key < metrics.BySide[j].Key })
	return metrics
}
func groupMetrics(key string, trades []Trade) GroupMetrics {
	g := GroupMetrics{Key: key, TradeCount: len(trades)}
	wins := 0
	winPnL, lossAbs := 0.0, 0.0
	holding := int64(0)
	for _, t := range trades {
		g.NetPnL += t.NetPnL
		g.Fees += t.Fees
		g.Funding += t.FundingPnL
		holding += t.HoldingMs
		if t.NetPnL > 0 {
			wins++
			winPnL += t.NetPnL
		} else if t.NetPnL < 0 {
			lossAbs += -t.NetPnL
		}
	}
	if len(trades) > 0 {
		g.WinRate = float64(wins) / float64(len(trades))
		g.AverageHoldingMs = holding / int64(len(trades))
	}
	if lossAbs > 0 {
		g.ProfitFactor = winPnL / lossAbs
	} else if winPnL > 0 {
		g.ProfitFactor = winPnL
	}
	return g
}
func riskRatios(points []EquityPoint, interval time.Duration) (float64, float64) {
	if len(points) < 2 {
		return 0, 0
	}
	returns := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		prev := points[i-1].Equity
		if prev > 0 {
			returns = append(returns, (points[i].Equity-prev)/prev)
		}
	}
	if len(returns) < 2 {
		return 0, 0
	}
	mean := averageFloat(returns)
	std := stdFloat(returns, mean)
	downs := make([]float64, 0)
	for _, r := range returns {
		if r < 0 {
			downs = append(downs, r)
		}
	}
	annual := math.Sqrt(intervalAnnualPeriods(interval))
	sharpe := 0.0
	if std > 0 {
		sharpe = mean / std * annual
	}
	sortino := 0.0
	if len(downs) > 0 {
		downStd := stdFloat(downs, 0)
		if downStd > 0 {
			sortino = mean / downStd * annual
		}
	}
	return sharpe, sortino
}
func averageFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range values {
		s += v
	}
	return s / float64(len(values))
}
func stdFloat(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range values {
		d := v - mean
		s += d * d
	}
	return math.Sqrt(s / float64(len(values)))
}
