# Protocol

1. Exhaustively scan Binance official CMS catalogId=48 for 2023-2024 titles containing "Trading Bots".
2. Parse article body mechanically. New-pair section is the text from "open trading for" to the bot-service section. Bot section begins at "enable Trading Bots services" (or equivalent "Trading Bots services for") and ends before "Start Trading".
3. Extract BASE/QUOTE pairs from each section. A token base is eligible as a raw bot-support event only if it appears in the bot section and the same base does not appear in that article's new-pair section.
4. Deduplicate the same base within one article. Do not split Spot Grid / Spot DCA / Rebalancing Bot into separate events.
5. Exclude fiat/stablecoin bases (USDT, USDC, FDUSD, TUSD, USDP, BUSD, DAI, EUR, TRY and other obvious fiat quote units used as base).
6. Direction is preregistered LONG: adding an automated trading access surface may increase persistent trading participation/liquidity for an already-listed token.
7. Before any post-event return is read, require same-name Binance USD-M history >=730 days at article publish time and prior 24 complete 1h QuoteVolume >=5M USDT.
8. Coverage gate: >=8 eligible token-events, >=8 unique tokens, and >=5 independent article batches.
9. Failure freezes the family without return inspection. Do not lower the 2-year/QV gates or merge new-pair bases back into the sample.
10. If coverage passes, freeze discovery outcome protocol before reading returns; 2025+ remains untouched unless later discovery gates pass.
