# DeFiLlama Protocol Open-Interest Momentum — Feasibility Protocol

1. Source must be the free no-auth DeFiLlama legacy open-interest endpoints: /overview/open-interest and /summary/open-interest/{protocol}.
2. Start from every protocol returned by overview. Use each summary endpoint's own non-empty token symbol; do not infer a token from protocol name, chain, gecko_id, or current popularity.
3. Multiple versions/child protocols with the same token symbol may later be aggregated by UTC date, but count as one token universe member.
4. Stablecoins, fiat-like symbols, wrapped-only symbols, LP tokens and symbols that cannot map unambiguously to same-name Binance USD-M are excluded.
5. Historical OI coverage must include meaningful 2023-2024 observations. Production eligibility for eventual signals will still require same-name Binance USD-M history >=730 days at signal date and prior-24h QuoteVolume >=5M USDT.
6. Feasibility coverage gate before returns: >=8 unique safely mapped token symbols with DeFiLlama OI observations in 2023-2024 and plausible same-name Binance USD-M history.
7. No token forward return may be read before the coverage gate is evaluated.
