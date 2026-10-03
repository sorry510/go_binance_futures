# Protocol

1. Source is the already archived Binance official 2023-2024 title corpus; candidate titles must explicitly be "Notice Regarding the Removal of Selected Liquidity Pools on Binance Liquid Swap".
2. Fetch each official article body and parse only pairs listed as pools being removed. Do not include pool additions, reward changes, farming promotions, or generic Liquid Swap notices.
3. For every removed pair, include each non-stablecoin/non-fiat crypto asset appearing in the pair; deduplicate the same token within one article batch.
4. Stablecoins and fiat-like assets are excluded before market eligibility. No manual token selection after outcomes.
5. Direction is preregistered SHORT: removal of an exchange-operated liquidity venue is treated as a negative liquidity/distribution event.
6. Before any post-event return is read, require same-name Binance USD-M history >=730 days at official publish time and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Coverage gate requires >=8 eligible token-events, >=8 unique tokens, and >=5 independent article batches.
8. Failure freezes the family without return inspection. Do not lower history/QV thresholds, split one article into fake independent batches, or merge Liquid Swap additions.
9. Only if coverage passes may a separate discovery protocol be frozen before returns.
