# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and 2023-2024 discovery before reading returns; 2025+ remains unread unless the gate passes.
2. Use Binance Vision USD-M 1h premiumIndexKlines. Only completed hourly premium bars are allowed.
3. The signal variable is the premium-index 1h close. A cross from <=0 to >0 emits SHORT; a cross from >=0 to <0 emits LONG.
4. Rationale is preregistered carry/reversion: positive perpetual premium implies rich perp / long-side funding pressure; negative premium implies cheap perp / short-side pressure.
5. Require the previous and current premium bars to be exactly one hour apart; gaps cannot manufacture a cross.
6. Enter at the next USD-M futures 1h open. Measure signed 1h/4h/12h price returns.
7. No premium magnitude threshold, z-score, re-arm threshold, trend, funding-rate magnitude, OI, taker, volume, or symbol-specific filter.
8. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan premium thresholds, 5m/4h intervals, add funding filters, delete one direction, or reverse to continuation.
10. This differs from Premium Index Extreme Reversal, which required a 24h causal z-score extreme |z|>=3 and re-arm; v148 tests the natural premium sign boundary only.
