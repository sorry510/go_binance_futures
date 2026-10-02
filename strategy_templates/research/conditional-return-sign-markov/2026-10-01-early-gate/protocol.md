# Protocol

1. Freeze six-symbol 2023-2024 discovery before reading returns.
2. Each completed 1h candle is state +1 if close>open and -1 if close<open; zero-return bars are ignored as unusable state.
3. At each signal hour, use the most recent 24 fully observed sign transitions, including the transition into the current completed hour.
4. Build the 2x2 transition counts: ++, +-, -+, --.
5. Given current + state, predictor=(++ - +-)/(++ + +-). Given current - state, predictor=(-+ - --)/(-+ + --).
6. Predictor <=0 to >0 emits LONG; >=0 to <0 emits SHORT. Enter next complete 1h open.
7. No smoothing prior, magnitude threshold, price/volume/taker/funding/OI filter, streak rule, or symbol-specific parameter.
8. Early gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, both 2023 and 2024 12h means >0.
9. Failure freezes this family: no 12/48/72-transition scan, no probability threshold, no side deletion or reversal.
10. 2025+ OOS remains unread unless early gate passes.
