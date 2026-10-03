# DeFiLlama Protocol Open-Interest Momentum — Feasibility

Hypothesis: growth/decay in decentralized-derivatives protocol open interest could provide a fundamental demand/risk signal for the protocol token, distinct from Binance Futures OI.

Data audit used only free no-auth DeFiLlama legacy endpoints:
- `/overview/open-interest`;
- `/summary/open-interest/{protocol}`.

Mapping was deliberately strict: only each summary endpoint's own non-empty `symbol` was accepted. Protocol names, chains, CoinGecko IDs or manual guesses were not used to manufacture token mappings.

## Coverage audit

- 137 open-interest protocols from overview.
- 137/137 protocol summaries fetched; **0 fetch errors**.
- 10 symbols had at least 30 historical OI observations across 2023-2024.
- Same-name Binance USD-M archive was checked at 2022-12 and 2024-12 as a conservative proxy for possible >=730d production eligibility by end-2024.
- Only **DYDX** passes that same-name two-year plausibility check.
- GMX, JUP, THE and the remaining protocol symbols either started too late on Binance Futures or lack a same-name mature USD-M contract.

Coverage gate required >=8 unique safely mapped tokens.

## Decision

**Coverage blocked / freeze before returns.**

No protocol-token forward price return was read. Do not infer tokens from protocol names/chains, substitute protocol-underlying markets for protocol tokens, lower the two-year Binance rule, or change the universe to one-token/small-sample research.
