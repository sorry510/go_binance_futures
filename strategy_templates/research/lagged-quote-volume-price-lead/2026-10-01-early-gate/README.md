# v136 Lagged Quote-Volume → Price Lead — 2026-10-01 Early Gate

Hypothesis: a symbol's own recent relationship between quote-volume change and the following hour's return may reveal whether a fresh activity change predicts the next hour direction.

Frozen definition:
- x_t = log(QuoteVolume_t / QuoteVolume_{t-1}).
- y_{t+1} = log(Close_{t+1} / Open_{t+1}).
- Estimate centered covariance(x_t, y_{t+1}) on 24 fully historical pairs.
- Predictor = covariance * current completed-hour x.
- Predictor zero-up-cross = LONG; zero-down-cross = SHORT.
- Enter next complete 1h open.
- No threshold, trend, taker, funding, ATR, or symbol-specific filter.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

59,539 events; 1h signed mean -0.0061%, 4h -0.0060%, 12h -0.0090%; only 1/6 symbols positive at 12h; 95.023 events/symbol/week. 2023 12h -0.0069%, 2024 -0.0112%.

The post-hoc side split differs sharply (LONG +0.0751% at 12h, SHORT -0.0931%), but side deletion is prohibited because it would be selected after observing returns.

Decision: **early gate failed / freeze**. No 12h/48h lookback scan, covariance threshold, alternate volume transform, or direction deletion. Strict TP8/SL6 Engine and 2025+ OOS were not run.
