# Binance Earn Wednesday Flexible-Offer Activation → LONG — Feasibility

Hypothesis: when an ordinary token newly enters Binance Earn Wednesday's Simple Earn Flexible Products limited-time offer set, the incremental yield incentive may temporarily increase holding/subscription demand.

Event construction is state-based, not "every weekly mention":
- 2022-12-28 is used only as the pre-2023 warmup Flexible set.
- All 88 Earn Wednesday articles in 2023-2024 are parsed from structured table rows.
- Only the Flexible Products subsection is used; Locked, Staking, Liquidity Farming, Auto-Invest and Dual Investment are excluded.
- Stablecoin/fiat bases are excluded.
- An event occurs only when a token is present this week but absent from the immediately preceding Earn Wednesday Flexible set. Continuous repeated weekly offers do not retrigger; disappear/re-enter can retrigger.

The first CMS pass reached article 75 before rate limiting. Its exact parsed state is preserved in `results/audit.log`; `resume_audit.py` restored that checkpoint and fetched only the remaining 13 articles at a slower cadence. Final parser classification was also normalized to exclude AEUR/EURI/EURT.

## Coverage

- 88 discovery-period articles plus one warmup article.
- **212 raw activation events / 113 unique tokens / 76 independent weekly batches**.
- Production eligibility: same-name Binance USD-M history >=730 days at publish time and prior 24 complete 1h QuoteVolume >=5M USDT.
- **74 eligible events / 38 unique tokens / 43 independent batches**.
- Exclusions: 136 history/missing, 1 missing prior-24h window, 1 QV<5M.
- 2023: 44 events / 29 tokens / 23 batches.
- 2024: 30 events / 14 tokens / 20 batches.

Coverage gate passes. No post-event return was read during feasibility. Discovery is frozen separately under `../2023-2024-discovery/`.
