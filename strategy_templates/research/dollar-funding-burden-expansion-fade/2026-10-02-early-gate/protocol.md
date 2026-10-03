# Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only; 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision USD-M fundingRate monthly archives, 5m metrics daily archives, and 1h kline monthly archives.
3. Only normal funding settlements with funding_interval_hours=8 are eligible. The current settlement and the previous four settlements used for current/previous burden-ratio states must each be separated by 7.5h-8.5h.
4. For a settlement at time T, use the latest metrics observation strictly before or at T to get sum_open_interest_value. No future metrics row is permitted.
5. turnover8 = QuoteVolume from the eight complete 1h bars ending at T. burden = abs(funding_rate) * OI_notional / turnover8. Require all inputs >0.
6. baseline = mean burden of the immediately previous three eligible settlements. burden_ratio = current burden / baseline.
7. Trigger only when prior burden_ratio <=1 and current burden_ratio >1. Ratio 1 is the natural prior-day burden boundary; no magnitude threshold is added.
8. Direction is a preregistered crowding fade: current funding >0 -> SHORT; current funding <0 -> LONG; zero funding -> no signal.
9. Enter at the next complete USD-M 1h open after T; measure signed 1h/4h/12h returns. The 12h close endpoint must remain in the same calendar year as the signal.
10. No funding z-score, OI threshold, price/taker/trend filter, symbol-specific rule, or side deletion.
11. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
12. Failure freezes the family: do not scan 2/4/6 settlement baselines, burden-ratio thresholds, alternative turnover windows, continuation direction, or inspect OOS.
13. This is economically distinct from standalone funding extremes, funding sign streak, funding volatility expansion, OI/turnover, and OI creation-efficiency studies: it measures aggregate funding transfer burden relative to market turnover.
