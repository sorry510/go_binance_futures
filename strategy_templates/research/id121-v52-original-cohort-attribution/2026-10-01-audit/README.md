# ID121 vs v52 Original-Cohort Attribution — 2026-10-01

Purpose: test whether the already-frozen v52 2-3ATR precompression filter improves canonical ID121 on the original 15-symbol 2024+ production cohort.

This is descriptive validation, not fresh OOS. No parameter is changed.

Universe/start semantics are copied from the canonical ID121 exact-exit audit:
- Mature symbols start 2024-08-15.
- 1000PEPEUSDT starts 2025-05-05.
- SUIUSDT starts 2025-05-03.
- ONDOUSDT starts 2026-01-20.
- End 2026-09-01.
- Same 15 symbols as canonical ID121.

Both use strict ROI >= 8 || ROI <= -6, leverage 4, fee 0.0005/side, slippage 5bps/side, single position.

Interpretation:
- Compare aggregate PF, frequency, breadth, year stability, and exact-entry overlap.
- If v52 improves PF by deleting negative ID121-only long trades while retaining frequency >=0.30/symbol/week, it may be retained as a lower-frequency filtered variant.
- This cannot override the failed fresh-symbol holdout and does not prove universal cross-symbol generalization.

## Final attribution

On the original 15-symbol canonical cohort, ID121 produced 773 trades at PF 1.211221 and 0.532684 trades/symbol/week; v52 produced 509 trades at PF 1.212544 and 0.350758/week. Exact common entries were 500 trades at PF 1.217470. The 273 ID121-only trades filtered out by v52 were all LONG and had PF 1.199880 overall. By year, the removed cohort was negative in 2024 (45 trades, PF 0.844762), strongly positive in 2025 (143 trades, PF 1.481310), and neutral in 2026 (85 trades, PF 1.005553). Therefore the fixed 2-3ATR precompression filter is regime-dependent: it helps 2024 but discards substantial positive expectancy in 2025. v52 remains frozen; this audit does not alter the failed fresh-symbol generalization evidence.
