# v63 daily exit routing ablation

This is a development study on BTCUSDT, ETHUSDT, SOLUSDT, and XRPUSDT. It is specified while v62a's frozen 8-coin holdout is still acquiring data and before any holdout strategy result is available.

The v62a development replay has positive net four-year totals and frequencies above 0.9/week, but XRP's 2023-09..2024-08 and SOL's 2025-09..2026-08 windows lose money. XRP's losing window contains losses under both entry families and both exit families. Filtering the observed trade ledger by strong signal-candle closes also removes about a third of trades and does not eliminate those two weak years; this is descriptive attribution, not a replay or realized counterfactual.

Hypothesis: slower v29 exits should be selected only while daily trend strength and/or daily price/EMA alignment justify waiting. The v62 long routing guard has directional DI and EMA20 slope but no daily ADX strength/rising requirement and no EMA20/EMA50 alignment. Tightening routing changes closes without removing entry opportunities or adding indicators.

- v63a: add daily ADX >=20 and non-falling vs three days earlier to both slow-exit guards.
- v63b: add daily EMA20 > EMA50 and daily close > EMA20 to the long slow-exit guard; short routing stays as v62a.
- v63c: combine the two changes.

Every candidate keeps v29C + v48 entries, signal-confirmed close rules, live `[0]` usage, 8x leverage, margin sized at 10% of current available cash (constant fraction, compounding), outer +/-5% ROI gates, 0.05% fee and 5bps slippage each side, historical funding, and the full 2022-09-01..2026-08-31 period. Each full portable JSON is stored under `temp_strategy/v63`. Run the actual full matching engine for all three; retain failed candidates. None is a validated release before yearly and unseen-coin gates are proven.

## Completed 2026-10-03 same-source replay

All 16 v62/v63 × four-coin runs completed on canonical correction version `20261003-v2`. The stronger daily routing alternatives did not eliminate annual losses and generally reduced four-year net versus v62a. No v63 candidate is released or written to a database. See `2026-10-03-arm-v62-v65-summary.md` for exact annual arrays; source repairs never changed candidate hashes, dates or cost assumptions.
