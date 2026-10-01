# v127 1h TakerBuyRatio 8h Breakout — 2026-10-01 Discovery

Hypothesis: a completed hourly taker-buy ratio breaking beyond the prior eight completed hours marks a genuine order-flow regime breakout. LONG requires a new 8h high above 0.5; SHORT requires a new 8h low below 0.5. Entry also requires current price to break the trigger-hour high/low.

This is project-native and uses only existing 1h Kline TakerBuyRatio. No price-trend, funding, QPS, MarketCondition, Benchmark, or NowTime modulo filter is used.

Exact exits: ROI >= 8 || ROI <= -6. Discovery: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2025-01-01.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no clear 2023/2024 instability. Failure means no 4h/12h lookback scan, no 0.5 threshold change, no extra filters, and 2025/2026 OOS remain unread.

## Final result

Discovery failed decisively: 8460 trades, PF 0.819011, 0/6 symbols positive, 13.502052 trades/symbol/week. 2023 PF 0.817994; 2024 PF 0.819748. OOS was not evaluated. Freeze without lookback tuning, neutral-level tuning, or added filters.
