# Protocol

1. Freeze the 60-minute record definition, direction, universe, discovery window, and promotion gate before returns.
2. Use only project-native 1m Futures Kline QPS/OHLC. No MarketCondition, Benchmark, funding, taker flow, or NowTime modulo.
3. A burst occurs when current completed 1m QPS is strictly greater than every one of the previous 60 completed 1m QPS values.
4. LONG if the burst minute closes above its open; SHORT if it closes below its open.
5. Do not add price-breakout, volatility, trend, taker, or QPS-multiple filters.
6. EMA(64) exists only to request sufficient 1m history from DatasetBuilder; strategy rules do not reference it.
7. CLOSE_LONG and CLOSE_SHORT are exactly: ROI >= 8 || ROI <= -6.
8. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
9. Promotion gate: PF >=1.15, >=4/6 positive symbols, >=0.30 trades/symbol/week, and both 2023 and 2024 PF >1.
10. Failure freezes this family: no 30m/120m record scan, no QPS multiplier scan, no reversal, and OOS remains unread.
