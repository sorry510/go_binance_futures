# v119 Daily Engulfing + 4h Reversal Confirmation — 2026-10-01

Hypothesis: a completed daily bullish/bearish engulfing body can mark a regime reversal. The following day must produce a completed 4h candle in the engulfing direction, and current price must break that 4h extreme before entry.

Project-native only. No external data, MarketCondition, Benchmark, NowTime %, trend/volume/funding filter, or symbol-specific parameter. Exact exits are ROI >= 8 || ROI <= -6.

Time protocol is frozen before returns:
- Discovery: 2023-01-01 through 2025-01-01 (2023–2024 only)
- OOS1: 2025-01-01 through 2026-01-01, read only if discovery passes
- OOS2: 2026-01-01 through 2026-09-01, read only if OOS1 passes

Universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.

Discovery promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no severe year split instability. Failure freezes the family without changing engulfing definition or confirmation horizon.

## Final result

Discovery failed: 1716 trades, PF 0.809203, 0/6 symbols positive, 2.738714 trades/symbol/week. 2023 PF 0.759593 and 2024 PF 0.845043. OOS1 2025 and OOS2 2026 were not evaluated. Freeze without modifying engulfing or confirmation rules.
