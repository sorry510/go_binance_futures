# v174 GitHub Core Issue-Arrival Pressure — Discovery

Repository universe is reused unchanged from the frozen 10-repository GitHub Core Release research: BTC/ETH/BNB/XRP/ADA/AVAX/NEAR/TRX/OP/APT.

Signal construction:
- only GitHub `is:issue` objects, excluding Pull Requests;
- causal timestamp is `created_at`;
- daily issue-creation counts by UTC date;
- current pressure = mean(latest 7 complete UTC days);
- baseline = mean(previous 28 complete UTC days);
- score = current - baseline;
- fresh zero up-cross -> SHORT; fresh zero down-cross -> LONG;
- after signal day completes, enter next UTC-day Binance USD-M open.

No issue labels/titles/body/NLP, repo-specific normalization, issue-count threshold, price trend, funding, OI, or time-of-day filter.

## Data / eligibility audit

GitHub 2022-2024 histories were fully paginated from the official Search API, with yearly partitions kept below GitHub's 1,000-result cap. Network/SSL retries changed only transport behavior.

- 936 raw signals / 10 repositories.
- LONG 468 / SHORT 468 before market eligibility.
- Dynamic eligibility: same-name USD-M history >=730 days plus prior 24 complete 1h QuoteVolume >=5M USDT.
- **769 eligible signals / 10 symbols**.
- Eligible LONG 387 / SHORT 382.
- 167 signals excluded only for history <730d.
- Event frequency: **0.8770 per eligible-symbol/week**.
- No token return was read before eligibility completed.

## 2023-2024 discovery result

After excluding 13 preregistered year-end events whose 7d endpoint crossed the signal calendar year:

- **756 replayed events / 10 symbols**.
- 1d signed mean: **+0.0781%**.
- 3d signed mean: **+0.0838%**.
- 7d signed mean: **+0.2229%**.
- Breadth: **6/10 symbols positive**.
- 2023 7d: **-0.0651%**.
- 2024 7d: **+0.4945%**.
- LONG audit: 378 events, 7d **+1.5153%**.
- SHORT audit: 378 events, 7d **-1.0696%**.

The preregistered +0.25% aggregate gate is narrowly missed and the mandatory annual sign-consistency gate fails. The exact 6/10 breadth threshold passes.

The strong LONG/SHORT split is strictly post-result attribution. It does **not** permit deleting SHORT, reversing the issue-pressure interpretation, or testing LONG-only on 2025+ as if independently preregistered.

Decision: **freeze v174**. Do not alter 7d/28d windows, add issue-type filters, delete a direction/repository, reverse the mapping, inspect 2025+ for rescue, or run strict TP8/SL6. No DB import.
