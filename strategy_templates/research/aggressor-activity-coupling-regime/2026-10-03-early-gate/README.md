# v176 Aggressor-Activity Coupling Regime — Early Gate

Mechanism: use completed Binance USD-M 1h klines. Signed taker flow is `2*TakerBuyQuoteVolume/QuoteAssetVolume - 1`; activity change is `log(QuoteAssetVolume_t/QuoteAssetVolume_(t-1))`. Over the latest 48 complete hours compute Pearson correlation between signed aggressor flow and activity change. A fresh zero up-cross triggers LONG, a fresh zero down-cross triggers SHORT. Entry is next complete 1h open.

No correlation magnitude threshold, z-score, current-flow threshold, price-trend confirmation, funding, OI, QPS, trade-count, ATR, time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **4,977 events**.
- Frequency **7.9432 events/symbol/week**.
- 1h signed mean **+0.0062%**.
- 4h signed mean **+0.0369%**.
- 12h signed mean **+0.0466%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.0271%**.
- 2024 12h **+0.0669%**.
- LONG audit: 12h **+0.2052%**.
- SHORT audit: 12h **-0.1117%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic gate fails by a wide margin. The side split is post-result attribution only and does not permit deleting SHORT.

Decision: **freeze v176**. Do not scan windows, add correlation thresholds, substitute QuoteVolume level for change, delete one direction, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
