# Protocol

1. Freeze strategy JSON before reading returns.
2. Pressure = sum over four completed 1h bars of hourly return × quote volume.
3. LONG when pressure crosses from <=0 to >0; SHORT is symmetric.
4. Current price must break the latest completed 1h high/low.
5. No threshold multiplier or additional EMA/ADX/taker/funding filter.
6. Exact fixed exits: ROI >= 8 || ROI <= -6.
7. Core-4 early gate only.
8. Failure => no 2h/6h/8h window scan, no added filters, no reversal.
9. Pass => freeze and expand symbols before holdout.
