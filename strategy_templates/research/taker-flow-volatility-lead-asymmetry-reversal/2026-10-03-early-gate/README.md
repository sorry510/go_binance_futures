# v175 Taker-Flow Volatility-Lead Asymmetry Reversal — Early Gate

Mechanism: over the latest 48 completed one-hour pairs, compute Pearson correlation between signed taker flow `2*TakerBuyQuoteVolume/QuoteAssetVolume-1` and next-hour squared return. A fresh zero up-cross triggers SHORT; a fresh zero down-cross triggers LONG. Entry is the next complete USD-M 1h open.

No flow-magnitude threshold, z-score, current-flow confirmation, funding, OI, QPS, ATR, time-of-day, trend confirmation or symbol-specific rule.

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

The preregistered economic and breadth gates fail, and neither discovery year is positive. The side split is post-result audit only and cannot be used to delete SHORT.

Decision: **freeze v175**. Do not scan windows, add correlation thresholds, use absolute return instead of squared return, add current-flow sign confirmation, delete one direction, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
