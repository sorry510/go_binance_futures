# v139 Directional Taker-Flow Energy Asymmetry — 2026-10-01 Early Gate

Mechanism:
- Hourly signed taker flow = `2*TakerBuyQuoteVolume/QuoteVolume - 1`.
- Over trailing 24 completed 1h bars, square and sum positive flow separately from negative flow.
- Score = `log(buy_energy/sell_energy)`.
- Zero up-cross emits LONG; zero down-cross emits SHORT; enter next complete 1h open.

This differs from signed-flow mean/persistence and lagged-flow lead tests because it measures **directional second-moment energy/concentration**, allowing a few large aggressive-flow hours to dominate without adding price, QPS, OI, funding, or trend filters.

Frozen discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024.
Gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, both years positive.

## Result

4,742 events; 12h signed mean **-0.0024%**; 2/6 symbols positive; 7.568 events/symbol/week.
2023 12h **-0.0076%**; 2024 **+0.0029%**.
LONG 12h +0.0531%, SHORT -0.0578%, but this post-hoc side split is audit-only and cannot justify deleting SHORT.

Decision: **early gate failed / freeze**. Do not scan lookback, exponent, thresholds, direction, or add filters. Strict TP8/SL6 Engine and 2025+ OOS were not run/read.
