# v176 Aggressor-Activity Coupling Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h Klines.
3. signed_flow_t = 2*TakerBuyQuoteVolume_t/QuoteAssetVolume_t - 1, requiring positive QuoteAssetVolume.
4. activity_change_t = log(QuoteAssetVolume_t/QuoteAssetVolume_(t-1)), requiring continuous positive QuoteAssetVolume.
5. Over the latest 48 completed hours, score = Pearson corr(signed_flow_t, activity_change_t).
6. Fresh zero up-cross -> LONG: rising activity has become more associated with buy-side aggressor pressure. Fresh zero down-cross -> SHORT: rising activity has become more associated with sell-side aggressor pressure.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No correlation magnitude threshold, z-score, current-flow threshold, price-trend confirmation, funding, OI, QPS, trade-count, ATR, time-of-day, or symbol-specific rule.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 24h/72h/96h windows, add correlation thresholds, use level QuoteVolume instead of change, delete one direction, reverse the mapping, or inspect 2025+.
11. This is distinct from price-volume correlation, taker-flow autocorrelation, taker->future-return and v172/v175 volatility-lead families: v176 measures contemporaneous coupling between aggressor direction and activity expansion.
