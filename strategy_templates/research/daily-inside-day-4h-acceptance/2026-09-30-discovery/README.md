# v102 Daily Inside-Day 4h Acceptance — 2026-09-30 Discovery

Hypothesis: a completed daily inside day represents higher-timeframe range contraction. A completed 4h candle that crosses from inside that daily range to a close beyond the inside-day high/low represents directional acceptance; current price must then break the 4h signal candle extreme before entry.

Frozen before returns. No EMA trend, ADX, volume, funding, or body-size filter is used. Daily inside definition is strict High[1] < High[2] and Low[1] > Low[2]. Both close rules are exact fixed exits: `ROI >= 8 || ROI <= -6`.

Discovery universe is frozen to SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT from 2023-01-01 through 2026-09-01. Promotion gate: PF>=1.15, >=4/6 symbols positive, frequency>=0.30 trades/symbol/week, no clear multi-year collapse.

If discovery passes, parameters remain frozen and fresh holdout ALGO/INJ/LDO/PENDLE/PYTH is evaluated with the preregistered conservative >=2y starts. Failure leaves holdout untouched.

## Final result

Discovery failed: 975 trades, normalized PF 0.858241, 0/6 discovery symbols positive, frequency 0.849515 trades/symbol/week. Yearly PF 0.987893 / 0.840887 / 0.838170 / 0.720845. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remained untouched. Freeze without changing the inside-day or 4h-acceptance definition.
