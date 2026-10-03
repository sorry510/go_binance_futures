# v114 ID121-Regime Post-Breakout Inside Continuation — 2026-10-01 Discovery

Objective: test a second entry path inside the canonical ID121 quality regime. A canonical fresh 1h breakout/breakdown must occur, the immediately following completed 1h bar must be fully inside the breakout bar while closing on the accepted side of the original breakout boundary, and current price must then break the pause-bar extreme.

All ID121 regime, funding, RSI, ADX, ATR, breakout-window and impulse parameters are preserved. Exact exits are `ROI >= 8 || ROI <= -6`; leverage 4, fee 0.0005/side, slippage 5bps/side, single position.

Promotion gate was frozen before returns: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH stays unread unless discovery passes.

## Final result

- **46 trades**, LONG 24 / SHORT 22.
- normalized PF **1.2051**.
- normalized net **+8.4858%**.
- breadth **3/6 positive symbols**.
- frequency **0.0401 trades/symbol/week**.
- 23 TP / 23 SL.
- 2023: 5 trades, PF **0.6877**.
- 2024: 13 trades, PF **1.0296**.
- 2025: 21 trades, PF **1.1686**.
- 2026 through Aug: 7 trades, PF **2.9022**.

Per-symbol: SOL PF1.030 (4), DOGE 0.544 (3), LTC 3.104 (7), AVAX 0.993 (11), UNI no-loss 3-trade sample, ZEC 0.826 (18).

The aggregate PF passes, but breadth is only 3/6, frequency misses the 0.30 gate by a wide margin, and 2023 is clearly negative.

Decision: **freeze v114**. Do not add a 2-bar pause, loosen the inside-bar definition, modify ID121 parameters, or add filters. Fresh-symbol holdout remains unread.
