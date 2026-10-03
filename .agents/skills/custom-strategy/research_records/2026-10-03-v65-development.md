# v65 two-candle range-reclaim ablation

Frozen before replay. This is research, not a tradable release.

- v64a2 and v64b2 increased development frequency, but BTC/ETH/XRP still failed >=0.9/week and every candidate retained losing annual windows. Adding frequent weak signals reduced the base's four-year net returns. Do not hide those failures or promote the most profitable individual coin.
- One logical dimension: range-entry confirmation timing. Keep the same weak 4h ADX regime, price/ATR limits, candle quality, volume, taker direction, RSI, no-chase window, funding checks, entry-bound signal-confirmed closes and risk/cost settings.
- v65a: union of the v64 single-candle sweep/reclaim and a two-candle failed break. In the latter, candle `[2]` closes beyond the channel that existed at `[3]`, and `[1]` closes back inside. Compare against v64a2 to measure the additional pattern.
- v65b: only the delayed two-candle pattern plus the unchanged v29C trend entries. Compare against v65a to isolate the original single-candle contribution through full matching, not ledger filtering.
- No new indicator implementation, market-wide trend variable, symbol-specific parameter or threshold search.
- v65a: `temp_strategy/v65/01-trend-plus-range-reclaim-union.json`, portable SHA-256 `4e3e6c4f11af9594fe1857de153baddb72efb173c934337a41b609e5cb4b17d7`.
- v65b: `temp_strategy/v65/02-trend-plus-delayed-reclaim.json`, portable SHA-256 `f6d2d4e9358d181ac2e2dc2c46f4098b33193c6a3ae271c6f4895b93476e9a33`.
- Development: BTCUSDT, ETHUSDT, SOLUSDT, XRPUSDT, 2022-09-01..2026-08-31 UTC, actual standard 1m engine, correction version `20261003-v2` and the same immutable datasets as v64.
- Initial equity 1,000 USDT per coin, margin 10% of current available cash, 8x leverage, +/-5% outer gates, 0.05% fee and 5bps slippage per side, historical funding.
- Only if development satisfies every declared gate, validate the frozen final candidate on AAVEUSDT, ATOMUSDT, ETCUSDT, LINKUSDT. These symbols have not supplied returns in this study; require full-date eligibility before replay. The previous eight v62 validation symbols are now inspected and must not be relabeled untouched.
- Release requirements remain >=0.9/week per coin, positive real-cost net, no losing full-year window, cross-coin proof and recorded fill-minute liquidity. Avoid retuning against the predeclared fresh validation universe.

Normal range exits still require mean-price attainment or confirmed momentum deterioration for profit, and structure/directional failure for losses; the catastrophic exception remains -12% ROI. Trend entries retain v29C closes with a -20% emergency exception. The v64 close-decision matrix remains applicable; opening-expression hash bindings are regenerated for both new entry patterns.

## Completed comparison

All eight four-year development runs completed, and 72 deterministic Expr checks passed. v65a frequencies BTC/ETH/SOL/XRP = 1.006/1.049/1.212/0.886 per week; net = +401.985/+320.924/-40.325/+251.597 USDT. v65b frequencies = 0.723/0.723/0.930/0.647; net = +596.886/+585.608/-14.882/+473.362 USDT. Both retain losing annual windows. Do not relax the frequency gate or inspect the predeclared fresh universe to select an apparent winner.

Minute-liquidity audit: v65a 867 trades, v65b 631 trades; each has the same zero-activity BTC base fill reported for v64. Preserve the observed ledger and label execution uncertainty. No release, database insertion, assignment or activation took place. All per-coin/year evidence and source hashes are linked in `2026-10-03-arm-v62-v65-summary.md`.
