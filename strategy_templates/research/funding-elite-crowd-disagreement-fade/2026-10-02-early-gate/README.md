# v159 Funding–Elite/Crowd Disagreement Fade

At each normal 8h Binance USD-M funding settlement, use the last completed 5m metrics observation from the preceding UTC hour. Positioning score is `log(count_toptrader_long_short_ratio / count_long_short_ratio)`.

A disagreement exists when funding and positioning skew have opposite signs. Trigger only on a false->true disagreement transition:
- positive funding + negative elite-vs-crowd skew -> SHORT;
- negative funding + positive elite-vs-crowd skew -> LONG.

No magnitude threshold, smoothing, price, OI, premium or taker filter is used.

## Discovery — 2023-2024, SOL/DOGE/LTC/AVAX/UNI/ZEC

- **1,282 events**
- frequency **2.0461 events/symbol/week**
- 1h signed mean **-0.0193%**
- 4h signed mean **-0.0399%**
- 12h signed mean **+0.1005%**
- breadth **5/6 positive**
- 2023 12h **+0.1073%**
- 2024 12h **+0.0928%**
- LONG 252 events: 12h **+0.3060%**
- SHORT 1,030 events: 12h **+0.0502%**

Breadth, frequency and annual sign pass, but the combined 12h effect is only about half of the preregistered +0.20% economic gate. The side split is audit-only.

Decision: **freeze v159**. Do not scan funding/skew thresholds, retrigger persistent states, delete a side, reverse direction, or add filters. 2025+ and strict Engine remain unread/unrun.
