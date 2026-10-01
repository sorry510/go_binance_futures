# Protocol

1. Freeze canonical ID121 regime/funding/breakout parameters.
2. Fresh breakout/breakdown occurs on completed bar [2].
3. Bar [1] must be inside bar [2]: High[1] <= High[2] and Low[1] >= Low[2].
4. Bar [1] must close on the accepted side of the original breakout boundary.
5. Current price must break the pause bar high/low.
6. Exact exits are ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => no 2-bar pause, no loose inside threshold, no parameter changes or extra filters.
9. Pass => freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
