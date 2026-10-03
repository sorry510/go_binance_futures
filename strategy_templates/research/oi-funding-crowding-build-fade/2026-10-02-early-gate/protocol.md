# Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ stays unread unless the frozen gate passes.
2. Use Binance Vision USD-M fundingRate settlements, 5m metrics sum_open_interest_value, and 1h klines.
3. Only normal funding cadence is eligible: the three settlements needed for previous/current OI growth must each be separated by 7.5h-8.5h.
4. At each settlement T, use the latest metrics observation <=T for OI notional. No future metrics row is allowed.
5. Define current OI growth = log(OI_T / OI_prevSettlement), previous OI growth = log(OI_prevSettlement / OI_prev2Settlement).
6. Trigger only on the natural state transition previous_growth <=0 and current_growth >0. No OI magnitude threshold or z-score.
7. Direction is preregistered crowding fade: current funding >0 -> SHORT; current funding <0 -> LONG; exactly zero funding -> no signal.
8. Enter at the next complete 1h open after T. Measure signed 1h/4h/12h returns; 12h endpoint must remain in the signal calendar year.
9. No price-return, taker, QPS, trend, premium, volatility, or symbol-specific filter.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
11. Failure freezes the family: do not scan OI windows, add OI/funding magnitude thresholds, delete one direction, switch to continuation, or inspect OOS.
12. This is distinct from prior OI-price state transition (direction came from price), OI×taker alignment, standalone funding extremes/streaks, and v152 funding-burden expansion.
