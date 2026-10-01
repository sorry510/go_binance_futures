# v100 4h Quote-Volume-Weighted Return Pressure — 2026-09-30 Early Gate

Hypothesis: a zero-cross in the sum of hourly return × quote-volume over four completed hours marks a change in directional price pressure weighted by market participation. Entry waits for the next hour to break the latest completed 1h extreme.

The project DSL exposes quote volume as KLinePrice.Amount, so this definition is explicit and reproducible. It is not claimed to be standard Elder Force Index. Window is fixed at four completed 1h bars; no multiplier, EMA, ADX, taker-flow, funding, or extra filter is used. CLOSE rules are exactly `ROI >= 8 || ROI <= -6`.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Fixed execution: leverage 4, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, with no obvious multi-year instability. Failure freezes without changing the 4h window, adding filters, or reversing sign.

## Final result

Canonical strict TP8/SL6 early gate failed decisively: 6967 trades, normalized PF 0.823806, 0/4 symbols positive, 9.105489 trades/symbol/week; yearly PFs 0.804654 / 0.859325 / 0.833346 / 0.765014. Freeze without window tuning, filters, or reversal.
