# CoinMetrics Net Supply Growth — Feasibility

Fixed mature universe: BTC/ETH/XRP/ADA/LINK/BCH/LTC/DOGE/UNI/ZEC. Source is CoinMetrics Community daily `SplyCur`.

Coverage requirements:
- >=700 valid 2023-2024 daily observations;
- >=30 nonzero day-over-day supply changes;
- >=8 variable assets.

Result:
- BTC/ETH/XRP/ADA/BCH/LTC/DOGE/ZEC each have 731 days and 730 nonzero daily changes.
- LINK and UNI have complete 731-day coverage but `SplyCur` is constant in Community data.
- **8/10 assets are variable**, so the frozen coverage gate passes exactly.
- No price return was read during feasibility.

Discovery was frozen separately before returns under `../2026-10-02-discovery/`.
