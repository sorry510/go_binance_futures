# v166 Spot–Perp Tracking-Error Expansion Fade — Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ stays unread unless the frozen gate passes.
2. Align same-symbol Binance Spot and USD-M completed 1h bars by open timestamp.
3. Define hourly residual e_t = log(perp_close_t/perp_close_(t-1)) - log(spot_close_t/spot_close_(t-1)).
4. Current tracking error = RMS of e over the latest 24 completed hours. Baseline tracking error = RMS over the immediately preceding 168 completed hours.
5. score = log(current_tracking_error / baseline_tracking_error). Require exact continuous history and both RMS values >0.
6. Trigger only when score crosses from <=0 to >0. This marks a fresh expansion of perp-vs-spot tracking error above its preceding one-week baseline.
7. Direction is preregistered convergence fade: cumulative residual over the latest 24h >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
8. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h perp returns. The 12h endpoint must remain in the signal calendar year.
9. No tracking-error magnitude threshold, z-score, smoothing, basis-level filter, funding, OI, taker, volume, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes the family: do not scan 12h/48h current windows, 3d/14d baselines, add thresholds, delete a side, reverse to continuation, or combine with v161/v164 filters.
