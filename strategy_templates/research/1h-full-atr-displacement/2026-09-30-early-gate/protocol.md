# Protocol

1. Freeze strategy before returns.
2. Signal candle is the previous completed 1h bar.
3. Pre-signal volatility = ATR14[2].
4. LONG requires Close[1]-Open[1] >= ATR14[2]; SHORT is symmetric.
5. Current price must break the signal-hour high/low.
6. Strict fixed exits: `ROI >= 8 || ROI <= -6`.
7. Core-4 early gate only.
8. Failure => no 0.75/1.25/1.5 ATR scan, no range-vs-body substitution, no trend/volume/funding filter, no reversal.
9. Pass => freeze and expand symbols before holdout.
