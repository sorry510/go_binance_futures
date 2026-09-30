# DeFiLlama Stablecoin Borrow-Cost / Leverage-Demand — Feasibility

Hypothesis: chain-level stablecoin borrowing costs and utilization can proxy leveraged dollar demand. A broad rise in stablecoin borrow cost/utilization could indicate increasing leverage demand and potentially lead the chain native token.

No returns were inspected.

The free DeFiLlama yields pool catalog is currently accessible and contains approximately 17k pools, while the normal chart/{pool} endpoint provides historical deposit-side APY and TVL. However, the current free pool schema does not expose borrow APY or total borrow fields.

A tested Aave V3 USDT pool has free chart history back to 2023, but the dedicated chartLendBorrow/{pool} endpoint returns HTTP 402 paid API required. Substituting deposit APY would change the mechanism, and reconstructing historical borrow-cost indexes from today's surviving pool catalog would introduce survivorship and pool-selection bias.

Decision: data-access/data-quality blocked before any return inspection. Do not substitute deposit yield for borrow cost, do not hand-pick current pools, and do not proceed to discovery/OOS/exact replay.
