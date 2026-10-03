# v173 Volatility-Growth Acceleration Regime — Early Gate

Mechanism: split the latest 36 completed USD-M 1h returns into three adjacent, non-overlapping 12h realized-volatility blocks. Define g0 = log(RV0/RV1), g1 = log(RV1/RV2), score = g0-g1. A zero up-cross means volatility growth is accelerating and fades the completed trailing 4h price direction; a zero down-cross means volatility growth is decelerating and follows the trailing 4h direction. Entry is next complete 1h open.

## 2023-2024 discovery

- **12,184 events**.
- Frequency **19.4455 events/symbol/week**.
- 1h signed mean **+0.0019%**.
- 4h signed mean **+0.0051%**.
- 12h signed mean **+0.0114%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **-0.0235%**.
- 2024 12h **+0.0456%**.
- LONG audit: 12h **+0.1150%**.
- SHORT audit: 12h **-0.0924%**.
- Acceleration up-cross audit: 12h **+0.0305%**.
- Deceleration down-cross audit: 12h **-0.0076%**.

The aggregate effect is negligible, the +0.20% economic gate fails and 2023 is negative.

Decision: **freeze v173**. Do not scan block lengths, add acceleration thresholds, use overlapping blocks, delete one side/crossing direction, invert the mapping, or inspect 2025+. No strict Engine run and no DB import.
