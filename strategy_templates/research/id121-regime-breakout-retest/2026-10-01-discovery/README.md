# v113 ID121-Regime Breakout Retest — 2026-10-01 Discovery

Objective: test an orthogonal entry path inside the quality regime that still supports canonical ID121. Instead of entering the fresh 1h breakout immediately, require the next completed hour to retest the original breakout/breakdown boundary and close on the accepted side, then require current price to continue beyond the retest-hour extreme.

The long side preserves ID121's funding-supportive, daily directional, 4h trend/ADX, RSI55, fresh 12h breakout + 0.10 ATR buffer, and impulse-body requirements. The short side preserves ID121's funding non-negative, full bearish daily regime, 4h trend/ADX, RSI45, fresh 12h breakdown, and impulse-body requirements. No breakout, funding, RSI, ADX, ATR, or EMA parameter is changed. The only new mechanism is one-hour boundary retest-and-hold before entry.

Discovery is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Exact exits are ROI >= 8 || ROI <= -6; leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery produced 97 trades, PF 1.058233, 3/6 positive and only 0.084516 trades/symbol/week. SOL/DOGE/LTC were positive, but AVAX/UNI/ZEC failed; 2023/2024 were also below 1. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. Freeze without retest-tolerance or delay tuning.
