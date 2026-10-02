# Protocol

1. Source is Binance official CMS catalogId=49, exhaustively scanned over 2023-2024.
2. Include only the first announcement explicitly stating Binance will support a token swap, token migration, rebranding, or redenomination.
3. Exclude completion notices, resume-trading/deposit notices, Futures/Margin/Earn product-only updates, generic network upgrades/hard forks, airdrops, and delistings.
4. Event ticker is the pre-action token that already exists when the announcement becomes public. One token per corporate action is counted once; duplicate/update announcements are deduplicated.
5. Direction is preregistered LONG: explicit exchange support removes migration/execution uncertainty for existing holders.
6. Before any post-event return is read, require Binance USD-M history >=730 days at announcement time and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Coverage gate: >=8 eligible event-symbols and >=8 unique tokens. Failure freezes the family without return inspection.
8. If coverage passes, discovery return protocol is fixed before returns are read; no subtype/symbol/date selection may be introduced afterward.
9. Do not lower the two-year requirement or merge KuCoin/cross-exchange corporate actions into this Binance-native family.
