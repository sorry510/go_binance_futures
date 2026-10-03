# v171 Return–Future-Volatility Asymmetry Reversal — Early Gate

Mechanism: over the latest 48 completed one-hour return pairs, compute Pearson correlation between signed return r_t and next-hour squared return r_(t+1)^2. A fresh zero up-cross indicates upside shocks have become the volatility-generating side and triggers SHORT; a fresh zero down-cross indicates downside shocks have become the volatility-generating side and triggers LONG. Entry is the next complete USD-M 1h open.

No correlation magnitude threshold, z-score, return-size filter, funding, OI, volume, taker, ATR, time-of-day, trend confirmation or symbol-specific rule.

## 2023-2024 discovery

- **4,653 events**.
- Frequency **7.4261 events/symbol/week**.
- 1h signed mean **-0.0029%**.
- 4h signed mean **+0.0167%**.
- 12h signed mean **+0.0764%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **-0.00023%**.
- 2024 12h **+0.1503%**.
- LONG audit: 12h **+0.1545%**.
- SHORT audit: 12h **-0.0018%**.

Breadth and frequency pass, but the preregistered +0.20% economic gate fails and 2023 is not positive. The side split is audit-only and cannot be used to delete SHORT.

Decision: **freeze v171**. Do not scan windows, add correlation thresholds, replace squared return with absolute return, delete one direction, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
