# v101 ADX20 × ATR-Term Expansion Confluence — 2026-09-30 Discovery

Hypothesis: an hourly ADX14 transition from <=20 to >20 is more likely to represent a real trend start when short-horizon volatility is already expanded relative to the daily volatility regime. Volatility state is ATR14(1h)*sqrt(24)/ATR14(1d) > 1. DI dominance selects direction; current price must break the trigger-hour extreme.

This is a preregistered confluence of two project-native mechanisms previously tested separately. No thresholds were tuned from their outcomes: ADX=20 and ATR term ratio=1 are the natural fixed definitions already used. Exact exits are `ROI >= 8 || ROI <= -6`.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Core BTC/ETH/BNB/XRP results from the component studies are not used as the discovery result for this combination.

Discovery promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, with no obvious multi-year collapse. If it passes, parameters remain frozen and the next step is fresh symbol holdout on ALGO/INJ/LDO/PENDLE/PYTH with conservative >=2y local-history eligibility starts.

## Final result

Discovery failed decisively under strict fixed TP8/SL6: 485 trades, normalized PF 0.763035, 0/6 symbols positive, frequency 0.422579 trades/symbol/week; yearly PFs 0.689085 / 0.679165 / 0.933009 / 0.750535. Freeze without threshold/period/filter changes. Reserved fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated and remains untouched.
