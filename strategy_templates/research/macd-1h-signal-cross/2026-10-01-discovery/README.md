# v124 1h MACD Signal Cross — 2026-10-01 Discovery

Standard MACD(12,26,9) on completed 1h bars. LONG when DIF[1] crosses above DEA[1] from DIF[2] <= DEA[2]; SHORT is symmetric. Entry additionally requires current price to break the trigger-hour high/low.

Exact exits: ROI >= 8 || ROI <= -6. Discovery universe: SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT. Discovery window: 2023-01-01 through 2025-01-01.

Promotion gate: PF >= 1.15, >= 4/6 symbols positive, frequency >= 0.30 trades/symbol/week, and no clear year instability. Failure means no MACD period scan, no extra filters, and 2025/2026 OOS remain unread.

## Final result

Discovery failed decisively: 4362 trades, PF 0.797092, 0/6 symbols positive, 6.961696 trades/symbol/week. 2023 PF 0.823876; 2024 PF 0.777894. 2025 OOS1 and 2026 OOS2 were not evaluated. Freeze without MACD period tuning or added filters.
