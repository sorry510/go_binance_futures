# Mandatory Cycle Audit Backfill — v190/v192/v194/v196/v197

Triggered by the 2026-10-03 global protocol change requiring every sufficiently covered mechanism to be evaluated over the full crypto-cycle window 2023-01-01 through 2026-09-30.

Rules:
1. Source signal definitions remain exactly frozen from their original 2023-2024 studies; no parameter, side, threshold, window or universe changes.
2. Data source for the backfill is Binance Vision USD-M monthly Kline archives. 1h is used for v190/v194 and forward returns; 1m is used for v192/v196.
3. Before accepting 2025-2026 results, the script must reproduce the archived 2023-2024 event keys exactly for every source family.
4. v197 keeps the original fixed >=2-of-4 same-hour same-side and zero-opposite-vote rule.
5. Report 2023, 2024, 2025 and 2026 Jan-Sep separately plus aggregate.
6. No retuning based on 2025/2026. These years are mandatory cycle diagnostics, not parameter-development data.
7. Signals whose 12h forward endpoint crosses a calendar-year boundary or the 2026-10-01 audit cutoff are excluded exactly to keep annual attribution causal and contained.
8. No DB writes.
