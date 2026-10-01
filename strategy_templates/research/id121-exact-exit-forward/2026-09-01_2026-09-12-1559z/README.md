# ID121 Exact TP8/SL6 Forward OOS — 2026-09-01 to 2026-09-12 15:59 UTC

Purpose: re-evaluate the untouched September 2026 forward window using the canonical exact-exit ID121 strategy after the Engine exit-semantics correction.

The underlying local data coverage is unchanged from the original forward note: all 15 production symbols share complete row data through 2026-09-12 15:59 UTC. This audit does not fetch or backfill data.

To preserve production state and single-position semantics, each symbol is replayed from its original production eligibility start through 2026-09-12 16:00 UTC. Only trades with entry_time >= 2026-09-01 00:00 UTC are counted as forward trades.

Strategy entry, universe, starts, fees, slippage, leverage and single-position semantics are unchanged. CLOSE_LONG and CLOSE_SHORT are the frozen canonical exact rule: ROI >= 8 || ROI <= -6.

This is an audit only. No parameter changes or DB writes are allowed.

## Final result

Canonical exact-exit forward produced 9 trades, all LONG: 2 TP and 7 SL, normalized PF 0.532989, normalized net -0.059170824, and 1/15 symbols positive. This remains a forward warning, but the sample is small.

Historical context using all 773 canonical ID121 exact-exit trades: among 765 consecutive 9-trade windows, 119 (15.56%) had PF <= 0.532989 and 101 (13.20%) had <=2 wins. Historical 9-trade PF q10=0.330857, q25=0.584994, median=1.065986. Therefore the September window is poor but not an exceptionally rare historical tail. ID121 remains the current 2024+ formal candidate; continue untouched forward accumulation and do not tune against this window.
