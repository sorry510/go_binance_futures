# Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. Do not read 2025+ unless the frozen gate passes.
2. Source is Binance Vision USD-M daily metrics at 5m cadence. Use only completed rows.
3. For each UTC hour, retain the last completed metrics row in that hour. No intra-hour signal is allowed.
4. score = log(sum_toptrader_long_short_ratio / count_toptrader_long_short_ratio).
5. Interpretation: score > 0 means top-trader capital positioning is more long-biased than top-trader account headcount; score < 0 means more short-biased.
6. Up-cross from <=0 to >0 emits LONG. Down-cross from >=0 to <0 emits SHORT. No magnitude threshold, z-score, smoothing, or re-arm threshold.
7. Enter at the next complete 1h open. Measure signed 1h/4h/12h returns.
8. No price breakout, trend, funding, OI, QPS, taker, or symbol-specific filter.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and 2023/2024 12h means both >0.
10. Failure freezes the family: do not scan thresholds, smoothing windows, 5m entry, position/global ratio substitutions, delete one direction, or reverse the signal.
11. This differs from the 2026-09-23 study, which tested positioning as a v33 gate and positioning-change alignment around fresh 12h breakout events; this test is the positioning skew's own zero-cross event without a price setup.
12. A zero-cross is valid only when the current and previous hourly sampled metrics points are exactly one hour apart; gaps do not create synthetic crosses.
13. The last 5m metrics observation of hour H is treated as information available by the H+1:00 boundary; entry is the H+1:00 Binance USD-M 1h open.
14. The 12h endpoint must remain in the same calendar year as the signal so the 2024 annual check cannot read 2025 prices.
15. 2022-12-31 metrics may be downloaded only as causal warmup for the first 2023 zero-cross; no pre-2023 signal or return is included.
