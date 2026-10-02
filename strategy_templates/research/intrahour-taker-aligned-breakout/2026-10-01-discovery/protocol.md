# Protocol

1. Freeze strategy, discovery universe/window, and promotion gate before reading returns.
2. Use only project-native Futures Kline data; no MarketCondition, Benchmark, external data, or NowTime modulo.
3. Current partial 1h TakerBuyRatio is causal aggregation of completed 1m bars inside the still-forming hour.
4. LONG requires current partial 1h TakerBuyRatio > 0.5 and a fresh 1m close cross above previous completed 1h high.
5. SHORT requires current partial 1h TakerBuyRatio < 0.5 and a fresh 1m close cross below previous completed 1h low.
6. 0.5 is the natural buy/sell quote balance threshold; no threshold search is permitted.
7. No QPS, trend, ATR, RSI, ADX, funding, or symbol-specific filter.
8. CLOSE_LONG and CLOSE_SHORT are exactly: ROI >= 8 || ROI <= -6.
9. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
10. Promotion gate: PF >=1.15, >=4/6 positive symbols, >=0.30 trades/symbol/week, and both 2023/2024 PF >1.
11. Failure freezes this family: no 0.52/0.48 or 0.55/0.45 scan, no QPS/trend/funding filter, no reversal.
12. 2025 OOS1 and 2026 OOS2 remain unread unless the preceding stage passes.
