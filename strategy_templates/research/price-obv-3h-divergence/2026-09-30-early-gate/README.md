# v90 3h Price-OBV Divergence Reversal — 2026-09-30 Early Gate

Hypothesis: when price moves lower over the last three completed hourly observations while OBV moves higher, or vice versa, the price move is not confirmed by cumulative directional volume and may reverse. Entry waits for current price to break the most recent completed hourly extreme in the reversal direction.

Frozen before returns. Window is exactly the comparison between [1] and [3]. No EMA/ADX/RSI, no volume threshold, and no parameter scan. Both close rules use the exact fixed exits: `ROI >= 8 || ROI <= -6`.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Fixed execution: leverage 4, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no obvious multi-year instability. Failure freezes the family without changing divergence window, adding filters, or reversing sign.

## Final result

Canonical strict TP8/SL6 early gate failed decisively: 4224 trades, normalized PF 0.795049, 0/4 symbols positive, frequency 5.520538 trades/symbol/week; yearly PFs 0.801569 / 0.840378 / 0.776456 / 0.732058. Freeze without changing the 3h divergence window, adding filters, or reversing sign.
