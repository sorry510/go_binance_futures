# Corrected Feasibility Protocol

1. Supersedes the prior incomplete census that only saw the 2021-08-12 combined Tick Size + Step Size notice.
2. Census all official Binance Spot Step Size notices from 2021-2024, including the two exact 2024 titles now present in the archived official 2023-2024 corpus.
3. Parse only Step Size / quantity-increment rows, never Tick Size rows from mixed notices.
4. Use BASE/USDT spot rows only; exclude leveraged UP/DOWN tokens, stablecoin/fiat bases, and rows where before==after.
5. Pre-register mapping before returns: step-size decrease -> LONG USD-M (finer quantity granularity / reduced execution friction); step-size increase -> SHORT USD-M.
6. Before any post-event return, require same-name USD-M history >=730 days at effective time and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Coverage gate requires >=8 eligible token-events, >=8 unique tokens, and >=3 independent official announcement batches.
8. Failure freezes the family without return inspection. Do not count separate rows/effective sub-times inside one announcement as independent information batches, lower the history/QV rule, or merge Tick Size changes.
9. Only if coverage passes may a discovery protocol be frozen before returns.
