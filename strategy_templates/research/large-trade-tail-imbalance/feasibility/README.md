# Large-Trade Tail Imbalance — Feasibility

Hypothesis: directional imbalance among only the largest USD-M trades may contain information distinct from aggregate taker ratio, average trade size, VPIN, and trade-count features.

Before defining a return signal, data feasibility was checked for the fixed six-symbol discovery set (SOL/DOGE/LTC/AVAX/UNI/ZEC):

- Local `market_trades` in `go_bn_test`: **0 rows for all six symbols**.
- Binance Vision USD-M monthly `aggTrades` is available, but a single SOLUSDT month (2023-01) is about **262,724,502 bytes compressed (~263 MB)**.
- A complete six-symbol 2023-2024 archive would therefore require a substantial multi-GB / potentially tens-of-GB download before any hypothesis could even be evaluated.

Given the user's current storage constraint and the project's preference for lightweight, production-runnable research, this data cost is not justified for an unvalidated family.

Decision: **data/storage feasibility blocked**. No aggTrades archive was downloaded beyond the 1-byte HTTP range size probe, no market returns were read, and no DB data was written. Reopen only if a compact historical large-trade aggregate becomes available from an official source or the project later maintains the required trade history for production use.
