# Binance Portfolio Margin Collateral-Ratio — 2024 Feasibility Correction

This corrects the earlier feasibility audit. The complete official 2024 article/body reconstruction contains 55 directional asset-level collateral-ratio changes.

Frozen mapping:
- collateral-ratio increase -> LONG;
- collateral-ratio decrease -> SHORT.

Production eligibility was applied before any corrected-event return:
- same-name USD-M history >=730 days at effective time;
- prior 24 complete 1h QuoteVolume >=5M USDT.

Result:
- 55 raw events.
- **8 eligible events / 8 unique assets / 4 independent official articles**.
- 3 LONG / 5 SHORT.
- 9 no event-month USD-M history.
- 38 USD-M history <2 years.
- Coverage gate passes exactly at 8 events / 8 assets / 4 articles.
- No post-event return was read during feasibility.

The corrected eligible set is frozen under `../2024-corrected-exact-discovery/`.
