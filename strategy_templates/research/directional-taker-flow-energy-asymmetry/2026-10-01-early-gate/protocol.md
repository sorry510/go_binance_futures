# Protocol

1. Freeze six-symbol 2023-2024 discovery before returns.
2. Use only completed project-native USD-M 1h QuoteVolume and TakerBuyQuoteVolume.
3. Hourly signed flow = 2*TakerBuyQuoteVolume/QuoteVolume - 1.
4. Over trailing 24 completed hours, buy energy is the sum of squared positive flow; sell energy is the sum of squared absolute negative flow.
5. Score = log(buy energy / sell energy), requiring both energies >0.
6. Score <=0 to >0 emits LONG; >=0 to <0 emits SHORT. Enter next complete 1h open.
7. No magnitude threshold, trend/price/QPS/funding/OI filter, ratio normalization, or symbol-specific rule.
8. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes the family: no 12h/48h scan, no exponent scan, no side deletion, no reversal.
10. 2025+ OOS remains unread unless the early gate passes.
