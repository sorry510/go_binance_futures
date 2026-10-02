# Protocol

1. Freeze the six-symbol 2023-2024 discovery set before reading returns.
2. Use only completed project-native USD-M 1h Klines and QuoteVolume; no external market feature, MarketCondition, Benchmark, OI, funding, taker, or NowTime modulo.
3. For each completed hour, compute its log return from open to close and classify it positive or negative.
4. Over the trailing 24 completed hours, define upside impact = sum(positive log returns) / sum(QuoteVolume of positive-return hours).
5. Define downside impact = sum(abs(negative log returns)) / sum(QuoteVolume of negative-return hours).
6. Score = log(upside impact / downside impact). Require both sides to have nonzero return sum and volume.
7. A score cross from <=0 to >0 emits LONG; >=0 to <0 emits SHORT. Enter at the next complete 1h open.
8. No magnitude threshold, volume threshold, trend filter, or re-arm threshold is permitted.
9. Early gate: 12h event-weighted signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and 2023/2024 12h means both >0.
10. Failure freezes this family: no 12h/48h lookback scan, no impact threshold, no taker/ATR/trend filter, no reversal.
11. 2025+ OOS remains unread unless this gate passes.
