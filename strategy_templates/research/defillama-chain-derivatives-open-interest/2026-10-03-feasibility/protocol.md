# DeFiLlama Chain Derivatives Open-Interest — Feasibility

1. Source is the free no-auth DeFiLlama open-interest overview and protocol summary endpoints only.
2. Chain-level OI must be reconstructed only from each protocol summary's historical totalDataChartBreakdown chain keys. The overview breakdown is protocol-level and must not be misread as chain-level.
3. Native-token mapping is frozen before historical coverage inspection in inputs/chain_native_map.json. No chain/token is added after seeing coverage.
4. Mapping is restricted to direct native-chain tokens: Ethereum→ETH, BSC→BNB, Solana→SOL, Avalanche→AVAX, Fantom→FTM, Injective→INJ, NEAR→NEAR, Kava→KAVA, Celo→CELO, Moonbeam→GLMR, Cronos→CRO, Cardano→ADA, Tron→TRX, Bitcoin→BTC.
5. Arbitrum/OP/Polygon are deliberately excluded from this frozen map: ARB/OP lack two years of token history through much of 2023-2024, while Polygon's MATIC→POL migration breaks a single stable same-name token mapping.
6. For each mapped chain, aggregate all protocol OI values by complete UTC date. Require >=30 nonzero historical days in 2023-2024.
7. Conservative Binance production-coverage proxy: same-name USD-M monthly 1h archive must exist in both 2022-12 and 2024-12. This does not replace per-signal >=730d eligibility later.
8. Coverage gate before any token return: >=8 mapped chains/tokens pass both OI-history and Binance archive checks.
9. If coverage fails, freeze without price-return inspection. Do not add chains/tokens after seeing the result, infer wrapped/governance tokens, or lower the gate.
