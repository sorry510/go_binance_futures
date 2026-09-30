# DeFiLlama CEX Token Reserve / Inflow — Feasibility

Hypothesis: changes in centralized-exchange token balances or net inflows can proxy immediately sellable supply and may lead the same token's Binance USD-M return.

No price returns were inspected.

The public `https://api.llama.fi/cexs` endpoint is accessible, but it exposes only current exchange-level aggregate state such as `currentTvl`, `cleanAssetsTvl`, and rolling 24h/1w/1m aggregate inflows. It does not expose historical observations or per-token balance history.

The official DeFiLlama free OpenAPI contains 31 paths and no CEX reserve/history endpoint. The official Pro OpenAPI contains `GET /api/inflows/{protocol}/{timestamp}`; direct anonymous access returns an API-key error. This endpoint is protocol-level historical inflow/outflow and is not available through the free API.

Decision: data-access blocked before return inspection. Do not scrape frontend-private state or reconstruct incomplete historical token balances. Revisit only if an official auditable historical CEX token-balance/inflow source becomes available.
