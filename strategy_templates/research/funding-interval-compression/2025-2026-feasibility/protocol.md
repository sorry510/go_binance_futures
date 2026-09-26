# Protocol

Goal: determine whether Binance funding-interval compression can be tested under the project's production eligibility rules before looking at returns.

Primary event:
- first transition from repeated 8h/4h settlement to 1h settlement.
- proposed direction: opposite the funding sign that triggered the compression.

Fallback feasibility check:
- repeated 8h settlement -> 4h settlement.
- event must occur after the contract has at least 730 days of history.
- production liquidity floor is 5M USDT prior-24h quote volume if a replay is reached.

No history-age or liquidity rule may be relaxed to create samples.
