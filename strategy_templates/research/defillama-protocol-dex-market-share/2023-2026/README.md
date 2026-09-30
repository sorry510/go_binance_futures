# DeFiLlama Protocol DEX Market-Share Flow

Hypothesis: changes in a DEX protocol token's share of aggregate DeFi DEX volume may represent protocol-level competitive usage changes that lead the token price.

The protocol was frozen before reading returns. All DEX versions mapped to the same DeFiLlama protocol token symbol are aggregated. The signal is the zero-cross of log(recent 7-day average market share / prior 7-day average market share): positive cross LONG, negative cross SHORT, next UTC day open.

Production eligibility is dynamic Binance USD-M history >=2 years plus signal-day quote volume >=5M USDT. Discovery is 2023-2024. The frozen gate is mean signed 7d return >= +0.25% and >=60% positive symbols.

Discovery produced 519 events across 8 symbols with events. Mean signed endpoint return was -0.1242% at 1d, -0.3934% at 3d, and -0.4786% at 7d. Only 2/8 symbols had positive mean 7d return.

The discovery gate failed decisively. OOS 2025/2026 returns were not evaluated and exact TP8/SL6 replay was not entered. The family is frozen; no direction inversion, threshold scan, or symbol filtering is permitted.

RAYUSDT and DODOUSDT were retained in the mechanical universe but produced zero eligible discovery events under the frozen continuous-data/zero-cross rules.

Canonical evidence:
- protocol.md / config.json: pre-return frozen protocol.
- inputs/universe.json: mechanical symbol/slugs mapping and eligibility metadata.
- results/events.json: discovery events only because the gate failed.
- results/summary.json: discovery summary and explicit oos_evaluated=false.
- replay.py: stage-gated replay; OOS price windows are requested only after a passing discovery gate.
