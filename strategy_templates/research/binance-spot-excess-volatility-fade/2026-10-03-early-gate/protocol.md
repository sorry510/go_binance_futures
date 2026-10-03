# v169 Binance-Spot Excess-Volatility Fade — Protocol

1. Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ stays unread unless the frozen gate passes.
2. Align same-symbol Binance Spot 1h and Binance USD-M Index Price 1h bars by open timestamp.
3. Over the latest 24 completed hours, compute close-to-close one-hour log returns separately for Binance Spot and Index Price. RV = sqrt(sum(r^2)).
4. score = log(RV_spot / RV_index). Require both RV values >0 and exact continuous 24h aligned history.
5. Trigger only on a fresh zero up-cross from <=0 to >0, meaning Binance's own spot venue has just become more volatile than the external composite index.
6. Direction is preregistered local-overreaction fade: trailing 24h Binance Spot close-to-close return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
7. Entry is the next complete Binance USD-M 1h open. Measure signed 1h/4h/12h USD-M returns. The 12h endpoint must remain in the signal calendar year.
8. No RV magnitude threshold, z-score, smoothing, basis-level filter, funding, OI, taker, volume, time-of-day or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 12h/48h RV windows, add RV-ratio thresholds, test down-crosses, delete one side, reverse to continuation, or combine with Spot-vs-Index price-level filters.
11. This differs from the prior Spot-vs-Index Price Lead family, which used extreme price-level z-scores; v169 tests only relative realized-volatility state.
