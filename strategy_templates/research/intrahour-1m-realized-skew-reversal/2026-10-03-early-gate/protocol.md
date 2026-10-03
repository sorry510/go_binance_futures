# v191 Intrahour 1m Realized-Skew Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Reuse the project's audited `market_klines_1m_chunks` binary_v1+zstd decoder. Database access is read-only; no historical data is inserted or repaired for this study.
3. At each fully completed UTC hour t, require 61 consecutive one-minute closes ending at the minute hh:59. These create exactly 60 one-minute log returns internal to the completed hour.
4. score_t is the ordinary sample skewness of those 60 one-minute returns: third central moment divided by sample-standard-deviation cubed, with the finite-sample correction n/((n-1)(n-2)). Require non-zero variance.
5. A fresh zero up-cross from <=0 to >0 means the just-completed hour developed a positive intrahour tail; preregister SHORT. A fresh zero down-cross from >=0 to <0 means a negative intrahour tail; preregister LONG.
6. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
7. No skew magnitude threshold, multi-hour trend, volume/taker/funding/OI filter, time-of-day, symbol-specific rule, or current partial hour.
8. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
9. Failure freezes this intrahour-skew family: do not add +/-0.5 or +/-1 thresholds, scan 30m/120m formation windows, switch to kurtosis as a rescue, delete one side, reverse to continuation, or inspect 2025+.
10. This is intentionally distinct from the frozen 24/48h higher-moment studies: v191 measures the shape of the 60 one-minute returns *inside a single completed hour*, not a rolling distribution of hourly returns.
