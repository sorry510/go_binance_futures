# ID121 no-no-chase — 2026-09 Untouched Temporal Holdout

The mechanism ablation ended exactly at 2026-09-01 00:00 UTC. Therefore September 2026 was preregistered as an untouched temporal sanity window for canonical ID121 versus the generated no-no-chase simplification.

Before any strategy return was read, local execution/funding coverage was audited for the unchanged original 15-symbol cohort.

## Coverage result

Required per symbol:
- 2026-09-01 00:00 through 2026-09-30 23:59 UTC;
- 43,200 complete local 1m bars;
- September realized funding coverage through the final standard settlement.

Observed:
- Most symbols: **16,800 / 43,200 1m rows**, ending 2026-09-12 15:59 UTC, with only 36 funding rows.
- BTC: 17,814 rows; ETH: 17,821 rows; still far short of full September.
- No September 1m monthly chunks are present.
- Funding coverage is correspondingly partial.

`all_15_complete=false`.

## Decision

**Temporal holdout coverage blocked. No canonical/no-no-chase strategy return, trade, PF or PnL was read.**

No historical backfill or DB write was performed. Per the frozen protocol, do not shorten September to the available partial interval, choose symbols with slightly longer coverage, or fetch/backfill history without explicit authorization. The no-no-chase simplification remains an unvalidated generated hypothesis and cannot be promoted.
