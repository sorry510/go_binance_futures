# v179 Return-Energy Concentration Regime — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h closes.
3. Define hourly log return r_t. For a 24h block, energy weight w_i = r_i^2 / sum(r_j^2), and HHI = sum(w_i^2). Require positive total squared return.
4. Compare two adjacent non-overlapping completed 24h blocks: score = log(HHI_latest / HHI_previous).
5. A fresh zero up-cross means return energy has become more concentrated in fewer hours; preregister shock-reversal behavior and fade the completed trailing 4h price direction.
6. A fresh zero down-cross means return energy has become more diffuse; preregister distributed-move continuation and follow the completed trailing 4h price direction.
7. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No HHI magnitude threshold, z-score, return-size filter, funding, OI, volume, taker, ATR, time-of-day, symbol-specific rule or overlapping-block redesign.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 signals/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not scan 12h/48h blocks, add concentration thresholds, replace HHI with Gini/entropy, delete one crossing/side, invert the mapping, or inspect 2025+.
11. This is distinct from volume-concentration HHI and realized-kurtosis/jump studies: v179 measures how the realized return energy is distributed across time within a fixed rolling path.
