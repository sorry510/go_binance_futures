# Protocol

This is a pre-registered early gate before inspecting returns.

- Use only BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT.
- Use three fixed 3-day blocks in 2026: Jan 5-7, Apr 6-8, Jul 6-8.
- Metrics source is Binance Vision daily USD-M metrics.
- Implied contemporaneous mark proxy is sum_open_interest_value / sum_open_interest.
- 4h OI expansion transition: OI 4h log-change crosses from <=0 to >0. Trade in sign of 4h price return.
- 4h OI contraction transition: OI 4h log-change crosses from >=0 to <0. Trade opposite sign of 4h price return.
- Observe 1h/4h/12h signed forward returns only; this is not yet an Engine backtest.
- Causal as-of matching allows at most 30 minutes around required lookback/forward timestamps.
- Continue to full history / exact Engine only if overall 4h signed mean >= +0.10% and at least 3/4 symbols are positive.
- Do not search neighboring lookbacks, thresholds, sampling blocks, or reverse directions after seeing results.

Known limitation frozen in advance: historical Binance metrics publication latency is not independently verified; even a positive result requires a point-in-time availability audit before production use.

## Stage 2 frozen before reading 2025

Only oi_contraction_reversal passed the 2026 early gate. Its definition is frozen.
Validate on every available 2025 calendar day for BTCUSDT/ETHUSDT/BNBUSDT/XRPUSDT.
Use the identical 4h OI transition, opposite 4h price direction, horizons, and 30-minute as-of tolerance.
Require again overall 4h signed mean >= +0.10% and >=3/4 symbols positive before any 1m TP/SL replay.
No neighboring lookbacks, OI magnitude thresholds, price-return thresholds, or direction changes are permitted.
