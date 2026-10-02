# v138 Lagged Taker-Flow → Price Lead — 2026-10-01 Early Gate

Hypothesis: the symbol's own historical relationship between aggressive taker flow and the following hour return can predict the next hour when applied to current completed-hour flow.

Frozen definition:
- x_t = 2*TakerBuyQuoteVolume_t/QuoteVolume_t - 1.
- y_{t+1} = log(Close_{t+1}/Open_{t+1}).
- Estimate centered covariance(x_t,y_{t+1}) from 24 fully historical pairs ending at x_{i-1}->y_i.
- Predictor = covariance * current completed-hour x_i.
- Predictor zero-up-cross = LONG; zero-down-cross = SHORT; enter next complete 1h open.
- No flow threshold, persistence gate, price confirmation, QPS, trend, funding, OI, or symbol-specific rule.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

50,484 events; 1h signed mean -0.0004%, 4h -0.0010%, 12h +0.0007%; 3/6 symbols positive; 80.572 events/symbol/week. 2023 12h -0.0063%, 2024 +0.0076%.

Post-hoc side split: LONG +0.0769% at 12h, SHORT -0.0754%. This is not used to remove a side because it was observed after discovery returns.

Decision: **early gate failed / freeze**. No window scan, covariance threshold, alternate flow transform, side deletion, strict TP8/SL6 Engine, or 2025+ OOS.
