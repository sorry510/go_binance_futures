# Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC over 2023-2024. 2025+ stays unread unless the frozen gate passes.
2. Use Binance Vision USD-M 1h premiumIndexKlines and 1h contract klines only.
3. Premium range = High - Low of a completed 1h premium-index bar.
4. A record-expansion state is true when current premium range is strictly greater than the maximum range of the previous 24 completed hours.
5. Trigger only on false -> true transition of that record-expansion state; consecutive record hours do not repeatedly trigger.
6. Direction is preregistered basis-crowding fade: premium close >0 => SHORT; premium close <0 => LONG; exact zero => no signal.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h log returns.
8. No premium magnitude threshold, z-score, funding, OI, taker, price trend, QPS, or symbol-specific filter.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes the family: do not scan lookback 12h/48h, replace record with multiplier/z-score, delete one direction, add funding/OI filters, or reverse to continuation.
