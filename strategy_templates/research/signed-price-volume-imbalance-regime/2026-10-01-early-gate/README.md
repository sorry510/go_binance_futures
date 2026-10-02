# v137 Signed Price-Volume Imbalance Regime — 2026-10-01 Early Gate

Motivation: perpetual-futures literature reports a positive relationship between signed price-volume imbalance and future returns. The project-native approximation is `2*TakerBuyQuoteVolume - QuoteVolume`. The published work uses cross-sectional sorting; that would violate this project's no-Benchmark/no-cross-symbol rule, so this run preregistered a single-symbol time-series zero-cross adaptation instead.

Frozen definition:
- Per completed 1h bar: signed PVI = 2*TakerBuyQuoteVolume - QuoteVolume.
- Score = mean signed PVI over the last 24 completed 1h bars.
- Score <=0 -> >0 emits LONG; >=0 -> <0 emits SHORT.
- Enter next complete 1h open.
- No magnitude threshold, ratio normalization, breakout, trend, funding, OI, or symbol-specific rule.

Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

4,877 events; 1h signed mean -0.0039%, 4h -0.0117%, 12h +0.0356%; 5/6 symbols positive at 12h; 7.784 events/symbol/week. 2023 12h +0.0073%, 2024 +0.0665%.

Post-hoc side split: LONG 2,436 events with 12h +0.2064%; SHORT 2,441 events with 12h -0.1348%. This side asymmetry is not used to alter the frozen strategy because deleting SHORT after inspection would be selection on discovery results.

Decision: breadth/frequency/cross-year sign pass, but the preregistered economic gate of +0.20% 12h signed mean fails by a wide margin. **Freeze / no strict Engine / no OOS / no side deletion / no threshold or window scan.**
