# v145 Quote Ease of Movement (14) Zero-Cross — 2026-10-01 Early Gate

Hypothesis: price midpoint displacement achieved with relatively little quote-volume effort may identify directional movement that can continue far enough for the project's fast-move objective.

Frozen definition:
- Per completed 1h bar: `EOM = (midpoint_t - midpoint_{t-1}) * (High-Low) / QuoteVolume`.
- Fixed 14-bar arithmetic mean; no period search.
- Mean crossing <=0 to >0 -> LONG; >=0 to <0 -> SHORT.
- Entry next complete 1h open.
- No trend, ATR, taker, funding, magnitude threshold, alternate smoothing, or symbol-specific rules.
- Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024.
- Gate: 12h signed mean >= +0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, and both years positive.

## Final result

- Events: **12,085**
- Frequency: **19.287506 events/symbol/week**
- 1h: **-0.0116%**
- 4h: **-0.0355%**
- 12h: **-0.0201%**
- Breadth: **2/6 positive**
- 2023 12h: **+0.0078%**
- 2024 12h: **-0.0487%**
- LONG 6,043 events: 12h -0.0268%
- SHORT 6,042 events: 12h -0.0134%

The signal is frequent but economically negative, has weak breadth, and changes annual sign.

Decision: **freeze v145**. Do not scan EOM period/threshold/smoothing, add filters, remove a direction, reverse it, or read 2025+. No strict Engine candidate is generated.
