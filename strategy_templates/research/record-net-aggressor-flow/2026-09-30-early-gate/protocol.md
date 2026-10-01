# Protocol

1. Freeze entry and exit JSON before returns.
2. Compute completed-hour net aggressive quote flow as `2*TakerBuyAmount-Amount`.
3. Trigger only when the absolute signal-hour flow exceeds every one of the prior eight completed hours.
4. LONG if signal flow >0, SHORT if <0; current price must break the signal-hour high/low.
5. Strict fixed exits: `ROI >= 8 || ROI <= -6`.
6. Core-4 early gate only.
7. Failure => no 4h/12h lookback scan, no sigma/multiplier threshold, no trend/QPS/funding filter, no reversal.
8. Pass => freeze parameters and expand symbols before holdout.
