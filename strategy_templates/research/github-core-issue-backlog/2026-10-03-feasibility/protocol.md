# GitHub Core Issue-Backlog — Feasibility Protocol

1. Reuse the frozen 10-token core repository mapping unchanged from the GitHub Core Release family. No repository is added or replaced after coverage inspection.
2. Source is GitHub's public Issues/Search API. Use only `is:issue`; Pull Requests are excluded.
3. Historical timestamps must come from GitHub issue fields `created_at`, `closed_at`, and state. These are visibility/state timestamps and avoid the commit-author timestamp problem that blocked commit activity research.
4. Feasibility first counts issues created during 2023-2024 for every frozen repository without reading any token forward returns.
5. Full-history retrieval is considered reproducible only if the 2023-2024 range can be partitioned into fixed calendar months, each search partition has <=1000 issue results, and created/closed timestamps can therefore be exhaustively paginated.
6. Coverage gate before returns: >=8/10 mapped repositories each have >=100 issues created in 2023-2024 and all their monthly partitions are retrievable under the GitHub 1000-search-result cap.
7. Later market eligibility, if discovery is opened, remains dynamic same-name Binance USD-M history >=730 days plus prior-24h QuoteVolume >=5M USDT.
8. If coverage/data-cost fails, freeze without returns. Do not include PRs, infer issue severity from labels after outcomes, replace quiet repos, or lower the repo-count gate.
