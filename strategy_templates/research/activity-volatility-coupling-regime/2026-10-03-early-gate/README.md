# v181 Activity–Volatility Coupling Regime — Early Gate

Mechanism: over the latest 48 completed USD-M 1h bars, compute Pearson correlation between log QuoteVolume change and one-hour squared log return. A fresh zero up-cross follows the completed trailing 4h price direction; a zero down-cross fades it.

## 2023-2024 discovery

- **453 events**.
- Frequency **0.7230 events/symbol/week**.
- 1h signed mean **-0.0330%**.
- 4h signed mean **-0.1085%**.
- 12h signed mean **-0.0021%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **+0.0508%**.
- 2024 12h **-0.0466%**.
- LONG audit: 12h **+0.1193%**.
- SHORT audit: 12h **-0.1188%**.

The economic/breadth gates fail and the annual sign flips. The side split is audit-only.

Decision: **freeze v181**. Do not scan windows, add correlation/volume thresholds, replace squared return with absolute return, delete one crossing/side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
