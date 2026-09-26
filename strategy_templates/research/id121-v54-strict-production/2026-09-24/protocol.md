# Protocol

This comparison uses the production universe rule already fixed before candidate evaluation: a USD-M perpetual is eligible only after at least two years of contract history.

For each of the 15 symbols, ID121 and v54 must start a fresh Engine run from the **same** production-eligible timestamp and use the same dataset and execution configuration. A historical ID121 run started earlier must not be truncated and compared with a newly started v54 run because the Engine is path dependent through equity and position sizing.

The strict production window is:

- Twelve mature symbols: 2024-08-15 through 2026-09-01.
- 1000PEPEUSDT: 2025-05-05 through 2026-09-01.
- SUIUSDT: 2025-05-03 through 2026-09-01.
- ONDOUSDT: 2026-01-20 through 2026-09-01.

Execution is fixed at 4x leverage, TP 8, SL 6, fee 0.0005 per side, slippage 5 bps per side, and single-position semantics. No MarketCondition, Benchmark, `NowTime % ...`, symbol-specific tuning, or post-hoc eligibility changes are introduced by this comparison.

Two metric families are intentionally kept separate:

1. **Raw-dollar Engine metrics** use the Engine's realized `NetPnL` and preserve the equity/position-size path. These are the canonical production comparison and paired-attribution numbers recorded in `strategy_templates/result.md`.
2. **Fixed-notional normalized metrics** use `NetPnL / (EntryPrice * Quantity)`. `results/paired_normalized.csv` stores only this normalized return and cannot reconstruct raw-dollar attribution.

The v54 decision is frozen after the strict comparison: it is a frequency candidate alongside ID121, not a replacement and not an independent strong alpha. Nearby parameter search is prohibited.
