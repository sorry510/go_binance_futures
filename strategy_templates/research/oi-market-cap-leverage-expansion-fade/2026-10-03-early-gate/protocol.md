# v167 OI / Market-Cap Leverage Expansion Fade — Protocol

1. Discovery universe is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC for 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Market cap uses CoinMetrics Community API `CapMrktEstUSD` uniformly for all six assets. Do not mix `CapMrktCurUSD`.
3. OI notional uses Binance Vision USD-M 5m metrics field `sum_open_interest_value`. For each UTC date, use only that date's final completed 5m metrics row.
4. Daily leverage ratio = OI_notional_USD / CapMrktEstUSD. Both values must be positive and aligned to the same UTC date.
5. Baseline is the arithmetic mean of the previous 30 complete UTC-day leverage ratios, excluding the current date.
6. score = log(current_ratio / baseline). Trigger only on a fresh zero up-cross from <=0 to >0.
7. Direction is preregistered leverage-crowding fade: trailing 7 complete UTC-day USD-M close-to-close return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
8. Entry is the next UTC-day USD-M open. Measure signed 1d/3d/7d returns. The 7d endpoint must remain in the signal calendar year.
9. No leverage-ratio magnitude threshold, z-score, OI-change threshold, funding, taker, basis, volume-share, volatility, time-of-day, or symbol-specific filter.
10. Early gate: 7d signed mean >= +0.25%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 7d means >0.
11. Failure freezes this family: do not scan 7/14/60d baselines, add ratio thresholds, delete one side, reverse to continuation, or combine with funding/OI filters.
12. This is distinct from OI/turnover and OI-change families: the denominator is the asset's estimated USD market capitalization, so the state measures derivatives notional leverage relative to the asset's economic size.
