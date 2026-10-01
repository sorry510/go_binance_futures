# ID121 Fresh-Symbol Holdout — 2026-10-01

Purpose: test the canonical ID121 exact TP8/SL6 entry on a fresh symbol cohort that has not been used in prior ID121 tuning or validation.

Frozen holdout symbols and starts:
- ALGOUSDT: 2024-12-01
- INJUSDT: 2024-12-01
- LDOUSDT: 2024-12-01
- PENDLEUSDT: 2025-07-01
- PYTHUSDT: 2025-11-01

End: 2026-09-01.

The strategy snapshot is copied exactly from the canonical ID121 exact-exit audit. No entry parameter, indicator, funding rule, eligibility rule, or exit rule is changed. CLOSE_LONG and CLOSE_SHORT remain exactly ROI >= 8 || ROI <= -6.

Interpretation is pre-registered:
- Strong holdout support: aggregate PF >= 1.15 and at least 3/5 symbols positive.
- Weak/partial support: aggregate PF > 1 but below 1.15, or only 2/5 symbols positive.
- Failure: aggregate PF <= 1 or clear negative breadth.
- Frequency is reported but is not optimized here.
- Whatever the outcome, do not retune ID121 against these five symbols.

No DB write, no app.conf change, no commit/push.

## Final result

Fresh-symbol holdout failed decisively. ALGO/INJ/LDO/PENDLE/PYTH produced 238 trades, aggregate PF 0.847944, 0/5 symbols positive, frequency 0.629154 trades/symbol/week. Year PFs were 0.745679 for 2025 and 0.968217 for 2026. Long side: 117 trades, PF 0.850740. Short side: 121 trades, PF 0.845310. Therefore the failure is not isolated to one direction. ID121 remains evidence-backed only for its original 2024+ cohort; it is not supported as a cross-new-symbol universal strategy. No retuning is allowed from this holdout.
