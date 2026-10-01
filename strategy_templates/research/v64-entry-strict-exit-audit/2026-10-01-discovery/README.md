# v64 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-test the original v64 1h KDJ(9,3,3) cross + 4h EMA20/50 + ADX/DMI trend entry under corrected fixed-exit semantics.

The original LONG/SHORT entry code is copied unchanged. Only CLOSE_LONG and CLOSE_SHORT are replaced by exactly ROI >= 8 || ROI <= -6.

Discovery universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT. Discovery window: 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear 2023/2024 instability. Failure freezes the entry without changing KDJ periods, EMA/ADX/DMI trend definition, or adding filters. 2025/2026 OOS remain unread unless discovery passes.

## Final result

Strict fixed-exit discovery failed decisively: 4479 trades, PF 0.788039, 0/6 symbols positive, 7.148427 trades/symbol/week. 2023 PF 0.735167; 2024 PF 0.835213. The original entry was unchanged; only CLOSE semantics were corrected. 2025/2026 OOS was not evaluated. Freeze v64 entry without KDJ/trend-filter changes.
