# ID121 no-no-chase — 2026-09 Untouched Temporal Holdout Protocol

Purpose: perform a forward sanity check of the generated no-no-chase simplification using a time interval that was strictly outside the mechanism-ablation sample. This is not candidate promotion by itself.

1. The ablation sample ended exactly at 2026-09-01 00:00 UTC. Freeze the temporal holdout to [2026-09-01 00:00, 2026-10-01 00:00) UTC.
2. Use the unchanged original ID121 15-symbol cohort. Do not add/remove symbols from holdout outcomes.
3. Evaluate exactly two already-frozen strategies on the same holdout: canonical ID121 base and no_no_chase. no_no_chase removes only the existing no-chase clause; every other rule/threshold remains identical.
4. Exact costs/exits remain leverage=4, TP8, SL6, fee=0.0005/side, slippage=5bps/side, single-position semantics.
5. Before any strategy return is read, require complete local execution/history coverage for all 15 symbols through 2026-09-30 23:59 UTC. Coverage checking may read DB metadata/rows only; it may not backfill or write.
6. If coverage fails, stop without reading strategy returns. Do not fetch/backfill because the previously frozen second-holdout protocol requires explicit user authorization for historical backfill.
7. This one-month check is only a forward sanity gate. Even a positive result cannot promote no_no_chase while the independent second fresh-symbol holdout remains coverage-blocked.
8. A negative result is sufficient evidence to keep the simplified route frozen. No retuning, side deletion, threshold changes or alternate September subwindows are allowed.
