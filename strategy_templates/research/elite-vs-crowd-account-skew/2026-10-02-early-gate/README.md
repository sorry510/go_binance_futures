# v158 Elite-vs-Crowd Account Skew Zero-Cross

Binance Vision USD-M 5m metrics are sampled causally at the last completed observation of each UTC hour.

Score:
`log(count_toptrader_long_short_ratio / count_long_short_ratio)`.

Interpretation: positive score means top-trader account headcount is more long-biased than the global account population. Zero up-cross -> LONG; zero down-cross -> SHORT; entry at the next 1h open. Adjacent hourly metrics points must be exactly one hour apart.

This is distinct from v147, which compared top-trader position capital weighting with top-trader account headcount.

## Discovery result — SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024

- **3,917 events**
- frequency **6.2515 events/symbol/week**
- 1h signed mean **+0.0201%**
- 4h signed mean **+0.0809%**
- 12h signed mean **+0.0958%**
- breadth **5/6 positive**
- 2023 12h **+0.0557%**
- 2024 12h **+0.1316%**
- LONG: 1,961 events, 12h **+0.3580%**
- SHORT: 1,956 events, 12h **-0.1670%**

Breadth, frequency and annual sign are acceptable, but the preregistered combined economic gate requires >=+0.20% at 12h. The combined effect is less than half that threshold. The LONG/SHORT split is audit-only and does not authorize deleting SHORT after seeing outcomes.

Decision: **freeze v158**. Do not scan score thresholds/smoothing, switch to Position/Global or Position/Account, delete a side, reverse direction, or add price/funding/OI/taker filters. 2025+ and strict Engine remain unread/unrun.
