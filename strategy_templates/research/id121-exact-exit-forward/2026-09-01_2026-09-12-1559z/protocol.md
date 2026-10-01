# Protocol

1. Use the frozen ID121 exact-exit JSON without any entry edits.
2. Replay each symbol continuously from its original production eligibility start to 2026-09-12 16:00 UTC.
3. Preserve single-position state across 2026-09-01; do not reset the engine at the forward boundary.
4. Count only trades whose entry_time is >= 2026-09-01 00:00 UTC.
5. Exact exits are ROI >= 8 || ROI <= -6.
6. Do not fetch/backfill data, tune parameters, remove symbols, or reinterpret the direction after seeing results.
7. Compare only descriptively with the older non-canonical September forward note.
