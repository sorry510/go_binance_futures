# Protocol

1. Freeze JSON before reading returns.
2. Daily outside day: High[1] > High[2] and Low[1] < Low[2].
3. LONG only if outside day is bullish; SHORT only if bearish.
4. Completed 4h candle must cross from inside to close beyond outside-day high/low.
5. Current price must break the 4h signal candle high/low.
6. Exact fixed exits: ROI >= 8 || ROI <= -6.
7. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => no outside-day body threshold, no EMA/volume/funding filter, no reversal.
9. Pass => freeze and evaluate the preregistered fresh holdout.
