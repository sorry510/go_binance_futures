# ID121 vs v54 — Strict Production Audit

This bundle preserves the canonical `2026-09-24` comparison between the production benchmark ID121 and the v54 frequency candidate.

The only valid comparison basis is **dynamic >=2-year contract eligibility plus a fresh Engine start from exactly the same per-symbol production start time**. Earlier aggregates that truncated an already-running ID121 backtest are superseded because equity and position sizing are path dependent.

## Strategy identity

- ID121 is the production template named `v33 LONG + relaxed daily-ADX SHORT 正式候选` from read-only `go_bn_test.strategy_templates.id=121` at archive time. Its exact JSON snapshot is `strategy/id121-db-template-121.json`.
- `strategy/v33-source.json` is the repository v33 source from which ID121 was derived. It is **not** an exact substitute for ID121.
- ID121 and v33 have identical `technology`. Only strategy rule index 1 (`short`) differs: ID121 removes the daily ADX-rising requirement `adx_1d_14.ADX[1] >= adx_1d_14.ADX[3]` and the 4h ADX acceleration requirement `adx_4h_14.ADX[1] - adx_4h_14.ADX[3] >= 2`. The other three rules are identical.
- v54 is preserved exactly as `strategy/v54.json`.

## Strict raw Engine result

ID121 is the main benchmark. v54 remains a parallel frequency candidate and does not replace ID121.

| Metric | ID121 | v54 |
| --- | ---: | ---: |
| Trades | 519 | 563 |
| Raw PF | 1.395080 | 1.395847 |
| Raw net PnL | +17015.9 | +18131.7 |
| Positive symbols | 10/15 | 11/15 |
| Median raw symbol PF | 1.084 | 1.146 |
| Trades / symbol / week | 0.358 | 0.388 |

Raw annual PF: ID121 = 1.494 / 1.157 / 1.647 for 2024/2025/2026; v54 = 1.444 / 1.063 / 1.792.

## Fixed-notional normalized result

`results/paired_normalized.csv` stores one row per trade with `norm_return = NetPnL / (EntryPrice * Quantity)`.

| Metric | ID121 | v54 |
| --- | ---: | ---: |
| Trades | 519 | 563 |
| Normalized PF | 1.3857864308 | 1.3835154361 |
| Normalized net | +2.9435523306 | +3.1936493259 |
| Positive symbols | 13/15 | 13/15 |
| Median normalized symbol PF | 1.3070051107 | 1.3477793320 |
| LONG / SHORT | 271 / 248 | 315 / 248 |

Normalized annual PF: ID121 = 1.538963 / 1.239839 / 1.536392; v54 = 1.484502 / 1.159394 / 1.674758.

## Paired attribution

The canonical raw-dollar attribution recorded in `strategy_templates/result.md` is:

- COMMON: 509 trades, PF 1.410, net +17275.5.
- ID121-only: 10 trades, PF 0.724, net -259.5.
- v54-only: 54 trades, PF 0.972, net -142.1.
- v54-only annual PF: 2024 0.862, 2025 0.638, 2026 1.591.

`results/paired_normalized.csv` contains normalized returns only, so recomputing attribution from it produces a **different metric family**: COMMON PF 1.404777, ID121-only PF 0.525551, v54-only PF 1.199403. These normalized values must never overwrite the raw-dollar attribution above.

The interpretation remains frozen: v54 adds frequency at approximately zero overall PF cost and improves the cross-symbol median, but its incremental trades are not a standalone strong alpha and are regime dependent. Do not tune nearby ID121/v54 thresholds.

## Canonical evidence

- `strategy/id121-db-template-121.json`: exact production ID121 snapshot from the experiment DB, read-only.
- `strategy/v33-source.json`: repository source ancestor for ID121.
- `strategy/v54.json`: exact v54 candidate.
- `results/paired_normalized.csv`: normalized replay export from the strict same-start runs.
- `results/summary.json`: raw and normalized metrics with explicit metric provenance.
- `replay/paired_121_v54.go`: original paired replay helper used to produce the CSV; it reads ID121 from DB and is retained unchanged as historical evidence.
- `replay/v54_norm_compare.go`: original strict raw/normalized aggregation helper.
- `legacy/id121_prod_agg_mixed_start.go`: superseded mixed-start helper, retained only to explain why the benchmark was corrected.

No database writes are part of this archive. No large K-line archive is stored in Git.
