# Binance Margin Additions Direct-Long — 2024 Discovery Audit

This bundle preserves the evidence behind the `2026-09-26 — Binance Margin Additions Direct-Long：2024 Discovery` entry in `strategy_templates/result.md`.

The preregistered hypothesis was that a Binance Margin announcement adding a borrowable asset or new Cross/Isolated Margin pair could create incremental leveraged demand, so the event direction was LONG. The 2024 discovery set failed decisively and the family is frozen; 2025 was not opened as OOS.

Canonical result:

- 21 official Margin addition announcements.
- 168 token-level events / 121 unique bases.
- 52 eligible events / 34 bases after the production eligibility rules.
- 52 trades, 15 wins / 37 losses.
- 15 TP / 36 SL / 1 TIME.
- PF 0.5284822667154679.
- normalized net -116.44202292291679%.
- funding -2.1101966311707705%; fees 20.75321069220023%.
- single-position overlap skips: 0.

Canonical evidence is `inputs/events.json`, `inputs/eligibility.json`, `results/trades.json`, `results/summary.json`, and `replay.py`. `results/summary.json` can be regenerated from the per-trade results and has been cross-checked at archive time.

Two audit corrections are material:

1. The entry boundary is the next complete minute: `((publish_ms // 60000) + 1) * 60000`. The final 52 trades all satisfy this exactly.
2. Binance CMS table structure must preserve cell/row boundaries. The corrected parser avoids concatenating adjacent table cells into fake pairs and raised the event universe from the superseded 149 events / 47 eligible sample to the final 168 / 52 sample.

`legacy/superseded_run.log` is intentionally retained because it documents the pre-parser-fix 149/47 run. It is not final evidence and must not be used for metrics.

Status: **frozen / no 2025 OOS / no DB import**. Do not reverse the failed result into a SHORT thesis or post-hoc filter by symbol, month, quote asset, event subtype, first-minute reaction, or liquidity.
