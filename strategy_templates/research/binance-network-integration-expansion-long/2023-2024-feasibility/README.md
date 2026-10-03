# Binance Network Integration Expansion → LONG — Feasibility

Hypothesis: adding a new Binance deposit/withdraw network route for an already-listed token increases transfer accessibility and may act as a positive catalyst.

The frozen title rule used only official 2023-2024 "Binance Completes Integration of ..." announcements explicitly stating that deposits were opened. Web3 Wallet dApp integrations, stablecoins/fiat assets, and Mainnet-only titles without explicit deposit opening were excluded.

Audit:
- 26 raw token-events / 22 unique tokens / 26 independent batches.
- Production eligibility required same-name USD-M history >=730 days and prior 24 complete 1h QuoteVolume >=5M USDT.
- 8 events pass, spanning 8 independent batches but only **7 unique tokens**: BNB, BTC, DOT, ETH, FIL, GRT, SFP. ETH has two separate network-integration events.
- C98 fails only the QV gate (~4.55M); 17 events fail history/missing.

The preregistered gate requires >=8 events, >=8 unique tokens and >=8 batches. Unique-token coverage fails at 7.

Decision: **coverage blocked / freeze before returns**. Do not count repeated ETH integrations as an eighth token, lower the QV/history gates, or add ambiguous Mainnet-only/stablecoin events. No post-event return was read.
