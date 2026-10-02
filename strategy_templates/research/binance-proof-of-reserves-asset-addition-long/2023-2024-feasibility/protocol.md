# Protocol

1. Use Binance official CMS catalogId=49 and the archived exhaustive 2023-2024 title corpus.
2. Include only announcements that explicitly say one or more crypto assets are newly added to Binance's Proof-of-Reserves (PoR) system.
3. Exclude routine reserve attestations, Merkle/zk-SNARK methodology updates, audit reports, snapshot/report releases, stablecoin-only reserve commentary, and generic platform announcements.
4. A token is counted once, at its first explicit PoR-addition announcement. Multiple tokens in one announcement remain token-events but share one independent batch.
5. Stablecoins are excluded from directional trading.
6. Direction is preregistered LONG: first inclusion in PoR increases exchange-level custody transparency for the asset and removes a token-specific reserve-verification gap.
7. Before any post-event return is read, require the corresponding Binance USD-M contract to have >=730 days of history at announcement time and prior 24 complete 1h QuoteVolume >=5M USDT.
8. Coverage requires >=8 eligible token-events, >=8 unique eligible tokens, AND >=3 independent eligible announcement batches.
9. If coverage fails, freeze without reading returns. Do not split a single batch into fake independent samples or merge generic PoR reports.
10. If coverage passes, freeze the discovery return protocol before reading returns; no post-hoc token/subtype filtering.
