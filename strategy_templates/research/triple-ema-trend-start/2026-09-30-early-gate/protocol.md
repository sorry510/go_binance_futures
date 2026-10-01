# Protocol

Frozen before returns:
- BTC/ETH/BNB/XRP, 2023-01-01 through 2026-09-01.
- 4h EMA periods 3/7/15, both hierarchy crosses within the recent four completed bars.
- EMA3 slope must continue in entry direction for three completed values.
- LONG RSI6<80 and RSI14<75; SHORT mirror RSI6>20 and RSI14>25.
- No MarketCondition, Benchmark, NowTime modulo, volume/funding filter.
- Close rules false; fixed Engine 4x TP8/SL6, fee0.0005/side, slippage5bps/side, single position.
- Gate PF>=1.15, >=3/4 positive, frequency>=0.3/symbol/week, no obvious full-year collapse.
- Failure => freeze; no EMA period/cross-window/RSI tuning and no extra filters.
