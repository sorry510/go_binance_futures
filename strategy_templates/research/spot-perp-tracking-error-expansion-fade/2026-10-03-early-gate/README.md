# v166 Spot–Perp Tracking-Error Expansion Fade — Early Gate

Mechanism:
- align same-symbol Binance Spot and USD-M completed 1h bars;
- residual e_t = perp hourly log return - spot hourly log return;
- current tracking error = RMS residual over the latest 24h;
- baseline tracking error = RMS residual over the immediately preceding 168h;
- fresh zero up-cross of log(current/baseline) triggers;
- fade the latest 24h cumulative residual: perp relative outperformance -> SHORT, relative underperformance -> LONG;
- enter next complete USD-M 1h open.

No residual magnitude threshold, z-score, smoothing, basis-level filter, funding, OI, taker, volume, time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **2,893 events**.
- Frequency **4.6172 events/symbol/week**.
- 1h signed mean **+0.0281%**.
- 4h signed mean **+0.1050%**.
- 12h signed mean **+0.0524%**.
- Breadth **3/6 positive symbols**.
- 2023 12h **+0.0806%**.
- 2024 12h **+0.0231%**.
- LONG audit: 12h **+0.1985%**.
- SHORT audit: 12h **-0.0953%**.

The annual sign-consistency and frequency conditions pass, but both the +0.20% economic threshold and 4/6 breadth gate fail. The side split is audit-only and does not permit deleting SHORT.

Decision: **freeze v166**. Do not scan current/baseline windows, add tracking-error thresholds, delete one side, reverse to continuation, combine with v161/v164 filters, inspect 2025+, or run strict TP8/SL6.

The broader Spot–Perp state branch is now paused: v161 basis momentum, v163 volume-share expansion, v164 perp-excess volatility, and v166 tracking-error expansion all failed their frozen discovery gates.
