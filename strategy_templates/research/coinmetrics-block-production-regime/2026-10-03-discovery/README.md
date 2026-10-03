# v195 CoinMetrics Block-Production Regime — Discovery

Mechanism:
- CoinMetrics Community daily `BlkCnt`;
- baseline = arithmetic mean of previous 30 complete UTC days;
- score = log(current BlkCnt / baseline);
- fresh zero up-cross -> LONG; fresh zero down-cross -> SHORT;
- entry next UTC-day Binance USD-M open.

No magnitude threshold, z-score, PoW/PoS subgrouping, price/funding/OI overlay, side deletion or symbol-specific rule.

## Eligibility before outcomes

- 3,428 raw signals / 12 symbols.
- **3,426 eligible signals / 12 symbols**.
- LONG 1,711 / SHORT 1,715.
- No returns were read before eligibility completed.

## 2023-2024 discovery

After excluding 69 preregistered year-end signals whose 7d endpoint crossed the signal calendar year:

- **3,357 replayed events / 12 symbols**.
- Frequency **2.6789 events/symbol/week**.
- 1d signed mean **-0.1548%**.
- 3d signed mean **-0.0592%**.
- 7d signed mean **-0.1009%**.
- Breadth **4/12 positive symbols**.
- 2023 7d **-0.0551%**.
- 2024 7d **-0.1470%**.
- LONG audit: 7d **+0.9283%**.
- SHORT audit: 7d **-1.1318%**.

The preregistered economic, breadth and annual sign-consistency gates all fail. The large LONG/SHORT split is post-result attribution only and cannot justify deleting SHORT.

Decision: **freeze v195**. Do not scan 7/14/60d baselines, add block-count magnitude filters, split PoW vs PoS, delete SHORT, reverse the mapping, or inspect 2025+. No strict TP8/SL6 stage and no DB import.
