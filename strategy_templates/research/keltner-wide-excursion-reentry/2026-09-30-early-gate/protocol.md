# Protocol

Frozen before returns:
- Core-4 BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01.
- 4h KC narrow=(50,2.75), wide=(50,3.75), fixed from legacy line7.
- Wide excursion must appear in prior 10 completed 4h bars.
- Latest completed 4h close outside narrow KC; current price re-enters narrow KC.
- LONG requires previous completed daily close > prior daily close; SHORT exact mirror.
- No 12h data, no MarketCondition/Benchmark/NowTime modulo, no additional RSI/ADX/volume filters.
- Close rules disabled; standard Engine 4x TP8/SL6, fee0.0005/side, slippage5bps/side, single position determines exit.
- Gate: PF>=1.15, >=3/4 symbols positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- Failure => freeze; no KC multiplier/period/window tuning and no Bollinger substitution.
