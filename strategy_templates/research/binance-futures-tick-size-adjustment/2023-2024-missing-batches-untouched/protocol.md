# Missing-Batches Untouched Validation Protocol

1. The original 2026-09-27 Futures Tick-Size audit hard-coded 6 of the 15 official 2023-2024 Binance Futures tick-size announcements. Returns from its three eligible events (TRB/VET/SOL) are considered seen and are excluded from this untouched stage.
2. This stage uses only the nine official 2023-2024 announcements absent from that original script. Their article codes were fixed from the archived official corpus before any missing-batch return is read.
3. Parse only USDⓈ-M perpetual rows. COIN-M rows are excluded.
4. Preserve the original preregistered mapping: finer tick (new < old) -> LONG; coarser tick (new > old) -> SHORT. No side deletion.
5. Effective time is the explicit official adjustment time in each article. Production eligibility requires same-name USD-M history >=730 days at effective time and prior 24 complete 1h QuoteVolume >=5M USDT.
6. Feasibility gate before returns: >=8 eligible token-events across >=3 independent missing official articles. Failure freezes without reading any missing-batch return.
7. If feasibility passes, exact validation parameters are frozen before returns: leverage=4, TP=8, SL=6, fee=0.0005/side, slippage=5bps/side, funding included, max hold=72h, entry at next 1m open after effective time.
8. Exact validation requires normalized PF >=1.15, aggregate normalized net >0, >=60% independent article batches positive, and both LONG/SHORT are retained if present. Failure freezes the family.
9. Do not merge the three seen old events into untouched metrics, change eligibility, tune tick-change magnitude, select symbols/articles, or consume 2025+ to rescue failure.
