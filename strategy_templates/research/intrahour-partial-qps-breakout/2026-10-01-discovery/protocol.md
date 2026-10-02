# Protocol

1. Freeze strategy, universe, discovery window, and promotion gate before reading returns.
2. Use only project-native Futures Kline data. No MarketCondition, Benchmark, external data, or NowTime modulo.
3. Baseline QPS is the mean of the previous 8 completed 1h bars.
4. The current partial 1h QPS must already be >= 1.0x that completed-hour baseline.
5. LONG triggers only on a fresh 1m close cross above the previous completed 1h high.
6. SHORT is symmetric below the previous completed 1h low.
7. Entry evaluation is minute-close causal; the Engine executes its normal pending action semantics.
8. CLOSE_LONG and CLOSE_SHORT are exactly: ROI >= 8 || ROI <= -6.
9. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
10. Promotion gate: PF >=1.15, >=4/6 positive symbols, >=0.30 trades/symbol/week, and both 2023 and 2024 PF >1.
11. Failure freezes this family: no 0.8x/1.2x QPS scan, no 4h/12h baseline scan, no taker/trend filter, no reversal.
12. 2025 OOS1 and 2026 OOS2 remain unread unless the preceding stage passes.
