# v172 Taker-vs-Top-Position Skew — Early Gate

Mechanism: sample only the final completed Binance USD-M 5m metrics row of each UTC hour. score = log(sum_taker_long_short_vol_ratio / sum_toptrader_long_short_ratio). A zero up-cross means aggressive flow has become more long-biased than capital-weighted top-trader positioning and triggers LONG; zero down-cross triggers SHORT. Entry is next complete USD-M 1h open.

## 2023-2024 discovery

- **29,336 events**.
- Frequency **46.8199 events/symbol/week**.
- 1h signed mean **-0.0031%**.
- 4h signed mean **-0.0048%**.
- 12h signed mean **+0.0014%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **-0.0023%**.
- 2024 12h **+0.0083%**.
- LONG audit: 12h **+0.0922%**.
- SHORT audit: 12h **-0.0893%**.

The aggregate effect is effectively zero, breadth fails and 2023 is negative. The side split is audit-only and cannot justify deleting SHORT.

Decision: **freeze v172**. Do not scan ratio thresholds, smooth either series, substitute global/top-account positioning ratios, add price/funding/OI filters, delete one side, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
