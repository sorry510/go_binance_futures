# v192 Intrahour Volatility-Signature Noise Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Reuse the audited read-only `market_klines_1m_chunks` binary_v1+zstd loader from v191. No DB insert, repair, REST gap fill, app.conf edit, commit or push.
3. At each fully completed UTC hour require exactly 61 consecutive one-minute closes ending at hh:59, yielding 60 one-minute log returns internal to the hour.
4. RV1m = sum of squared all 60 one-minute returns.
5. RV5m uses the same hour and the same starting close, sampled every five minutes: 12 non-overlapping five-minute log returns ending at minutes 04,09,...,59. RV5m = sum of their squares.
6. score = log(RV1m/RV5m), requiring both >0. A fresh zero up-cross from <=0 to >0 means the finest sampling has just begun to report more realized variance than the 5m scale.
7. Trigger only that fresh up-cross. Direction is preregistered microstructure-noise reversal: completed 1h close-to-close return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
9. No ratio magnitude threshold, rolling baseline, alternate 2m/10m scale, skew/kurtosis, volume/taker/funding/OI filter, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this volatility-signature family: do not add RV-ratio thresholds, compare 1m/10m or 2m/5m, test the down-cross separately, add trend/filter conditions, delete one price direction, reverse to continuation, or inspect 2025+.
12. This is distinct from v191 realized skew and v179 energy concentration: v192 compares realized-variance estimates across sampling scales within the same completed hour.
