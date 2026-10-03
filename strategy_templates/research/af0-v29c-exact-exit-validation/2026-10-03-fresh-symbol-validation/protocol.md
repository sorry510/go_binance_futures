# AF0 v29C Exact-Exit Fresh-Symbol Validation — Protocol

1. Hypothesis source is the noncanonical 2026-10-03 diagnostic on BTCUSDT/ETHUSDT/SOLUSDT/XRPUSDT. That diagnostic used 8x and dynamic exits, so its returns are mechanism-generation evidence only and are NOT formal performance.
2. Freeze AF0 entry logic exactly from `temp_strategy/20261003-adverse-flow/01-v29c-observed-activity-control.json`. No entry threshold or indicator may change.
3. Replace both dynamic CLOSE rules only with exact `ROI >= 8 || ROI <= -6`.
4. Formal RunConfig is leverage=4, fee=0.0005/side, slippage=5bps/side, StopLossPct=6, TakeProfitPct=8, single-position Engine semantics.
5. Fresh validation universe is fixed BEFORE reading AF0 exact-exit returns: DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT, ADAUSDT. None participated in the AF0 diagnostic.
6. Validation interval is 2023-01-01T00:00:00Z through 2026-09-01T00:00:00Z. All six symbols have established USD-M history before the start.
7. Repository source MUST be nil/read-only. No gap repair, REST import, DB write, app.conf edit, commit or push. Any data gap blocks that symbol rather than being repaired.
8. Promotion gate: aggregate normalized PF >=1.15; >=4/6 symbols net-positive; frequency >=0.30 trades/symbol/week; and no clear chronological collapse (a full calendar year with material sample PF <0.90 fails).
9. If the gate fails, freeze AF0/v29C relaxed-funding+relaxed-ADX entry. Do not retune funding 0.0001, restore/remove ADX acceleration selectively, alter no-chase/impulse, delete one side, or reuse BTC/ETH/SOL/XRP for rescue.
10. If it passes, AF0 becomes a candidate only; broader untouched validation is still required before DB import.
