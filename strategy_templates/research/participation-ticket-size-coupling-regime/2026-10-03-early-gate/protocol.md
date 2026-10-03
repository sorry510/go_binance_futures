# v185 Participation–Ticket-Size Coupling Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h Klines with positive QuoteVolume and TradeCount.
3. Define count_change_t = log(TradeCount_t/TradeCount_(t-1)).
4. Define average_ticket_t = QuoteVolume_t/TradeCount_t and ticket_change_t = log(average_ticket_t/average_ticket_(t-1)).
5. Over the latest 48 complete hours compute score = Pearson corr(count_change_t, ticket_change_t).
6. Fresh zero up-cross means participation count and ticket size changes have become positively coupled; preregister coherent participation and follow the completed trailing 4h price return.
7. Fresh zero down-cross means count and ticket-size changes have become negatively coupled; preregister fragmented/churning participation and fade the completed trailing 4h price return.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
9. No correlation magnitude threshold, z-score, count/size shock threshold, funding, OI, taker, volatility, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbol means, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not scan windows, add thresholds, replace correlation with regression elasticity, delete one crossing/side, invert the mapping, or inspect 2025+.
12. This differs from average-trade-size shock and trade-count intensity: v185 tests the rolling relationship between changes in participation breadth and changes in average ticket size.
