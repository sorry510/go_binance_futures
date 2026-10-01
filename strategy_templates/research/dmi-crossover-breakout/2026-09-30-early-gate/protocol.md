# Protocol

1. Freeze strategy JSON before reading returns.
2. LONG trigger: completed 1h +DI crosses above -DI; SHORT is symmetric.
3. Entry requires current price to break the completed trigger-hour high/low.
4. No ADX threshold, EMA, volume/QPS, funding, or extra filters.
5. Exact fixed exits: ROI >= 8 || ROI <= -6.
6. Core-4 early gate only.
7. Failure => no DI period/threshold scan, no ADX filter, no trend/volume filter, no reversal.
8. Pass => freeze parameters and expand symbols before holdout.
