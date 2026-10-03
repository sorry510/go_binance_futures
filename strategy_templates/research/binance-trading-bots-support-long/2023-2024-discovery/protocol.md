# Discovery Protocol

1. Event universe and eligibility are frozen by ../2023-2024-feasibility before any post-event return. Copy the 26 eligible events unchanged.
2. Direction is LONG for every event. No token/category/bot-type/severity filtering is allowed.
3. Signal time is Binance official article publishDate. Entry is the first whole-hour Binance USD-M 1h open strictly after publishDate; an article at 09:00:02 cannot use the 09:00 open.
4. Measure raw signed log returns from entry open to the close after 1h, 4h and 12h. These are only an early economic gate, not final TP8/SL6 performance.
5. Gate: event-weighted 12h mean >= +0.50%; batch-equal 12h mean >= +0.50%; >=60% of unique token means positive; >=60% of independent article-batch means positive; 2024 event-weighted 12h mean >0.
6. 2023 has only BTC+ETH from one independent batch, so it is reported transparently but is not a standalone annual sign gate.
7. Failure freezes the family: do not change horizon, bot subtype, pair quote, token list, direction, entry delay or thresholds; do not inspect 2025+.
8. Only if discovery passes may an exact replay be defined with leverage=4, TP=8, SL=6, fee=0.0005/side, slippage=5bps/side and single-position semantics.
