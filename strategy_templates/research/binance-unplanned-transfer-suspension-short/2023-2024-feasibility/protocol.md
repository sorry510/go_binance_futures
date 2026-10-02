# Protocol

1. Exhaustively scan Binance Maintenance Updates catalogId=157 for 2023-2024 titles indicating deposit/withdrawal suspension, pause, or disabling.
2. Exclude planned network upgrades and routine maintenance notices; retain only an already-occurring disruption/security situation.
3. Parse only explicitly named affected tokens from the official body.
4. Use official publication time as the information timestamp; do not backdate to an earlier operational suspension unless an auditable contemporaneous announcement is present in the same fixed census.
5. Direction is preregistered SHORT if production eligibility passes.
6. Before any return is read, require USD-M history >=730 days at publication and prior 24 complete 1h QuoteVolume >=5M USDT.
7. Require at least 8 eligible unique token-symbol observations. Failure freezes without post-event return evaluation.
8. A single common-incident batch is reported explicitly and cannot be represented as independent event-time replication.
