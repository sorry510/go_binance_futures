# Legacy-Derived 3m Exhaustion Wick Reversal

Status: **DATA-PATH BLOCKED before returns**.

Mechanism: after a strongly one-sided 3m run, the latest completed 3m candle makes a fresh 20-bar extreme with a long rejection wick; entry requires the current 3m bar to break the rejection candle in the reversal direction.

The strategy was pre-registered before returns and is directly expressible in the current JSON DSL. During DatasetBuilder loading, BTCUSDT 3m failed with historical gaps `[[1672495200000 1756619999999]]`. No trade result was generated.

Decision: freeze as currently unverifiable in the existing local historical-data path. Do not auto-download/backfill 3m data, do not write DB, and do not reinterpret this as alpha failure.

Canonical evidence: `strategy.json`, `protocol.md`, `config.json`, `results/summary.json`, `results/engine.log`, `replay.go`, `provenance.json`, `manifest.sha256`.

## Strict-exit audit status

The prior conditional-exit result is non-canonical and preserved under legacy/conditional-exit/. A canonical rerun with ROI >= 8 || ROI <= -6 could not start because BTCUSDT 3m history has a large local gap. No strict-return evidence was inspected and no DB backfill was performed. Status: data-blocked.

## Strict-exit audit status

The original conditional-exit result is non-canonical. A strict TP8/SL6 rerun was attempted with unchanged entry logic, but DatasetBuilder failed before returns because the current local BTCUSDT 3m history is incomplete. No DB backfill was performed. Therefore v78 is now strict-exit data-blocked, not a valid pass/fail return result.
