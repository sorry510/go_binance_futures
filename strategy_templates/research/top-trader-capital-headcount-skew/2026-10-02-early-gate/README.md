# v147 Top-Trader Capital-vs-Headcount Skew Zero-Cross

Signal uses Binance Vision USD-M 5m metrics. For every UTC hour, only the last completed metrics row is retained. Score is `log(sum_toptrader_long_short_ratio / count_toptrader_long_short_ratio)`. Up-cross of zero emits LONG; down-cross emits SHORT; entry is the next complete 1h open.

This is distinct from the 2026-09-23 positioning study: that work used positioning as a v33 filter and positioning-change alignment around 12h breakout events. v147 tests the positioning skew's own zero-cross without a price setup.

Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC over 2023-2024. No magnitude threshold, smoothing, trend, funding, OI, QPS, taker or symbol-specific filter. Adjacent hourly metrics points must be exactly one hour apart. 2025+ remains unread unless the frozen gate passes.

## Final result

- 1,561 events.
- Frequency: **2.4913 events/symbol/week**.
- 1h signed mean: **+0.0219%**.
- 4h signed mean: **-0.0406%**.
- 12h signed mean: **-0.0365%**.
- Breadth: **4/6 symbols positive**.
- 2023 12h: **-0.0129%**.
- 2024 12h: **-0.0471%**.
- LONG: 782 events, 12h **+0.0585%**.
- SHORT: 779 events, 12h **-0.1318%**.

The event rate is sufficient, but the preregistered economic gate fails and both discovery years are negative. The LONG/SHORT split is audit-only and does not permit deleting SHORT after seeing returns.

Decision: **freeze v147**. Do not scan thresholds/smoothing, switch to 5m entries, substitute top-position/global ratio, add a price setup, delete one direction, or reverse the signal. No strict Engine/OOS/DB import.
