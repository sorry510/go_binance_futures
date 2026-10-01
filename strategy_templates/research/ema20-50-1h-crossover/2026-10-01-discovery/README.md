# v123 1h EMA20/50 Crossover — 2026-10-01

Hypothesis: the standard hourly EMA20/EMA50 crossover is a simple cross-symbol trend-start baseline. Entry is delayed until current price breaks the completed crossover bar's extreme, reducing same-bar ambiguity.

No extra trend, volume, funding, taker-flow, volatility or time filter. Exact exits are ROI >= 8 || ROI <= -6.

Time protocol:
- Discovery: 2023-01-01 through 2025-01-01
- OOS1: 2025-01-01 through 2026-01-01, only if discovery passes
- OOS2: 2026-01-01 through 2026-09-01, only if OOS1 passes

Universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.
Discovery gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no severe 2023/2024 instability.

## Final result

Discovery failed: 1015 trades, PF 0.851023, 1/6 symbols positive, 1.619927 trades/symbol/week. 2023 PF 0.831853 and 2024 PF 0.867421. OOS1/OOS2 were not evaluated. Freeze without changing EMA periods or adding filters.
