# v181 Activity–Volatility Coupling Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h klines.
3. activity_change_t = log(QuoteAssetVolume_t / QuoteAssetVolume_(t-1)), requiring positive quote volume and continuous bars.
4. volatility_t = r_t^2 where r_t is the one-hour close-to-close log return.
5. Over the latest 48 completed hours compute score = Pearson corr(activity_change_t, volatility_t).
6. Fresh zero up-cross means activity expansion has become positively coupled to high-volatility hours; preregister activity-confirmed continuation and follow the completed trailing 4h price return.
7. Fresh zero down-cross means activity and volatility have decoupled; preregister exhaustion/noise reversal and fade the completed trailing 4h price return.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
9. No correlation magnitude threshold, z-score, volume shock threshold, funding, OI, taker, ATR, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not scan windows, add correlation/volume thresholds, substitute absolute return, delete one crossing/side, invert the mapping, or inspect 2025+.
12. This differs from price-volume correlation and v136 volume->future-return lead: v181 measures whether activity expansion is associated with contemporaneous volatility magnitude, then uses that state only to choose continuation versus fade.
