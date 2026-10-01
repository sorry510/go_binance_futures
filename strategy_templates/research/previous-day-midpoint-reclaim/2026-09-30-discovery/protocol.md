# Protocol

1. Freeze the strategy JSON before reading returns.
2. Midpoint = (prior completed daily High + Low) / 2.
3. LONG requires completed 1h bar to open at/below midpoint and close above it; current price must break that 1h high.
4. SHORT is symmetric.
5. CLOSE rules are exactly ROI >= 8 || ROI <= -6.
6. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Promotion gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
8. Failure => no close-weighted midpoint, no 25/75% range levels, no added filters or reversal.
9. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.
