# v191 Intrahour 1m Realized-Skew Reversal — Early Gate

Mechanism: each fully completed UTC hour is decomposed into exactly 60 one-minute close-to-close log returns using the project's existing `market_klines_1m_chunks` binary_v1+zstd history. Compute finite-sample realized skewness inside that single completed hour. A fresh zero up-cross triggers SHORT; a fresh zero down-cross triggers LONG. Entry is the next complete USD-M 1h open.

The research reuses the shared audited read-only ARM minute-chunk decoder. No database row was inserted, repaired or modified.

## Data audit

All six discovery symbols have identical coverage:
- **1,052,700 one-minute rows per symbol**.
- **17,543 complete 60-return formation hours per symbol**.
- No symbol-specific minute coverage anomaly affected the result.
- Shared loader snapshot SHA-256: `a96e8d07cbf54846b435fbab672f91df7416bcf243e7e925620f1583d8f29f2d`.

## 2023-2024 discovery

- **50,344 events**.
- Frequency **80.3484 events/symbol/week**.
- 1h signed mean **-0.00320%**.
- 4h signed mean **-0.00210%**.
- 12h signed mean **+0.00076%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **+0.00489%**.
- 2024 12h **-0.00334%**.
- LONG audit: 12h **+0.08440%**.
- SHORT audit: 12h **-0.08289%**.

The aggregate effect is economically zero, breadth fails, and the annual sign flips. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v191**. Do not add +/-0.5 or +/-1 skew thresholds, scan 30m/120m formation windows, switch to intrahour kurtosis as a rescue, delete one side, reverse to continuation, or inspect 2025+. No strict Engine run and no DB import.
