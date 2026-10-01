# Protocol

1. Freeze JSON before returns.
2. Trigger when completed 1h ADX14 crosses from <=20 to >20.
3. Direction is DI dominance on the trigger hour.
4. Entry requires current price to break trigger-hour high/low.
5. Fixed project execution: 4x TP8 SL6 fee/slippage, single position.
6. Core-4 early gate only.
7. Failure => no ADX18/25 scan, no EMA/volume/funding/Donchian filter, no reversal.
8. Pass => freeze and expand symbols before holdout.

## Audit correction

The initial run used a conditional CLOSE expression. In this Engine, RunConfig TP/SL values gate close-rule evaluation but do not force closure. Therefore that run did not implement strict fixed TP8/SL6 and is non-canonical. Its files are preserved under `legacy/conditional-exit/`. The canonical rerun keeps entry logic unchanged and uses `ROI >= 8 || ROI <= -6` for both close rules.
