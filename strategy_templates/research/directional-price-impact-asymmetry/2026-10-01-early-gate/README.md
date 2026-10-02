# Directional Price-Impact Asymmetry — 2026-10-01 Early Gate

Hypothesis: if the same token requires less quote-volume to move upward than downward, upside liquidity is effectively thinner and subsequent returns may continue upward; mirror state favors SHORT.

Frozen definition: trailing 24 completed 1h bars; upside impact = positive log-return sum / positive-hour QuoteVolume sum; downside impact = absolute negative log-return sum / negative-hour QuoteVolume sum; log ratio zero-cross upward LONG, downward SHORT; next 1h open entry.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only. Gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, both years positive.

## Final result

Early gate failed: 9,907 events, 15.811 events/symbol/week. Event-weighted signed mean was +0.0097% at 1h, -0.0018% at 4h, and only **+0.0004% at 12h**. Breadth was **3/6**.

12h by symbol: SOL -0.0041%, DOGE +0.0263%, LTC -0.0360%, AVAX +0.0143%, UNI +0.0598%, ZEC -0.0545%. By year: 2023 +0.0278%, 2024 -0.0290%.

Decision: freeze. Economic magnitude is effectively zero and the annual sign flips. Do not scan 12h/48h lookback, add thresholds/filters, reverse the signal, or enter strict Engine/OOS.
