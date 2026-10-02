# v135 Intrahour Range-Expansion Previous-Hour Breakout — 2026-10-01 Discovery

Hypothesis: a breakout is more likely to reach the fixed TP8/SL6 targets when the still-forming hour has already expanded beyond the entire range of the previous completed hour, indicating unusually fast path expansion.

Definition:
- Require current partial 1h high-low range > previous completed 1h high-low range.
- LONG on a fresh 1m close cross above previous completed 1h high.
- SHORT mirrors below previous completed 1h low.
- No QPS, taker, trend, funding, or external-data filter.

EMA(2) 1m/1h entries are only data-loading scaffolding. Exact exits are `ROI >= 8 || ROI <= -6`.
Discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024 only.

## Final result

Discovery failed decisively: 10,446 trades, normalized PF 0.817519, 0/6 symbols positive, 16.671683 trades/symbol/week. 2023 PF 0.821409; 2024 PF 0.814619. LONG PF 0.792320; SHORT PF 0.840750.

Together with v132 and v134, three distinct current-hour acceleration signals (partial QPS, partial taker alignment, partial range expansion) all produce high-frequency stable negative expectancy. Freeze v135 and pause this intrahour-acceleration branch; do not scan range multiples, add filters, or reverse it. OOS remains unread.
