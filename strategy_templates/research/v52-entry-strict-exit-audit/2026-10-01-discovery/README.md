# v52 Entry Strict TP8/SL6 Audit — 2026-10-01 Discovery

Purpose: re-evaluate historical v52 under corrected Engine exit semantics. Entry and indicators are copied unchanged from temp_strategy/v52/01-pre-range-2-3atr-long.json. Only CLOSE_LONG/CLOSE_SHORT are replaced by ROI >= 8 || ROI <= -6.

Discovery: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT from 2023-01-01 through 2025-01-01. 2025 OOS1 and 2026 OOS2 remain unread unless discovery passes.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no clear year instability. No entry retuning is permitted.

## Final result

Strict-exit discovery retained positive expectancy but failed promotion: 151 trades, PF 1.297779, 4/6 positive, frequency 0.240994 trades/symbol/week. 2023 PF 0.978071 and 2024 PF 1.532837 show year instability. 2025/2026 OOS remained unread. Freeze without changing the 2–3 ATR definition or other entry parameters.

## Side attribution

On the same frozen discovery sample, LONG had 38 trades with PF 1.482913 and SHORT had 113 trades with PF 1.239595. Positive aggregate expectancy is therefore not solely a base-SHORT artifact. This is descriptive attribution only; neither side is removed and OOS remains locked because the preregistered frequency/year-stability gates failed.
