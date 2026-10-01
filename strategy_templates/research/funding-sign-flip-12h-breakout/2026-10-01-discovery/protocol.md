# Protocol

1. Freeze strategy JSON and all dates before reading returns.
2. FundingRate.Data[0] is the latest settled funding rate as of the evaluation point; Data[1] is the previous settled rate.
3. LONG funding event: Data[0] > 0 and Data[1] <= 0. SHORT funding event: Data[0] < 0 and Data[1] >= 0.
4. Price trigger uses the completed signal hour [1] crossing the high/low of completed hours [2:14].
5. Current price must then break the trigger-hour extreme.
6. CLOSE_LONG and CLOSE_SHORT are exactly ROI >= 8 || ROI <= -6.
7. Discovery only: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-01-01 to 2025-01-01.
8. If discovery fails, do not read 2025 OOS1 or 2026 OOS2; do not add funding thresholds, alter 12h lookback, reverse direction, or add filters.
9. If discovery passes, freeze parameters and evaluate 2025 OOS1; only an OOS1 pass unlocks 2026 OOS2.
