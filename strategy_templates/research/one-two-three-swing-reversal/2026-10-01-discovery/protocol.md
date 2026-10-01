# Protocol

1. Freeze v125 JSON before reading returns.
2. Use the exact 1h point-1 / point-2 / point-3 definitions in strategy.json.
3. Current price must break point-2 high/low before entry.
4. Both close rules are exactly ROI >= 8 || ROI <= -6.
5. Fixed 4x leverage, fee0.0005/side, slippage5bps/side, single position.
6. Discovery universe is SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-01-01 to 2025-01-01.
7. Failure => no pivot-width changes, no extra trend/volume filters, no reversal of direction.
8. Only a discovery pass may unlock 2025 OOS1 and then 2026 OOS2.
