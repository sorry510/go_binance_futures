# v191 / v193 Full-Cycle Audit — 2023 through 2026-09

The archived 2023-2024 event sets were preserved exactly. The original signal definitions were then extended without tuning to 2025-01-01 through 2026-09-30 using complete Binance Vision USD-M monthly 1m and 1h Klines.

Each later-year symbol has **963,360 one-minute rows** including the December-2024 warmup month; the actual extension window itself is complete through 2026-09-30.

## v191 — Intrahour 1m Realized-Skew Reversal

- 95,534 full-cycle events.
- Frequency: 81.414 events/symbol/week.
- Full-cycle 12h signed mean: **-0.00359%**.
- Positive symbols: **1/6**.
- Positive years: **2/4**.
- Yearly 12h: 2023 +0.00489%, 2024 -0.00334%, 2025 -0.01721%, 2026 +0.00344%.
- LONG +0.03290%, SHORT -0.04007% is attribution only; deleting a side is prohibited.

Decision: no edge; frozen.

## v193 — Intrahour Price-Staleness Activation Reversal

- 16,152 full-cycle events.
- Frequency: 13.765 events/symbol/week.
- Full-cycle 12h signed mean: **+0.02188%**.
- Positive symbols: **3/6**.
- Positive years: **2/4**.
- Yearly 12h: 2023 -0.05653%, 2024 +0.06973%, 2025 +0.06320%, 2026 **-0.11974%**.
- LONG +0.08908%, SHORT -0.04334% is post-hoc attribution only.

The large 2026 reversal confirms that the apparent 2024-2025 effect is not cycle-stable. Its magnitude is also far below the +0.20% raw economic gate.

Decision: frozen; no threshold changes to zero-count, no timeframe variants, no side deletion, no strict TP8/SL6 promotion, and no DB write.
