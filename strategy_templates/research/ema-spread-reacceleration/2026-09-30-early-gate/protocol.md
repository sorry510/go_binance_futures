# Protocol

1. Freeze strategy JSON before reading returns.
2. Define spread = EMA20 - EMA50 on completed 1h bars.
3. LONG requires spread>0 and spread delta to turn from <=0 to >0; SHORT is symmetric.
4. Current price must break the trigger-hour high/low.
5. No RSI/ADX/volume/QPS/funding filters.
6. Exact fixed exits: ROI >= 8 || ROI <= -6.
7. Core-4 early gate only.
8. Failure => no EMA-period scan, no spread threshold/multiplier, no added filters, no reversal.
9. Pass => freeze and expand symbols before holdout.
