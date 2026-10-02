# Binance Corporate-Action Support → USD-M LONG — 2023-2024 Feasibility

Hypothesis: Binance's first explicit support announcement for a token swap, migration, rebranding, or redenomination removes execution/migration uncertainty for existing holders and therefore defines a preregistered LONG event.

Official-source audit:
- Binance CMS catalogId=49 (Latest Binance News), exhaustively scanned across 2023-2024.
- 1,071 titles were present in the period.
- 30 titles matched token-swap/migration/rebranding/redenomination keywords.
- Completion notices, follow-up updates, and the generic BNB Beacon Chain sunset/migration notice were excluded before market returns.
- 14 first-support token-events / 14 unique pre-action tickers remained: MATIC, FRONT, RNDR, STRAX, PLA, TVK, TOMO, MC, AVA, QUICK, COCOS, SXP, BNX, GTO.

Production eligibility was checked before any post-event return:
- Binance USD-M history >=730 days at announcement time.
- Prior 24 complete 1h QuoteVolume >=5M USDT.
- Coverage gate >=8 eligible events and >=8 unique tokens.

## Final result

Only **3/14** candidates pass production eligibility: **MATIC, TOMO, SXP**. The other 11 fail because the USD-M contract either did not exist or did not yet have two years of history at announcement time. All three eligible events also pass the 5M QuoteVolume threshold.

Coverage gate fails (3 < 8). No post-event return was read or computed.

Decision: **feasibility blocked / freeze**. Do not lower the two-year requirement, use post-migration replacement-ticker history, merge completion notices as separate events, or combine KuCoin corporate actions to manufacture coverage.
