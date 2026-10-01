# Protocol

1. Freeze entry/exit JSON and discovery universe before reading confluence returns.
2. Trend start: completed 1h ADX14 crosses from <=20 to >20.
3. Volatility expansion: ATR14(1h)*sqrt(24)/ATR14(1d) > 1 on the trigger hour.
4. Direction uses +DI/-DI dominance.
5. Entry requires current price to break the completed trigger-hour high/low.
6. Exact fixed exits: ROI >= 8 || ROI <= -6.
7. Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC only, 2023-01-01..2026-09-01.
8. Gate: PF>=1.15, >=4/6 positive, >=0.30 trades/symbol/week, no clear yearly collapse.
9. Failure => freeze; do not tune ADX level, ATR ratio, ATR/ADX periods, or add filters.
10. Pass => freeze parameters and run fresh holdout ALGO/INJ/LDO/PENDLE/PYTH with conservative >=2y local-history starts. Do not use holdout to tune.
