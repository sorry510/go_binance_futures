# DeFiLlama Chain Stablecoin Concentration (HHI)

Hypothesis: rising concentration of a chain's USD-stablecoin liquidity increases dependence on fewer settlement assets and is negative for the chain native token; increasing diversification is positive.

Before returns, all 339 DeFiLlama `peggedUSD` assets were audited. Per-stablecoin historical balances matched chain aggregate supply essentially exactly on the target chains. Valid signal days require at least 3 positive-balance USD stablecoins and per-asset sum / aggregate in [0.98, 1.02].

Frozen signal: 7d HHI-change zero-cross; concentration rising -> SHORT, diversification rising -> LONG; next UTC day Binance USD-M open; contract history >=2y and signal-day quote volume >=5m. Discovery=2023-2024 and the full 7d endpoint cannot cross into 2025.

Canonical discovery: 823 events / 11 symbols. 1d signed mean -0.0518%, 3d +0.0305%, 7d -0.1168%, win7 51.28%. Breadth was 7/11=63.64%, but the economic gate failed. 2023 7d mean was +0.3663%; 2024 flipped to -0.5692%. Symbol-equal 7d mean was -0.0292%.

Data checks: no duplicate symbol/date events; LONG/SHORT counts 412/411; event HHI range 0.2949-0.9673; event coverage ratio 0.99999986-1.00000009.

Decision: freeze. 2025-2026 OOS was not evaluated. Do not reverse the sign, remove weak chains, replace HHI with entropy/effective-N variants, or tune the 7d window.