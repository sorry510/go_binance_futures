# v183 24h Price-Monotonicity Regime Cross — Early Gate

Mechanism: over the latest 24 completed USD-M 1h closes, compute Spearman rank correlation between chronological rank 1..24 and Close rank. A fresh zero up-cross triggers LONG; a fresh zero down-cross triggers SHORT.

## 2023-2024 discovery

- **4,613 events**.
- Frequency **7.3623 events/symbol/week**.
- 1h signed mean **+0.0176%**.
- 4h signed mean **+0.0833%**.
- 12h signed mean **+0.0906%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0149%**.
- 2024 12h **+0.1704%**.
- LONG audit: 12h **+0.1203%**.
- SHORT audit: 12h **+0.0608%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic threshold fails. Both sides are positive, but the aggregate effect is still too small to promote.

Decision: **freeze v183**. Do not scan windows, add rank-correlation thresholds, replace Spearman with regression slope, delete one side, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
