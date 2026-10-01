# v108 4h Directional Excursion Imbalance — 2026-10-01 Discovery

Hypothesis: the balance between total upside excursion from each hourly open and total downside excursion from each hourly open can identify a directional micro-regime without relying on candle color, volume, taker flow, or external data.

For four completed 1h bars, define up = sum(High-Open) and down = sum(Open-Low). LONG triggers when up/down crosses from <=1 to >1; SHORT mirrors from >=1 to <1. Current price must then break the most recent completed 1h high/low.

Discovery is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Exact exits are ROI >= 8 || ROI <= -6. Fixed leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 15,268 trades, PF 0.811203, 0/6 symbols positive, 13.302962 trades/symbol/week. Yearly PFs were 0.788434, 0.784422, 0.860612, and 0.801883. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without window/threshold tuning, added filters, or reversal.
