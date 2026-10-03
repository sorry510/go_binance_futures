# v186 Rolling-4h Extreme-Order Continuation — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h OHLC bars. No current partial bar.
3. Over the latest four completed 1h bars, locate the earliest occurrence of the maximum High and earliest occurrence of the minimum Low.
4. score = high_index - low_index, with indices 0..3 in chronological order.
5. score > 0 means the rolling path formed its low before its high; score < 0 means the high occurred before the low. score = 0 means both extremes occurred in the same 1h bar and is neutral.
6. Trigger only on fresh sign transitions: previous score <=0 and current score >0 -> LONG; previous score >=0 and current score <0 -> SHORT.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No range magnitude threshold, close-location/body/wick rule, volume, taker, funding, OI, ATR, time-of-day, trend overlay or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 3h/6h/8h windows, choose latest instead of earliest tied extrema after outcomes, add amplitude filters, delete one side, reverse the mapping, or inspect 2025+.
11. This is a path-topology mechanism, distinct from CLV/body-range, Kaufman efficiency, Spearman monotonicity, Donchian breakout and return magnitude.
