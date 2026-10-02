# Protocol

1. Freeze SOL/DOGE/LTC/AVAX/UNI/ZEC and 2023-2024 discovery before reading returns; 2025/2026 remain unread.
2. Use only project-local Binance USD-M funding settlements and 1h Klines. No MarketCondition, Benchmark, OI, taker, external data, or NowTime modulo.
3. Recent funding volatility = population standard deviation of the latest 3 settlements (roughly one normal 24h funding cycle).
4. Baseline funding volatility = population standard deviation of the 21 settlements immediately before those 3 (roughly one week).
5. To isolate rate dispersion from the already-studied funding-interval-compression mechanism, all settlement gaps needed for the current and immediately previous ratio must be regular 8h cadence, defined ex ante as 7.5h–8.5h. Otherwise no signal is evaluated.
6. Trigger only when recent_std / baseline_std crosses from <=1 to >1. Baseline std must be >0.
7. Direction is a preregistered crowding fade: sum(latest 3 funding)>0 => SHORT; sum<0 => LONG; exact zero => no signal.
8. Entry is the next complete 1h open strictly after the triggering settlement. Measure signed 1h/4h/12h returns.
9. No funding magnitude threshold, no z-score, no price/taker/trend filter, no symbol-specific rule.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes the family: do not scan 2/4/6 recent settlements, 14/30 baseline settlements, ratio thresholds, direction, cadence tolerance, or add filters.
12. Only if promoted may 2025/2026 be read and a strict Engine version be produced with leverage=4, TP8/SL6, fee=0.0005/side, slippage=5bps/side, single-position, and exact exits `ROI >= 8 || ROI <= -6`.
