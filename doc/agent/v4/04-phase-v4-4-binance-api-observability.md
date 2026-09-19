# Phase V4-4：Binance API Usage Observability

## 1. 目标

在优化 API 使用前，先统一回答：

```text
现在每分钟用了多少 Binance REST？
哪些 endpoint 最重？
是谁调用的？
当前 Used Weight / Order Count 到多少？
有没有 429 / 418？
哪些调用在重复？
```

V4-4 只负责“看清楚”，不急着改变业务行为。

## 2. 统一观测位置

优先在 Binance HTTP Client 的 Transport 层包装统一 `RoundTripper`，而不是要求每个 wrapper 手工埋点。

这样可以覆盖：

- Futures REST。
- Spot REST。
- Delivery REST。
- Agent / Feature / Ownership / Background 调用。

Testnet/Mainnet 分开统计。

## 3. 每次请求采集

至少采集：

- product：futures / spot / delivery。
- environment：mainnet / testnet。
- HTTP method。
- normalized path。
- status code。
- latency。
- estimated request weight（已知时）。
- response header：`X-MBX-USED-WEIGHT-*`。
- response header：`X-MBX-ORDER-COUNT-*`。
- `Retry-After`。
- 是否 429。
- 是否 418。

绝对不能记录：

- API Key。
- Signature。
- 完整 signed query string。
- Secret。

## 4. 权威数据优先级

本地 estimated weight 只用于 endpoint 分析。

实际限额状态优先使用 Binance response headers，例如：

```text
X-MBX-USED-WEIGHT-1M
X-MBX-ORDER-COUNT-10S
X-MBX-ORDER-COUNT-1M
```

因为具体 endpoint 权重和交易所规则可能变化。

## 5. 聚合方式

不建议每个 REST 请求都写一条 DB 记录。

使用内存 rolling window：

- 10s。
- 1m。
- 5m。

按 endpoint 聚合：

```text
count
estimated weight
avg / p95 latency
429 count
418 count
last error
```

必要时每分钟输出一次聚合日志；是否长期持久化在完成 V4-4 实测后再决定。

## 6. System Dashboard

在现有“系统看板”增加 Binance API 区块：

- Futures Used Weight 1m。
- Order Count 10s / 1m。
- 当前预算百分比。
- 最近 429 / 418。
- Top Endpoints by Count。
- Top Endpoints by Estimated Weight。
- 最近一分钟请求数。

增加只读 API，例如：

```text
GET /system/binance-api-usage
```

## 7. Source Attribution

如果只看 endpoint 不够，可以通过 context/request metadata 在关键路径附带 source：

```text
start_trade
ownership_reconcile
agent_trade
historical_market
funding_rate
market_intelligence
manual_api
```

Transport 层仍可工作；source 只是额外标签。

## 8. 输出报告

V4-4 完成后产出一份真实运行报告：

```text
doc/agent/v4/v4-4-binance-api-usage-report.md
```

至少列出：

- Top 20 endpoint。
- 实际 1m 峰值。
- 开多币时的调用放大链路。
- 可以通过 WS/local/caching/reuse 优化的 endpoint。
- 必须保留实时调用的 endpoint。

## 9. Gate

- 所有 go-binance REST 请求基本都能被统一观测。
- WebSocket 不计入 REST weight。
- signed query 不泄露。
- 429 / 418 可在 Dashboard 明确看到。
- Trade/Testnet 请求和普通 Read 请求可区分。
- 观测本身不会明显增加交易延迟。

## 10. 本阶段不做

- 不改变 Rate Limit。
- 不自动 sleep 所有请求。
- 不自动 retry Trade Mutation。
- 不修改 Binance 业务逻辑。
