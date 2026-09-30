# DeFiLlama Chain Stablecoin Velocity

Hypothesis: DEX turnover per dollar of chain stablecoin liquidity measures on-chain transaction/speculative velocity and may lead the native token.

Frozen feature: weekly velocity = sum of 7 consecutive UTC days of DeFiLlama DEX dailyVolume divided by mean stablecoin supply over those 7 days. flow = log(recent 7d velocity / prior non-overlapping 7d velocity). Upward zero-cross LONG, downward zero-cross SHORT, next UTC day Binance USD-M open.

Dynamic eligibility is USD-M history >=2 years plus signal-day quote volume >=5M USDT. Discovery gate was frozen before returns: mean signed 7d >= +0.25% and >=60% positive symbols.

Discovery 2023-2024: 806 events / 11 symbols; 1d +0.1142%, 3d +0.5436%, 7d +1.0474%; 10/11 symbols positive. Discovery passed strongly.

Untouched 2025 OOS: 482 events / 12 symbols; 7d -0.0218%, 8/12 symbols positive. The economic edge collapsed to slightly negative.

2026 OOS: 360 events / 12 symbols; 7d +0.2701%, but only 6/12 symbols positive. The endpoint recovered modestly while cross-symbol breadth failed.

Decision: freeze. The strong discovery did not survive consistently across OOS regimes. Do not enter exact TP8/SL6, filter weak symbols, change the window, add a velocity threshold, or reverse the signal.

Canonical evidence:
- protocol.md / config.json: pre-return frozen definition and gate.
- inputs/source_coverage.json: 15-chain stablecoin source coverage.
- inputs/universe.json: dynamic eligibility/event coverage.
- results/events.json: discovery and permitted OOS events.
- results/summary.json: discovery/OOS summaries.
- replay.py: stage-gated replay.
