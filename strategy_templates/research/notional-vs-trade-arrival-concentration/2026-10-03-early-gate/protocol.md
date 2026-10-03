# v184 Notional-vs-Trade-Arrival Concentration Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h Klines: QuoteAssetVolume and TradeCount.
3. Over the latest 24 completed hours, normalize QuoteVolume into hourly shares and compute HHI_QV = sum(q_i^2).
4. Independently normalize TradeCount into hourly shares and compute HHI_Count = sum(c_i^2). Require positive 24h sums for both.
5. score = log(HHI_QV / HHI_Count).
6. Fresh zero up-cross means dollar notional has become more temporally concentrated than transaction arrivals; preregister informed/notional-burst continuation and follow the completed trailing 4h price direction.
7. Fresh zero down-cross means transaction arrivals are more concentrated than dollar notional; preregister fragmented/noise crowding and fade the completed trailing 4h price direction.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
9. No concentration magnitude threshold, z-score, average-trade-size threshold, funding, OI, taker, volatility, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not scan windows, add HHI-ratio thresholds, substitute entropy/Gini, delete one crossing/side, invert the mapping, or inspect 2025+.
12. This is distinct from standalone QuoteVolume HHI and average-trade-size shocks: it compares the temporal concentration of dollars with the temporal concentration of transaction arrivals inside the same path.
