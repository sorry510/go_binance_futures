# v182 Per-Trade Volatility-Impact Regime — Early Gate

Mechanism: for each completed 24h block, compute `sum(one-hour log-return^2) / sum(TradeCount)`, then compare the latest block with the immediately preceding 24h block. A fresh zero up-cross of log(impact_latest/impact_previous) fades the completed trailing 4h price direction; a zero down-cross follows it.

## 2023-2024 discovery

- **8,137 events**.
- Frequency **12.9865 events/symbol/week**.
- 1h signed mean **+0.0134%**.
- 4h signed mean **+0.0171%**.
- 12h signed mean **+0.0327%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.0250%**.
- 2024 12h **+0.0399%**.
- LONG audit: 12h **+0.2164%**.
- SHORT audit: 12h **-0.1548%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails by a wide margin. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v182**. Do not scan block lengths, add impact thresholds, substitute range for variance, delete one crossing/side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
