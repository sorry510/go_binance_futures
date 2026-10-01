# Protocol

1. Freeze strategy JSON and time windows before reading returns.
2. LONG trigger: completed 1h EMA20 crosses above EMA50; SHORT trigger is symmetric.
3. Entry requires current price to break the crossover bar's high/low.
4. Exact exits are ROI >= 8 || ROI <= -6.
5. Discovery is 2023–2024 only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
6. OOS1 2025 remains unread unless discovery passes.
7. OOS2 2026 remains unread unless OOS1 passes.
8. Failure => no EMA period scan, no added filters, no direction reversal.
