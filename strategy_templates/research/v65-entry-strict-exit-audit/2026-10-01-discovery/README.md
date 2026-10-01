# v65 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-test the original v65 1h doji/narrow-body breakout + 4h trend entry under the corrected fixed-exit semantics.

The original LONG/SHORT entry code is copied unchanged from temp_strategy/v65/01-doji-breakout-4h-trend.json. Only CLOSE_LONG and CLOSE_SHORT are replaced by exactly ROI >= 8 || ROI <= -6.

Discovery universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT. Discovery window: 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear 2023/2024 instability. Failure freezes the entry without changing its 20% body threshold, EMA/ADX/DMI trend definition, or breakout rule. 2025/2026 OOS remain unread unless discovery passes.

## Final result

Strict fixed-exit discovery failed decisively: 3557 trades, PF 0.860536, 0/6 symbols positive, 5.676927 trades/symbol/week. 2023 PF 0.792273; 2024 PF 0.920704. The original entry was unchanged; only CLOSE semantics were corrected. 2025/2026 OOS was not evaluated. Freeze v65 entry without threshold or trend-filter changes.
