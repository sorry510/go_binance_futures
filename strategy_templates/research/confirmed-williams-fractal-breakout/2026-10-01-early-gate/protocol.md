# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and 2023-2024 discovery before reading returns; 2025/2026 remain unread.
2. Use only completed project-native USD-M 1h OHLC. No MarketCondition, Benchmark, volume, taker, funding, external data, or NowTime modulo.
3. Use the standard 5-bar Williams fractal. A high fractal centered at j requires High[j] strictly greater than High[j-2], High[j-1], High[j+1], High[j+2]; low fractal is symmetric.
4. A fractal is eligible only after both right-side bars are completed, so there is no future leakage.
5. At each completed 1h bar, search only the trailing 24 completed hours and choose the most recent confirmed high fractal and most recent confirmed low fractal independently.
6. LONG if the previous completed 1h close was <= the selected fractal high and the current completed 1h close is > it. SHORT is the symmetric fresh close break below the selected fractal low.
7. If both sides would trigger on one bar, skip the bar. Entry is next complete 1h open.
8. No breakout-distance threshold, trend/volume filter, retest, fractal-strength score, or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 12h/48h horizon, 3-bar/7-bar fractals, add filters, reverse it, or change level selection.
11. Only if promoted may 2025/2026 be read and a strict Engine version be generated with leverage=4, TP8/SL6, fee=0.0005/side, slippage=5bps/side, single-position and exact exits `ROI >= 8 || ROI <= -6`.
