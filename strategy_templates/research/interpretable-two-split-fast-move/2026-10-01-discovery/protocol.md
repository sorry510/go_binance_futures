# Protocol

- 2023 train, 2024 validation; 2025/2026 remain unread.
- Six symbols: SOL, DOGE, LTC, AVAX, UNI, ZEC USDT perpetuals.
- Depth exactly 2; thresholds only train q10/q25/q50/q75/q90; minimum train child 2000 resolved samples.
- Fixed nine DSL-translatable 1h features; no symbol feature.
- Label uses next-1m-open entry with 5bps entry slippage and future 1m CLOSE path up to 24h; gross 4x ROI TP +8 before SL -6. Unresolved samples are not used to fit.
- Select train leaves only when mean reward (+8 TP / -6 SL) is at least +1.0.
- Validation gate: at least 1000 resolved selected samples, mean reward at least +0.5, and at least 4/6 symbols positive.
- Failure freezes this route without tuning depth, quantile grid, features, leaf size or reward thresholds.
