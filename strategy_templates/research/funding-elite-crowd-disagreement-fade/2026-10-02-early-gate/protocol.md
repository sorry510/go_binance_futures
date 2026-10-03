# v159 Funding–Elite/Crowd Disagreement Fade

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. 2025+ remains unread unless the frozen gate passes.
2. Use project-local Binance USD-M funding settlements, Binance Vision USD-M 5m metrics, and project-local 1h Klines.
3. Only normal consecutive funding settlements separated by 7.5h-8.5h are eligible.
4. At settlement T, positioning score = log(count_toptrader_long_short_ratio / count_long_short_ratio) from the last completed 5m metrics observation in the UTC hour immediately preceding T. No post-T metrics are allowed.
5. A disagreement state exists when funding_rate * positioning_score < 0.
6. Direction is preregistered from the crowding interpretation:
   - funding > 0 and score < 0 -> SHORT;
   - funding < 0 and score > 0 -> LONG.
7. Trigger only on false->true disagreement transitions between consecutive eligible settlements; persistent disagreement does not retrigger.
8. Entry is the next complete 1h open strictly after T. Measure signed 1h/4h/12h log returns; 12h endpoint must remain in the same calendar year.
9. No funding magnitude threshold, positioning magnitude threshold, smoothing, price breakout, OI, premium, taker, or symbol-specific rule.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 symbols positive, >=0.30 events/symbol/week, and both 2023/2024 means >0.
11. Failure freezes the family: do not scan thresholds, trigger every persistent settlement, delete one direction, reverse the signal, or add filters.
