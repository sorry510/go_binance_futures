# v89 1h Full-ATR Displacement Continuation — 2026-09-30 Early Gate

Hypothesis: a completed one-hour candle whose directional body is at least one full pre-signal ATR14 represents a genuine displacement rather than ordinary drift. If the following hour continues beyond the signal candle extreme, the move may have enough path expansion to match fixed TP8/SL6.

The displacement denominator is `ATR[2]`, i.e. ATR known before the signal candle, so the large signal candle cannot inflate its own threshold. No trend, volume, taker-flow, funding, or time filter is used.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Exact fixed exits: `ROI >= 8 || ROI <= -6`. Promotion gate: normalized PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year instability.

## Final result

Strict fixed-exit early gate failed decisively: 5570 trades, normalized PF 0.824723, 0/4 symbols positive, every yearly PF below 1. Freeze without reversal, ATR-threshold tuning, or extra filters.
