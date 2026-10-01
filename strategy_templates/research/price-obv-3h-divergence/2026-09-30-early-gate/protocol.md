# Protocol

1. Freeze strategy JSON before reading returns.
2. LONG divergence: Close[1] < Close[3] and OBV[1] > OBV[3].
3. SHORT divergence is symmetric.
4. Current price must break the latest completed 1h high/low in the reversal direction.
5. No EMA/ADX/RSI or extra volume threshold.
6. Exact fixed exits: ROI >= 8 || ROI <= -6.
7. Core-4 early gate only.
8. Failure => no 2h/4h/6h scan, no trend/filter additions, no sign reversal.
9. Pass => freeze parameters and expand symbols before holdout.
