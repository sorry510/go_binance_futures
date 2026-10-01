# Protocol

1. Freeze strategy before returns.
2. Current rolling 24h return is project runtime `NowSymbolPercentChange`, causally reconstructed in backtest.
3. Prior completed-hour 24h return approximation is Close[1] vs Close[25].
4. LONG: current >0 and prior <=0; SHORT symmetric.
5. Current price must break previous completed-hour high/low.
6. Strict fixed exits: `ROI >= 8 || ROI <= -6`.
7. Core-4 early gate only.
8. Failure => no +/-1%, +/-2%, alternate rolling-window, trend/volume/funding filter, or reversal.
9. Pass => freeze and expand symbols before holdout.
