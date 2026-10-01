# v108 Funding-Price 8h Divergence Transition — 2026-09-30 Discovery

Hypothesis: when the funding-sign crowd and price direction disagree, a zero-cross in the opposite 8h price return can mark the crowded side losing control. The 8h horizon is fixed by the standard funding interval, not selected from returns.

LONG: latest known funding <0 and completed 8h return crosses from <=0 to >0; current price then breaks the trigger 4h high. SHORT is symmetric for funding >0 and 8h return crossing below zero. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Fixed leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, with no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH stays unread unless discovery passes.

## Final result

Discovery failed decisively: 6671 trades, PF 0.815039, 0/6 symbols positive, 5.812422 trades/symbol/week; yearly PF 0.821633 / 0.800275 / 0.863991 / 0.744108. Fresh holdout was not evaluated. Freeze without changing the 8h horizon, funding magnitude threshold, added filters, or reversal.
