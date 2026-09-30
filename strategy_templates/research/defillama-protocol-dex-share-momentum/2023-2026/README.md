# DeFiLlama Protocol DEX Share Momentum

Hypothesis: sustained improvement in a DEX protocol token's share of global DEX volume may lead token returns because it reflects protocol-level competitive/usage gains rather than broad crypto trading activity.

Frozen signal: trailing-7-day protocol DEX volume / trailing-7-day global DEX volume; take the 7-day log change of that share and trade zero-crosses. Positive cross = LONG, negative cross = SHORT, entry next UTC day open.

Eligibility is dynamic: Binance USD-M history >=2 years at signal time and completed signal-day QuoteVolume >=5m USDT. Discovery was frozen to 2023-2024 with a +0.25% 7d mean / 60% positive-symbol breadth gate.

Canonical discovery after identity audit: 519 events / 8 symbols. Signed mean returns were -0.1460% at 1d, -0.2756% at 3d, and -0.5292% at 7d. Only 1/8 symbols had positive 7d mean. The discovery gate failed decisively.

No 2025/2026 OOS return was read and no exact TP8/SL6 replay was run. The family is frozen; no reverse-direction, window scan, or symbol filtering is allowed.

Audit correction: the first discovery mistakenly allowed ticker-only LIT mapping. DeFiLlama LIT here is Lighter, while Binance historical LITUSDT was Litentry. That superseded run is preserved in legacy/pre-identity-correction/. Removing LIT made the already-negative result more negative and reduced positive breadth from 2/9 to 1/8.

Canonical evidence: protocol.md, config.json, provenance.json, replay.py, inputs/components.json, inputs/universe.json, results/discovery_events.json, results/discovery_summary.json.
