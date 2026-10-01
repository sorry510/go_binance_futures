# v118 Daily 3-Bar Streak Reversal — 2026-10-01 Discovery

Hypothesis: three consecutive completed daily candles in the same direction can create short-term exhaustion. During the following day, a completed 4h candle in the opposite direction acts as reversal confirmation; entry requires current price to break that 4h candle's extreme.

The setup is fully project-native. No MarketCondition, Benchmark, NowTime %, external data, RSI/ADX/volume/funding filter, or symbol-specific parameter is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed decisively: 3629 trades, PF 0.792235, 0/6 symbols positive, 3.161937 trades/symbol/week; yearly PF 0.731850 / 0.778893 / 0.830850 / 0.825879. Fresh holdout was not evaluated. Freeze without changing streak length, filters, or reversing into continuation.

## Final result

Discovery failed decisively: 3629 trades, PF 0.792235, 0/6 symbols positive, 3.161937 trades/symbol/week. Yearly PFs were 0.731850, 0.778893, 0.830850, and 0.825879. Fresh holdout was not evaluated. Freeze without changing streak length, adding filters, or reversing the family into continuation.
