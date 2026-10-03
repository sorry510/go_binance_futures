# v167 OI / Market-Cap Leverage Expansion Fade — Early Gate

Mechanism:
- CoinMetrics Community daily `CapMrktEstUSD` is used uniformly for SOL/DOGE/LTC/AVAX/UNI/ZEC.
- Binance Vision USD-M 5m metrics provides `sum_open_interest_value`; only the final completed 5m row of each UTC day is used.
- Daily leverage ratio = OI notional / estimated USD market cap.
- score = log(current ratio / mean(previous 30 complete UTC-day ratios)).
- A fresh zero up-cross triggers.
- Direction is a preregistered crowding fade against the trailing 7 complete UTC-day USD-M return: positive trend -> SHORT, negative trend -> LONG.
- Entry is next UTC-day USD-M open.

No magnitude threshold, z-score, OI-change threshold, funding, taker, basis, volume-share, volatility, time-of-day or symbol-specific filter.

## 2023-2024 discovery

- **240 events**.
- Frequency **0.3830 events/symbol/week**.
- 1d signed mean **-0.5897%**.
- 3d signed mean **-0.5235%**.
- 7d signed mean **-1.7403%**.
- Breadth **1/6 positive symbols**.
- 2023 7d **-1.2042%**.
- 2024 7d **-2.2764%**.
- LONG: 7d **-1.7466%**.
- SHORT: 7d **-1.7355%**.

The preregistered fade is strongly negative across both years and both sides. Frequency barely clears the minimum, but expectancy and breadth fail decisively.

Decision: **freeze v167**. Do not reverse to continuation after seeing the result, scan 7/14/60d baselines, add leverage-ratio thresholds, delete one side, or combine with funding/OI filters. 2025+ remains unread; no strict Engine run and no DB import.
