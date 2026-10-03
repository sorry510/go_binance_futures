# v180 24h Range-Occupancy Regime Cross — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h OHLC bars.
3. For the latest 24 complete hours, define range midpoint m = (max(High)+min(Low))/2.
4. occupancy = count(Close_i > m)/24. A close exactly at m is not counted above.
5. Recompute the same statistic one hour earlier using that earlier 24h window.
6. Fresh crossing from occupancy_prev <=0.5 to occupancy_now >0.5 -> LONG. Fresh crossing from occupancy_prev >=0.5 to occupancy_now <0.5 -> SHORT.
7. 0.5 is the natural majority/acceptance boundary; no magnitude threshold or re-arm threshold is used.
8. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
9. No trend, funding, OI, taker, volume, ATR, volatility, time-of-day or symbol-specific filter.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not scan 12h/48h windows, move the occupancy threshold, weight closes by volume/time, require consecutive acceptance, add breakout confirmation, delete one side, reverse the mapping, or inspect 2025+.
12. This differs from v110 Donchian midpoint cross: v110 used the terminal price crossing the midpoint; v180 measures path occupancy/acceptance across the full rolling window.
