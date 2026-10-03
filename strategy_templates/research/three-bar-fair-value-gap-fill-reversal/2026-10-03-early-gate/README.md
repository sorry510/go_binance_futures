# v194 3-Bar Fair-Value-Gap Fill Reversal — Early Gate

Mechanism: on completed Binance USD-M 1h bars, a bullish FVG is `Low_t > High_(t-2)`; a bearish FVG is `High_t < Low_(t-2)`. The preregistered mapping is gap-fill reversal: bullish FVG -> SHORT, bearish FVG -> LONG. Entry is the next complete 1h open. No gap-size or middle-bar filter.

## 2023-2024 discovery

- **18,521 events**.
- Frequency **29.5593 events/symbol/week**.
- 1h signed mean **+0.0172%**.
- 4h signed mean **+0.0298%**.
- 12h signed mean **+0.0435%**.
- Breadth **5/6 positive symbols**.
- 2023 12h **+0.0415%**.
- 2024 12h **+0.0453%**.
- LONG audit: 12h **+0.1613%**.
- SHORT audit: 12h **-0.0648%**.

Frequency, breadth and annual sign consistency pass, but the preregistered +0.20% economic gate fails materially. The side split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v194**. Do not add gap-size/body filters, scan other timeframes, switch to continuation, delete one side, or inspect 2025+. No strict Engine run and no DB import.
