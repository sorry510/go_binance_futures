# v194 3-Bar Fair-Value-Gap Fill Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h OHLC bars.
3. A bullish FVG exists at completed bar t when Low_t > High_(t-2), leaving a non-overlapping upward price interval across the three-bar sequence.
4. A bearish FVG exists when High_t < Low_(t-2).
5. Direction is preregistered gap-fill reversal: bullish FVG -> SHORT; bearish FVG -> LONG.
6. If both conditions are impossible/false there is no signal. No extra body requirement on the middle bar and no minimum gap size.
7. Entry is the next complete USD-M 1h open. Measure signed 1h/4h/12h returns. The 12h endpoint must remain in the signal calendar year.
8. No gap-size threshold, ATR normalization, volume/taker/funding/OI condition, trend filter, time-of-day, symbol-specific rule, or fill-before-entry requirement.
9. Early gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
10. Failure freezes this family: do not add gap-size/body filters, scan 30m/4h versions, switch to continuation, delete one side, or inspect 2025+.
11. This is distinct from ordinary bar gaps, Donchian breakout, range overlap and liquidity-sweep studies: v194 requires a true three-bar non-overlap interval.
