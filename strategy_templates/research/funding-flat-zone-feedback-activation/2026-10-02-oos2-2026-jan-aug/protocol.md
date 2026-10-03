# OOS2 Protocol — frozen before 2026 outcomes

1. Reuse v154 discovery/OOS1 rule exactly. No signal, side, threshold or timing change.
2. OOS2 is 2026-01-01 through 2026-09-01 on the same six symbols.
3. The cutoff is fixed because September 2026 monthly fundingRate is available but the Binance Vision September 1h kline monthly archive is not yet sealed (404 on 2026-10-02). Do not mix a partial September source into this test.
4. Only adjacent normal 8h settlements are eligible. Previous funding must equal 0.0001 and current must leave it; >0.0001 SHORT, <0.0001 LONG.
5. Entry is next complete 1h open; measure 1h/4h/12h.
6. OOS2 pass gate remains identical: 12h mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week.
7. Failure freezes v154. Passing permits exact TP8/SL6 Engine-path validation; no parameter tuning is permitted either way.
8. Side imbalance is audit-only. Do not delete SHORT/LONG, add magnitude/premium/OI filters, or change equality tolerance.
