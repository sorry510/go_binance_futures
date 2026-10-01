# ID121 / v54 Exact TP8-SL6 Audit — 2026-09-30

Purpose: verify whether the current best candidates remain valid when the user's fixed exit requirement is interpreted exactly by the actual Backtest Engine.

Engine audit established that RunConfig TakeProfitPct=8 and StopLossPct=6 are close-rule gates, not unconditional forced exits. The archived ID121/v54 strategy JSONs use conditional close expressions that can defer exits beyond +8/-6 ROI. Therefore the 2026-09-24 production comparison is not an exact fixed-exit test.

This audit changes **only** CLOSE_LONG/CLOSE_SHORT to:

ROI >= 8 || ROI <= -6

All entry rules, technology indicators, universe, production eligibility starts, fees, slippage, leverage, single-position semantics and data are unchanged.

Universe and starts are copied from the frozen 2026-09-24 production protocol:
- mature 12 symbols: 2024-08-15
- 1000PEPEUSDT: 2025-05-05
- SUIUSDT: 2025-05-03
- ONDOUSDT: 2026-01-20
- end: 2026-09-01

No DB write is performed. Original ID121/v54 snapshots and results are preserved unchanged.

## Final canonical result

Exact fixed-exit audit is complete. ID121: 773 trades, normalized PF 1.211221, 12/15 symbols positive, 0.532684 trades/symbol/week; yearly PF 1.241297 / 1.179101 / 1.245726 for 2024/2025/2026. V54: 830 trades, PF 1.162986, 11/15 positive, 0.571963/week; yearly PF 1.157365 / 1.128126 / 1.214292.

Paired attribution by symbol+entry_time+side: 770 common trades PF 1.214441; ID121-only 3 trades PF 0.568207; V54-only 60 trades PF 0.653374, with yearly PF 0.649524 / 0.542524 / 0.824697. Therefore v54's extra frequency comes from a clearly negative-expectancy marginal cohort and lowers aggregate PF. Under the project rule against buying frequency with negative expectancy, ID121 remains the formal candidate and v54 is demoted/frozen. Older PF≈1.39 candidate labels are superseded for exact TP8/SL6 evaluation.
