# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and 2023-2024 before reading returns.
2. Use only completed project-native USD-M 1h QuoteVolume and TakerBuyQuoteVolume.
3. x_t = 2*TakerBuyQuoteVolume_t/QuoteVolume_t - 1.
4. y_{t+1} = log(Close_{t+1}/Open_{t+1}).
5. At signal hour i estimate centered covariance(x_t,y_{t+1}) from 24 fully historical pairs ending with x_{i-1}->y_i.
6. Predictor = covariance * current completed-hour x_i. No return from hour i+1 or later enters the predictor.
7. Predictor <=0 -> >0 emits LONG; >=0 -> <0 emits SHORT. Enter at next complete 1h open.
8. No flow magnitude threshold, persistence requirement, breakout, QPS, trend, ATR, funding, OI, MarketCondition, Benchmark, or NowTime modulo.
9. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and 2023/2024 12h means both >0.
10. Failure freezes this family: no 12h/48h lookback scan, covariance threshold, alternate flow transform, side deletion, or reversal.
11. 2025+ OOS and strict TP8/SL6 Engine remain unread unless early gate passes.
