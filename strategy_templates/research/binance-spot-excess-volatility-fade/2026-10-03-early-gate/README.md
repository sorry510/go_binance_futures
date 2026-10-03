# v169 Binance-Spot Excess-Volatility Fade — Early Gate

Mechanism: align same-symbol Binance Spot and Binance USD-M Index Price 1h bars. Over the latest 24 hours compute realized volatility as sqrt(sum(one-hour log-return^2)). A fresh zero up-cross of log(RV_spot/RV_index) triggers, meaning Binance's own spot venue has just become more volatile than the external composite Index Price. Fade the trailing 24h Binance Spot direction and enter next complete USD-M 1h open.

## 2023-2024 discovery

- **3,841 events**.
- Frequency **6.1302 events/symbol/week**.
- 1h signed mean **+0.0098%**.
- 4h signed mean **-0.0085%**.
- 12h signed mean **+0.0273%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.0089%**.
- 2024 12h **+0.0452%**.
- LONG audit: 12h **+0.2358%**.
- SHORT audit: 12h **-0.1676%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails by a wide margin. The side split is post-result audit only.

Decision: **freeze v169**. Do not scan RV windows, add ratio thresholds, delete one side, reverse to continuation, combine with Spot-vs-Index price-level filters, inspect 2025+, or run strict TP8/SL6.
