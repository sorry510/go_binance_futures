# v183 24h Price-Monotonicity Regime Cross — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h closes.
3. For the latest 24 complete hours, compute Spearman rank correlation between chronological position 1..24 and the 24 Close values. Price ties, if any, use average ranks.
4. Recompute the same statistic for the 24h window ending one hour earlier.
5. Fresh zero up-cross from <=0 to >0 -> LONG. Fresh zero down-cross from >=0 to <0 -> SHORT. Zero is the natural boundary between predominantly rising and predominantly falling ranked paths.
6. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
7. No correlation magnitude threshold, z-score, price-return threshold, breakout, funding, OI, taker, volume, ATR, time-of-day or symbol-specific rule.
8. Early gate: 12h signed mean >=+0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan 12h/48h windows, add rank-correlation thresholds, replace Spearman with linear-regression slope, delete one side, reverse the mapping, or inspect 2025+.
10. This is distinct from Aroon/time-since-extreme, path efficiency and return autocorrelation: v183 uses the full ordinal path of all 24 closes.
