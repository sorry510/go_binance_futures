# v164 Perp-Excess-Volatility Fade — Early Gate

Mechanism:
- align same-symbol Binance Spot and USD-M completed 1h bars;
- over the latest 24 hours, compute realized volatility as sqrt(sum(one-hour log-return^2)) separately for perp and spot;
- score = log(RV_perp / RV_spot);
- trigger only on a fresh zero up-cross, meaning derivatives volatility has just exceeded spot volatility;
- fade trailing 24h USD-M price direction: positive 24h return -> SHORT, negative -> LONG;
- enter at the next complete USD-M 1h open.

No RV-ratio magnitude threshold, z-score, smoothing, basis, volume-share, funding, OI, taker, time-of-day or symbol-specific filter.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

- **3,606 events**.
- Frequency **5.7551 events/symbol/week**.
- 1h signed mean **+0.0114%**.
- 4h signed mean **+0.0792%**.
- 12h signed mean **+0.1082%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0143%**.
- 2024 12h **+0.2034%**.
- LONG audit: 12h **+0.2750%**.
- SHORT audit: 12h **-0.0582%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails materially. The side split is post-result audit only and does not permit deleting SHORT.

Decision: **freeze v164**. Do not scan 12h/48h RV windows, add RV-ratio thresholds, test down-crosses as a separate direction, delete one side, reverse to continuation, add filters, inspect 2025+, or run strict TP8/SL6.
