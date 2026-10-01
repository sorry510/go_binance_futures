# Protocol

1. Freeze strategy before reading returns.
2. LONG requires Close[2] and Close[1] both above Bollinger(20,2) upper band. SHORT mirrors below lower band.
3. Current price must break the second outside bar's high/low.
4. Exact exits: ROI >= 8 || ROI <= -6.
5. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
6. Failure => no 1-bar/3-bar walk, no Bollinger parameter scan, no trend/volume/funding filter, no reversal.
7. Pass => freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
