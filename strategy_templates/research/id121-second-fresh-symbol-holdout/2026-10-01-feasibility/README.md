# ID121 Second Fresh-Symbol Holdout — 2026-10-01 Feasibility

Goal: construct a second independent, untouched 5-symbol holdout for canonical ID121 without selecting symbols by strategy returns.

Selection rule frozen before any ID121 replay:
- exclude the original ID121 15-symbol cohort;
- exclude the first fresh holdout ALGOUSDT / INJUSDT / LDOUSDT / PENDLEUSDT / PYTHUSDT;
- require current QuoteVolume >= 5,000,000 USDT;
- require local 1m history starting no later than 2023-01-01;
- require local 1m history covering through 2026-08-31 23:59 UTC;
- use only already-present local data; no DB backfill/write;
- require at least 5 eligible symbols before running any ID121 returns.

Result: only LITUSDT satisfies all criteria in the current local cache. Eligible count = 1, below the pre-registered minimum cohort size of 5.

Decision: DATA/COVERAGE BLOCKED. Do not run a one-symbol pseudo-holdout. No ID121 returns were inspected for LITUSDT. Revisit only if at least four additional untouched, eligible symbols are already present locally or the user explicitly authorizes historical-data backfill.
