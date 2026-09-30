# DeFi On-chain Liquidation Stress — Feasibility

Hypothesis: large lending-protocol liquidation waves can be an exogenous chain/protocol deleveraging signal distinct from centralized futures liquidation data.

No returns were inspected.

The legacy free DeFiLlama overview/summary liquidation endpoints were tested for aggregate, Aave V3 and Compound V3 histories and returned HTTP 500. Current DeFiLlama liquidation history is exposed on the paid/API-key surface. Open-source dimension adapters do contain protocol-specific on-chain liquidation reconstruction logic, so the data is theoretically reconstructible, but a proper 2023-2026 universe would require years of RPC/event-log replay across multiple protocols and chains.

Decision: data-access/data-cost blocked before returns. Do not substitute one protocol or one chain and treat it as a broad liquidation family.
