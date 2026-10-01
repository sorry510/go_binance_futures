# Protocol

1. Freeze strategy JSON and time windows before reading returns.
2. QVWC20 is sum(Close * Amount) / sum(Amount), where Amount is QuoteAssetVolume, over completed 1h bars.
3. LONG requires Close[1] above its QVWC20 after Close[2] was at/below its own prior QVWC20. SHORT is symmetric.
4. Current price must break the trigger bar's high/low.
5. Exact exits are ROI >= 8 || ROI <= -6.
6. Discovery is 2023–2024 only on SOL/DOGE/LTC/AVAX/UNI/ZEC.
7. OOS1 2025 stays unread unless discovery passes; OOS2 2026 stays unread unless OOS1 passes.
8. Failure means no period scan, no true-VWAP substitution, no added filters, and no direction reversal.
