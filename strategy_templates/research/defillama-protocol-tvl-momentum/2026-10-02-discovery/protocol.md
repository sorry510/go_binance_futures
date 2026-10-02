# Protocol

1. Reuse the exact unique protocol.symbol universe already frozen by defillama-protocol-tvl-shock; do not add/remove protocols based on this test.
2. Exclude the same CEX, Chain, Canonical Bridge and identity-collision cases already excluded by the source universe.
3. Use DeFiLlama protocol totalLiquidityUSD on complete UTC dates only.
4. For each date t, score = log(TVL_t / TVL_{t-7}). Require both values >0 and all dates needed for the signal to exist.
5. score crossing from <=0 to >0 emits LONG; crossing from >=0 to <0 emits SHORT. No magnitude threshold or re-arm threshold.
6. Enter at the next UTC day's Binance USD-M open. Measure signed 1d/3d/7d log returns.
7. Dynamic production eligibility at every signal: USD-M history >=730 days and signal-day QuoteVolume >=5M USDT.
8. Discovery is 2023-2024 only. 2025/2026 must remain unread unless the frozen discovery gate passes.
9. Gate: 7d event-weighted signed mean >=+0.25%, >=60% of triggered symbols positive, >=0.30 eligible signals/symbol/week, and 2023/2024 7d means both >0.
10. Failure freezes the family: do not scan 3d/14d/30d TVL windows, add TVL-size/category filters, delete one direction, residualize by price, or reverse the signal.
11. This is distinct from the prior protocol-TVL shock family: that family tested isolated z<=-3 daily shocks; this tests persistent seven-day capital-flow direction.
12. Before outcome computation, require the 7d return endpoint to remain in the same calendar year as the signal; this prevents 2024 labels from reading 2025 and keeps annual sign checks independent.
13. Frequency denominator is the number of unique symbols with at least one eligible discovery signal times the full 2023-2024 week count; breadth denominator is the same triggered-symbol set.
14. Binance price/QuoteVolume input is read-only public USD-M 1d REST; no market data is imported into the project DB.
