# Mandatory 2023–2026-09 Cycle Audit Backfill — v179/v180/v181/v182/v183/v184/v185/v186/v187/v188

## Frozen protocol before later-year returns

This bundle applies the 2026-10-03 global cycle-audit rule to ten already-defined mechanisms that were originally evaluated only on 2023-2024.

Period:
- start inclusive: 2023-01-01T00:00:00Z
- end exclusive: 2026-10-01T00:00:00Z
- report 2023 / 2024 / 2025 / 2026 Jan-Sep separately and the full aggregate.

Frozen universe:
- SOLUSDT
- DOGEUSDT
- LTCUSDT
- AVAXUSDT
- UNIUSDT
- ZECUSDT

Frozen mechanisms:
- v179 Return-Energy Concentration Regime
- v180 24h Range-Occupancy Regime Cross
- v181 Activity–Volatility Coupling Regime
- v182 Per-Trade Volatility-Impact Regime
- v183 24h Price-Monotonicity Regime Cross
- v184 Notional-vs-Trade-Arrival Concentration Regime
- v185 Participation–Ticket-Size Coupling Regime
- v186 Rolling-4h Extreme-Order Continuation
- v187 Taker-vs-Global Account Skew
- v188 Corwin–Schultz Liquidity-Stress Reversal

No signal direction, threshold, window, universe rule, entry timing, or forward-return endpoint is changed from the archived 2023-2024 replay. Entry remains next 1h open; diagnostics are signed 1h/4h/12h forward log returns.

Canonical parity rule:
1. Preserve each archived 2023-2024 event file unchanged as the canonical discovery slice.
2. Copy the original replay implementation and alter only its later-year date/source ranges.
3. Run 2025-01-01 through 2026-09-30 from Binance Vision; v179-v186/v188 use USD-M 1h Klines, while v187 additionally uses the original 5m Futures metrics source.
4. Concatenate archived 2023-2024 events with the newly evaluated 2025-2026 events for the canonical full-cycle summary.
5. Do not retune after reading 2025/2026.

This is a diagnostic cycle audit, not a strict TP8/SL6 promotion by itself. No DB write, app.conf change, commit, or push is authorized.
