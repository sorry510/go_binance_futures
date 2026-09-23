# V4-4 Binance API Usage Report

## Status

**Instrumentation implementation: complete.**

**Real live sampling: pending backend restart with the V4-4 binary.**

This report deliberately does not invent runtime numbers from source code. The actual 1-minute peak and Top-20 endpoint ranking must come from `GET /system/binance-api-usage` after the new binary has observed real traffic.

## 1. Live snapshot to capture

After restart, allow the normal application workload to run for at least 5 minutes, and preferably include:

- normal Futures trading loop
- User Data WS/local-mirror state
- ownership reconcile
- market intelligence / Agent analysis if enabled
- funding-rate task if enabled
- one historical-data/backtest prefetch period if that workload is part of normal use

Then capture:

```text
GET /system/binance-api-usage
```

The report should record:

- maximum observed `X-MBX-USED-WEIGHT-1M`
- corresponding weight percentage
- 1-minute request-count peak
- 1-minute estimated-weight peak
- 429 count
- 418 count
- Top 20 by count
- Top 20 by estimated weight
- source distribution

## 2. Code-level amplification paths already identified

These are **code inspection findings, not runtime ranking**.

### A. Account-wide open orders

Endpoint family:

```text
GET /fapi/v1/openOrders
```

When called without `symbol`, the estimator treats it as a high-weight request. It appears in account/open-order synchronization and can become expensive if it is repeated from multiple workflows.

This is a primary V4-5 candidate for reuse of the User Data WS/local mirror or a short-lived shared snapshot.

### B. Position-risk reads

Endpoint family:

```text
GET /fapi/v2/positionRisk
GET /fapi/v3/positionRisk
```

Position reads can occur in:

- trading/account snapshots
- ownership safety checks/reconcile
- Agent Trade risk checks
- manual account API

V4-4 source attribution is specifically intended to show whether these calls duplicate each other within short windows.

### C. Depth

Endpoint:

```text
GET /fapi/v1/depth
```

Depth is used for fill-price/average-price decisions and market intelligence. When many symbols are evaluated concurrently, this is a natural per-symbol multiplier.

Whether it can be cached or replaced by local WS data depends on the caller and freshness requirement and should be decided in V4-5 from live data.

### D. Historical K-lines

Endpoint:

```text
GET /fapi/v1/klines
```

Historical prefetch is paginated and can generate sustained weight over a short period. It is already marked `source=historical_market`, making it separable from trading-loop pressure.

Existing local-first historical cache/chunk logic remains unchanged.

### E. Funding / market-intelligence reads

Relevant endpoint families include:

```text
/fapi/v1/premiumIndex
/futures/data/openInterestHist
/futures/data/takerlongshortRatio
```

These should be evaluated by actual count and estimated weight before deciding whether they require reuse/caching.

## 3. Calls that generally must remain fresh

The following categories should not be removed merely because they appear in the Top list:

- order placement
- algo-order placement
- order cancel
- uncertain-submit reconciliation
- ownership safety reconciliation when local state is insufficient
- leverage / margin-type mutation when actually required
- reads required to confirm a trade mutation

V4-5 may avoid redundant repeats, but must preserve trading safety and idempotency guarantees.

## 4. Calls likely eligible for V4-5 reuse

Subject to the live report:

- repeated account-wide open-orders reads
- repeated position-risk snapshots in the same short interval
- market-data reads already available from local WS state
- repeated per-symbol depth when multiple callers need equivalent freshness
- repeated exchangeInfo/static metadata
- historical requests whose data already exists in local chunks/cache

## 5. Live Top-20

Pending restart/live collection.

| Rank | Product | Env | Source | Type | Method + Path | Count (5m) | Est. Weight (5m) | P95 | Errors |
| ---: | --- | --- | --- | --- | --- | ---: | ---: | ---: | ---: |
| - | - | - | - | - | - | - | - | - | - |

## 6. Live peak

Pending restart/live collection.

```text
Peak Used Weight 1m: pending
Weight Limit 1m: pending
Peak Percent: pending
Peak Requests / 1m: pending
Peak Estimated Weight / 1m: pending
429: pending
418: pending
```

## 7. Completion rule

V4-4 can be marked fully complete after:

1. the new backend is restarted;
2. at least one normal 5-minute collection window is observed;
3. the live Top-20 and peak section above are filled with measured values;
4. those values are used to choose V4-5 optimization priorities.
