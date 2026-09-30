# Snapshot Governance Turnout Shock — Feasibility

Hypothesis: unusually high governance participation at proposal close may represent protocol attention / commitment and could affect the token after the vote is complete.

Causality rule: final scores_total may only be used at proposal end, never at proposal creation. Any eventual signal would compare log(scores_total) against prior already-closed proposals from the same Snapshot space.

Before inspecting returns, the existing DeFiLlama-governanceID safe mapping was audited. The proposal corpus contains only seven mapped Binance USD-M symbols: APE, ENS, GTC, LDO, LINA, LRC and REN. LINA/LRC/REN are additionally very sparse.

Decision: coverage blocked below the standing minimum of eight symbols. No returns inspected; do not broaden identity mapping or use ambiguous Snapshot spaces.
