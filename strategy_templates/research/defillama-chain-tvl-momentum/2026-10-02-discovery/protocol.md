# Protocol

1. Reuse the exact 15 native-chain -> Binance USD-M mapping already audited by defillama-chain-fee-tvl-yield. Do not add/remove chains based on v150 outcomes.
2. Use DeFiLlama historicalChainTvl daily chain TVL and Binance Vision USD-M monthly 1d Klines only.
3. For date t, score = log(TVL_t / TVL_{t-7}). Require every UTC date t-8 through t to exist with TVL > 0.
4. Up-cross from prior score <=0 to current >0 emits LONG; down-cross from >=0 to <0 emits SHORT.
5. No magnitude threshold, smoothing, price residualization, chain-category filter, stablecoin/fee/DEX filter, or symbol-specific rule.
6. Entry is next UTC-day USD-M open. Measure signed 1d/3d/7d log returns.
7. Dynamic event eligibility: signal date >= the pre-audited two-year USD-M eligible_from date and signal-day QuoteVolume >=5M USDT.
8. Discovery is 2023-2024 only. A 7d endpoint must remain in the same calendar year as the signal; 2025+ remains unread unless discovery passes.
9. Gate: 7d event-weighted signed mean >=+0.25%, >=60% of triggered symbols positive, >=0.30 eligible signals/symbol/week, and both 2023/2024 7d means >0.
10. Failure freezes the family: do not scan 3d/14d/30d TVL windows, add price-residual/fee/stablecoin filters, delete one direction, restrict chains, or reverse the signal.
11. This is distinct from chain TVL price-residual flow (90d beta residualized against token return) and protocol TVL momentum (protocol-level capital rather than whole-chain capital).
