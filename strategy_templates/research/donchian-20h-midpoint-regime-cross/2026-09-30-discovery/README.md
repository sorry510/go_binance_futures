# v110 Donchian 20h Midpoint Regime Cross — 2026-09-30 Discovery

Hypothesis: the midpoint of the prior 20 completed hourly range can act as a rolling equilibrium level. A completed 1h candle crossing that midpoint may mark a short-horizon regime transition; entry waits for the current hour to break the trigger-hour extreme.

The midpoint is computed causally from High/Low[2:22], excluding the trigger hour. This study is fully project-native and adds no trend, volume, funding, taker-flow, or time-modulo filter. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed decisively: 10146 trades, PF 0.815047, 0/6 symbols positive, 8.840179 trades/symbol/week. Yearly PFs were 0.828710 / 0.810168 / 0.852102 / 0.738282. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without changing the 20-hour lookback, adding filters, or reversing the signal.

## Final result

Discovery failed: 10,146 trades, PF 0.815047, 0/6 symbols positive, 8.840179 trades/symbol/week. Yearly PFs were 0.828710, 0.810168, 0.852102, and 0.738282. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without lookback tuning, added filters, or reversal.

## Final result

Discovery failed decisively: 10146 trades, PF 0.815047, 0/6 symbols positive, 8.840179 trades/symbol/week. Yearly PFs were 0.828710, 0.810168, 0.852102, and 0.738282. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without lookback tuning, filters, or reversal.
