# v153 OI × Funding Crowding-Build Fade — Early Gate

At normal 8h funding settlements, OI notional is sampled from the latest Binance Vision metrics observation at or before settlement. Current OI growth is log(OI_T/OI_prev), previous growth is log(OI_prev/OI_prev2). A natural state transition from previous growth <=0 to current growth >0 triggers a crowding fade: positive funding -> SHORT, negative funding -> LONG.

Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024 only. No OI magnitude threshold, funding threshold, price-return, taker, QPS, trend, premium, volatility or symbol-specific filter.

## Final result

- **3,379 events**
- Frequency: **5.3928 events/symbol/week**
- 1h signed mean: **-0.0134%**
- 4h signed mean: **+0.0206%**
- 12h signed mean: **-0.0805%**
- Breadth: **1/6 symbols positive**
- 2023 12h: **-0.0549%**, 2/6 positive
- 2024 12h: **-0.1064%**, 1/6 positive

The preregistered economic, breadth and annual gates all fail.

Decision: **freeze v153**. Do not scan OI windows, add OI/funding magnitude thresholds, delete one direction, switch to continuation, or inspect 2025+. No strict Engine and no DB import.
