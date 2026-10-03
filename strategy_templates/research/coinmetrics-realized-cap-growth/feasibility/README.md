# CoinMetrics Realized-Cap Growth — Feasibility

Proposed mechanism: use daily `CapRealUSD` growth as a slow on-chain cost-basis/capital-flow signal, distinct from the previously tested MVRV extreme ratio.

Coin Metrics documents `CapRealUSD` as realized capitalization: the sum USD value of native units valued at the USD closing price on the day each unit last moved on-chain. It can be interpreted as an aggregate holder cost basis.

## Access audit

On 2026-10-02, the Coin Metrics Community API was queried through the same asset-metrics endpoint used by prior project research:
- `PriceUSD`: available.
- `CapMVRVCur`: available.
- `CapMrktCurUSD`: available.
- `CapRealUSD`: **HTTP 403 Forbidden**.

This isolates the blocker to direct community access for the proposed metric rather than a general API outage.

Although `CapRealUSD` is algebraically derivable from market cap and MVRV according to the documented formula, that would change the data acquisition definition after the direct-metric access blocker was observed. No proxy/derived reconstruction was used.

Decision: **data-access blocked / freeze feasibility**. No signal parameters were frozen, no market returns were read, no OOS/exact replay, and no DB write.
