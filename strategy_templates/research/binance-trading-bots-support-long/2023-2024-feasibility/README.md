# Binance Trading Bots Support → LONG — Feasibility

Official Binance CMS catalogId=48 was exhaustively scanned across 2023-2024. All 64 titles containing Trading Bots were audited at body level.

The event definition excludes any base that also appears in the same article's new Spot-pair opening section, so the family isolates bot-service support for already-listed pairs. Fiat/stablecoin bases are excluded. Spot Grid / Spot DCA / Rebalancing Bot are not split into separate events.

Audit result:
- 64/64 articles parsed.
- 84 raw token-events / 58 unique tokens / 33 independent batches.
- Strict production eligibility: same-name USD-M history >=730 days and prior 24 complete 1h QuoteVolume >=5M USDT.
- 26 eligible events / 13 unique tokens / 13 batches.
- All 58 exclusions were history<730d or missing; no QV failure.
- Eligible tokens: ADA, AVAX, BCH, BNB, BTC, CHZ, DOGE, ETH, LINK, LTC, MASK, SOL, XRP.

Coverage gate passes. No post-event return was read during feasibility. Discovery is frozen separately in ../2023-2024-discovery/.
