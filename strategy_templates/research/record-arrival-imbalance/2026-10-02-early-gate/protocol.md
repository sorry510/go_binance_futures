# v160 Record-Arrival Imbalance — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed project-local Binance USD-M 1h closes; no external data, MarketCondition, Benchmark, funding, OI, taker or NowTime modulo.
3. For each signal hour, scan the trailing 24 completed 1h closes in chronological order. The first close seeds both running high and running low and contributes no count.
4. Up-record count increments whenever a later close is strictly greater than every earlier close in that 24h window. Down-record count increments whenever a later close is strictly lower than every earlier close.
5. score = up-record count - down-record count.
6. score crossing from <=0 to >0 emits LONG; crossing from >=0 to <0 emits SHORT. No magnitude threshold, smoothing, re-arm threshold, or price-return filter.
7. Enter at the next complete 1h open. Measure signed 1h/4h/12h log returns.
8. Early gate: 12h event-weighted signed mean >=+0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes the family: do not scan 12h/48h windows, require count gaps, add trend/volume filters, delete a direction, or reverse.
10. Only if promoted will this exact fixed logic be translated into an Engine-compatible strategy representation before OOS.
11. The trailing 24h window and the required 12h forward path must consist of strictly consecutive 1h bars; any gap invalidates the event.
12. For annual-sign checks, the 12h endpoint must remain in the same calendar year as the signal.
