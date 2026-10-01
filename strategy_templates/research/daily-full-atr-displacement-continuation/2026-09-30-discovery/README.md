# v107 Daily Full-ATR Displacement Continuation — 2026-09-30 Discovery

Hypothesis: a completed daily candle whose body is at least one pre-signal daily ATR14 represents a genuine regime displacement. Continuation is only entered after a completed 4h candle on the following day crosses from inside the impulse-day range to a close beyond the impulse-day high/low, followed by current price breaking the trigger 4h extreme.

ATR uses Data[2], so the signal day's own range does not inflate its threshold. This is project-native and materially different from the failed 1h full-ATR continuation family.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Exact exits are ROI >= 8 || ROI <= -6. Fixed leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=4/6 positive symbols, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH remains unread unless discovery passes.

## Final result

Discovery failed: 550 trades, PF 0.743470, 1/6 symbols positive, 0.479213 trades/symbol/week. Yearly PFs were 0.895204, 0.836659, 0.583453, and 0.690213. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. The family is frozen without ATR-multiplier tuning, added filters, or reversal.
