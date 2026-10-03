# v187 Taker-vs-Global-Account Skew — Early Gate

Mechanism: sample the final completed Binance USD-M 5m metrics row of each UTC hour and compute:

`score = log(sum_taker_long_short_vol_ratio / count_long_short_ratio)`.

A fresh zero up-cross means aggressive taker flow has become more long-biased than the global account headcount and triggers LONG; a fresh zero down-cross triggers SHORT. Entry is the next complete USD-M 1h open.

No magnitude threshold, smoothing, price trend, funding, OI, QPS, ATR, time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **13,224 events**.
- Frequency **21.1053 events/symbol/week**.
- 1h signed mean **-0.0024%**.
- 4h signed mean **+0.0067%**.
- 12h signed mean **+0.0085%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0081%**.
- 2024 12h **+0.0089%**.
- LONG audit: 12h **+0.1022%**.
- SHORT audit: 12h **-0.0852%**.

Frequency, breadth and annual sign consistency are adequate, but the preregistered +0.20% economic threshold fails by a very large margin. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v187 and close the natural pairwise positioning audit space**. Do not substitute Top Position/Top Account after outcomes, add thresholds, delete one side, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
