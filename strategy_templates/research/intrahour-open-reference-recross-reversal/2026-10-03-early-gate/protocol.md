# v196 Intrahour Open-Reference Recross Activation Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Reuse the audited read-only `market_klines_1m_chunks` loader. No DB insert/repair, REST gap fill, app.conf edit, commit or push.
3. At each fully completed UTC hour t require the prior minute close at t-1m plus exactly 60 consecutive one-minute closes t..t+59m.
4. The fixed reference for that hour is the prior minute close at t-1m. For each subsequent close compute deviation from this fixed reference.
5. Ignore exact-zero deviations for side state. recross_count is the number of sign changes between consecutive nonzero deviations across the 60 closes.
6. Trigger only when the previous completed hour had recross_count=0 and the current completed hour has recross_count>0. This is a natural activation from one-sided path to reference recrossing.
7. Direction is preregistered failed-auction reversal: completed-hour final close above the reference -> SHORT; below the reference -> LONG; exact zero -> no signal.
8. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns; the 12h endpoint must remain in the signal calendar year.
9. No minimum recross count beyond one, no crossing-speed/amplitude threshold, no volume/taker/funding/OI/spread/ATR/time-of-day/symbol-specific filter.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not test >=2/3/5 recrosses, moving-average/VWAP references, 30m/120m windows, continuation, side deletion, additional filters, or inspect 2025+.
12. This is distinct from v193 price staleness (exact zero 1m returns), return autocorrelation/runs tests, intrahour skew and volatility signature. v196 measures path recrossing of a fixed opening reference.
