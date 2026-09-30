# DeFiLlama Chain Stablecoin-to-TVL Capital Rotation

Mechanism: chain USD stablecoin supply relative to chain TVL is a capital-allocation ratio. Rising stablecoin/TVL dominance is interpreted as rotation toward cash/risk-off and fixed SHORT the native token; falling dominance is fixed LONG.

Signal was frozen before returns: D_t = log[(stablecoin_supply/TVL)_t / (stablecoin_supply/TVL)_{t-7}], zero-cross. Entry next UTC daily open. Eligibility requires >=2 years Binance USD-M history and signal-day QuoteVolume >=5m USDT. Discovery is 2023-2024; 7d endpoints may not cross the partition boundary.

Frozen discovery gate: aggregate 7d signed mean >=+0.25%, >=60% symbols positive, and both 2023 and 2024 aggregate mean7 >0.

Canonical discovery: 1124 events / 11 symbols. Mean 1d +0.1096%, 3d -0.1083%, 7d -0.0818%; only 4/11 symbols positive. 2023 mean7 +0.2393%, 2024 -0.3818%.

Decision: discovery failed. OOS remains untouched. Do not reverse direction, change 7d window, filter symbols, or enter exact TP8/SL6.
