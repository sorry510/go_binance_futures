# OOS1 Protocol — frozen before 2025 outcomes

1. Reuse v154 discovery rule exactly. No threshold, symbol, side, horizon or timing change.
2. OOS1 is calendar 2025 only on the same six symbols.
3. Only adjacent normal 8h funding settlements separated by 7.5h-8.5h are eligible.
4. Previous funding must equal the exchange flat state 0.0001 exactly at stored precision; current funding must leave it. >0.0001 -> SHORT; <0.0001 -> LONG.
5. Entry is the next complete 1h open. Measure signed 1h/4h/12h; 12h endpoint must remain in calendar 2025.
6. Binance's 2025-09 funding formula update does not alter the 0.01% standard 8h interest state; this test also rejects non-8h settlements.
7. OOS1 pass gate is intentionally identical to discovery: 12h mean >=+0.20%, >=4/6 symbol means positive, and >=0.30 events/symbol/week.
8. Failure freezes v154 without reading 2026. Passing permits frozen OOS2 on 2026.
9. Do not delete a side, add funding magnitude/premium/OI filters, change equality tolerance, or change the entry/horizon after seeing OOS1.
