# v163 Spot–Perp Volume-Share Expansion Confirmation — Early Gate

Mechanism: compare the same-symbol USD-M/Spot quote-volume ratio over the latest 24 completed 1h bars with the immediately preceding 7-day baseline. A fresh zero up-cross of `log(current_ratio / baseline_ratio)` marks derivatives participation expanding above baseline. Direction follows the already-observed trailing 24h USD-M close-to-close return: positive -> LONG, negative -> SHORT.

This uses only Binance Vision Spot + USD-M 1h data, same symbol, timestamp aligned. No basis, funding, OI, taker, threshold magnitude, time-of-day or symbol-specific parameter.

## 2023-2024 discovery

- **2,066 events**.
- Frequency: **3.2973 events/symbol/week**.
- 1h signed mean: **+0.0279%**.
- 4h signed mean: **-0.0636%**.
- 12h signed mean: **-0.0602%**.
- Breadth: **1/6 positive symbols**.
- 2023 12h: **-0.1874%**.
- 2024 12h: **+0.0658%**.
- LONG: 12h **+0.0710%**.
- SHORT: 12h **-0.1860%**.

The event frequency is sufficient, but the economic/breadth gates fail and annual sign flips. The side split is post-result audit only and does not permit deleting SHORT.

Decision: **freeze v163**. Do not scan participation/baseline windows, add magnitude thresholds, test contraction as a separately mapped direction, delete SHORT, reverse the price mapping, inspect 2025+, or run strict TP8/SL6.
