# v186 Rolling-4h Extreme-Order Continuation — Early Gate

Mechanism: over the latest four completed Binance USD-M 1h bars, locate the earliest highest High and earliest lowest Low. score = high_index - low_index. A fresh transition from <=0 to >0 means the rolling path formed its low before its high and triggers LONG; >=0 to <0 means high before low and triggers SHORT. score=0 is neutral when both extremes occur in the same 1h bar.

No amplitude threshold, close-location/body/wick rule, volume, taker, funding, OI, ATR, time-of-day, trend overlay or symbol-specific rule.

## 2023-2024 discovery

- **26,136 events**.
- Frequency **41.7127 events/symbol/week**.
- 1h signed mean **+0.0024%**.
- 4h signed mean **+0.0091%**.
- 12h signed mean **-0.0124%**.
- Breadth **1/6 positive symbols**.
- 2023 12h **-0.0058%**.
- 2024 12h **-0.0190%**.
- LONG audit: 12h **+0.0377%**.
- SHORT audit: 12h **-0.0619%**.

The frequency is extremely high, but expectancy, breadth and both yearly gates fail. The side split is post-result attribution only.

Decision: **freeze v186**. Do not scan 3h/6h/8h windows, change tied-extrema handling after outcomes, add range/amplitude filters, delete one side, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
