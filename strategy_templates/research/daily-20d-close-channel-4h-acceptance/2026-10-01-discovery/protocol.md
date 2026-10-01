# Protocol

1. Freeze strategy before returns.
2. LONG level = max of prior 20 completed daily closes; SHORT level = min.
3. Completed 4h bar must cross from inside to a close outside the level.
4. Current price must break that 4h bar's high/low.
5. Exact close rule on both sides is ROI >= 8 || ROI <= -6.
6. Discovery uses SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only.
7. Failure => no lookback scan, no high/low-channel substitution, no added filters.
8. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
