# Protocol

Frozen before returns:
- Entry rules copied unchanged from `hourly-trend-strength-breakout-v2.json`: 4h Supertrend(10,3), 1h ADX14 threshold/slope/DMI, 1h Donchian20 breakout, 1h ATR14 candle-body bounds, live-hold constraint.
- BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01.
- CLOSE_LONG/CLOSE_SHORT are `false`; exits standardized to Engine leverage4, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.
- No MarketCondition, Benchmark, NowTime modulo.
- Gate: PF>=1.15, >=3/4 positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- Failure => freeze entry family; no ADX/DI/ATR/Donchian/Supertrend tuning.
