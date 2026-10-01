# Protocol

1. Freeze strategy JSON before reading returns.
2. Daily impulse body must be >= pre-signal ATR14 Data[2].
3. LONG requires impulse day bullish; completed current-day 4h opens at/below impulse high and closes above it; current price then breaks the trigger 4h high.
4. SHORT is symmetric below impulse low.
5. Exact CLOSE rules: ROI >= 8 || ROI <= -6.
6. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Promotion gate PF>=1.15, >=4/6 positive, frequency>=0.30/symbol/week, no clear multi-year instability.
8. Failure => no 0.75/1.25 ATR multiplier, no trend/volume/funding filters, no fade reversal.
9. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.
