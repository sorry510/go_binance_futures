# v106 Classic Daily Pivot R1/S1 Acceptance — 2026-09-30 Discovery

Hypothesis: classic prior-day pivot levels provide a fixed, widely used reference that is distinct from prior-day high/low breakouts. P=(H+L+C)/3, R1=2P-L, S1=2P-H, using the prior completed daily bar. A completed 1h candle must cross from the inside through R1/S1 and close beyond it; current price must then break that trigger hour's extreme.

This is project-native and uses only existing 1d/1h Kline data. No trend, volume, funding, taker-flow, volatility, or time-modulo filter is added. Exact exits are ROI >= 8 || ROI <= -6.

Discovery universe is frozen before returns: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT; 2023-01-01 through 2026-09-01. Fixed leverage4, fee0.0005/side, slippage5bps/side, single position.

Promotion gate: normalized PF >=1.15, >=4/6 symbols positive, frequency >=0.30 trades/symbol/week, and no clear multi-year instability. Only if promoted may the fresh holdout ALGO/INJ/LDO/PENDLE/PYTH be read.

## Final result

Discovery failed decisively: 4846 trades, PF 0.815244, 0/6 symbols positive, 4.222305 trades/symbol/week; yearly PF 0.819051 / 0.790347 / 0.819823 / 0.857128. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. Freeze without R2/S2, alternate pivot formulas, filters, or reversal.

## Final result

Discovery failed: 4846 trades, PF 0.815244, 0/6 symbols positive, 4.222305 trades/symbol/week. Yearly PFs were 0.819051, 0.790347, 0.819823, and 0.857128. Fresh holdout ALGO/INJ/LDO/PENDLE/PYTH was not evaluated. This family is frozen without alternate pivot formulas, extra filters, or reversal.
