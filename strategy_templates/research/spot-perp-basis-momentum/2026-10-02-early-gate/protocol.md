# v161 Spot–Perp Basis Momentum — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Spot and USD-M 1h bars are timestamp-aligned using Binance Vision. No cross-symbol benchmark is used.
3. basis_t = log(perp_close_t / spot_close_t).
4. momentum_t = basis_t - basis_(t-24h), using completed 1h closes only.
5. Up-cross from momentum<=0 to >0 emits LONG; down-cross from >=0 to <0 emits SHORT.
6. Adjacent signal observations must be exactly one hour apart and the 24h lag must exist exactly 24 hours earlier. Gaps do not create synthetic zero-crosses.
7. Entry is the next complete USD-M 1h open. Signed returns are measured at 1h/4h/12h. The 12h endpoint must remain in the signal calendar year.
8. No basis magnitude threshold, z-score, smoothing, funding/OI/taker/volume/price-trend filter, re-arm threshold, or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and both 2023 and 2024 12h means >0.
10. Failure freezes basis-momentum continuation: do not scan 6h/12h/48h momentum windows, add magnitude thresholds, delete one direction, or reverse to mean reversion.
11. This is distinct from the prior spot-perp basis-convergence family, which tested 720h z-score extremes followed by convergence. v161 tests the basis trend's own zero-cross with no extreme event.
