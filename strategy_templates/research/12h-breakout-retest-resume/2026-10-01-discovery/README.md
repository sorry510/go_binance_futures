# v116 12h Breakout Retest Resume — 2026-10-01 Discovery

Hypothesis: immediate breakouts often fail, but a fresh 1h close beyond the prior 12 completed-hour range followed by a one-hour retest that touches the breakout level and closes back outside it may represent genuine acceptance. Entry waits for current price to resume beyond the retest-hour extreme.

This is a standalone project-native setup and does not relax or modify ID121. No EMA/ADX/QPS/Taker/Funding/time filter is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed decisively: 2914 trades, PF 0.785588, 0/6 symbols positive, 2.538959 trades/symbol/week. Yearly PFs were 0.755243 / 0.779204 / 0.796861 / 0.812602. Fresh holdout was not evaluated. Freeze without lookback, retest-horizon, filter, or direction changes.
