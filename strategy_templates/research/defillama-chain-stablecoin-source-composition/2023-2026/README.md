# DeFiLlama Chain Stablecoin Source Composition

Mechanism: whether a chain's USD-stablecoin liquidity is increasingly bridge-sourced relative to locally minted supply. This differs from total stablecoin supply, stablecoin price/depeg stress, DEX velocity, and full bridge netflow.

For each chain/day with both components present:
- composition = log(totalBridgedToUSD.peggedUSD / totalMintedUSD.peggedUSD)
- flow7 = composition_t - composition_t-7d
- zero-cross upward => LONG native token
- zero-cross downward => SHORT
- entry = next UTC daily open

The chain-to-native-token map and >=2-year Binance USD-M eligibility were inherited unchanged from the audited Chain Stablecoin Supply Growth universe. Signal-day QuoteVolume must be >=5m USDT. Discovery is 2023-2024 with the complete 7d return endpoint inside discovery.

Frozen gate: 7d signed mean >= +0.25% and >=60% triggering symbols positive.

Discovery: 810 events / 9 symbols. 1d +0.2175%, 3d -0.1447%, 7d -0.0632%; only 3/9 symbols positive at 7d.

Decision: discovery failed. 2025-2026 OOS was not evaluated. Freeze without reversing direction, changing the 7d window, or adding composition thresholds.
