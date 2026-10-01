# v88 Record Net Aggressor Flow Continuation — 2026-09-30 Early Gate

Hypothesis: an unusually large one-hour net aggressive quote-flow imbalance can identify a genuine short-horizon displacement. Net aggressive quote flow is computed entirely from project-native Kline fields as `2*TakerBuyAmount - Amount`. A signal hour must have larger absolute net flow than each of the preceding eight completed hours. Direction is the sign of net flow; current price must break the signal-hour high/low before entry.

Frozen before returns. No sigma threshold, no QPS/EMA/ADX/Funding filter, and no parameter scan. Strict fixed exits are `ROI >= 8 || ROI <= -6` for both sides.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability.

## Final result

Strict fixed-exit early gate failed decisively: 4902 trades, normalized PF 0.827068, 0/4 symbols positive, every yearly PF below 1. Freeze without direction reversal or lookback/filter tuning.
