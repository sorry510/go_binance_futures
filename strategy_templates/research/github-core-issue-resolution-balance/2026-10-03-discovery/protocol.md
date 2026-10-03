# v177 GitHub Core Issue-Resolution Balance — Discovery Protocol

1. Reuse exactly the v174 10-repository map and its archived issue-creation `created_at` histories. No repo replacement or symbol selection.
2. Fetch only GitHub `is:issue` closures; exclude Pull Requests; use server-side `closed_at` only. An issue created before 2022 but closed during the study window still counts as a closure on its close date.
3. Construct UTC daily counts for creations and closures.
4. At completed UTC day t, balance_t = sum(closures over t-6..t) - sum(creations over t-6..t).
5. Trigger only fresh zero-crosses: balance <=0 to >0 = issue resolution has overtaken issue arrival -> LONG; balance >=0 to <0 = backlog pressure has resumed -> SHORT. Zero is the natural net-backlog boundary.
6. Signal acts only after UTC day t completes; entry is next UTC-day Binance USD-M open.
7. Dynamic market eligibility: same-name USD-M history >=730 calendar days and prior 24 complete 1h QuoteVolume >=5M USDT.
8. Measure raw signed 1d/3d/7d returns; 7d endpoint must remain within the signal calendar year.
9. No title/label/severity/NLP filters, no repo normalization, no closure-age weighting, no 14d/28d smoothing, no price/funding/OI overlay, no symbol-specific rule.
10. Discovery gate: 7d signed mean >=+0.25%, >=6/10 symbol means positive, >=0.30 events per eligible-symbol/week, and both 2023/2024 7d means >0.
11. Failure freezes the family: do not change the 7d window, add issue-type filters, use close/open ratios, delete repos/sides, reverse the mapping, or inspect 2025+.
12. Only if discovery passes may exact TP8/SL6 implementation be defined under fixed project execution constraints.
