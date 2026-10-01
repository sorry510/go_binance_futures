# v90 Rolling-24h Return Zero-Cross — 2026-09-30 Early Gate

Hypothesis: the project-native rolling 24h ticker return changing sign can mark a short-horizon momentum regime transition. Backtest reconstructs `NowSymbolPercentChange` causally at each historical minute from the prior 24h execution bars, so no current/future ticker leakage is used.

LONG requires current rolling-24h percent change >0 while the latest completed-hour approximation of the prior 24h return was <=0; current price must then exceed the latest completed 1h high. SHORT is symmetric.

Frozen before returns. No nonzero percent threshold, EMA/ADX/volume/funding filter, or time modulo. Strict exits: `ROI >= 8 || ROI <= -6`.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability.

## Final result

Strict fixed-exit early gate failed decisively: 4295 trades, normalized PF 0.822372, 0/4 symbols positive, every yearly PF below 1. Freeze without nonzero-threshold or alternate-window tuning.
