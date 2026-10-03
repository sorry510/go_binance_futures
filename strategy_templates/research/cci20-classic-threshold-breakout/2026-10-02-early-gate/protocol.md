# Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. 2025+ remains unread unless the frozen early gate passes.
2. Use completed Binance USD-M 1h Klines already available through the project historical repository.
3. CCI is calculated exactly as the project CalculateCCI formula: typical price=(High+Low+Close)/3, 20-bar mean, mean absolute deviation, divisor 0.015.
4. LONG only when completed 1h CCI crosses from <=+100 to >+100. SHORT only when it crosses from >=-100 to <-100.
5. Enter at the next completed 1h bar open; measure signed 1h/4h/12h log returns. The 12h endpoint must remain in the same calendar year as the signal.
6. No price breakout confirmation, EMA/ADX/ATR/funding/taker/QPS filter, symbol-specific rule, or time-of-day rule.
7. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
8. Failure freezes the CCI family: do not scan CCI period, +/-50 or +/-200 thresholds, use threshold re-entry/mean-reversion, delete a side, reverse direction, or add filters.
9. Only if early gate passes may a separate strict Engine strategy be frozen before OOS, with leverage=4, TP8/SL6, fee=0.0005/side, slippage=5bps/side, single-position, exact close ROI>=8 || ROI<=-6.
