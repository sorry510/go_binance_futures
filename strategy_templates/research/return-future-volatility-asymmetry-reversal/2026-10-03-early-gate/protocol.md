# v171 Return–Future-Volatility Asymmetry Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h closes. Define one-hour log return r_t = log(C_t/C_(t-1)).
3. For the latest 48 completed return pairs, compute score = Pearson correlation between r_t and r_(t+1)^2. This is a causal rolling estimate of whether positive or negative returns are associated with larger next-hour variance.
4. Require exact continuous history and non-zero variance in both correlation inputs.
5. A fresh zero up-cross (score <=0 to >0) means upside shocks have become the volatility-generating side; preregister SHORT. A fresh zero down-cross (score >=0 to <0) means downside shocks have become the volatility-generating side; preregister LONG.
6. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
7. No correlation magnitude threshold, z-score, return-size filter, funding, OI, volume, taker, ATR, time-of-day, trend confirmation, or symbol-specific rule.
8. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan 24h/72h/96h windows, add correlation thresholds, use absolute return instead of squared return, delete one direction, invert the mapping, or inspect 2025+.
10. This is distinct from volatility clustering (autocorrelation of squared returns), semivariance, realized skewness and jump detection: v171 measures signed-return to next-period variance asymmetry.
