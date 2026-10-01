# Protocol

1. Freeze strategy JSON before reading returns.
2. Compute prior-day classic pivot P, R1 and S1 from completed daily bar [1].
3. LONG requires completed 1h bar to open at/below R1 and close above R1; current price then breaks that 1h high.
4. SHORT is symmetric at S1.
5. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
6. Discovery uses only SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Promotion gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
8. Failure => no R2/S2, Camarilla/Fibonacci pivot, EMA/ADX/QPS/funding filter, or reversal.
9. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.
