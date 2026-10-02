# Protocol

1. Keep the previously frozen production-eligible event cohort unchanged: 9 events / 8 symbols.
2. Do not select events according to whether a precise timestamp is easy to find.
3. Require a verifiable UTC unlock timestamp for every fixed event before any exact 1m replay.
4. Date-only records are insufficient because an assumed 00:00 UTC changes entry and TP8/SL6 first-hit paths.
5. A timestamp inferred from a recurring schedule is insufficient unless the historical event itself is independently documented.
6. Prefer project/on-chain/block-explorer records or a provider event record carrying an explicit timestamp.
7. If complete timestamp coverage cannot be obtained from auditable accessible sources, freeze as coverage blocked and do not run exact returns.
8. Do not substitute reconstructed hourly templates from the paper repository for raw market observations.
