# v164 Perp-Excess-Volatility Fade — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision Spot and USD-M 1h klines for the same symbol only; align by open timestamp.
3. For each completed hour t, compute 24 one-hour log returns for Spot and USD-M over t-23..t. Realized volatility is sqrt(sum(r^2)); no demeaning or annualization.
4. score = log(RV_perp_24h / RV_spot_24h). Require both realized volatilities >0 and exact 24h aligned history.
5. Trigger only when score crosses from <=0 to >0. This marks a fresh transition to derivatives volatility exceeding spot volatility.
6. Direction is a preregistered leverage-stress fade: trailing 24h USD-M close-to-close return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No volatility magnitude threshold, z-score, smoothing, basis, volume-share, funding, OI, taker, time-of-day, or symbol-specific filter.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and both 2023 and 2024 12h means >0.
10. Failure freezes this family: do not scan 12h/48h RV windows, add RV-ratio thresholds, test down-crosses as a separate mapping, delete one side, reverse to continuation, or add filters.
11. This is distinct from v161 basis momentum and v163 volume-share expansion. v164 uses only the relative realized-volatility state between perp and spot.
