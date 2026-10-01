# Protocol

1. Freeze strategy before returns.
2. Rolling VWAP uses sum(QuoteVolume)/sum(Volume) across exactly four completed 1h bars.
3. Trigger is a completed-hour close cross of rolling VWAP relative to the prior completed hour's own 4h VWAP.
4. Current price must break the trigger-hour extreme.
5. Fixed project 4x TP8 SL6 fee/slippage single-position execution.
6. Core-4 early gate only.
7. Failure => no 3h/6h/8h window scan, no deviation threshold, no trend/volume/funding filter, no reversal.
8. Pass => freeze and expand symbols before holdout.
