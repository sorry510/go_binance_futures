# Protocol

1. Copy v67 LONG/SHORT entry logic unchanged.
2. Replace both v67 conditional CLOSE rules with exact fixed exits: ROI >= 8 || ROI <= -6.
3. Do not tune 0.5 TakerBuyRatio center, QPS baseline, EMA periods, ADX threshold, or confirmation.
4. Discovery only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
5. Promotion gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear multi-year instability.
6. If discovery fails, freeze this strict audit and do not inspect fresh holdout.
7. If discovery passes, evaluate fresh holdout ALGO/INJ/LDO/PENDLE/PYTH with parameters frozen.
