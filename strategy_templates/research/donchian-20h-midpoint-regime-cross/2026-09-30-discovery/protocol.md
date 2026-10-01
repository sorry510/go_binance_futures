# Protocol

1. Freeze strategy JSON before returns.
2. Use only completed hourly bars [2:22] to define the prior-20h range midpoint.
3. LONG requires trigger bar [1] to cross midpoint from below; SHORT mirrors from above.
4. Current price must break the trigger-hour high/low.
5. Exact exit rules are ROI >= 8 || ROI <= -6.
6. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Failure means no 10h/40h lookback scan, no trend/volume/funding/taker filter, and no reversal.
8. Pass means freeze parameters and evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH.
