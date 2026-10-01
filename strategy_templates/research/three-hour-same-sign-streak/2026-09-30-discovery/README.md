# v108 Three-Hour Same-Sign Streak Continuation — 2026-09-30 Discovery

Hypothesis: three consecutive completed 1h candles with the same sign represent short-horizon directional persistence. Entry is delayed until current price breaks the third candle's extreme, avoiding entry on the streak close itself.

This is a project-native price-path test. No EMA/ADX/ATR/QPS/Taker/Funding filter is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Promotion gate: PF>=1.15, >=4/6 symbols positive, frequency>=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH stays unread unless discovery passes.

## Final result

Strict TP8/SL6 discovery: 13874 trades, PF 0.778936, 0/6 symbols positive, 12.088374 trades/symbol/week; yearly PF 0.798713 / 0.745244 / 0.810188 / 0.751123. Fresh holdout remained unread. The family is frozen without changing streak length, body threshold, filters, or direction.
