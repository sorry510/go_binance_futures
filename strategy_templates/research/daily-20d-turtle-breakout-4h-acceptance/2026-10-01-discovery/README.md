# v122 Daily 20d Turtle Breakout + 4h Acceptance — 2026-10-01

Hypothesis: a fresh breakout beyond the prior 20 completed daily highs/lows is a classic medium-term trend signal. To avoid repeated entries while price remains outside the channel, a completed 4h bar must newly close across the fixed prior-20d boundary, and current price must then break that trigger 4h extreme.

Project-native only. No external data, MarketCondition, Benchmark, NowTime %, funding, volume, taker-flow, or symbol-specific parameter. Exact exits are ROI >= 8 || ROI <= -6.

Time protocol:
- Discovery: 2023-01-01 through 2025-01-01
- OOS1: 2025-01-01 through 2026-01-01, only if discovery passes
- OOS2: 2026-01-01 through 2026-09-01, only if OOS1 passes

Universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.

Discovery gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, with no severe 2023/2024 instability.

## Final result

Discovery failed: 657 trades, PF 0.843833, 1/6 symbols positive, 1.048564 trades/symbol/week. 2023 PF 0.908893 and 2024 PF 0.796182. OOS1/OOS2 were not evaluated. Freeze without changing 20d lookback or adding filters.
