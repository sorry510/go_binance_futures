# Token Unlock Supply-Shock — 2026-10-01 Exact-Timestamp Recheck

This reopens only the prior **data blocker**, not the strategy parameters or event cohort.

The previously frozen production-eligible cohort remains exactly 9 events / 8 symbols: FTM, IMX, AXS, RON, APT (two events), ID, FET, SEI. The earlier event study showed unusually large raw 72h downside, so this family is worth preserving if exact execution timestamps become auditable.

## Source re-audit

The public replication repository's actual `01_binance_token_unlock_events_2023_2025.csv` contains `unlock_date` but **does not contain `unlock_timestamp_utc` or `timestamp`**.

This conflicts with the repository documentation:
- `DATA_SOURCES.md` defines event T as an on-chain block timestamp in UTC with hour-level precision.
- `DATA_DOCUMENTATION.md` instructs replication code to read `unlock_timestamp_utc` from File01.

Git history was checked for the CSV itself. It has a single public commit (2026-04-20), so there is no earlier public revision containing the missing timestamp columns.

Tokenomist currently exposes timestamped unlock-event products, but accessible public pages/API history do not provide auditable complete coverage for this fixed 2024-2025 cohort. Individual old events may be discoverable through isolated articles or third-party posts; using only those events would create availability selection and is prohibited.

## Decision

**Timestamp coverage remains blocked.** No exact 1m TP8/SL6 replay was run and no new post-event returns were read.

Do not assume 00:00 UTC, infer timestamps from recurring schedules, or shrink the cohort to events with easy-to-find timestamps. Reopen only after all nine frozen events have independently verifiable exact UTC times (ideally provider event records or on-chain transactions) so the exact 1m Engine can be run without timestamp-selection bias.
