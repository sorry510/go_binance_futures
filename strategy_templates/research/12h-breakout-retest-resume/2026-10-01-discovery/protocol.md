# Protocol

1. Freeze strategy before reading returns.
2. LONG breakout bar [2] must close above the max High of bars [3:15], i.e. the prior 12 completed hours.
3. Retest bar [1] must touch or cross that breakout level with Low[1] <= level but close back above it.
4. Current price must then break retest-bar High[1]. SHORT is symmetric.
5. Exact exits are ROI >= 8 || ROI <= -6.
6. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Failure => no 6h/24h lookback scan, no multi-bar retest, no trend/volume/funding filter, no reversal.
8. Pass => freeze and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
