# Protocol

1. Freeze strategy JSON and all windows before reading returns.
2. Bullish engulfing: daily [2] bearish, daily [1] bullish, Open[1] <= Close[2], Close[1] >= Open[2].
3. Bearish engulfing is symmetric.
4. Entry requires completed 4h candle in reversal direction plus current price breaking its high/low.
5. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
6. Discovery is 2023–2024 only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. Only if discovery gate passes may 2025 OOS1 be read.
8. Only if OOS1 remains positive may 2026 OOS2 be read.
9. Failure means no body-ratio threshold, no wick filter, no 1h/8h confirmation change, no trend/volume/funding filter, no sign reversal.
