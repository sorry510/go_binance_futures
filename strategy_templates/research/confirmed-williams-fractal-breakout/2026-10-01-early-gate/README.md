# v144 Confirmed 1h Williams-Fractal Breakout — 2026-10-01 Early Gate

Hypothesis: a break of the most recent causally confirmed local swing can represent structure migration without requiring a Donchian/global-window extreme.

Frozen definition:
- Standard five-bar Williams fractal: center high/low strictly exceeds/falls below two completed bars on each side.
- Only already confirmed fractals are eligible; no future leakage.
- At each completed 1h bar, independently select the most recent confirmed high and low fractal inside the trailing 24 completed hours.
- Fresh close cross above latest high fractal -> LONG; below latest low fractal -> SHORT.
- If both would trigger, skip. Entry is next complete 1h open.
- No volume, funding, taker, trend, breakout-distance, retest, or strength filter.
- Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024.
- Gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both years positive.

## Final result

- Events: **10,035**
- Frequency: **16.015732 events/symbol/week**
- 1h signed mean: **-0.0036%**
- 4h signed mean: **-0.0418%**
- 12h signed mean: **-0.0776%**
- Breadth: **0/6 positive**
- 2023 12h: **-0.0689%**
- 2024 12h: **-0.0859%**
- LONG: 5,154 events, 12h +0.0171%
- SHORT: 4,881 events, 12h -0.1776%

The combined preregistered strategy is negative on every symbol and in both years. Direction attribution is retained only for audit and cannot be used to delete one side after seeing outcomes.

Decision: **freeze v144**. Do not scan 12h/48h search horizon, 3/7-bar fractals, add filters/retests, delete SHORT, reverse the family, or read 2025+. No strict Engine candidate is generated.
