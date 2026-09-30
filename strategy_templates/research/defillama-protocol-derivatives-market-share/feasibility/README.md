# DeFiLlama Protocol Derivatives Market-Share — Feasibility

Hypothesis: changes in a decentralized perpetual protocol's share of aggregate derivatives volume could proxy protocol usage and fee-source competitiveness and may lead the protocol token.

No returns were inspected. Feasibility was checked before any signal replay.

The public DeFiLlama protocols directory still exposes derivatives adapter mappings for examples including dYdX, GMX, Gains Network, and Synthetix. However, both the aggregate derivatives overview endpoint and tested single-protocol derivatives summary endpoints now return the paid-API message instead of historical series.

Without a complete aggregate denominator and protocol-level historical daily volume, a mechanically auditable market-share history cannot be reconstructed from the currently accessible public API.

Decision: data-access blocked. Do not scrape visualization pages, combine partial third-party histories, or substitute current snapshots. No discovery/OOS/exact replay and no DB import.
