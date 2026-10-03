# v190 Trading-Invariant Stress Reversal — Early Gate

Mechanism: for each completed 24h Binance USD-M block, compute the theory-fixed trading invariant

`I = QuoteVolume * realized_volatility / TradeCount^(3/2)`.

At completed hour t, compare the latest 24h invariant with the immediately preceding non-overlapping 24h block. A fresh zero up-cross of `log(I_current/I_previous)` marks increased exchanged-risk per `N^(3/2)`; fade the completed trailing 4h price direction and enter at the next complete 1h open.

The 3/2 exponent, 24h block, direction mapping and zero boundary were frozen before returns.

## 2023-2024 discovery

- **3,840 events**.
- Frequency **6.1286 events/symbol/week**.
- 1h signed mean **+0.0264%**.
- 4h signed mean **+0.0653%**.
- 12h signed mean **+0.1196%**.
- Breadth **4/6 positive symbols**.
- 2023 12h **+0.1331%**.
- 2024 12h **+0.1058%**.
- LONG audit: 12h **+0.3251%**.
- SHORT audit: 12h **-0.0926%**.

Frequency, minimum breadth and annual sign consistency pass, but the preregistered +0.20% economic gate fails. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v190**. Do not fit the 3/2 exponent, scan 12h/48h blocks, add invariant thresholds, replace the definition with an MDH residual, delete one side, reverse to continuation, or inspect 2025+. No strict Engine run and no DB import.
