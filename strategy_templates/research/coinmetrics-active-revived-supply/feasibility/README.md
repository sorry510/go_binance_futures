# CoinMetrics Active / Revived Supply — Feasibility

Proposed on-chain mechanisms:
- SplyAct30d: unique native supply active at least once in the trailing 30 days.
- SplyActPct1yr: fraction of supply active in the trailing year.
- SplyRvv30d: supply that became active after at least 30 days of dormancy.

These would measure supply mobility/dormant-coin reactivation rather than transaction count or active-address count.

Community API access audit on 2026-10-02:
- SplyAct30d: HTTP 403 Forbidden.
- SplyActPct1yr: HTTP 403 Forbidden.
- SplyRvv30d: HTTP 403 Forbidden.

Together with prior blockers for CapRealUSD, SOPR/NUPL, NVT and supply-concentration fields, advanced CoinMetrics supply-cohort research is frozen under the current community-data constraint. No proxy or paid-data assumption is introduced.

Decision: **data-access blocked / freeze feasibility**. No market returns read and no DB write.
