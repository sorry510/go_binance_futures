# v185 Participation–Ticket-Size Coupling Regime — Early Gate

Mechanism: over the latest 48 completed USD-M 1h bars, compute the Pearson correlation between changes in TradeCount and changes in average ticket notional (QuoteVolume/TradeCount). A fresh zero up-cross follows the completed trailing 4h price direction; a zero down-cross fades it.

## 2023-2024 discovery

- **295 events**.
- Frequency **0.4708 events/symbol/week**.
- 1h signed mean **-0.0135%**.
- 4h signed mean **-0.1038%**.
- 12h signed mean **+0.0239%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **-0.0091%**.
- 2024 12h **+0.1225%**.
- LONG audit: 12h **+0.1283%**.
- SHORT audit: 12h **-0.0917%**.

The economic/breadth gates fail and 2023 is negative.

Decision: **freeze v185**. Do not scan windows, add thresholds, replace correlation with regression elasticity, delete one side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
