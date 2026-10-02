# v141 1h Range-Overlap Value Migration — 2026-10-01 Early Gate

Mechanism:
- Compute adjacent completed 1h high-low range overlap as intersection length divided by the smaller range.
- Compare the current overlap with the mean of the previous 24 pair-overlap ratios.
- Trigger only on a fresh crossing from at/above baseline to below baseline.
- LONG when the current range midpoint migrated upward versus the previous hour; SHORT when it migrated downward.
- Enter the next complete 1h open.

This targets abrupt auction/value-area migration rather than ordinary return momentum, inside/outside bars, or raw range expansion.

Frozen discovery: SOL/DOGE/LTC/AVAX/UNI/ZEC, 2023-2024.
Gate: 12h signed mean >=+0.20%, >=4/6 positive symbols, >=0.30 events/symbol/week, both years positive.

## Result

26,175 events; 12h signed mean **-0.0266%**; 2/6 positive; 41.775 events/symbol/week.
2023 12h **-0.0145%**; 2024 **-0.0387%**.
LONG/SHORT post-hoc split is audit-only and is not used to alter direction.

Decision: **early gate failed / freeze**. Do not scan overlap lookback, add fixed overlap thresholds/body/wick filters, delete a side, or reverse. Strict Engine and 2025+ OOS were not run/read.
