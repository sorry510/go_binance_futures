# v112 Daily First-4h Opening Range Breakout — 2026-10-01 Discovery

Hypothesis: the first completed 4h range of each UTC day is a natural opening range. Acceptance outside that range later in the same day may represent intraday price discovery strong enough to match the fixed TP8/SL6 path.

The strategy identifies the first 4h bar without NowTime modulo: its Open equals the current daily Open. The first 4h bar must already be completed, so it is searched only in kline_4h[1:6]. A completed 1h candle must cross from inside to a close outside the opening-range high/low, and current price must then break the trigger-hour extreme.

Discovery is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Exact exits are ROI >= 8 || ROI <= -6. Fixed leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 8,807 trades, PF 0.799160, 0/6 symbols positive, 7.673513 trades/symbol/week. Yearly PFs were 0.809173, 0.770946, 0.841836, and 0.762457. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without opening-range duration variants, added filters, or reversal.
