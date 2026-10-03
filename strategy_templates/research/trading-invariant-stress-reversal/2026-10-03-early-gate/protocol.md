# v190 Trading-Invariant Stress Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h klines: Close, QuoteAssetVolume, TradeCount.
3. For a completed 24h block:
   - dollar volume V = sum QuoteAssetVolume of its 24 hourly bars;
   - trade count N = sum TradeCount of those same 24 bars;
   - realized volatility sigma = sqrt(sum of the 24 squared close-to-close one-hour returns bounded by the block's start/end closes).
   - invariant I = V * sigma / N^(3/2).
4. At hour t, current V/N use bars t-23..t and sigma uses returns from close(t-24) through close(t); baseline V/N use bars t-47..t-24 and sigma uses returns from close(t-48) through close(t-24). The blocks are non-overlapping in traded hours.
5. Require both I values >0. score = log(I_current/I_previous).
6. Trigger only on a fresh zero up-cross from <=0 to >0, representing increased exchanged-risk per N^(3/2).
7. Direction is preregistered liquidity-stress reversal: trailing completed 4h return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns; 12h endpoint must remain in signal calendar year.
9. No change to the 3/2 exponent, no z-score, magnitude threshold, taker/funding/OI/basis filter, time-of-day or symbol-specific rule.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, both 2023 and 2024 means >0.
11. Failure freezes this family: do not scan 12h/48h blocks, replace 3/2 with a fitted exponent, use MDH regression residuals as rescue, delete one side, reverse to continuation or inspect 2025+.
12. This is distinct from per-trade variance, average-ticket and count/volume correlation families because the state is the theory-fixed risk-transfer invariant V*sigma/N^(3/2).
