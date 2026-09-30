# Funding Interval Compression Feasibility

## Motivation

Binance can shorten funding settlement frequency during stressed conditions. The intended research family was to trade against the crowded side when settlement frequency compresses.

## Data audit

After 2025-05-02, the local `market_funding_rates` archive contains:
- 8h gaps: 26,936 observations across 23 symbols.
- 4h gaps: 10,997 observations across 5 symbols.
- 1h gaps: **0 observations**.

Therefore the primary 8h/4h -> 1h mechanism cannot be reconstructed from the current local archive.

## Fallback 8h -> 4h feasibility

A strict search requiring repeated 8h gaps before the transition found only one event:
- SOLUSDT, 2022-11-09 20:00 UTC.
- Previous funding rate: -0.015.
- Contract-history age under the local 1h archive: -43.5 days.
- Production eligibility (>=730 days): **false**.

Production-eligible sample size is therefore **0**.

## Decision

This family is **not classified as a failed alpha**. It is frozen as **currently unverifiable / insufficient sample**.

Do not relax the >=2-year rule, liquidity floor, or merge unrelated funding-frequency events merely to create a sample. Revisit only if a future local archive preserves 1h funding observations or another point-in-time source becomes available.

## Binance Vision full-universe audit (2026-09-28)
Re-audited official Binance Vision monthly fundingRate archives for the formal 15-symbol universe across 2024-01..2026-08. Fourteen symbols contain only 8h intervals; ONDOUSDT contains only 4h intervals from the beginning of its available archive. There are zero 1h rows and zero interval-change transitions in the formal universe.

This confirms the earlier blocker is not a local-table artifact. There is no production-eligible compression event to test. The family remains permanently frozen unless Binance later publishes a genuinely point-in-time interval-change history with new events.
