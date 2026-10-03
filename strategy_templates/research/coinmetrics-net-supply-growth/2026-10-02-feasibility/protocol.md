# Feasibility Protocol

1. Fixed universe is BTC/ETH/XRP/ADA/LINK/BCH/LTC/DOGE/UNI/ZEC, identical to the mature CoinMetrics network-activity universe already used in this research program.
2. Source is CoinMetrics Community daily `SplyCur`. No paid metric, proxy reconstruction or alternative supply vendor is allowed after seeing coverage.
3. Feasibility period is 2023-01-01 through 2024-12-31. No market returns are read.
4. A variable asset must have >=700 valid daily `SplyCur` observations and >=30 nonzero day-over-day supply changes during the period.
5. Coverage gate requires >=8 variable assets. Failure freezes the family before any trading signal or return analysis.
6. If coverage passes, the trading transform/direction/gate will be frozen separately before any price return is read.
7. This is distinct from the frozen Issuance-Rate Shock family: `SplyCur` is net outstanding/current supply and can change through issuance, burn and token unlock/distribution even when `IssTotNtv` is zero.
