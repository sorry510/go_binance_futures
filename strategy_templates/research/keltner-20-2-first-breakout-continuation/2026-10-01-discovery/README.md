# v115 Keltner 20/2 First-Breakout Continuation — 2026-10-01 Discovery

Hypothesis: the first completed 1h close outside a standard Keltner(20,2) channel marks a volatility-adjusted directional breakout. Entry waits for the current hour to continue beyond the trigger-hour extreme.

This is intentionally standalone and project-native. No extra EMA/ADX/Funding/QPS/Taker filter is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 4597 trades, PF 0.831587, 0/6 symbols positive, 4.005352 trades/symbol/week. Yearly PFs were 0.839302 / 0.810809 / 0.864484 / 0.809687. Fresh holdout was not evaluated. Freeze without multiplier, filter, or direction changes.
