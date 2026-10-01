# v85 Rolling-4h VWAP Reclaim — 2026-09-30 Early Gate

Hypothesis: a completed hourly close reclaiming a rolling four-hour executed VWAP can mark a directional value shift. VWAP is computed directly from project-native Kline QuoteVolume/Volume, not from external trades. Entry waits for the next hour to break the trigger-hour extreme.

Frozen before returns. Window is exactly four completed 1h bars; no deviation threshold, EMA, ADX, funding or volume filter. Core-4 early gate: BTC/ETH/BNB/XRP, 2023-01-01..2026-09-01. Fixed 4x/TP8/SL6/fee/slippage/single-position execution.

Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability. Failure freezes without trying 3h/6h/8h VWAP or added filters.

## Final result

**DSL BLOCKED BEFORE RETURNS.** `KLinePrice.Amount` is QuoteAssetVolume and `Qps` is quote-volume-per-second; base volume is not exposed. Therefore true rolling VWAP cannot be expressed in the current strategy DSL. No return evidence was inspected. No engine change or DB write was made.
