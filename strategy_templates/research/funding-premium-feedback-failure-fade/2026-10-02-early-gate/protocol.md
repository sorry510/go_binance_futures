# Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. 2025+ remains unread unless the frozen gate passes.
2. Use only normal consecutive 8h Binance USD-M funding settlements, Binance Vision 1h premiumIndexKlines, and USD-M 1h Klines.
3. At a settlement T, observe the first complete premium-index 1h bar after settlement (the bar opened at T and closed by T+1h).
4. feedback_failure(T) is true iff funding_rate(T) and that post-settlement premium close are both nonzero and have the same sign.
5. Trigger only when the previous consecutive normal 8h settlement had feedback_failure=false and the current settlement has feedback_failure=true. This is a state activation, not a repeated signal.
6. Direction is preregistered basis fade: positive premium/funding -> SHORT; negative premium/funding -> LONG.
7. Enter at the USD-M 1h open at T+1h, after the observed premium bar is complete. Measure signed 1h/4h/12h returns.
8. Require current and previous funding settlements to be 7.5h-8.5h apart. The 12h endpoint must remain in the same calendar year as the signal.
9. No premium/funding magnitude threshold, z-score, OI/taker/QPS/price/trend filter, smoothing, or symbol-specific rule.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
11. Failure freezes the family: do not scan post-settlement observation windows, add thresholds/filters, delete one direction, or reverse the signal.
12. This is distinct from v148 premium zero-cross and v154 funding flat-zone activation: it measures whether a funding payment fails to remove the basis sign one hour later.
13. Discovery warmup is conservative: no pre-2023 premium state is inferred. A signal is allowed only after two consecutive normal 8h settlements within available discovery data both have a fully observed feedback state.
