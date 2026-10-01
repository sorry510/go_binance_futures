# Protocol

1. Freeze JSON before returns.
2. Direction requires two completed 1h TakerBuyRatio observations on same side of 0.5.
3. Both signal-hour QPS values must be below mean QPS of hours [3:11].
4. Entry requires current price to break the high/low of the two signal hours.
5. Fixed 4x TP8 SL6 fee/slippage single-position execution.
6. Core-4 early gate only.
7. Failure => no 0.52/0.48 threshold tuning, no 3h persistence, no trend/funding filter, no reversal.
8. Pass => freeze parameters and expand symbols before holdout.

## Audit correction

The initial run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. Therefore that run did not implement strict fixed TP8/SL6 and is non-canonical. Its files are preserved under `legacy/conditional-exit/`. The canonical rerun keeps entry logic unchanged and uses `ROI >= 8 || ROI <= -6` for both close rules.
