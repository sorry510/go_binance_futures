# Open-Interest / Price State Transition

## Hypothesis

Test whether a fresh transition in 4h open-interest state contains a large, fast directional edge without tuning magnitude thresholds:
- OI expansion transition: continuation in the existing 4h price direction.
- OI contraction transition: reversal against the existing 4h price direction.

## Frozen protocol

Stage 1 used fixed 2026 Jan/Apr/Jul three-day blocks on BTC/ETH/BNB/XRP. Only a family with overall 4h signed mean >= +0.10% and at least 3/4 symbols positive could advance.
The contraction-reversal family alone passed, so its definition was frozen and tested on every available 2025 day. No lookback, threshold, direction, or symbol rule was changed.

## Stage 1 — 2026 early gate

Expansion-continuation: 132 events; 4h signed mean -0.0653%; positive symbols 2/4; failed.

Contraction-reversal: 129 events; 1h +0.0359%; 4h +0.1765%; 12h +0.5865%; positive symbols 3/4; passed the pre-registered early gate.

## Stage 2 — full 2025 historical OOS

Contraction-reversal: 7,989 events; 1h -0.0064%; 4h -0.0267%; 12h -0.0094%; positive symbols 0/4.

By-symbol 4h signed mean: BTC -0.0009%; ETH -0.0551%; BNB -0.0265%; XRP -0.0211%.
## Decision

The apparent 2026 contraction-reversal edge completely failed on the much larger untouched 2025 sample. The effect is regime/sample instability, not a candidate alpha.

Freeze the entire OI-price-state transition family. Do not search 2h/6h/12h lookbacks, OI magnitude thresholds, price thresholds, alternative zero-cross definitions, or opposite directions. Do not add an OI indicator to the production engine.

Because Stage 2 failed before the Engine gate, no 1m TP8/SL6 replay was run.

## Known data limitation

Binance Vision daily metrics are reconstructable, but historical point-in-time publication latency of metrics/open-interest observations is not independently verified. A positive result would have required a separate availability audit; the negative OOS result makes that unnecessary.

## Canonical evidence

- config.json
- protocol.md
- provenance.json
- replay.py
- validate_2025.py
- results/summary.json and results/events.csv
- results/summary_2025.json and results/events_2025.csv
- inputs/README.md
- manifest.sha256
