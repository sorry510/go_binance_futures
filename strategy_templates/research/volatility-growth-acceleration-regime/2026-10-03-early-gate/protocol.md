# v173 Volatility-Growth Acceleration Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h closes. Define one-hour log return r_t.
3. At each completed hour form three adjacent, non-overlapping 12h realized-volatility blocks ending at the signal hour: RV0 (latest 12h), RV1 (preceding 12h), RV2 (preceding 12h). RV = sqrt(sum(r^2)).
4. Define current volatility growth g0 = log(RV0/RV1), previous growth g1 = log(RV1/RV2), and acceleration score = g0 - g1. Require all RV values >0 and exact continuous history.
5. Trigger only on score zero-crosses. Up-cross from <=0 to >0 means volatility growth is accelerating; preregister a fade of the completed trailing 4h price direction. Down-cross from >=0 to <0 means volatility growth is decelerating; preregister continuation of the trailing 4h price direction.
6. Trailing 4h return >0 defines positive price direction; <0 negative; exact zero creates no event.
7. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No RV magnitude threshold, z-score, ATR, funding, OI, volume, taker, time-of-day, symbol-specific rule or smoothing.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 6h/24h blocks, add acceleration thresholds, use overlapping blocks, delete one crossing direction, invert the mapping, or inspect 2025+.
11. This differs from simple realized-vol expansion and volatility clustering: the state is the second difference of log realized volatility across adjacent non-overlapping blocks.
