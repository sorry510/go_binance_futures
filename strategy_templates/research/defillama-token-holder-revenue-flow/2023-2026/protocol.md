# Protocol — DeFiLlama Token-Holder Revenue Flow

1. Source universe is frozen before returns using DeFiLlama fee adapters whose methodology exposes direct `HoldersRevenue`, mapped to Binance USD-M token identity. Ticker collisions and adapters that are not actually token-holder revenue are excluded before any return inspection.
2. Signal uses direct token-holder value accrual only: buybacks/burns, revenue distributions, or fees paid to token stakers/locked governance holders. Protocol treasury revenue and supply-side LP revenue are not substitutes.
3. Daily flow = log(sum holder revenue over the latest 7 completed UTC days / sum over the preceding 7 completed UTC days). Both 7-day sums must be positive and all 14 calendar dates must be present in the aggregated source history.
4. Zero-cross upward => LONG; zero-cross downward => SHORT. Entry is the next UTC daily open.
5. Dynamic eligibility at signal: actual Binance USD-M history >=2 years and completed signal-day QuoteVolume >=5m USDT.
6. Discovery signal dates are in 2023-01-01 through 2024-12-31, and an event is included only if its full 7d endpoint is no later than 2024-12-31. Endpoint diagnostics are signed 1d/3d/7d log returns from the frozen next-day open.
7. Discovery gate frozen before returns: >=80 eligible events, >=8 triggering symbols, aggregate 7d mean >=+0.25%, >=60% triggering symbols with positive mean7, and aggregate 2023 and 2024 mean7 both >0.
8. Only a discovery pass permits untouched OOS. OOS uses the exact same frozen source map and signal; no symbol additions/deletions, no adapter substitutions, no window/threshold/direction changes.
9. OOS gate frozen before reading OOS returns: because the superseded initial discovery run had indirectly inspected aggregate outcomes through 2025-01-07 before the boundary leak was detected, truly untouched OOS starts 2025-01-08; combined 2025-01-08 through 2026-08-24 must have >=80 events, >=8 triggering symbols, aggregate 7d mean >=+0.25%, breadth >=60%, and 2025 and 2026 mean7 both >0.
10. Exact 1m TP8/SL6 with leverage=4, fee=0.0005/side, slippage=5bps/side, funding and single-position semantics is allowed only if OOS passes. No post-hoc reversal or subgroup rescue.
