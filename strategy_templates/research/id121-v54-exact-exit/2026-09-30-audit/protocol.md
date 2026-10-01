# Protocol

1. Source entry rules are the archived 2026-09-24 snapshots, not the live DB.
2. Only close_long and close_short code is replaced with exact ROI >= 8 || ROI <= -6.
3. ID121 and v54 use the same technology and same dataset per symbol.
4. Each strategy starts fresh from the same production-eligible timestamp per symbol.
5. Execution: leverage 4, fee 0.0005/side, slippage 5bps/side, single position.
6. Report both raw NetPnL PF and fixed-notional normalized PF = NetPnL/(EntryPrice*Quantity), plus annual and per-symbol results.
7. No threshold tuning, no entry modification, no eligibility change.
8. This is an audit, not a new alpha discovery. The result determines whether old candidate labels remain valid under the exact-exit interpretation.
