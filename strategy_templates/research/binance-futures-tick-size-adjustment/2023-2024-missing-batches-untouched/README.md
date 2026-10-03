# Binance Futures Tick-Size Adjustment — Missing-Batches Untouched Exact Validation

The original 2026-09-27 archive accidentally hard-coded only 6 of the 15 official 2023-2024 Binance Futures tick-size notices. Three old eligible events (TRB/VET/SOL) had already been inspected and are therefore excluded from this stage.

Nine previously omitted official notices were frozen before any of their returns were read. Body parsing recovered 54 untouched USD-M tick-size events / 53 symbols / 9 articles. The original mapping was preserved unchanged: finer tick -> LONG, coarser tick -> SHORT.

Production eligibility (event-time USD-M history >=730 days plus prior-24h QuoteVolume >=5M) left **10 events / 10 symbols / 5 independent articles**:
OCEAN SHORT; OMG/SUSHI/EGLD/ENJ/GMT/LRC/KSM/MASK/SAND LONG.

Exact replay was then run with the project's audited event semantics:
- next 1m open after effective time;
- leverage 4;
- exact `ROI >= 8 || ROI <= -6`;
- minute-close trigger, next-minute-open exit;
- fee 0.0005/side;
- adverse slippage 5bps/side;
- funding included;
- maximum hold 72h.

## Final result

**10 / 10 trades hit SL.**
- TP: **0**
- SL: **10**
- wins/losses: **0 / 10**
- PF: **0.000**
- normalized net: **-69.7510%**
- average: **-6.9751%**
- positive independent articles: **0/5**
- 2023: 3/3 losses, net -20.8463%
- 2024: 7/7 losses, net -48.9047%
- LONG: 9/9 losses, net -62.2911%
- SHORT: 1/1 loss, net -7.4599%

A trade-level timing audit confirms every exit followed the frozen minute-close trigger -> next-minute-open rule; fee and funding signs are consistent.

Decision: **permanently freeze Binance Futures Tick-Size Adjustment family**. The old 3-event positive observation was sample noise and is decisively rejected by untouched validation. Do not consume 2025+ to rescue it, retune tick-change magnitude, delete a side, alter TP/SL, or merge the old seen events into validation metrics.
