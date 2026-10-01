# v113 v63 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-evaluate the exact v63c engulfing + 4h trend entry under the project's canonical fixed exits. The original v63c used conditional trend-failure CLOSE rules and therefore did not represent strict fixed TP8/SL6.

Entry logic is copied unchanged from temp_strategy/v63/03-engulfing-trend-native-trigger.json. Only CLOSE_LONG/CLOSE_SHORT are replaced by ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Fresh holdout ALGOUSDT, INJUSDT, LDOUSDT, PENDLEUSDT, PYTHUSDT remains unread unless discovery passes.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. No engulfing, EMA, ADX, confirmation, or direction tuning is allowed after reading discovery.

## Final result

Strict fixed TP8/SL6 audit failed decisively: 4864 trades, PF 0.824296, 0/6 symbols positive, 4.237989 trades/symbol/week. Yearly PFs were 0.789602 / 0.823139 / 0.853645 / 0.827873. The earlier v63 conditional-exit strength does not survive canonical fixed exits. Fresh holdout was not evaluated. Freeze without entry tuning.
