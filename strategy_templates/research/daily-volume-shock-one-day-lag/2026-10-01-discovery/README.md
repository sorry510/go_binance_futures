# v117 Daily Volume Shock One-Day Lag — 2026-10-01 Discovery

Hypothesis: the return effect of an unusual daily participation shock may be delayed rather than immediate. A fresh daily quote-volume expansion is identified two completed days ago. The direction is the shock day's candle sign. After one full intervening day, the current day can enter only when a completed 4h candle aligns with the shock direction and current price breaks that 4h extreme.

This differs from v111, which traded the immediate day after the completed expansion signal. This study tests exactly one full-day lag only. It does not scan 2d/3d lags or add EMA/ADX/Taker/Funding filters.

All inputs are project-native. Amount is QuoteAssetVolume. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01.

Promotion gate: normalized PF >= 1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. If discovery fails, freeze the family without testing other lags or filters. No holdout is read unless discovery passes.

## Final result

Discovery failed: 2490 trades, PF 0.817541, 0/6 symbols positive, 2.169529 trades/symbol/week. Yearly PFs were 0.690112, 0.856469, 0.781452, and 0.971257. No holdout was evaluated. The one-day-lag volume-shock continuation family is frozen; no 2d/3d lag scan, alternate baseline, filter, or reversal is permitted.

## Final result

Discovery failed: 2490 trades, PF 0.817541, 0/6 symbols positive, 2.169529 trades/symbol/week. Yearly PFs were 0.690112, 0.856469, 0.781452, and 0.971257. No holdout was evaluated. The one-day-lag volume-shock continuation family is frozen; no 2d/3d lag scan, alternate baseline, filter, or reversal is permitted.
