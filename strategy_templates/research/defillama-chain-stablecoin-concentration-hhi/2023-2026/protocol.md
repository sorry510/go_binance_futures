# Protocol

1. Enumerate every current DeFiLlama catalog entry with `pegType=peggedUSD`; do not select stablecoins by present size or by post-return relevance.
2. For each chain/day, reconstruct positive nominal `circulating.peggedUSD` balances from `/stablecoin/{id}` and compare their sum with `/stablecoincharts/{chain}` `totalCirculating.peggedUSD`.
3. A day is valid only when at least 3 USD stablecoins have positive balances and per-asset sum / aggregate is between 0.98 and 1.02.
4. HHI uses the reconstructed per-asset sum as the share denominator. Compute 7-day HHI change. A zero-cross to positive (concentration rising) is SHORT; a zero-cross to negative (diversification rising) is LONG. Do not scan thresholds or windows.
5. Enter next UTC daily open. Require Binance USD-M history >=2 years at the signal date and signal-day quote volume >=5m USDT.
6. Discovery is 2023-2024; the entire 7-day forward endpoint must remain <=2024-12-31. Promote only if signed 7d mean >=+0.25% and >=60% of event-bearing symbols have positive 7d mean.
7. If discovery fails, do not inspect 2025-2026, reverse the economic interpretation, remove weak chains, or alter the active-stablecoin/data-quality rules.