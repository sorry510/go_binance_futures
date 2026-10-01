# Protocol

1. Freeze all canonical ID121 regime and breakout parameters before returns.
2. Breakout/breakdown signal occurs on completed 1h bar [2].
3. The immediately following completed bar [1] must touch the original breakout boundary and close on the accepted side.
4. Current price must continue beyond the retest bar high/low.
5. No retest tolerance, alternate delay, EMA pullback, or threshold scan is allowed.
6. Exact exits are ROI >= 8 || ROI <= -6.
7. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
8. Failure => freeze this breakout-retest mechanism; do not tune delay or retest tolerance.
9. Pass => freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
