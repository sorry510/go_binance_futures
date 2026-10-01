# Protocol

1. Freeze strategy before returns.
2. Baseline = mean absolute body of completed bars [4:12].
3. Bars [3] and [2] must both have body < baseline.
4. Release bar [1] must have body > baseline; candle sign gives direction.
5. Current price must break release-bar high/low.
6. Exact close rules are ROI >= 8 || ROI <= -6.
7. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC in 2023-2024.
8. Failure => no multiplier/window/compression-length tuning and no added filters.
9. 2025 OOS1 and 2026 OOS2 remain unread unless prior stage passes.
