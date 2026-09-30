# Stablecoin Depeg Stress — Feasibility (Superseded)

This initial feasibility note concluded that DeFiLlama did not expose a direct historical stablecoin market-price series.

It is **superseded** by the later canonical research bundle:
`strategy_templates/research/defillama-chain-stablecoin-depeg-stress/2023-2026/`.

That later audit identified an outcome-independent same-source proxy:
`implicit_peg = totalCirculatingUSD.peggedUSD / totalCirculating.peggedUSD`,
then froze and tested a 7d stress-flow rule. Canonical 2023–2024 discovery failed (996 events / 11 symbols, 7d -0.4077%, 4/11 positive), so the family is frozen.

Use the canonical chain-level bundle for the final family decision. This feasibility file is retained only for audit history.
