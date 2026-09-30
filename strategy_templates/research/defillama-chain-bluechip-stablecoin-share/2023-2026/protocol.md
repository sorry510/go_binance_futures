# Protocol

1. Source is DeFiLlama only: USDT id=1 and USDC id=2 chainBalances `tokens[].circulating.peggedUSD`, divided by chain `stablecoincharts` `totalCirculating.peggedUSD`.
2. Use nominal circulating amounts, not current price, so historical depeg mark-to-market is not injected.
3. For each chain compute share and its 7-day change. First positive zero-cross is LONG; first negative zero-cross is SHORT. No threshold scan.
4. Enter at the next UTC daily Binance USD-M open. Require contract history >=2 years at signal date and signal-day quote volume >=5m USDT.
5. Discovery is 2023-2024 and every 7-day endpoint must remain inside 2024. Promote only if signed 7d mean >=0.25% and >=60% of event-bearing symbols have positive 7d mean.
6. If discovery fails, do not inspect 2025-2026, reverse direction, alter the 7-day window, or filter symbols post hoc.