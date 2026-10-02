# Protocol

1. Freeze six-symbol 2023-2024 discovery before reading returns.
2. Use only completed project-native USD-M 1h Klines and QuoteVolume.
3. Define x_t = log(QuoteVolume_t / QuoteVolume_{t-1}).
4. Define y_{t+1} = log(Close_{t+1} / Open_{t+1}).
5. At each signal hour i, estimate centered covariance(x_t, y_{t+1}) from the 24 fully historical pairs ending with x_{i-1} -> y_i. No future return enters the estimate.
6. Multiply covariance by current completed-hour x_i. Positive predictor means LONG expectation for the next hour; negative means SHORT.
7. Emit only predictor zero-cross events: <=0 to >0 LONG, >=0 to <0 SHORT. Enter at next complete 1h open.
8. Window=24 pairs and zero threshold are frozen; no magnitude filter or trend/taker/funding/ATR overlay.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
10. Failure freezes this family: no 12h/48h scan, no covariance threshold, no alternate volume transform, no reversal.
11. 2025+ OOS remains unread unless the early gate passes.
