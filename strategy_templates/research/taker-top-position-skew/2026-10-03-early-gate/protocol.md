# v172 Taker-vs-Top-Position Skew — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision USD-M 5m metrics. For each UTC hour keep only the final completed metrics row in that hour.
3. Taker flow state = sum_taker_long_short_vol_ratio. Capital-weighted top-trader positioning state = sum_toptrader_long_short_ratio. Both must be positive.
4. score = log(taker_ratio / top_position_ratio). score >0 means current aggressive flow is more long-biased than top-trader capital positioning; score <0 means it is more short-biased.
5. Trigger only on exact hourly consecutive observations: zero up-cross -> LONG; zero down-cross -> SHORT. Gaps do not create synthetic crosses.
6. Entry is the next complete Binance USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
7. No magnitude threshold, z-score, smoothing, price trend, funding, OI, QPS, volatility, time-of-day or symbol-specific filter.
8. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan ratio thresholds, smooth either series, substitute global-account ratio, add price/funding/OI filters, delete one side, reverse the mapping, or inspect 2025+.
10. This is distinct from v147/v158 positioning-skew families and standalone taker-flow families: v172 compares current aggressive execution flow directly with capital-weighted top-trader position bias.
