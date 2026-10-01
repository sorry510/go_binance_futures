# v128 1h Body Compression -> Expansion Release — 2026-10-01 Discovery

Hypothesis: two consecutive small real bodies after an eight-hour normal-body baseline represent directional compression. A subsequent completed 1h candle whose body expands back above that baseline may mark release; entry waits for current price to break the release candle's extreme.

The baseline is the simple mean of absolute Close-Open body size over completed bars [4:12]. Bars [3] and [2] must both be smaller than that mean. Bar [1] must be larger than the mean and its sign defines LONG/SHORT.

This is project-native and uses only 1h OHLC. No ATR/range, volume, funding, taker flow, MarketCondition, Benchmark, or NowTime modulo filter.

Exact exits: ROI >= 8 || ROI <= -6. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, no clear 2023/2024 instability. Failure => no body multiplier, no 1/3-bar compression variant, no trend/volume filter, and OOS remains unread.
