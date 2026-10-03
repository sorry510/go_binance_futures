# v187 Taker-vs-Global-Account Skew — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use Binance USD-M 5m metrics. For each UTC hour, use only that hour's final completed metrics row.
3. score = log(sum_taker_long_short_vol_ratio / count_long_short_ratio), requiring both ratios >0.
4. A fresh zero up-cross means aggressive taker flow has become more long-biased than the global account headcount and triggers LONG. A fresh zero down-cross triggers SHORT.
5. Entry is the next complete Binance USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
6. No ratio magnitude threshold, smoothing, price trend, funding, OI, QPS, ATR, time-of-day or symbol-specific rule.
7. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
8. Failure freezes the positioning-pair audit space: do not substitute Top Account/Top Position after outcomes, add thresholds, delete one side, reverse the mapping, or inspect 2025+.
9. This is the final natural pairwise positioning audit: Top Position/Top Account, Top Account/Global and Taker/Top Position are already covered elsewhere.
