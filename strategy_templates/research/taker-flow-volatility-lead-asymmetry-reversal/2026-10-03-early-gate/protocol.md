# v175 Taker-Flow Volatility-Lead Asymmetry Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h klines.
3. Define signed taker flow for each completed hour as f_t = 2*TakerBuyQuoteVolume_t/QuoteAssetVolume_t - 1, requiring positive QuoteAssetVolume.
4. Define one-hour log return r_t = log(C_t/C_(t-1)).
5. Over the latest 48 completed pairs, compute score = Pearson correlation between f_t and r_(t+1)^2.
6. A fresh zero up-cross means buy-side aggressor flow has become the side associated with larger next-hour variance; preregister SHORT. A fresh zero down-cross means sell-side aggressor flow has become the volatility-generating side; preregister LONG.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No flow magnitude threshold, z-score, current-flow confirmation, funding, OI, QPS, volume filter, ATR, time-of-day, trend confirmation or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 24h/72h/96h windows, add correlation thresholds, replace squared return with absolute return, add current-flow sign confirmation, delete one direction, invert the mapping, or inspect 2025+.
11. This is distinct from taker-flow autocorrelation, taker->future-return covariance and v171 return->future-volatility asymmetry: v175 asks which aggressor side is associated with next-period variance.
