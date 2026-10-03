# v158 Elite-vs-Crowd Account Skew Zero-Cross

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC in 2023-2024 only. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision USD-M 5m metrics. For each UTC hour retain only the last completed observation; adjacent sampled points must be exactly one hour apart.
3. score = log(count_toptrader_long_short_ratio / count_long_short_ratio).
4. score > 0 means top-trader account headcount is more long-biased than the global account population; score < 0 means relatively more short-biased.
5. Zero up-cross (<=0 to >0) -> LONG; zero down-cross (>=0 to <0) -> SHORT. Enter at the next 1h open.
6. No TopPosition field, smoothing, magnitude threshold, z-score, price breakout, funding, OI, taker, or symbol-specific filter.
7. 12h endpoint must remain in the same calendar year as the signal.
8. Early gate: event-weighted 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes the family: do not scan thresholds/smoothing, substitute Position/Global or Position/Account, delete a side, reverse direction, or add filters.
10. This is distinct from v147: v147 compared top-trader position capital weighting to top-trader account headcount; v158 compares top-trader account headcount directly with all-market account headcount.
