# v174 GitHub Core Issue-Arrival Pressure — Discovery Protocol

1. Reuse exactly the frozen 10 core repo→token mappings from GitHub Release research: BTC/ETH/BNB/XRP/ADA/AVAX/NEAR/TRX/OP/APT. No repo replacement or symbol selection from returns.
2. Fetch only GitHub `is:issue` objects, excluding Pull Requests. Use only `created_at`; do not use issue titles, labels, bodies, comments, severity guesses or NLP.
3. Construct a UTC daily issue-creation count. 2022 is warmup only; discovery outcomes are 2023-01-01 through 2024-12-31.
4. At completed UTC day t, current pressure = mean issue creations over t-6..t; baseline = mean over t-34..t-7. score = current - baseline.
5. Trigger only fresh zero-crosses. score <=0 to >0 = rising issue-arrival pressure -> SHORT. score >=0 to <0 = easing issue-arrival pressure -> LONG. Zero is the natural activity-balance boundary.
6. Signal may act only after UTC day t completes. Entry is the next UTC-day Binance USD-M open.
7. Dynamic market eligibility at each signal: same-name Binance USD-M has >=730 calendar days of history at signal time and prior 24 complete 1h QuoteVolume >=5M USDT.
8. Measure raw signed 1d/3d/7d returns. The 7d endpoint must remain in the same calendar year as the signal.
9. No issue-count threshold, label filtering, repo-specific normalization, smoothing beyond frozen 7d-vs-28d means, release/PR overlay, price trend, funding, OI, time-of-day or symbol-specific rule.
10. Discovery gate: 7d signed mean >=+0.25%, >=6/10 symbol means positive, >=0.30 events per eligible symbol-week, and both 2023 and 2024 7d means >0.
11. Failure freezes the family: do not change 7/28 windows, reverse direction, add issue-type filters, use PRs, delete repos/sides, or inspect 2025+.
12. Only if discovery passes may an exact TP8/SL6 implementation be defined; fixed project execution constraints remain leverage=4, TP8, SL6, fee0.0005/side, slippage5bps/side, single position.
