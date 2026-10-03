# v170 Quote-Volume Persistence Emergence Momentum — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h klines and QuoteAssetVolume. No cross-symbol or benchmark input.
3. Transform each completed hour's activity as x_t = log(QuoteAssetVolume_t), requiring positive QuoteAssetVolume.
4. Over the latest 48 complete hours, compute ordinary lag-1 Pearson autocorrelation corr(x_t, x_(t-1)).
5. Trigger only on a fresh zero up-cross from <=0 to >0. This marks the emergence of persistent/clustered trading activity. Zero is the natural autocorrelation boundary.
6. Direction is preregistered activity-confirmed momentum: trailing completed 4h USD-M close-to-close return >0 -> LONG; <0 -> SHORT; exact zero -> no signal.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No volume magnitude threshold, z-score, QPS, trade-count, taker, funding, OI, volatility, time-of-day, symbol-specific rule or re-arm threshold.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 24h/72h/96h windows, add autocorrelation thresholds, use abs-return/volume interactions, delete one side, reverse to fade, or inspect 2025+.
11. This is distinct from single-hour QV shocks, QPS record bursts and return-volatility clustering: the state variable is serial dependence in the activity level itself.
