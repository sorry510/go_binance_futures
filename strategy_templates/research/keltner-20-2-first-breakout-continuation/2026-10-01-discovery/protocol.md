# Protocol

1. Freeze strategy JSON before reading returns.
2. LONG: completed bar [2] is not above Keltner upper and bar [1] closes above upper. SHORT mirrors lower band.
3. Current price must break trigger-hour high/low.
4. Exact exits are ROI >= 8 || ROI <= -6.
5. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
6. Failure => no multiplier scan, no trend/volume/funding/taker filter, no reversal.
7. Pass => freeze and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
