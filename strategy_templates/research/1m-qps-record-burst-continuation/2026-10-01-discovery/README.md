# v133 1m QPS Record-Burst Continuation — 2026-10-01 Discovery

Hypothesis: a one-minute quote-volume-rate record over the previous hour is an information/liquidity event; if the same minute has directional price displacement, continuation may be strong enough for the fixed TP8/SL6 path.

Mechanism:
- Current completed 1m QPS must be strictly greater than every one of the previous 60 completed 1m QPS values.
- LONG follows a bullish burst minute; SHORT follows a bearish burst minute.
- No partial-hour accumulation, previous-hour breakout, trend, taker, funding, volatility, or symbol-specific filter.
- EMA(64) exists only to expose sufficient 1m history to the strategy VM.

Exact exits: `ROI >= 8 || ROI <= -6`.
Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT, 2023-01-01 through 2025-01-01.

## Final result

Discovery failed decisively: 17,440 trades, normalized PF 0.812907, 0/6 symbols positive, 27.834017 trades/symbol/week. 2023 PF 0.807354; 2024 PF 0.816619. LONG PF 0.810856 and SHORT PF 0.814634. Every symbol PF is between 0.771215 and 0.846128.

This is a clean high-frequency negative result: quote-notional burst intensity itself does not create continuation compatible with fixed TP8/SL6 after costs. Freeze the family. Do not scan 30m/120m record windows, QPS multiples, reverse the direction, or add trend/taker filters. 2025 OOS1 and 2026 OOS2 were not evaluated.
