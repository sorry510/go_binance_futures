# Protocol

- Sources: DeFiLlama stablecoincharts chain history, DeFiLlama historicalChainTvl, Binance Vision USD-M 1d klines.
- Stablecoin amount uses nominal totalCirculating.peggedUSD to avoid mixing depeg price noise into the capital-allocation ratio.
- Ratio = nominal USD stablecoin supply / chain TVL.
- Signal = 7d log-change of ratio; upward zero-cross SHORT, downward zero-cross LONG.
- Entry = next UTC daily open.
- Event eligibility: actual USD-M history >=2 years and complete signal-day QuoteVolume >=5m USDT.
- Discovery partition = 2023-01-01 through 2024-12-31, with the full 7d endpoint required to remain inside the discovery partition.
- Gate = aggregate mean7 >=+0.25%, >=60% positive symbols, 2023 aggregate mean7 >0 and 2024 aggregate mean7 >0.
- OOS may be evaluated only if discovery passes. No post-hoc reversal/window/symbol filtering.
