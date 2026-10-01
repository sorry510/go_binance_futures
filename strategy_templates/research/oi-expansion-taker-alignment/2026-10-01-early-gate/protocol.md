# Protocol

1. Freeze symbols, quarterly sample dates, 4h OI lookback, 4h taker aggregation, direction rule, and gate before returns.
2. Raw source is Binance Vision USD-M daily metrics only.
3. Price proxy is sum_open_interest_value / sum_open_interest, matching the prior OI research family.
4. OI event: current 4h log OI change >0 and previous observation's 4h log OI change <=0.
5. Taker direction: arithmetic mean of log(sum_taker_long_short_vol_ratio) across observations in the trailing 4h. Positive => LONG, negative => SHORT.
6. No taker magnitude threshold and no price-direction condition.
7. Evaluate signed 1h, 4h and 12h forward log returns.
8. Promote only if signed 4h mean >=0.10% and >=4/6 symbols have positive 4h mean.
9. Failure => no OI magnitude threshold, no 2h/6h/12h lookback scan, no opposite direction, no symbol filtering, no full-history download.
10. Pass => freeze this exact event definition and run full 2023-2024 discovery before any production/Engine work.
