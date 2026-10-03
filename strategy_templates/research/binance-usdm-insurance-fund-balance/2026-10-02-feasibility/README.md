# Binance USD-M Insurance Fund Balance — 2026-10-02 Feasibility

Hypothesis: abnormal insurance-fund depletion could identify system-wide loss absorption / deleveraging stress distinct from symbol-level funding, OI and liquidation signals.

Feasibility audit:
- Binance officially added `GET /fapi/v1/insuranceBalance` on 2025-04-23 as an insurance-fund balance snapshot endpoint.
- Current anonymous endpoint returns groups of symbols and current asset margin balances with current updateTime.
- Supplying historical-looking `startTime`, `timestamp`, or `limit` query parameters returns the same current snapshot; they do not expose historical state.
- No 2023-2024 point-in-time insurance-balance archive is available through this endpoint.
- No returns were inspected.

Decision: **data-history blocked / freeze**. Do not scrape current snapshots backward, infer historical balances from today's symbol groups, or substitute another exchange's insurance fund as Binance history.
