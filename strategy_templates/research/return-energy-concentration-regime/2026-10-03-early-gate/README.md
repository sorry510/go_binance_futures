# v179 Return-Energy Concentration Regime — Early Gate

Mechanism: within each 24h block, define hourly return-energy weights from squared one-hour log returns and compute HHI. Compare the latest completed 24h block with the immediately preceding non-overlapping 24h block. A fresh zero up-cross of log(HHI_latest/HHI_previous) fades the trailing 4h price direction; a zero down-cross follows it.

## 2023-2024 discovery

- **9,361 events**.
- Frequency **14.9400 events/symbol/week**.
- 1h signed mean **-0.0045%**.
- 4h signed mean **+0.0077%**.
- 12h signed mean **-0.0238%**.
- Breadth **2/6 positive symbols**.
- 2023 12h **+0.0287%**.
- 2024 12h **-0.0713%**.
- LONG audit: 12h **+0.0981%**.
- SHORT audit: 12h **-0.1503%**.

The economic/breadth gates fail and the annual sign flips.

Decision: **freeze v179**. Do not scan block lengths, add HHI thresholds, substitute Gini/entropy, delete one crossing/side, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
