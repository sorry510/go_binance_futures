# Protocol

1. Copy canonical ID121 exact-exit strategy snapshot before reading holdout returns.
2. Holdout symbols are fixed: ALGOUSDT, INJUSDT, LDOUSDT, PENDLEUSDT, PYTHUSDT.
3. Start dates are fixed to the previously reserved production-eligibility dates.
4. End date is 2026-09-01.
5. Use existing project historical data only. Do not backfill or write DB for this audit.
6. Engine execution is fixed: leverage 4, TP8, SL6, fee 0.0005 per side, slippage 5bps per side, single position.
7. Exact close rule is ROI >= 8 || ROI <= -6.
8. Report per-symbol PF, aggregate PF, breadth, frequency, long/short split, year split and exit reasons.
9. Do not tune ID121 based on the holdout result.
