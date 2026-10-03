# v178 Carry-Adjusted Momentum Zero-Cross — Early Gate

Mechanism: at each completed Binance USD-M 1h bar, compute the prior 24h continuously-held LONG mark-to-market return after realized funding cash flows:

`score = log(C_t/C_(t-24)) - sum(funding rates with fundingTime in (t-24h,t])`.

A fresh zero up-cross triggers LONG; a fresh zero down-cross triggers SHORT. Entry is the next complete USD-M 1h open. The 24h horizon corresponds to three standard 8h funding intervals and zero is the natural carry break-even boundary.

No funding magnitude threshold, z-score, basis, OI, taker, volume, volatility, trend overlay, time-of-day or symbol-specific rule.

## 2023-2024 discovery

- **10,287 events**.
- Frequency **16.4179 events/symbol/week**.
- 1h signed mean **+0.0043%**.
- 4h signed mean **-0.0181%**.
- 12h signed mean **-0.0448%**.
- Breadth **0/6 positive symbols**.
- 2023 12h **-0.0325%**.
- 2024 12h **-0.0579%**.
- LONG audit: 12h **+0.0876%**.
- SHORT audit: 12h **-0.1773%**.

Frequency is abundant, but expectancy, breadth and both yearly gates fail decisively. The LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v178**. Do not scan 8h/48h windows, multiply funding by leverage, add thresholds/filters, delete one direction, reverse the mapping, or inspect 2025+. No strict Engine run and no DB import.
