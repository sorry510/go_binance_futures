# Protocol

1. Freeze JSON before returns.
2. LONG sweep: daily Low[1] below min Low[2:22] and daily Close[1] back above that prior low.
3. SHORT sweep is symmetric at max High[2:22].
4. A completed 4h candle must cross the sweep-day midpoint back toward the old range.
5. Current price must break the 4h signal candle high/low.
6. Exact exits: ROI >= 8 || ROI <= -6.
7. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => no 10d/30d scan, no wick/body threshold, no trend/volume/funding filter, no reversal.
9. Pass => freeze and evaluate preregistered fresh holdout.
