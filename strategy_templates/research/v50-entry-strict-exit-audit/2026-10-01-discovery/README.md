# v50 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-evaluate the historical v50 entry under the corrected Engine exit semantics. Entry logic and all indicators are copied unchanged from temp_strategy/v50/01-controlled-breakout-long.json. Only CLOSE_LONG/CLOSE_SHORT are replaced by exact fixed exits ROI >= 8 || ROI <= -6.

Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT from 2023-01-01 through 2025-01-01. OOS 2025 and 2026 remain unread unless discovery passes.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no clear year instability. No entry retuning is permitted.

## Final result

Strict-exit discovery failed: 142 trades, PF 1.065596, 4/6 symbols positive, but only 0.226630 trades/symbol/week. 2023 PF was 0.784899 and 2024 PF was 1.263188. OOS remained unread. Freeze without retuning the v50 entry.
