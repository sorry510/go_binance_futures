# Discovery Protocol

1. Copy the 137 eligibility-passing token-events unchanged from the frozen feasibility audit; 50 unique tokens / 11 independent Binance article batches.
2. Direction is SHORT for every event. No token, pool quote, batch, liquidity size, or repeat-event filtering is allowed.
3. Signal time is Binance official article publishDate, not the future effective pool-removal time. Entry is the first whole-hour Binance USD-M 1h open strictly after publishDate.
4. Measure raw signed log returns at 1h, 4h and 12h from entry open. This is an early economic gate, not exact TP8/SL6 performance.
5. Gate: event-weighted 12h mean >= +0.50%; batch-equal 12h mean >= +0.50%; >=60% of unique-token means positive; >=60% of independent article-batch means positive.
6. All 11 candidate announcements were published in 2023 (the article titled 2024-01-05 was published 2023-12-29), so no artificial calendar-year breadth gate is added. Report signal-month/batch concentration transparently.
7. Failure freezes this family: do not switch to effective removal time, remove repeated tokens, select particular pools/quotes, add liquidity-size/severity filters, change horizon, or reverse LONG.
8. Only if discovery passes may 2024+ later events be sought as OOS and an exact replay be defined with leverage=4, TP8, SL6, fee=0.0005/side, slippage=5bps/side and single-position semantics.
