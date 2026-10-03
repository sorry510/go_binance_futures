# v180 24h Range-Occupancy Regime Cross — Early Gate

Mechanism: over the latest 24 complete USD-M 1h bars, define the rolling high-low midpoint and measure the fraction of closes above it. A fresh majority cross above 0.5 triggers LONG; below 0.5 triggers SHORT.

## 2023-2024 discovery

- **8,808 events**.
- Frequency **14.0575 events/symbol/week**.
- 1h signed mean **+0.0310%**.
- 4h signed mean **+0.0195%**.
- 12h signed mean **+0.0036%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **+0.0038%**.
- 2024 12h **+0.0034%**.
- LONG audit: 12h **+0.0810%**.
- SHORT audit: 12h **-0.0760%**.

The 12h effect is effectively zero and breadth fails. The side split is audit-only.

Decision: **freeze v180**. Do not move the 0.5 threshold, scan windows, volume-weight occupancy, require consecutive acceptance, add breakout confirmation, delete one side, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
