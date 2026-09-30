# Spot-vs-Perp Order-Book Imbalance Divergence — Feasibility

Hypothesis: divergence between Spot and USD-M near-book bid/ask depth could reveal real-demand versus leveraged-market disagreement.

Binance Vision USD-M exposes daily bookDepth history, but the official Spot daily archive exposes only aggTrades, klines, and trades. There is no corresponding Spot bookDepth archive under the official public data tree.

Decision: same-source historical-data blocked before any return inspection. Do not combine Binance Futures depth with a different vendor's Spot depth because sampling, levels, and timestamp semantics would be inconsistent.
