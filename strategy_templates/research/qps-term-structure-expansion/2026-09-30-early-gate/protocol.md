# Protocol

1. Freeze entry JSON before returns.
2. Baseline = previous completed daily Qps.
3. Trigger when completed 1h Qps crosses from <= baseline to > baseline.
4. Direction = trigger-hour candle sign; current price must break trigger-hour extreme.
5. Exact fixed exit rules are `ROI >= 8 || ROI <= -6` for both CLOSE_LONG and CLOSE_SHORT.
6. Core-4 early gate only.
7. Failure => no baseline multiplier/window scan, no trend/taker/funding filters, no reversal.
8. Pass => freeze and expand symbols before holdout.

Technical note: an initial dry run used false close rules and was invalid because RunConfig ROI thresholds are gates rather than forced exits. No valid research conclusion was taken from that run; the canonical run uses the exact fixed exit rules above.
