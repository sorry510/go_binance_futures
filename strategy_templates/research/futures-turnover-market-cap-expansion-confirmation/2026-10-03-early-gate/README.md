# v168 Futures Turnover / Market-Cap Expansion Confirmation — Early Gate

Mechanism: CoinMetrics Community daily `CapMrktEstUSD` plus Binance USD-M daily QuoteVolume. Daily turnover = USD-M QuoteVolume / market cap. A fresh zero up-cross of current turnover relative to the previous 30 complete UTC-day mean triggers. Direction follows the trailing 7-day USD-M price direction; entry is the next UTC-day open.

No turnover magnitude threshold, z-score, funding, OI, taker, basis, volatility, time-of-day or symbol-specific filter.

## 2023-2024 discovery

- **602 events**.
- Frequency **0.9608 events/symbol/week**.
- 1d signed mean **+0.3254%**.
- 3d signed mean **+0.0898%**.
- 7d signed mean **+0.0103%**.
- Breadth **3/6 positive symbols**.
- 2023 7d **+1.2379%**.
- 2024 7d **-1.3015%**.
- LONG audit: 7d **+0.9570%**.
- SHORT audit: 7d **-1.1935%**.

The 7d aggregate effect is effectively zero, breadth fails, and the annual sign flips strongly. The LONG/SHORT split is post-result audit only and cannot be used to delete SHORT.

Decision: **freeze v168**. Do not scan baseline windows, add turnover thresholds, delete one side, reverse to fade, combine with v167, or inspect 2025+. No strict Engine run and no DB import.
