# Protocol

1. Freeze JSON before returns.
2. Regime transition is previous completed daily TakerBuyRatio crossing 0.5.
3. During the current day, use only same-direction completed 1h candles; entry requires current price to break their extreme.
4. Fixed 4x TP8 SL6 fee/slippage single-position execution.
5. Core-4 early gate only.
6. Failure => no 0.52/0.48 threshold, no EMA/ADX/QPS/funding filters, no reversal.
7. Pass => freeze parameters and expand symbols before holdout.

## Audit correction

The initial run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. Therefore that run did not implement strict fixed TP8/SL6 and is non-canonical. Its files are preserved under `legacy/conditional-exit/`. The canonical rerun keeps entry logic unchanged and uses `ROI >= 8 || ROI <= -6` for both close rules.
