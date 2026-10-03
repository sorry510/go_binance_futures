# DeFiLlama Chain Derivatives Open-Interest — Feasibility

Hypothesis: growth in decentralized-derivatives open interest on a chain may proxy native-chain trading/speculation demand and predict the native token.

Chain OI was reconstructed correctly from each protocol summary's historical `totalDataChartBreakdown`; the overview breakdown was not used as chain data. Native-token mappings were frozen before coverage inspection.

## Audit

- DeFiLlama overview: 137 OI protocols.
- 34 protocols belonged to at least one frozen mapped native chain.
- 34/34 relevant summaries fetched with 0 errors.
- Frozen native mappings: 14 chains/tokens.
- Required >=30 nonzero chain-OI days in 2023-2024 plus conservative same-name Binance USD-M monthly archive presence in 2022-12 and 2024-12.
- Passing mappings: **ETH, BNB, SOL, INJ** only.
- AVAX has full historical chain OI but fails the conservative 2024-12 same-name USD-M archive proxy; FTM/NEAR/KAVA/CELO/ADA/TRX/BTC have no meaningful 2023-2024 chain OI in this source; other frozen mappings fail one or both conditions.

Coverage gate required >=8 tokens.

## Decision

**Coverage blocked / freeze before returns.**

No native-token forward return was read. Do not add ARB/OP after seeing coverage, reintroduce the MATIC→POL migration as a stable mapping, infer non-native governance tokens, lower the two-year rule, or run a four-token special case.
