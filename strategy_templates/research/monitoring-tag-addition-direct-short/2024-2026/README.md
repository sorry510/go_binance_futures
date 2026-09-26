# Binance Monitoring Tag Addition Direct-Short — 2024–2026 Audit

This bundle preserves the historical discovery/OOS evidence behind the Monitoring Tag addition event family.

The preregistered mechanism was: when Binance adds a token to the Monitoring Tag, treat the official announcement as a negative risk-rating event and SHORT the still-trading USD-M perpetual from the next 1m open. Direction, eligibility, execution parameters, and OOS sequence were frozen before later-year results were inspected.

The recorded research sequence was:

- 2024 discovery: 9 eligible events, 5 wins / 4 losses, historical PF 1.374, historical normalized net +10.84%.
- 2025 OOS: 9 eligible events, 5 wins / 4 losses, historical PF 1.107, historical normalized net +7.77%.
- 2026 second OOS: 17 eligible events, 8 wins / 9 losses, historical PF 0.910, historical normalized net -8.13%.
- Three-year historical aggregate: 35 events, 18 wins / 17 losses, PF about 1.054, normalized net about +10.49%.

The final recorded decision is **frozen / no DB import / not a formal candidate** because the positive 2024/2025 behavior did not persist in the larger 2026 OOS and event-gap tail losses were large.

## Known funding-parser issue

All three archived replay helpers contain a historical parser bug. They assume Binance Vision monthly `fundingRate` CSV has a fourth `mark_price` column:

`calc_time,funding_interval_hours,last_funding_rate,mark_price`

The actual archive schema used in this research has only:

`calc_time,funding_interval_hours,last_funding_rate`

The helpers therefore silently skip funding rows. The production Engine instead falls back to the corresponding 1m `bar.Close` when a funding record has no MarkPrice.

For this reason, the PF/net figures above are retained as **historical values pending corrected-parser replay**, not as corrected final metrics. Fee and slippage logic remains represented in the helpers. This archive does not reopen the research family or use later results to change the frozen rule.

## Evidence retained

- `inputs/eligible_events.json`: the 35 eligible events encoded in the three archived helpers.
- `replay/2024-discovery.py`, `replay/2025-oos.py`, `replay/2026-oos.py`: exact historical helpers copied from `/tmp` before loss.
- `results/summary.json`: historical metrics with an explicit validation status.
- `config.json`: frozen execution and eligibility semantics.
- `protocol.md`: discovery-to-OOS decision protocol.
- `provenance.json`: file hashes, source references, and known limitations.

The complete pre-eligibility event universe and exclusion rows were not preserved in `/tmp`; `strategy_templates/result.md` remains the canonical narrative source for the official batch counts and eligible sets. This limitation is explicit so future audits do not mistake the archived eligible list for the complete event universe.
