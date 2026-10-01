# Protocol

1. Freeze strategy JSON and time windows before reading returns.
2. Boundary is the max/min of the prior 20 completed daily bars.
3. A completed 4h bar must freshly close across that boundary relative to the prior 4h close.
4. Current price must then break the trigger 4h high/low.
5. Exact exits are ROI >= 8 || ROI <= -6.
6. Discovery is 2023–2024 only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. OOS1 2025 stays unread unless discovery passes.
8. OOS2 2026 stays unread unless OOS1 passes.
9. Failure => no 10d/55d lookback scan, no filters, no direction reversal.
