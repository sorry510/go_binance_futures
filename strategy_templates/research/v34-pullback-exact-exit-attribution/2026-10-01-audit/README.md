# v34 Pullback Exact-Exit Attribution — 2026-10-01 Audit

Objective: determine whether the legacy v34 rule named funding_pullback_resume_short_v34 contributes positive expectancy under the current canonical fixed-exit semantics.

This is an attribution audit, not parameter research. The original v34 entry logic is preserved. Two variants are run on the same 15-symbol production eligibility universe:
- FULL: all original v34 entry rules enabled.
- BASE: identical strategy with only funding_pullback_resume_short_v34 disabled.

Both variants use exact CLOSE_LONG/CLOSE_SHORT = ROI >= 8 || ROI <= -6. Execution remains leverage4, fee0.0005/side, slippage5bps/side, single position.

Interpretation is based on paired trades keyed by symbol, entry_time and side. If FULL-only trades are positive and aggregate FULL improves over BASE, pullback-resume is a viable orthogonal setup candidate. If FULL-only is negative or FULL degrades BASE, the mechanism is frozen. No threshold tuning or rule edits are permitted after seeing results.

## Final result

Enabling funding_pullback_resume_short_v34 increased frequency from 0.256350 to 0.494782 trades/symbol/week but reduced aggregate PF from 1.006467 to 0.922210. The FULL-only marginal cohort had 356 trades, PF 0.842817 and normalized net -0.565383; yearly PFs were 0.717990 / 0.746190 / 1.207816 for 2024/2025/2026. The mechanism is therefore frozen as negative-expectancy frequency expansion. No threshold or rule tuning is permitted.
