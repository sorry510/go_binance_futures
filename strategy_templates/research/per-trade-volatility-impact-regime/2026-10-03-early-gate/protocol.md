# v182 Per-Trade Volatility-Impact Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h Klines: Close and TradeCount.
3. For each non-overlapping 24h block, impact = sum(r_t^2) / sum(TradeCount_t), requiring positive total TradeCount and continuous bars.
4. Compare the latest completed 24h block with the immediately preceding 24h block: score = log(impact_latest/impact_previous).
5. Fresh zero up-cross means more realized variance is being generated per transaction; preregister liquidity/absorption deterioration and fade the completed trailing 4h price direction.
6. Fresh zero down-cross means variance per transaction has fallen; preregister improved absorption and follow the completed trailing 4h price direction.
7. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain within the signal calendar year.
8. No impact magnitude threshold, z-score, quote-volume filter, funding, OI, taker, ATR, time-of-day or symbol-specific rule.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan block lengths, add impact thresholds, substitute range for variance, delete one crossing/side, invert the mapping, or inspect 2025+.
11. This is distinct from Amihud (return per dollar volume), average trade size (dollars per trade), and Kyle-like flow impact: v182 normalizes realized price variance by actual transaction count.
