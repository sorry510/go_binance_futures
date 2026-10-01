# Protocol

1. Freeze entry and exit rules before reading returns.
2. Previous completed daily quote volume must cross from at-or-below its prior 20-day mean to above that mean.
3. Daily candle direction defines the candidate side.
4. Completed 4h candle must align with that side, and current price must break its extreme.
5. Both close rules use fixed ROI take-profit 8 and stop-loss -6.
6. Discovery symbols are SOL, DOGE, LTC, AVAX, UNI, and ZEC only.
7. On failure, do not change the 20-day baseline, add filters, reverse direction, or inspect reserved holdout symbols.
8. Only a passing discovery may proceed to the reserved holdout set.
