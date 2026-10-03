# 2024 Feasibility Correction Protocol

1. Supersedes the incomplete 2026-09-27 census that hard-coded 5 of the 8 official 2024 Portfolio Margin collateral-ratio update notices.
2. Rebuild all 8 official 2024 notices from Binance article bodies before reading any post-event return.
3. Parse every crypto asset row with explicit before/after collateral ratio. Stablecoin/fiat-like assets may remain in the raw audit but are excluded from directional token events before eligibility.
4. Preserve the original preregistered mapping: collateral-ratio increase -> LONG; decrease -> SHORT.
5. Effective time is the official "from YYYY-MM-DD HH:MM (UTC)" timestamp, not publish time.
6. Production eligibility is unchanged: same-name USD-M history >=730 days at effective time and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Preserve the original minimum useful sample gate: >=8 eligible token-events. No return may be read unless this passes.
8. If the gate passes, freeze a separate discovery protocol before returns; do not change direction, eligibility, asset list or event time.
9. Do not use current collateral ratios to backfill history, split one article into fake independent studies, or drop weak rows after outcomes.
