# v143 Funding Volatility Expansion Fade — 2026-10-01 Early Gate

Hypothesis: a sudden expansion in funding-rate dispersion indicates destabilized leverage crowding. When the latest normal 24h funding cycle becomes more volatile than the preceding week, fade the sign of the recent funding balance.

Frozen definition:
- Recent volatility: population std of latest 3 regular-cadence funding settlements.
- Baseline volatility: population std of the immediately prior 21 settlements.
- All gaps needed for current/previous ratio must be 7.5h–8.5h, isolating rate dispersion from funding-interval compression.
- Trigger only when recent_std / baseline_std crosses from <=1 to >1.
- Sum of latest 3 funding >0 => SHORT; <0 => LONG.
- Next complete 1h open; diagnostic signed returns at 1h/4h/12h.
- SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.
- Gate: 12h >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both 2023/2024 12h means >0.

## Final result

- Events: **863**
- Frequency: **1.377337 events/symbol/week**
- 1h signed mean: **-0.0054%**
- 4h signed mean: **-0.0069%**
- 12h signed mean: **+0.2197%**
- Breadth: **5/6 positive**
- 2023 12h: **-0.0358%**
- 2024 12h: **+0.4721%**
- LONG: 149 events, 12h +0.6627%
- SHORT: 714 events, 12h +0.1272%

Economic magnitude, breadth, and frequency pass, but the preregistered cross-year stability condition fails because 2023 is negative. The strong 2024 result is therefore treated as regime dependence rather than permission to tune.

Decision: **freeze v143**. Do not scan recent/baseline windows, ratio thresholds, cadence tolerance, remove SHORT/LONG after seeing side attribution, reverse direction, or add filters. Strict Engine and 2025/2026 OOS remain unread.
