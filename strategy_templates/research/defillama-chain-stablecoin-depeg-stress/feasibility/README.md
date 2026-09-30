# DeFiLlama Chain Stablecoin Depeg Stress — Feasibility

Hypothesis: a material under-peg in a chain's supply-weighted USD stablecoin basket is a chain-specific liquidity/credit stress event and should pressure the native token.

Historical price semantics were validated before returns: chain stablecoincharts expose both totalCirculating.peggedUSD and totalCirculatingUSD.peggedUSD. Their ratio reproduced the March 2023 USDC depeg (e.g. Ethereum ~0.9764 on 2023-03-12).

Frozen event definition before returns: first daily ratio <=0.995 after re-arm at >=0.999, fixed SHORT native token. To prevent one small illiquid stablecoin from dominating measurement, event-day nominal peggedUSD supply must be >=50m USD. Contract must also have >=2 years Binance USD-M history. Minimum coverage is 8 symbols and 8 distinct UTC trigger dates before return inspection.

Raw 2023-2024 coverage: 57 chain-events / 12 symbols / 46 UTC dates. After the frozen >=50m supply and >=2-year futures-history quality gates: 34 events / 7 symbols / 27 UTC dates. Eligible symbols were ETH, BNB, SOL, AVAX, NEAR, MATIC and FTM.

Decision: coverage blocked (7 < 8) before any return inspection. Do not remove the 50m quality gate, add immature OP/APT/ARB/SUI, or treat repeated cross-chain USDC depeg observations as independent alpha evidence.
