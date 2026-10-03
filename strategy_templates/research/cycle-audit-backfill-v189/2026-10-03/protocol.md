# v189 Frozen-Model Full-Cycle Extension

The original v189 model was trained only on 2023 and frozen before 2024 returns were read.

This audit keeps the exact archived model SHA-256 and applies it unchanged to:
- archived untouched validation: 2024
- new temporal OOS1: 2025
- new temporal OOS2: 2026-01-01 through 2026-09-30

No retraining, pruning, leaf selection, threshold change, feature change, symbol-specific rule, or prediction-magnitude filter is allowed.

Universe remains the original ten symbols:
BTCUSDT, ETHUSDT, BNBUSDT, XRPUSDT, SOLUSDT, DOGEUSDT, LTCUSDT, AVAXUSDT, UNIUSDT, ZECUSDT.

Signal remains a fresh zero-cross of the frozen tree prediction:
- prediction <= 0 to > 0: LONG
- prediction >= 0 to < 0: SHORT
- entry: next complete 1h open
- diagnostics: signed 1h / 4h / 12h log return
- signal and 12h endpoint must remain inside the same calendar year.

Canonical full-cycle reporting preserves the archived 2024 validation events and appends only 2025 and 2026 events from this run. 2023 remains training and is not counted as strategy validation performance.

No strict TP8/SL6 promotion, DB write, app.conf modification, commit, or push unless the frozen model passes the declared raw gate.
