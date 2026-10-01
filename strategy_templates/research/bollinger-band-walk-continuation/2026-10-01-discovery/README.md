# v114 Bollinger Band-Walk Continuation — 2026-10-01 Discovery

Hypothesis: two consecutive completed 1h closes outside the same Bollinger(20,2) band identify unusually persistent directional displacement. If the current hour then breaks the second outside-bar extreme, continuation may be strong enough to reach the project's fixed TP8 before SL6.

The rule is intentionally minimal and project-native. No EMA/ADX/QPS/Taker/Funding/time filter is used. No Bollinger parameter scan is allowed.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Exact exits are ROI >= 8 || ROI <= -6.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 4366 trades, PF 0.875411, 0/6 symbols positive, 3.804083 trades/symbol/week. Yearly PFs were 0.886550 / 0.825510 / 0.871729 / 0.960308. Fresh holdout was not evaluated. Freeze the standard two-close Bollinger band-walk definition without parameter or direction changes.
