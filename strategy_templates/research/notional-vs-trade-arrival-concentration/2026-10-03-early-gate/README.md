# v184 Notional-vs-Trade-Arrival Concentration Regime — Early Gate

Mechanism: over the latest 24 completed hours, compare the temporal HHI of QuoteVolume shares with the temporal HHI of TradeCount shares. A fresh zero up-cross of log(HHI_QV/HHI_Count) follows the completed trailing 4h price direction; a zero down-cross fades it.

## 2023-2024 discovery

- **1,068 events**.
- Frequency **1.7045 events/symbol/week**.
- 1h signed mean **-0.0897%**.
- 4h signed mean **-0.1088%**.
- 12h signed mean **-0.1091%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **-0.0728%**.
- 2024 12h **-0.2218%**.
- LONG audit: 12h **+0.2624%**.
- SHORT audit: 12h **-0.4962%**.

The preregistered expectancy, breadth and annual gates fail. The side split is post-result attribution only.

Decision: **freeze v184**. Do not scan windows, add HHI-ratio thresholds, substitute entropy/Gini, delete one crossing/side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
