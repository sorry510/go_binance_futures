# v192 Intrahour Volatility-Signature Noise Reversal — Early Gate

Mechanism: for each fully completed UTC hour, reuse the project's audited read-only 1m chunk history and compute realized variance at two sampling scales from the same price path:
- RV1m = sum of 60 squared one-minute log returns;
- RV5m = sum of 12 squared non-overlapping five-minute log returns using the same hour and starting close;
- score = log(RV1m / RV5m).

A fresh zero up-cross from <=0 to >0 means the finest sampling has just begun to report more realized variance than the 5m scale. Preregistered interpretation is microstructure-noise / short-cycle churning; fade the just-completed one-hour price direction and enter at the next complete USD-M 1h open.

No ratio threshold, rolling baseline, alternate sampling scale, skew/kurtosis, volume/taker/funding/OI filter, time-of-day or symbol-specific rule.

## Data audit

All six discovery symbols:
- **1,052,700 one-minute rows per symbol**.
- **17,543 complete formation hours per symbol**.
- Existing database history was read-only; no REST repair/import and no DB write.

## 2023-2024 discovery

- **23,871 events**.
- Frequency **38.0978 events/symbol/week**.
- 1h signed mean **+0.0134%**.
- 4h signed mean **+0.0424%**.
- 12h signed mean **+0.0800%**.
- Breadth **6/6 positive symbols**.
- 2023 12h **+0.0424%**.
- 2024 12h **+0.1174%**.
- LONG audit: 12h **+0.1656%**.
- SHORT audit: 12h **-0.0051%**.

This is unusually consistent in breadth and annual sign, but the preregistered +0.20% economic threshold fails materially. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v192**. Do not add RV-ratio thresholds, compare 1m/10m or 2m/5m, test the down-cross separately, add trend/filter conditions, delete one price direction, reverse to continuation, or inspect 2025+. No strict Engine run and no DB import.
