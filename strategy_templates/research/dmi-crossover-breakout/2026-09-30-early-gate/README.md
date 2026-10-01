# v88 1h DMI Crossover Breakout — 2026-09-30 Early Gate

Hypothesis: a completed 1h +DI/-DI crossover marks a directional state change. Entry is delayed until current price breaks the trigger-hour extreme in the crossover direction.

Frozen before returns. No ADX threshold, EMA trend filter, volume/QPS, funding, or parameter scan. CLOSE_LONG/CLOSE_SHORT are the exact fixed exits `ROI >= 8 || ROI <= -6`.

Core-4 early gate: BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT; 2023-01-01 through 2026-09-01. Fixed execution: leverage 4, TP8, SL6, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=3/4 symbols positive, frequency >=0.30 trades/symbol/week, and no obvious multi-year instability. Failure freezes the family without ADX-level tuning, EMA/volume filters, or direction reversal.

## Final result

Canonical strict TP8/SL6 early gate failed decisively: 2990 trades, normalized PF 0.773559, 0/4 symbols positive, frequency 3.907767 trades/symbol/week; yearly PFs 0.700582 / 0.807557 / 0.825641 / 0.727144. Freeze without ADX/EMA filters, period tuning, or reversal.
