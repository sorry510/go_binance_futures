# Protocol

1. Freeze strategy before reading returns.
2. LONG requires FundingRate.Data[0] > 0 and Data[1] <= 0.
3. SHORT requires FundingRate.Data[0] < 0 and Data[1] >= 0.
4. Latest settled funding age must be >=1h and <2h so kline_1h[1] is the first complete post-settlement hour.
5. That completed 1h candle must align with the new funding sign; current price must break its high/low.
6. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
7. Discovery uses SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.
8. Failure => no funding magnitude threshold, no opposite/crowding direction, no different event-age window.
9. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
