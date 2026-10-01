# v125 Range Expansion + Taker Alignment — 2026-10-01 Discovery

Hypothesis: a completed 1h bar whose range expands above the prior 8-hour average and whose aggressive taker flow agrees with the bar direction may represent a genuine directional impulse rather than a low-quality breakout. Entry waits for current price to break the trigger-hour extreme.

LONG requires: trigger-hour range > prior-8h average range, bullish trigger candle, TakerBuyRatio > 0.5, then break of trigger high. SHORT is symmetric.

This is fully project-native. No MarketCondition, Benchmark, NowTime %, external exchange, symbol-specific threshold, QPS filter, funding filter, or parameter scan is used. Exact exits are ROI >= 8 || ROI <= -6.

Discovery: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT from 2023-01-01 through 2024-12-31. 2025 is OOS1; 2026 through 2026-09-01 is OOS2. OOS is not read unless the prior stage passes.

Promotion gate: PF >= 1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear 2023/2024 conflict.

## Final result

Discovery failed: 11745 trades, PF 0.802924, 0/6 symbols positive. 2023 PF 0.823800 and 2024 PF 0.788860. OOS1/OOS2 were not read. Freeze this family without threshold or filter changes.
