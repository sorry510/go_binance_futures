# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and 2023-2024 discovery before reading returns; 2025/2026 remain unread.
2. Use only completed project-native USD-M 1h High/Low/QuoteVolume. No MarketCondition, Benchmark, funding, taker, external data, or NowTime modulo.
3. Per-bar Quote-EOM = midpoint displacement from previous bar * current high-low range / current QuoteVolume. Bars with nonpositive QuoteVolume are invalid.
4. Smooth with a fixed 14-bar arithmetic mean, following the standard EOM period convention. No period search.
5. Mean EOM crossing from <=0 to >0 emits LONG; crossing from >=0 to <0 emits SHORT. Zero is the natural directional boundary.
6. Entry is next complete 1h open. Measure signed returns at 1h/4h/12h.
7. No magnitude threshold, trend filter, volume normalization, ATR, symbol-specific rule, or alternate direction.
8. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan EOM period, threshold, smoothing type, add filters, delete a direction, or reverse it.
10. Only if promoted may 2025/2026 be read and a strict Engine version be generated with leverage=4, TP8/SL6, fee=0.0005/side, slippage=5bps/side, single-position and exact exits `ROI >= 8 || ROI <= -6`.
