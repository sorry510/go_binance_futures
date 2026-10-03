# v175 Taker-Flow Volatility-Lead Asymmetry Reversal — Early Gate

Mechanism: use completed Binance USD-M 1h klines. Signed taker flow is `2*TakerBuyQuoteVolume/QuoteAssetVolume - 1`. Over the latest 48 completed pairs compute Pearson correlation between signed taker flow at hour t and squared return at t+1. A fresh zero up-cross triggers SHORT; a fresh zero down-cross triggers LONG. Entry is the next complete 1h open.

## 2023-2024 discovery

- **4,880 events**.
- Frequency **7.7884 events/symbol/week**.
- 1h signed mean **-0.0046%**.
- 4h signed mean **+0.0033%**.
- 12h signed mean **-0.0156%**.
- Breadth **2/6 positive symbols**.
- 2023 12h **-0.0316%**.
- 2024 12h **-0.0009%**.
- LONG audit: 12h **+0.0634%**.
- SHORT audit: 12h **-0.0946%**.

Frequency is abundant, but expectancy, breadth and annual gates all fail. The side split is audit-only and cannot be used to delete SHORT.

Decision: **freeze v175**. Do not scan windows, add correlation or flow thresholds, use absolute-return volatility, add current-flow sign confirmation, delete one side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
