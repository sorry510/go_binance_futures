# v188 Corwin–Schultz Liquidity-Stress Reversal — Early Gate

Mechanism: compute the Corwin–Schultz two-period high-low bid-ask spread estimator on completed Binance USD-M 1h OHLC bars. Compare the mean estimator over the latest 24 completed hours with the immediately preceding non-overlapping 24h block. A fresh zero up-cross of log(current spread / prior spread) marks new liquidity stress; fade the completed trailing 4h price direction and enter at the next 1h open.

## 2023-2024 discovery

- **4,229 events**.
- Frequency **6.7494 events/symbol/week**.
- 1h signed mean **-0.0019%**.
- 4h signed mean **+0.0201%**.
- 12h signed mean **+0.0340%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0044%**.
- 2024 12h **+0.0628%**.
- LONG audit: 12h **+0.1446%**.
- SHORT audit: 12h **-0.0716%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails by a large margin. The side split is post-result attribution only.

Decision: **freeze v188**. Do not scan block lengths, substitute Abdi-Ranaldo as a rescue estimator, add spread thresholds, test contraction separately, delete one price direction, invert to continuation, or inspect 2025+. No strict Engine run and no DB import.
