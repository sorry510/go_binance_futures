# v195 CoinMetrics Block-Production Regime — Frozen Protocol

1. Candidate universe is fixed before returns from the Community metric inventory: ADAUSDT, ALGOUSDT, BCHUSDT, BTCUSDT, DOGEUSDT, ETCUSDT, ETHUSDT, LTCUSDT, TRXUSDT, XLMUSDT, XRPUSDT, ZECUSDT. These are the 12 mature candidate assets with Community 1d `BlkCnt`.
2. Discovery period is 2023-01-01 through 2024-12-31. CoinMetrics daily metrics are only actionable from the next UTC day open; no same-day use.
3. For each date t with a positive `BlkCnt`, baseline = arithmetic mean of the previous 30 complete UTC-day `BlkCnt` observations, excluding t.
4. score_t = log(BlkCnt_t / baseline). Fresh zero up-cross from <=0 to >0 -> LONG; fresh zero down-cross from >=0 to <0 -> SHORT.
5. Economic interpretation: above-baseline block production indicates improved/normalizing chain production; below-baseline production indicates relative network slowdown. No magnitude threshold or z-score.
6. Production eligibility is checked before returns: same-name Binance USD-M history >=730 days at signal date plus prior 24 complete 1h QuoteVolume >=5M USDT. Stablecoins are not in the universe.
7. Entry = next UTC-day Binance USD-M open. Measure signed 1d/3d/7d returns. The 7d endpoint must remain in the signal calendar year.
8. Early gate: >=80 eligible events total, >=8 eligible symbols, 7d signed mean >=+0.25%, >=60% eligible symbols positive, frequency >=0.30 events/eligible-symbol/week, and both 2023/2024 7d means >0.
9. Failure freezes this family: no 7/14/60d baseline scans, no block-count magnitude filters, no PoW/PoS subgroup selection, no side deletion, no reversal mapping, no 2025+ rescue.
10. If the gate passes, only then consider strict TP8/SL6 or 2025+ OOS. No DB write at any stage without explicit user instruction.
