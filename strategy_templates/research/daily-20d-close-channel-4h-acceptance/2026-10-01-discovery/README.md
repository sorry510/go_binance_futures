# v126 20d Close-Channel Breakout + 4h Acceptance — 2026-10-01 Discovery

Hypothesis: a breakout through the highest/lowest closing price of the prior 20 completed daily bars may be a cleaner regime transition than a raw high/low Turtle breakout. A completed 4h candle must cross from inside the close-channel level to a close outside it; current price must then break that 4h candle's extreme.

Project-native only. No external data, MarketCondition, Benchmark, NowTime %, trend filter, volume filter, funding filter, or symbol-specific parameter.

Exact exits: ROI >= 8 || ROI <= -6. Discovery universe is frozen: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear 2023/2024 instability. Failure means no 10d/40d scan, no switch back to high/low levels, no extra filters, and 2025/2026 OOS stay unread.

## Final result

Discovery failed: 1469 trades, PF 0.894135, 2/6 symbols positive, 2.344505 trades/symbol/week. 2023 PF 0.903069; 2024 PF 0.886376. 2025 OOS1 and 2026 OOS2 were not evaluated. Freeze without lookback tuning, high/low substitution, reversal, or added filters.
