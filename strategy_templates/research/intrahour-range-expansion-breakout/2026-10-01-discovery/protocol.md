# Protocol

1. Freeze strategy, discovery universe/window, and promotion gate before reading returns.
2. Use only project-native Futures Kline data; no MarketCondition, Benchmark, external data, or NowTime modulo.
3. Compare the current partial 1h high-low range with the immediately previous completed 1h high-low range.
4. Require current partial range > previous completed range; the 1.0 ratio is a natural equality boundary and is not scanned.
5. LONG requires a fresh 1m close cross above previous completed 1h high; SHORT mirrors below previous completed 1h low.
6. No QPS, taker, trend, ATR, funding, RSI, ADX, or symbol-specific filter.
7. CLOSE_LONG and CLOSE_SHORT are exactly: ROI >= 8 || ROI <= -6.
8. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
9. Promotion gate: PF >=1.15, >=4/6 positive symbols, >=0.30 trades/symbol/week, and both 2023/2024 PF >1.
10. Failure freezes this family: no 0.8x/1.2x range ratio, no multi-hour baseline, no filters, no reversal.
11. 2025 OOS1 and 2026 OOS2 remain unread unless the preceding stage passes.
