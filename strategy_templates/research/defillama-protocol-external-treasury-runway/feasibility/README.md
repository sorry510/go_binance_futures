# DeFiLlama Protocol External-Treasury Runway — Feasibility

Hypothesis: growth or depletion of a protocol treasury's non-native-token assets changes operating runway and resilience, creating a protocol-token catalyst distinct from TVL, fees, or holder revenue.

No returns were inspected.

To avoid passive USD mark-to-market of the protocol's own token, only treasury branches that are not `OwnTokens` would be eligible for the eventual signal.

Universe construction was outcome-independent: start from the current DeFiLlama /protocols catalog entries with a non-null treasury adapter and a unique token ticker, then require a matching Binance USD-M contract. The mechanically identified mature-looking candidates were API3, CVX, ENS, GNS, HFT, LDO, PERP, and RSR. All eight treasury endpoints expose non-own-token historical series.

Production-history audit:
- API3USDT first monthly archive: 2022-02 -> eligible around 2024-02.
- CVXUSDT: 2022-09 -> eligible around 2024-09.
- ENSUSDT: 2021-11 -> eligible around 2023-11.
- GNSUSDT: no Binance USD-M archive.
- HFTUSDT: 2023-04 -> not 2y eligible until 2025.
- LDOUSDT: 2022-09 -> eligible around 2024-09.
- PERPUSDT: 2023-03 -> not 2y eligible until 2025.
- RSRUSDT: 2020-10 -> eligible throughout discovery.

Thus only five symbols can be production-eligible during 2023-2024: API3, CVX, ENS, LDO, RSR.

Decision: coverage blocked (<8) before return inspection. Do not manually add famous parent protocols such as UNI/AAVE merely because their treasury endpoint is known to exist; the parent/child catalog metadata does not provide a mechanically complete enumeration.
