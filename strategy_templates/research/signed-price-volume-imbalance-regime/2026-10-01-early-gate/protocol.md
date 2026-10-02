# Protocol

1. Freeze six-symbol 2023-2024 discovery before reading returns.
2. Use only completed project-native USD-M 1h QuoteVolume and TakerBuyQuoteVolume.
3. Per bar signed price-volume imbalance = 2*TakerBuyQuoteVolume - QuoteVolume.
4. Score = arithmetic mean of the last 24 completed 1h signed price-volume imbalance values.
5. Score <=0 to >0 emits LONG; score >=0 to <0 emits SHORT. Enter at the next complete 1h open.
6. The 24h window is the shortest one-day formation horizon motivated by perpetual-futures price-volume literature; zero is the natural buyer/seller balance. No threshold search.
7. This is not a replication of the paper's cross-sectional quintile sort; cross-symbol ranking is explicitly prohibited. It is a single-symbol time-series adaptation.
8. No QPS, trend, ATR, funding, OI, MarketCondition, Benchmark, or NowTime modulo.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes the family: no 12h/48h window scan, no magnitude threshold, no ratio normalization, no breakout/trend filter, no reversal.
11. 2025+ OOS remains unread unless early gate passes.
