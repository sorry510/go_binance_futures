# v129 1h Three-Return Acceleration — 2026-10-01 Discovery

Hypothesis: three consecutive completed 1h close-to-close returns in the same direction, with each newer return larger in magnitude than the previous one, represent directional acceleration rather than a simple price streak.

LONG requires r1 > r2 > r3 > 0; SHORT requires r1 < r2 < r3 < 0, where r1 is the newest completed 1h return. Current price must also break the newest completed hour's high/low.

Project-native only. No thresholds, EMA/ADX/volume/funding/taker filters, MarketCondition, Benchmark, or NowTime modulo.

Exact exits: ROI >= 8 || ROI <= -6. Discovery only: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT during 2023-2024.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no clear 2023/2024 instability. Failure => no 2/4-return variant, no magnitude threshold, no added filters, OOS remains unread.

## Final result

Discovery failed: 2166 trades, PF 0.809991, 0/6 symbols positive, 3.456908 trades/symbol/week. 2023 PF 0.897722; 2024 PF 0.744514. OOS was not evaluated. Freeze without changing return count, adding magnitude thresholds, reversing direction, or adding filters.
