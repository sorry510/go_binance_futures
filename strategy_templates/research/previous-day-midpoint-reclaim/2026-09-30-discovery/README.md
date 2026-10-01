# v107 Previous-Day Midpoint Reclaim — 2026-09-30 Discovery

Hypothesis: the midpoint of the prior completed daily range is a simple equilibrium reference. A completed 1h candle reclaiming the midpoint from below/above, followed by a break of that signal hour's extreme, may indicate directional acceptance away from prior-day equilibrium.

The definition is fixed before returns: midpoint=(prior-day High+Low)/2. No trend, volume, taker-flow, funding, ATR or time filter. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Promotion requires PF>=1.15, >=4/6 symbols positive, frequency>=0.30 trades/symbol/week and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless promoted.

## Final result

Strict TP8/SL6 discovery: 7106 trades, PF 0.825813, 0/6 symbols positive, 6.191436 trades/symbol/week; yearly PF 0.803968 / 0.792657 / 0.903401 / 0.782233. Fresh holdout remained unread. The family is frozen without changing midpoint definition, range fractions, filters, or direction.
