# Chronological Discovery Protocol

1. The 137 eligibility-passing token-events and 11 independent Binance Liquid Swap removal batches are frozen by the feasibility audit before returns.
2. Direction is SHORT for every token-event. No token/pair/stable-quote/severity/pool-count filtering is allowed.
3. Signal time is official Binance article publishDate. Entry is the first whole-hour Binance USD-M 1h open strictly after publishDate.
4. Measure raw signed log returns from entry open to closes after 1h, 4h, and 12h. These are an early economic gate, not final TP8/SL6 performance.
5. Because all 11 publishDate batches occur in 2023, temporal validation is frozen chronologically instead of pretending to have a 2024 sample.
6. Stage A = first 6 batches by publishDate (2023-07-28 through 2023-10-20), 66 eligible token-events. Stage B = last 5 batches (2023-11-03 through 2023-12-29), 71 events.
7. Stage A gate: event-weighted 12h mean >= +0.50%, batch-equal 12h mean >= +0.50%, >=60% of unique-token means positive, and >=60% of independent batch means positive.
8. Stage B post-event prices must remain unread unless Stage A passes. If Stage A passes, Stage B must independently satisfy the same four thresholds.
9. Failure at either stage freezes the family. Do not delete weak tokens/batches, select only USDT pools, change horizon/entry delay, use effective-removal time instead of announcement time, reverse to LONG, or tune thresholds.
10. Only if both stages pass may later-year event coverage/OOS and an exact replay be defined with leverage=4, TP=8, SL=6, fee=0.0005/side, slippage=5bps/side, and single-position semantics.
