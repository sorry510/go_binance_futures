# v193 Intrahour Price-Staleness Activation Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Reuse the audited read-only `market_klines_1m_chunks` binary_v1+zstd loader. No DB insert, repair, REST gap fill, app.conf edit, commit or push.
3. For each fully completed UTC hour require exactly 61 consecutive one-minute closes ending at hh:59, yielding 60 one-minute close-to-close returns internal to the completed hour.
4. zero_count_t is the number of those 60 returns whose two adjacent close prices are exactly equal.
5. Trigger only a fresh staleness activation: previous completed hour zero_count == 0 and current completed hour zero_count > 0. No minimum positive zero_count beyond one.
6. Direction is preregistered liquidity-friction reversal: completed hour close-to-close return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No zero-count threshold search, rolling baseline, volume/taker/funding/OI/spread/ATR filter, time-of-day or symbol-specific rule.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this staleness family: do not test >=2/3/5 zero minutes, 5m zeros, zero-volume conditioning, continuation, side deletion, or inspect 2025+.
11. This is distinct from volatility signature, return autocorrelation, Roll spread and entropy: the state variable is explicit high-frequency price staleness.
