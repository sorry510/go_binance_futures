# Protocol

1. Freeze strategy JSON and time windows before reading returns.
2. Daily regime uses completed 1d ROC14 at [1].
3. Trigger is completed 4h ROC14 crossing zero into the same direction at [1].
4. Entry requires current price to break trigger 4h high/low.
5. Exact exits are ROI >= 8 || ROI <= -6.
6. Discovery is 2023–2024 only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. OOS1 2025 stays unread unless discovery passes.
8. OOS2 2026 stays unread unless OOS1 passes.
9. Failure => no ROC period scan, no extra trend/volume/funding filters, no sign reversal.
