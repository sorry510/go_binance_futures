# Protocol

1. Freeze strategy before returns.
2. LONG: TakerBuyRatio[1] > max(TakerBuyRatio[2:10]) and >0.5.
3. SHORT: TakerBuyRatio[1] < min(TakerBuyRatio[2:10]) and <0.5.
4. Current price must break trigger-hour high/low.
5. Exact close rules are ROI >= 8 || ROI <= -6.
6. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024.
7. Failure => no lookback or neutral-level tuning, no extra trend/QPS/funding filters.
8. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
