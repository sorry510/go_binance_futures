# DeFiLlama Chain Bridge Netflow — Feasibility

Hypothesis: chain-specific cross-chain bridge capital inflow should support the chain native token, while net outflow should pressure it. Planned signal was the zero-cross of the trailing 7 UTC-day sum of `depositUSD - withdrawUSD`, entered at the next UTC-day open.

The required historical bridge endpoints are no longer freely accessible from the research host: `bridges.llama.fi/bridgevolume/Ethereum` and the bridge catalog both returned HTTP 402. The alternate `api.llama.fi` bridgevolume path did not provide a usable response.

Decision: **data-access blocked before any return inspection**. Do not scrape visual pages or use an incomplete subset to manufacture a backtest.

2026-09-29 source-code recheck: the open-source DefiLlama bridges-server confirms that historical USD deposits/withdrawals live in Postgres daily/hourly aggregate tables and exposes historical-day query logic, but the deployed netflows and bridgedaystats routes all currently return HTTP 402. The public llama-bridges-data S3 bucket does not allow listing; known readable objects contain only block-progress metadata. Rebuilding the history would require replaying many bridge adapters/on-chain logs and is therefore data-cost blocked for an unvalidated signal. The family remains frozen before returns.
