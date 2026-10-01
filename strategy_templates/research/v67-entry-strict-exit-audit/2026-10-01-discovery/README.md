# v108 v67 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-evaluate the exact v67 entry logic under the project's true fixed exit semantics. The original v67 used conditional trend-failure CLOSE rules, which are not equivalent to forced TP8/SL6 in the current Engine. This is not retroactive archival of v67; it is a new strict-exit audit of the entry mechanism.

Entry logic is frozen exactly from v67: 4h EMA20/50 + ADX14/DMI direction, completed 1h TakerBuyRatio cross of 0.5, signal-hour QPS at/above prior 8h mean, and current-price break of the trigger-hour extreme.

Canonical exits are exactly ROI >= 8 || ROI <= -6.

Discovery universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Promotion gate: PF >=1.15, >=4/6 positive symbols, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH is unread unless discovery passes.

## Final result

Strict fixed TP8/SL6 audit failed decisively: 3830 trades, PF 0.854210, 0/6 symbols positive, 3.337067 trades/symbol/week. Yearly PFs were 0.842015 / 0.888021 / 0.822679 / 0.868086. The earlier v67 conditional-exit strength does not survive canonical fixed exits. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. Freeze without tuning the v67 entry.
