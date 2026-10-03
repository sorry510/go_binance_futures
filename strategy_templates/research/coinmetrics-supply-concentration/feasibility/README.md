# CoinMetrics Supply Concentration — Feasibility

Proposed signal: top-1%-address supply share = SplyAdrTop1Pct / SplyCur, or the related Supply Equality Ratio (SER), to measure whale concentration/distribution independently from price and transaction activity.

Community API access audit on 2026-10-02:
- SplyCur: available for all 10 fixed CoinMetrics assets (BTC/ETH/XRP/ADA/LINK/BCH/LTC/DOGE/UNI/ZEC).
- SplyAdrTop1Pct: HTTP 403 Forbidden.
- SER: HTTP 403 Forbidden.

The concentration numerator/ratio is therefore not reproducible under the project's no-paid-data assumption. No alternative address-threshold metric or proxy was substituted after observing the blocker.

Decision: **data-access blocked / freeze feasibility**. No returns read and no DB write.
