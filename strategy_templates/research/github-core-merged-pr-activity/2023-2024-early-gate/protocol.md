# Protocol

1. Canonical repositories are frozen before returns and inherited from github-core-release-catalyst.
2. Monthly merged PR counts come from GitHub Search using server-side `merged:YYYY-MM-DD..YYYY-MM-DD`.
3. Feature uses only completed months: log1p(current count) minus the mean log1p count of the previous three months.
4. Positive feature => LONG; negative => SHORT. Entry is next month first day 00:00 UTC USD-M 1h open.
5. Discovery only: signal months 2023-01 through 2024-11, so the full 7d endpoint remains inside 2024.
6. Eligibility: actual futures history >=2 years and trailing completed 24h QuoteVolume >=5m USDT.
7. Gate: mean7 >= +0.25%, >=3/4 symbols positive, and 2023/2024 aggregate mean7 both positive.
8. No threshold, repository, direction, or window changes after returns are read. No 2025+ retrieval unless the gate passes.
