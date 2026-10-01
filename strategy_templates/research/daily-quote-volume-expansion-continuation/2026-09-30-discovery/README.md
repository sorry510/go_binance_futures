# v111 Daily Quote-Volume Expansion Continuation — 2026-09-30 Discovery

Hypothesis: a completed daily quote-volume expansion above its prior 20-day mean can mark a participation regime shift. Direction is the completed daily candle sign. Entry requires a completed 4h candle in the same direction and a subsequent break of that 4h extreme.

All inputs are project-native. Amount is the strategy DSL field for QuoteAssetVolume. No EMA/ADX/Taker/Funding or time-modulo filter is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, with no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 2743 trades, PF 0.814165, 0/6 symbols positive, 2.389968 trades/symbol/week. Yearly PFs were 0.759099 / 0.929956 / 0.740517 / 0.789727. Reserved holdout symbols were not evaluated. The family is frozen without changing the 20-day volume baseline, adding filters, or reversing the signal.
