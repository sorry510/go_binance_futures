# v29 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-evaluate the previously important v29 entry logic under the user's exact fixed exit requirement. The source strategy is copied from the frozen project template. Only CLOSE_LONG/CLOSE_SHORT are changed to exact ROI >= 8 || ROI <= -6. Entry logic, indicators and parameters remain unchanged.

Discovery universe is SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT from 2023-01-01 through 2026-09-01. This is not treated as a fresh holdout; it is a canonical strict-exit audit.

Promotion gate: aggregate PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Failure freezes the v29 entry family. No parameter changes are allowed from the result.

No DB write, app.conf change, commit or push.

## Final result

Strict TP8/SL6 discovery: 242 trades, PF 0.989065, 4/6 symbols positive, but only 0.210854 trades/symbol/week. 2025 PF was 0.726235. The v29 entry family fails both the economic and minimum-frequency requirements and is frozen. Fresh holdout was not evaluated; no entry parameters are retuned.
