# v89 1h EMA Spread Re-acceleration — 2026-09-30 Early Gate

Hypothesis: when EMA20 is already on the trend side of EMA50, but the EMA20-EMA50 spread's first difference turns back in the trend direction after a non-expanding hour, trend acceleration may be restarting. Entry waits for price to break the completed trigger-hour extreme.

Frozen before returns. No RSI, ADX, volume/QPS, funding, or threshold multiplier is used. EMA periods are fixed at 20/50. Both close rules are exact fixed exits: `ROI >= 8 || ROI <= -6`.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Fixed execution: leverage 4, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.30 trades/symbol/week, and no obvious multi-year instability. Failure freezes without changing EMA periods, adding filters, or reversing direction.

## Final result

Canonical strict TP8/SL6 early gate failed decisively: 3376 trades, normalized PF 0.870173, 0/4 symbols positive, frequency 4.412248 trades/symbol/week; yearly PFs 0.757349 / 0.923771 / 0.950142 / 0.807918. Freeze without EMA-period tuning, extra filters, or direction reversal.
