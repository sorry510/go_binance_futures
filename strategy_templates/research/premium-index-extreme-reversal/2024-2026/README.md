# Premium Index Extreme Reversal

## Hypothesis

The Binance Premium Index measures impact bid/ask pressure relative to the spot price index and feeds directly into funding-rate calculation. Extreme positive premium is treated as crowded long pressure and traded SHORT; extreme negative premium is traded LONG.

This is distinct from the previously frozen mark-vs-last dislocation and funding-rate families.

## Discovery — 2024

- 8,434 events.
- 1h signed mean: **+0.0031%**.
- 4h signed mean: **+0.0057%**.
- 12h signed mean: **+0.0756%**.
- Only **2/10** symbols had positive 4h mean.

## OOS1 — 2025

- 9,603 events.
- 1h signed mean: **+0.0033%**.
- 4h signed mean: **+0.0063%**.
- 12h signed mean: **+0.0898%**.
- 6/10 symbols had positive 4h mean.

## OOS2 — 2026

- 5,894 events.
- 1h signed mean: **+0.0375%**.
- 4h signed mean: **+0.0806%**.
- 12h signed mean: **+0.0489%**.
- 9/10 symbols had positive 4h mean.

## Decision

The 2024 discovery edge is economically negligible and fails cross-symbol consistency. 2025 remains near zero. Although 2026 is stronger, it is a later regime and still below the project's approximate round-trip fee+slippage requirement.

The 2026 improvement must not be used to retroactively tune the earlier weak years.

**Freeze the Premium Index extreme-reversal family.** Do not search 2σ/4σ, 12h/48h baselines, alternative re-arm levels, symbol-specific thresholds, or continuation direction. Do not enter exact 1m TP8/SL6 Engine validation.

Canonical evidence:
- results/summary.json
- results/events.csv
- replay.py
- export_prices.go
