# v170 Quote-Volume Persistence Emergence Momentum — Early Gate

Mechanism: over the latest 48 complete Binance USD-M 1h bars, compute lag-1 Pearson autocorrelation of log QuoteAssetVolume. A fresh zero up-cross marks the emergence of serially persistent trading activity. Follow the completed trailing 4h price direction and enter at the next 1h open.

No volume magnitude threshold, z-score, QPS, trade-count, taker, funding, OI, volatility, time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **25 events**.
- Frequency **0.0399 events/symbol/week**.
- 1h signed mean **-0.0632%**.
- 4h signed mean **-0.2253%**.
- 12h signed mean **+0.5417%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **-0.1517%**.
- 2024 12h **+1.4242%**.
- SOL: 0 events; DOGE 2; LTC 3; AVAX 1; UNI 5; ZEC 14.
- LONG audit: 12h **+0.9508%**.
- SHORT audit: 12h **-0.0720%**.

The natural zero boundary almost never triggers because log-volume autocorrelation is generally positive. Frequency misses the preregistered 0.30/week threshold by nearly an order of magnitude, and 2023 is negative.

Decision: **freeze v170**. Do not introduce arbitrary autocorrelation thresholds, scan windows, switch to volume-change autocorrelation, delete one side, reverse to fade, or inspect 2025+. No strict Engine run and no DB import.
