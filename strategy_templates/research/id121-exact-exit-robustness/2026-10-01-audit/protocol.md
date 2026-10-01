# Protocol

1. Use only canonical ID121 rows from the exact-exit audit trades.csv.
2. Use normalized return = raw NetPnL / entry notional.
3. Compute per-symbol PF/net and each leave-one-symbol-out portfolio PF.
4. Aggregate by UTC calendar month and quarter; do not split trades across blocks.
5. Bootstrap entire monthly blocks with replacement, using the same number of months as observed, 10,000 replicates, deterministic seed 121.
6. Report PF quantiles and fraction of bootstrap PF <=1; do not optimize or remove bad months/symbols.
7. This audit is descriptive evidence only; it cannot alter ID121 parameters.
