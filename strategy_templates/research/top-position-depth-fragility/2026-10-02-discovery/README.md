# v149 Top-Position Depth Fragility — 2026-10-02 Discovery

Hypothesis: top-trader directional position skew should be interpreted relative to visible opposite-side order-book capacity. A market with long-heavy top positioning and weak bid-side capacity is structurally fragile to downside liquidation, with the mirror structure implying upside fragility.

Frozen score:
- Binance Vision USD-M metrics: sum_toptrader_long_short_ratio.
- Binance Vision USD-M bookDepth: cumulative -1% bid and +1% ask notional.
- Hourly sample = last valid completed observation in each UTC hour for each source.
- score = log(TopPositionRatio * AskDepth1% / BidDepth1%).
- zero up-cross -> SHORT; zero down-cross -> LONG.
- adjacent combined samples must be exactly one hour apart.
- next USD-M 1h open; signed 1h/4h/12h forward returns.
- no price, funding, OI-magnitude, taker, volatility, threshold, smoothing, or symbol-specific filter.

Stage A is 2023 only. Before returns, every symbol had to reach >=95% combined hourly feature coverage. Stage B 2024 is prohibited unless Stage A passes.

## Final result

Data gate passes for all six symbols: combined hourly coverage ranges from **97.41% to 97.68%**.

2023 Stage A:
- **10,153 events**.
- frequency **32.4525 events/symbol/week**.
- breadth **5/6 positive symbols**.
- 1h signed mean **+0.0031%**.
- 4h signed mean **+0.0025%**.
- 12h signed mean **+0.0146%**.
- LONG: 5,079 events, 12h **+0.1426%**.
- SHORT: 5,074 events, 12h **-0.1136%**.

The combined 12h edge is far below the preregistered +0.20% gate despite adequate breadth and frequency. The side split is post-result audit only and cannot justify deleting SHORT.

Decision: **freeze v149 at Stage A**. Do not change depth band, add magnitude thresholds/smoothing, delete one side, reverse the rule, or add OI/price/funding filters. 2024 and 2025+ remain unread; no strict Engine candidate and no DB write.
