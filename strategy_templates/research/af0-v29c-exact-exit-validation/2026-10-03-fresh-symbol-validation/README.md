# AF0 v29C Exact-Exit Fresh-Symbol Validation

Generated hypothesis source:
- A noncanonical diagnostic on BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT using 8x leverage, 5/5 RunConfig gates and dynamic strategy exits.
- AF0 was the only diagnostic branch that was positive on all four source symbols.
- Those diagnostic returns were never eligible for promotion; they served only to generate this frozen exact-exit hypothesis.

Frozen AF0 entry differences versus old v29:
- LONG funding cap relaxed from <=0 to <=0.0001.
- The explicit 4h ADX acceleration requirement `ADX[1]-ADX[3] >= 2` is removed; monotonic `ADX[1] >= ADX[3]` remains.
- `kline_1h.Amount[0] > 0` is retained as a current-hour data/activity validity guard.
- All other AF0 entry logic is preserved exactly from the source diagnostic snapshot.

Formal validation:
- Fresh symbols, fixed before exact-exit returns: DOGE/LTC/AVAX/UNI/ZEC/ADA.
- None were in the AF0 source diagnostic.
- 2023-01-01 through 2026-09-01.
- leverage=4, fee=0.0005/side, slippage=5bps/side.
- exact CLOSE_LONG/CLOSE_SHORT: `ROI >= 8 || ROI <= -6`.
- Repository source=nil: existing historical data only; no REST gap repair/import and no DB write.

## Final result

- **915 trades**: LONG 580 / SHORT 335.
- Aggregate normalized PF **0.891839**.
- Normalized net **-0.993108**.
- Positive symbols **1/6**.
- Frequency **0.797237 trades/symbol/week**.
- 517 stop-loss exits / 398 take-profit exits.

By symbol:
- DOGE: 150 trades, PF **0.862197**.
- LTC: 128, PF **0.791630**.
- AVAX: 159, PF **0.919431**.
- UNI: 137, PF **0.913158**.
- ZEC: 190, PF **0.831099**.
- ADA: 151, PF **1.056730**.

By year:
- 2023: 168 trades, PF **0.716677**.
- 2024: 182, PF **1.115203**.
- 2025: 330, PF **0.862425**.
- 2026 Jan-Aug: 235, PF **0.916551**.

By side:
- LONG: 580 trades, PF **0.853064**.
- SHORT: 335 trades, PF **0.962215**.

Decision: **fresh validation fails decisively; freeze AF0/v29C relaxed-funding + relaxed-ADX entry family**. The earlier BTC/ETH/SOL/XRP diagnostic strength was cohort/exit-semantics dependent and did not survive the user’s canonical 4x TP8/SL6 semantics on unseen AF0 symbols.

Do not retune the 0.0001 funding cap, partially restore the ADX acceleration threshold, delete a side, modify no-chase/impulse, or reuse the four diagnostic symbols as rescue evidence. No DB import.
