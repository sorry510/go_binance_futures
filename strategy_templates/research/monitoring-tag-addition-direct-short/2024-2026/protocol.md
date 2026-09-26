# Protocol

The discovery family is defined before outcome inspection as Binance official **Monitoring Tag additions** followed by a direct SHORT in the corresponding still-trading USD-M perpetual.

Production eligibility requires the event-month Binance Vision USD-M 1m archive, the corresponding month 24 months earlier, and a post-announcement tradable 1m window. Monitoring Tag removals and Seed Tag changes belong to separate event families and are excluded.

Execution is frozen at 4x leverage, TP 8, SL 6, fee 0.0005 per side, 5 bps slippage per side, maximum 72h hold, and single-position semantics. Entry is the next 1m open after the announcement. TP/SL is gated by minute close and executed at the following minute open, so event gaps can materially overshoot nominal SL6.

The validation sequence is chronological and one-way:

1. 2024 is discovery.
2. Only because 2024 discovery was positive was the unchanged rule applied to 2025 OOS.
3. Only because 2025 remained positive was the unchanged rule applied to 2026 second OOS.
4. The negative 2026 OOS froze the family. No symbol, event reason, year, first-minute reaction, TP/SL, hold time, or eligibility filter may be selected after seeing the result.

The archived replay helpers have a known funding parser bug and are preserved as historical evidence. Corrected-parser replay, if ever performed for audit accuracy, must keep the exact same event sets and frozen execution parameters and may only correct funding cashflow accounting; it must not reopen parameter search.
