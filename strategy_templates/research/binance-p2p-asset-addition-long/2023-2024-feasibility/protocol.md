# Protocol

1. Exhaustively scan Binance official CMS catalogId=49 across 2023-2024 for P2P titles.
2. Keep only explicit token-level additions where Binance P2P newly supports buying/selling an ordinary crypto asset.
3. Direction is preregistered LONG because first P2P support broadens fiat access/on-off-ramp utility for that token.
4. Deduplicate by crypto token: later support for another fiat currency, country, payment method, or P2P corridor is not a new token event.
5. Exclude stablecoin/fiat-only additions, merchant/fee promotions, payment-method changes, contests, and general P2P platform changes.
6. Before any post-event return is read, require the token's Binance USD-M history >=730 days at announcement time and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Coverage gate: >=8 eligible events and >=8 unique tokens. Failure freezes without return inspection.
8. Do not reinterpret repeated regional expansions as independent events or merge Spot/Convert/Pay additions to manufacture coverage.
