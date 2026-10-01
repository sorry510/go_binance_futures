# v88 Daily Open Reclaim — 2026-09-30 Early Gate

Hypothesis: the current UTC daily open acts as a project-native intraday reference price. A completed 1h candle crossing from below to above the current daily open may signal upside acceptance; the symmetric cross may signal downside acceptance. Entry waits for current price to break the trigger-hour extreme.

Frozen before returns. No trend, volume, taker-flow, funding, or volatility filter. Exact exits are ROI >= 8 || ROI <= -6.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Fixed leverage 4, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: normalized PF >= 1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Failure freezes the family without changing the reference price or adding filters.

## Final result

Canonical strict TP8/SL6 result: 5700 trades, PF 0.777092, 0/4 symbols positive, 7.449589 trades/symbol/week; yearly PF 0.707557 / 0.798062 / 0.831435 / 0.729939. The daily-open reclaim family is frozen without reversal or added filters.
