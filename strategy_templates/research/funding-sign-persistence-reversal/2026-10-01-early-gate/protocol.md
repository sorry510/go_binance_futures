# Protocol

1. Freeze the six-symbol 2023-2024 discovery cohort before reading returns.
2. Use only project-local Binance USD-M funding history and 1h Klines; Repository source is nil, so the run cannot fetch/import/write missing market data.
3. Restrict to normal funding cadence: adjacent funding settlements must be 7.5h to 8.5h apart. This explicitly excludes funding-interval compression, which is a separately frozen family.
4. A positive streak event occurs only when three consecutive regular-cadence funding settlements are >0 and the immediately preceding settlement does not extend the same regular-cadence streak. Direction is SHORT.
5. A negative streak event is symmetric for three settlements <0. Direction is LONG.
6. Zero funding breaks the streak. Funding magnitude is ignored.
7. Three settlements is fixed because it corresponds to one normal 24h funding cycle; no 2/4/5-streak scan is permitted.
8. Enter at the next full 1h open after the third settlement, then measure causal signed 1h/4h/12h returns.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
10. Failure freezes the family: no funding magnitude threshold, no streak-length scan, no trend/taker/OI filter, no reversal to continuation.
11. 2025+ OOS remains unread unless early gate passes. Strict TP8/SL6 Engine is run only after early-gate promotion.
