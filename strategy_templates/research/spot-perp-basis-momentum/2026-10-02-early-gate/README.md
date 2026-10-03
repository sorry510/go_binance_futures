# v161 Spot–Perp Basis Momentum — Early Gate

Mechanism:
- basis = log(USD-M perpetual 1h close / Binance Spot 1h close);
- momentum = basis_now - basis_24h_ago;
- zero up-cross -> LONG; zero down-cross -> SHORT;
- next complete USD-M 1h open;
- no magnitude threshold, z-score, smoothing, funding/OI/taker/price filter or symbol-specific rule.

Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. Spot and perpetual bars are timestamp-aligned. 2025+ was not requested or read by this replay.

## Final result

- **42,516 events**.
- Frequency **67.855 signals/symbol/week**.
- 1h signed mean **-0.0127%**.
- 4h signed mean **-0.0151%**.
- 12h signed mean **-0.0159%**.
- **0/6 symbols positive** at 12h.
- 2023 12h **-0.0168%**.
- 2024 12h **-0.0150%**.
- LONG audit: 21,270 events, 12h **+0.0708%**.
- SHORT audit: 21,246 events, 12h **-0.1028%**.

The combined preregistered rule fails the economic and breadth gates, and both years are negative. The side split is post-result audit only and does not permit deleting SHORT. Reversing the entire rule would imply only about +0.016% before costs, economically irrelevant.

Decision: **freeze v161 / basis-momentum continuation family**. Do not scan 6h/12h/48h basis-momentum windows, add magnitude thresholds, delete a side, reverse to mean reversion, inspect 2025+, or run strict TP8/SL6.
