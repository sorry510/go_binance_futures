# v178 Carry-Adjusted Momentum Zero-Cross — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use completed Binance USD-M 1h closes plus realized Binance Vision fundingRate settlements only.
3. At completed hour t, compute 24h price log return = log(C_t/C_(t-24)).
4. Sum all realized funding rates whose fundingTime lies in the causal interval (t-24h, t]. A LONG pays positive funding and receives negative funding.
5. score_t = 24h price log return - trailing realized funding sum. This is the 1x carry-adjusted mark-to-market return of a continuously held LONG over the prior 24h, ignoring trading fees because no trade is implied inside the signal window.
6. Fresh zero up-cross from <=0 to >0 -> LONG. Fresh zero down-cross from >=0 to <0 -> SHORT. Zero is the natural carry break-even boundary.
7. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h price returns as early gate. The 12h endpoint must remain in the signal calendar year.
8. No funding magnitude threshold, z-score, basis, OI, taker, volume, volatility, trend overlay, time-of-day or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 8h/48h windows, multiply funding by leverage, add thresholds/filters, delete one direction, reverse the mapping, or inspect 2025+.
11. This differs from v108: v108 required funding sign disagreement plus an 8h price zero-cross and a 4h breakout. v178 uses the actual accumulated funding cash flow directly inside a 24h net-holding-return state.
