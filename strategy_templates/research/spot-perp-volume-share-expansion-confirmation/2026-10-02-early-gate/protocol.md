# v163 Spot–Perp Volume-Share Expansion Confirmation — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use Binance Vision Spot and USD-M 1h klines for the same symbol only; align bars by open timestamp. No cross-symbol Benchmark/MarketCondition is used.
3. For completed hour t:
   - current_ratio = sum(USD-M QuoteVolume over t-23..t) / sum(Spot QuoteVolume over t-23..t);
   - baseline_ratio = sum(USD-M QuoteVolume over t-191..t-24) / sum(Spot QuoteVolume over t-191..t-24);
   - score = log(current_ratio / baseline_ratio).
4. Trigger only when score crosses from <=0 to >0. This represents a new expansion of derivatives participation above the immediately preceding seven-day baseline.
5. Direction follows the same-symbol trailing 24h USD-M close-to-close return: positive -> LONG, negative -> SHORT; exact zero -> no signal.
6. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
7. No volume-share magnitude threshold, price-return threshold, z-score, smoothing, funding/OI/taker filter, time-of-day rule or symbol-specific parameter.
8. Early gate: 12h signed mean >= +0.20%, >=4/6 symbols positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this family: do not scan 6h/12h/48h participation windows, 3d/14d baselines, add thresholds, test contraction as a separately mapped direction, delete a side, or reverse the price-direction mapping.
10. The mechanism is distinct from v161 basis momentum: v163 uses relative quote-volume participation and only asks whether a fresh derivatives-share expansion confirms the already-observed same-symbol price direction.
