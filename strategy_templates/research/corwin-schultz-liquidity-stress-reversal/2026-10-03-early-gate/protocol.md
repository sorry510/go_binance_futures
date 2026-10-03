# v188 Corwin–Schultz Liquidity-Stress Reversal — Protocol

1. Discovery is fixed to SOL/DOGE/LTC/AVAX/UNI/ZEC during 2023-2024. 2025+ remains unread unless the frozen gate passes.
2. Use only completed Binance USD-M 1h OHLC bars.
3. For each adjacent two-hour pair, compute the Corwin–Schultz high-low spread estimator:
   - beta = ln(H1/L1)^2 + ln(H2/L2)^2;
   - gamma = ln(max(H1,H2)/min(L1,L2))^2;
   - alpha = (sqrt(2*beta)-sqrt(beta))/(3-2*sqrt(2)) - sqrt(gamma/(3-2*sqrt(2)));
   - if alpha <= 0, spread = 0; otherwise spread = 2*(exp(alpha)-1)/(1+exp(alpha)).
4. At completed hour t, current spread state is the arithmetic mean of pair-spread estimates wholly contained in bars t-23..t. Baseline is the arithmetic mean over the immediately preceding non-overlapping 24h block t-47..t-24.
5. Require both block means >0. score = log(current_mean / previous_mean).
6. Trigger only on a fresh zero up-cross from <=0 to >0. This is a new transition to higher estimated trading-cost/liquidity stress versus the prior day.
7. Direction is preregistered liquidity-provision reversal: trailing completed 4h price return >0 -> SHORT; <0 -> LONG; exact zero -> no signal.
8. Entry is next complete USD-M 1h open. Measure signed 1h/4h/12h returns; the 12h endpoint must remain in the signal calendar year.
9. No spread magnitude threshold, z-score, volume, taker, funding, OI, ATR, time-of-day, symbol-specific rule, or contraction signal.
10. Early gate: 12h signed mean >= +0.20%, >=4/6 symbol means positive, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.
11. Failure freezes this family: do not scan block lengths, use Abdi-Ranaldo as a rescue estimator, add spread thresholds, test contraction separately, delete one price direction, invert to continuation, or inspect 2025+.
12. This is distinct from Amihud, Roll-spread, Parkinson variance and effort-vs-result: the state variable is a volatility-adjusted OHLC bid-ask spread estimate.
