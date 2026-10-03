# v152 Dollar Funding Burden Expansion Fade — Early Gate

Mechanism: at each normal 8h funding settlement, compute `burden = abs(funding_rate) * OI_notional / trailing_8h_quote_volume`. Compare current burden with the mean of the previous three eligible settlements. A first cross of burden ratio from <=1 to >1 triggers a crowding fade: positive funding -> SHORT, negative funding -> LONG.

Discovery is SOL/DOGE/LTC/AVAX/UNI/ZEC in 2023-2024 only. Binance Vision fundingRate, 5m metrics and 1h klines are used causally. No funding z-score, price/taker/trend filter, symbol-specific rule, or side deletion.

## Final result

- 3,045 events; frequency **4.8598 events/symbol/week**.
- 1h signed mean **+0.0491%**.
- 4h signed mean **+0.0680%**.
- 12h signed mean **+0.0892%**.
- 5/6 symbols positive over combined discovery.
- 2023: 1,533 events, 12h **+0.1524%**, 6/6 positive.
- 2024: 1,512 events, 12h **+0.0250%**, only 3/6 positive.

The preregistered +0.20% economic gate fails and 2024 breadth materially deteriorates.

Decision: **freeze v152**. Do not scan baseline length, burden-ratio thresholds, alternative turnover windows, continuation direction, or add filters. 2025+ and strict Engine remain unread/unrun; no DB import.
