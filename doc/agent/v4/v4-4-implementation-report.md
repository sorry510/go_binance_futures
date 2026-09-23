# V4-4 Binance API Usage Observability - Implementation Report

## 1. Scope

V4-4 only adds observability. It does **not** change Binance rate-limit behavior, add global sleeps, change retry policy, cache REST results, merge requests, or modify trade decision logic.

No database schema or `app.conf` changes are required.

## 2. Unified REST instrumentation

All production `go-binance` REST clients are wrapped at the HTTP `RoundTripper` layer:

- USD-M Futures
- Spot
- COIN-M Delivery
- Futures Testnet when `binance::testnet=true`

The wrapper is implemented under:

```text
service/binanceapiusage/
  collector.go
  transport.go
  weights.go
```

The wrapper sits outside the existing proxy transport. Therefore proxy rotation, SOCKS/HTTP behavior, request signing, timestamp retry, and the actual Binance request path are unchanged.

WebSocket traffic is not instrumented and does not enter REST usage metrics.

## 3. Per-request fields

Each observed request keeps only:

- product: futures / spot / delivery
- environment: mainnet / testnet
- source
- request_type: read / trade
- HTTP method
- normalized URL path
- HTTP status code
- latency
- estimated endpoint weight
- 429 / 418 flags
- sanitized transport-error marker

It never stores:

- API key
- secret
- signature
- timestamp query
- symbol query
- full query string
- signed URL
- raw transport error text

The transport uses only `req.URL.Path` for persisted endpoint identity. Query parameters are inspected in-memory only long enough to estimate endpoint weight and are never placed into the collector.

## 4. Binance authoritative usage state

When present, response headers are read directly:

```text
X-MBX-USED-WEIGHT-1M
X-MBX-ORDER-COUNT-10S
X-MBX-ORDER-COUNT-1M
Retry-After
```

The current used weight and order counts shown in the Dashboard come from these response headers. Estimated endpoint weight is explicitly labeled as best-effort and is used only for endpoint ranking/analysis, never as the authoritative current limit state.

The denominator used for the 1-minute weight percentage is:

1. `exchangeInfo.rateLimits` with `REQUEST_WEIGHT / MINUTE / 1`, when available.
2. A product reference limit only as fallback.

The Dashboard exposes `limit_source` internally so the caller can distinguish `exchange_info` from `reference`.

## 5. Rolling windows

The collector is process-local and keeps a fixed-capacity ring buffer of at most 50,000 request events.

Writes are O(1): once the buffer is full, the oldest slot is overwritten in place. The write path no longer prunes/copies the retained slice under the global mutex.

Windows are calculated from event timestamps at snapshot time:

- 10 seconds
- 1 minute
- 5 minutes

For each window:

- request count
- estimated weight
- error count
- 429 count
- 418 count
- average latency
- p95 latency

The collector does not write one database row per request.

A hard in-memory cap prevents unbounded growth. If the cap causes an event that still belongs to the active 5-minute window to be overwritten, the snapshot exposes:

```text
truncated=true
dropped_events
last_dropped_at
retained_events
```

The System Dashboard shows an explicit warning so a truncated window cannot silently look complete.

## 6. Endpoint aggregation

The 5-minute endpoint aggregation key is:

```text
product
+ environment
+ source
+ request_type
+ method
+ normalized path
```

Each row contains:

- count
- estimated weight
- error count
- 429 count
- 418 count
- average latency
- p95 latency
- last sanitized error
- last error time

Two Top-20 views are produced:

- by request count
- by estimated weight

## 7. Source attribution

The Transport always works even without source metadata. Untagged production calls fall back to `go_binance`.

Important paths explicitly propagate a source through `context.Context`:

- `start_trade`
- `ownership_reconcile`
- `agent_trade`
- `historical_market`
- `funding_rate`
- `market_intelligence`
- `manual_api`
- `system_health`
- owner-specific managed mutations such as `new_coin_rush` and `notice_auto_order`

Existing public wrappers remain compatible. Context-aware variants were added only where attribution is needed, so callers that do not need attribution do not need to change.

## 8. Trade/read distinction

`request_type=trade` is attached to mutation endpoints such as:

- Futures order / algo order
- Futures leverage
- Futures margin type
- Spot order

Normal market/account reads remain `request_type=read`.

The environment is separately recorded as `mainnet` or `testnet`, so a Testnet trade can be distinguished from a Mainnet read without parsing URLs or queries.

## 9. 429 / 418

The collector records 429 and 418 separately and keeps the most recent rate-limit events, including:

- time
- product
- environment
- source
- request type
- method/path
- status
- Retry-After

V4-4 only observes these responses. Existing rate-limit notification/retry behavior is not replaced or modified.

## 10. Read-only API

Added:

```text
GET /system/binance-api-usage
```

The endpoint returns an in-memory snapshot only. It does not call Binance and therefore does not increase Binance API usage when the Dashboard refreshes.

## 11. System Dashboard

The existing System Dashboard now includes a Binance API Usage section showing:

- requests in the last minute
- estimated weight in the last minute
- p95 latency
- 429 / 418 in the last 5 minutes
- response-header Used Weight / Order Count
- reference/exchangeInfo budget percentage
- 10s / 1m / 5m rolling totals
- Top Endpoints by estimated weight
- Top Endpoints by request count
- source attribution
- recent 429 / 418

The same existing dashboard refresh cycle fetches the read-only snapshot.

## 12. Security regression coverage

Permanent tests verify:

- signed query values never enter endpoint/error output
- raw transport errors containing a signed URL are replaced by `transport_error`
- response-header usage is parsed correctly
- rolling 10s / 1m / 5m aggregation
- Top endpoint ordering
- read/trade classification
- dynamic `exchangeInfo` weight limits override and remain above fallback reference values
- the 50,000-event ring keeps fixed capacity and reports active-window truncation
- overwriting events already older than 5 minutes does not falsely mark the active window truncated

## 13. Runtime behavior intentionally unchanged

V4-4 does not:

- change existing REST request cadence
- remove duplicate requests
- add cache
- add global semaphore
- add automatic delay
- change the existing `-1021` retry handling
- retry uncertain trade mutations
- change ownership behavior
- change Line/Coin strategy behavior

Those optimizations belong to V4-5.

## 14. Live validation boundary

A production-like live sample requires the newly built backend process to be running long enough to exercise normal trading/background workflows.

The existing backend process must not be silently restarted by development automation. Therefore the code-level V4-4 implementation can be fully validated now, while the peak/top-endpoint section of the live report is completed only after the user restarts the new binary and lets it collect traffic.
