# V4-5 Binance API Budget & Optimization 实施报告

## 1. 结论

V4-5 不以 V4-4 当前运行时 Top Endpoints 作为唯一依据。

原因是本项目很多功能由配置开关控制；未开启的功能不会出现在短期用量指标中，但其代码一旦启用仍可能形成高频 REST 调用。因此本阶段同时采用：

1. **静态全仓 API 调用审计**：从 Binance wrapper、业务调用点、主循环调度频率反向分析。
2. **V4-4 运行指标**：用于验证实际热点与优化效果，不作为唯一优化清单。
3. **交易安全分级**：只对可证明幂等/只读的数据做缓存、singleflight、WS/local mirror 替代；交易 mutation 与执行不确定状态继续走 authoritative reconcile。

核心原则：

> 优先消除重复 REST；优先复用已有 WS/local mirror；不能因为节省 API 降低 Ownership、下单、保护单与 uncertain execution 的安全性。

---

## 2. 静态 API 调用矩阵

| 功能 | 当前 REST | 调用频率/触发 | WS / local / cache 替代 | 是否必须保留 REST | V4-5 优化结果 |
|---|---|---:|---|---|---|
| StartTrade | Position、OpenOrders、Kline、Depth、Order、Leverage、MarginType | 2s | User Data mirror、cycle snapshot、Kline WS、Depth 750ms cache、trade-config 5s successful-state | **部分必须**：下单、最终安全检查 fallback、配置 mutation | 全账户 Position/OpenOrders 不再随候选数线性放大；最终安全检查仍按候选执行 target-symbol 读取；相同配置 mutation 去重 |
| PositionConvertNotice | Position | 10s | User Data mirror + mark-price WS | mirror 不健康时必须 | 正常路径不再轮询账户 REST |
| 测试策略扫描 | Kline | 1.5s | canonical Kline cache + combined WS | bootstrap/gap recovery 必须 | 稳态由 WS 增量维护 |
| 测试仓位平仓检查 | Kline | 1.5s | 同上 | bootstrap/gap recovery 必须 | 与策略扫描共享缓存 |
| Futures 新币 Rush | ticker、ExchangeInfo、MarginType、Leverage、Order | 100ms | all-market ticker WS；ExchangeInfo listing-gated；trade-config 5s | **Order 必须**；ExchangeInfo/配置仅按需 | 订单尝试仍 100ms；ticker 不再 100ms REST；WS 故障时 ExchangeInfo 最快 5s probe |
| Spot 新币 Rush | ticker、ExchangeInfo、Order | 100ms | Spot all-market ticker WS；ExchangeInfo listing-gated | **Order 必须**；ExchangeInfo 按需 | 订单尝试仍 100ms；WS 故障时 ExchangeInfo 最快 10s probe |
| Futures/Spot 价格通知 | ticker | 2s | all-market ticker WS；REST fallback 1s TTL + singleflight | WS stale 时必须 | 多功能共享 fallback，避免 Rush/Notice 各自重复请求 |
| Futures/Spot Kline 监听 | Kline | 2s | canonical cache + combined Kline WS | bootstrap/gap recovery 必须 | 重复指标读取命中本地 |
| Funding Rate 监听 | PremiumIndex | 120s | all-market mark-price/premium-index WS | WS stale/bootstrap 时必须 | 稳态不再每 120s REST |
| 策略 Funding history | FundingRateHistory | 按策略求值 | `market_funding_rates` DB-first + 30m cache | 本地不足/过旧时必须 | 已有数据时不重复历史查询 |
| Market/Agent 单币分析 | Depth、PremiumIndex、OI、OI Stats、Taker Ratio、Kline | 人工/Agent | Funding/Kline WS；Depth 750ms、OI 2s、ratio 30s + singleflight | **部分必须** | 并发/重复分析请求被合并 |
| Ownership periodic reconcile | Order、Position | 1m | 全账户 Position snapshot 可共享；User Data mirror 仅辅助 | **必须**：逐单/uncertain authoritative reconcile | 保留交易安全；多 owner 顺序执行共享账户短缓存 |
| User Data mirror | Position、OpenOrders、listenKey | startup、reconnect、30m full sync；20m keepalive | User Data WS + local tables | **必须**：每 generation authoritative full sync/listenKey | reconnect 后未 full sync 前绝不信任旧 mirror |
| ExchangeInfo / precision | ExchangeInfo | 12h/按需 | 12h cache + local symbols | 新币/缓存失效时必须 | 常规路径避免重复全量 ExchangeInfo |
| Historical backfill | Kline、Funding history | 人工/任务 | local/public-data first | **必须**：缺失历史区间 | 统一进入 P3 budget，不与交易抢额度 |
| Funding arbitrage controller | ticker、Position、OpenOrders、Order | 人工 | ticker WS；Position/OpenOrders symbol-specific | **Order/安全读取必须** | 避免全账户高权重查询 |
| Legacy coin selector | account allOrders | StartTrade 旧选币路径 | 10s cache + singleflight；Smart Local V2 完全 local | legacy 模式仍需 | 2s 循环不再重复 allOrders |
| Agent Trade | Position、OpenOrders、Depth、Leverage、Order | Proposal 执行 | account cache、Depth cache | **Leverage/Order 必须** | 显式交易 mutation 保持 fail-on-error，不用推测状态替代 |
| 手工账户/API 页面 | Account、Position、OpenOrders、Funding history | 用户请求 | Position/OpenOrders 短缓存；其余按需 | **需要实时语义时保留** | 不为低频人工请求引入长期缓存 |
| System Dashboard health | `/fapi/v1/time` | 打开/手工刷新看板 | 无；统一 Budget Transport | **必须**：用于真实 REST 可达性检查 | 已纳入 source/weight/budget，不做无意义缓存 |
| Delivery | Account、ExchangeInfo | 当前非主流程 | 现有 ticker WS；统一 Budget Transport | 当前保留 | 纳入 headers/budget；不为未启用主流程构建额外 mirror |
| 旧 `ListenRateEat` | `/fapi/v1/income` | 当前 main 中禁用；旧设计逐 symbol | 暂无安全 bulk 替代 | 启用后仍需 | 记录为后续项；不同 cursor/分页语义下不做错误合并 |

---

## 3. StartTrade：消除候选数量线性放大的账户查询

原调用结构可能出现：

```text
StartTrade
  ├─ GetPosition(all)
  ├─ GetOpenOrders(all)
  └─ candidate x N
       └─ ensureAccountOpenSlotAvailable
            ├─ GetPosition
            └─ GetOpenOrders
```

在 User Data WS 未启用时，候选数量增加会把账户安全检查放大；其中 Futures 全账户 OpenOrders 是高权重请求。

V4-5 改为：

```text
StartTrade
  ├─ load account snapshot once
  ├─ cycleAccount
  │    ├─ positions
  │    ├─ open orders
  │    └─ current-cycle pending opens
  └─ candidate x N
       └─ final safety check
            ├─ healthy User Data mirror -> local
            └─ fallback -> target-symbol Position + OpenOrders
```

成功创建订单后立即把 pending open 写入 cycle snapshot，后续候选不会因为复用旧快照而越过 FutureMaxCount 或重复占用同一方向 slot。

---

## 4. User Data WS：可用性不仅看“socket 在线”

Futures User Data WS 现在是强制基础设施：只要 Futures API Key 已配置，程序启动时就会连接，不再提供配置开关。账户 mirror 只有满足以下条件才替代 Position/OpenOrders REST：

1. 当前 User Data WS generation 存活；
2. 当前 generation 已完成一次 authoritative Position + OpenOrders full sync；
3. full sync 年龄不超过 35 分钟。

每次 WS reconnect 都会生成新的 generation。

旧 generation 的 full sync 不能授权新连接继续使用本地账户镜像。新连接建立后 watcher 会执行一次 fresh full sync；同步完成前账户读取自动回退 REST。

另外 listenKey keepalive 与对应 websocket 生命周期绑定，连接结束后旧 keepalive goroutine 会退出，避免每次 reconnect 遗留一个永久的 20 分钟 REST 定时器。

---

## 5. 自动策略活动订单：不再每 2 秒逐单 REST reconcile

静态审计发现另一个隐藏热点：

```text
StartTrade every 2s
  -> every active auto_strategy managed order
     -> GET order
  -> if any active order
     -> fresh all-account Position
```

LIMIT 单持续挂单时，这会长期重复读取相同订单状态。

V4-5 改为：

- User Data mirror 有更新时，直接从 `futures_orders` 的 WS 镜像读取该 ClientOrderID，并应用累计成交量；
- 正常稳定的 submitted/partially-filled 订单，REST reconcile 最多约每 5 秒一次；
- 只有实际观察到累计 `FilledQty` 增长时，才额外 fresh 一次账户 Position；
- `reconcile_required`、没有 ExchangeOrderID、执行结果不确定等状态不做降频，继续 fail-closed authoritative reconcile。

因此优化只针对“已知正常且稳定”的挂单轮询，不降低 uncertain execution 安全性。

---

## 6. 行情：WS 优先，REST 只作为 bootstrap / fallback

### 6.1 Futures / Spot ticker

Futures 与 Spot 全市场 ticker WS 均属于强制基础设施，程序启动时自动连接并持续重连，不再提供 `WsFuturesEnable` / `WsSpotEnable` 运行开关。

同一条 WS 数据同时用于：

- Rush、Notice、EatRate 等运行时价格；
- symbols / spot_symbols 的行情更新；
- Futures market event bus 等附加处理。

因此这些功能不会因为人为关闭 WS 而退回高频 REST。只有 WS freshness 不满足时才进入受控 REST fallback。

WS snapshot 过旧时才 REST fallback，并使用：

- 1 秒 TTL；
- per-symbol singleflight。

因此 100ms Rush 在 WS 临时故障时也不会退化成同一 Symbol 每秒 10 次 ticker REST。

### 6.2 Mark Price / Premium Index

使用 Futures all-market mark-price stream（3 秒频率）维护：

- mark price；
- index price；
- funding rate；
- next funding time。

`GetFundingRateContext` 与本地仓位 PnL 优先读取 <=6s 的 WS snapshot。

### 6.3 Kline

Futures / Spot Kline 使用：

```text
first read
  -> REST bootstrap
  -> canonical symbol+interval cache
  -> combined Kline WS incremental update
```

当发现 bar gap 时立即使缓存失效，下一次读自动 REST authoritative bootstrap。

这样不能连续维护的数据不会继续被当成有效 Kline 使用。

---

## 7. Depth 为什么没有强行改成 WS

Depth 当前采用：

- REST；
- 750ms TTL；
- singleflight。

本系统调用的是 top-N depth 并计算数量加权平均价格。

如果改成 Binance diff-depth WS，要正确实现：

1. REST order-book snapshot；
2. `lastUpdateId`；
3. WS buffered delta；
4. U/u/pu 连续性校验；
5. gap 后重新 bootstrap；
6. 本地 bid/ask book 更新与截断。

仅用“最近一条 depth WS”不能等价替代当前 REST order book。

V4-5 不为了节省少量 REST 引入错误的盘口状态，因此本阶段明确保留 REST + 极短 cache。

---

## 8. ExchangeInfo / 新币 Rush

ExchangeInfo 属于低频静态元数据：

- 普通 Futures / Spot 调用使用 12 小时 cache；
- 交易精度正常情况下优先读取本地 symbols；
- Rush 只有在 target symbol 的 ticker WS 已出现后才 force refresh ExchangeInfo。

如果 ticker WS 故障：

- Futures Rush ExchangeInfo fallback probe：最长每 5 秒一次；
- Spot Rush fallback probe：最长每 10 秒一次。

不会恢复旧的 100ms ExchangeInfo polling。

---

## 9. MarginType / Leverage mutation 去重

这些接口是 mutation，不能像普通 GET 一样长期缓存结果。

但 Rush 100ms 循环中，如果订单因为市场条件暂时失败，原逻辑可能重复发送完全相同的：

```text
SetMarginType
SetLeverage
```

V4-5 增加 5 秒的“最近成功配置”状态：

- key = symbol + marginType + leverage；
- singleflight 合并并发相同配置；
- 只有两项配置成功后才记录；
- Binance -4046（margin type already set）按已满足处理；
- 其它 MarginType 错误会立即停止，不继续发送 Leverage；
- 任意直接 `SetMarginTypeContext` / `SetLeverageContext` 成功后立即失效该 symbol 的短状态；
- User Data `ACCOUNT_UPDATE` / `ACCOUNT_CONFIG_UPDATE` 观察到该 symbol 的账户配置变化时也立即失效。

这不是缓存 mutation response，而是避免短时间重复设置已确认的相同目标状态；同时保留外部/其它路径修改配置后的快速失效，避免 5 秒 TTL 遮蔽真实账户状态变化。

---

## 10. Funding 数据

### 当前 Funding Rate

`UpdateSymbolsFundingRates` 使用 all-market mark-price WS snapshot，不再需要每 120 秒重复调用 PremiumIndex REST。

### 历史 Funding Rate

自定义策略的 FundingRate 环境改为：

```text
market_funding_rates
  -> >=16 rows 且最新 <=12h
     -> use local
  -> 否则 REST fallback
```

结果再缓存 30 分钟。

这样已有历史数据时不会因为策略每轮计算重复请求 FundingRateHistory。

---

## 11. 全局 API Budget

所有正常 Futures / Spot / Delivery SDK HTTP client 统一经过 Budget Transport。

优先级：

| Priority | 类型 |
|---|---|
| P0 | Trade mutation、execution uncertain reconcile |
| P1 | Account / risk / order / listenKey |
| P2 | 普通市场读取 |
| P3 | Historical / maintenance / system health / ExchangeInfo |

状态：

- Normal：<70%
- Warning：>=70%
- Critical：>=85%
- Exchange Throttled：429/418 + Retry-After/cooldown

Critical 时 P2/P3 延后，给账户与真实交易请求保留额度。

P0 不是“无限制放行”：如果 Binance 已明确进入 throttle 或 order-count 接近限额，仍然 fail closed。

本地 `EstimateWeight` 仅用于请求调度和预算预估；Binance 响应头中的实际 used weight / order count 才是权威值。未显式列入估算表的 endpoint 可能按默认值估计，因此文档中的“估算权重”不应理解为交易所最终计费值。

### mutation 的特殊处理

Budget defer 发生在 HTTP base RoundTripper **之前**，因此可以确定该 mutation 没有发到 Binance。

这类错误被标记为本地 deterministic pre-send failure，不进入：

```text
unknown execution
 -> Lookup
 -> reconcile_required
```

真正的 timeout / connection uncertainty 仍保持现有 ClientOrderID reconcile 逻辑，不自动重复下单。

---

### Spot 对齐范围

Spot 侧与 Futures 的高频读路径按适用范围做了对齐，而不是只改 Futures：

- all-market ticker WS 内存快照，stale 时走 1 秒 REST fallback + singleflight；
- Kline 使用 REST bootstrap + canonical cache + combined Kline WS 增量更新，gap 时失效并重新 bootstrap；
- ExchangeInfo 使用 12 小时 cache + singleflight；Spot Rush 仅在 ticker WS 已确认新 Symbol 后 force refresh，WS 故障 fallback probe 最快 10 秒一次；
- Spot SDK HTTP 同样进入统一 Binance API Usage / Budget Transport。

Spot 没有 Futures 的 Position/OpenOrders/User Data/Funding 等账户与衍生品语义，因此不机械复制这些 cache。

---

### 市场数据节流权移交

旧 `market_data_guard` 中固定 **1000 weight/min** 的市场数据节流已删除，只保留 Binance `-1003` 封禁冷却。Kline / 历史数据等正常请求的限速职责统一交给 Global API Budget：

- Normal（<70%）不再叠加旧固定 sleep/1000-weight 阈值；
- Warning / Critical 时按 P2/P3 优先级主动延后或拒绝；
- 429/418 / Retry-After 进入 exchange throttle，阻止 retry storm。

这属于“节流权移交”，不是取消限流保护。

---

## 12. Account / Market cache

当前主要 TTL：

| 数据 | TTL / 策略 |
|---|---:|
| all-account Position | 2s |
| all-account OpenOrders | 5s |
| ticker REST fallback | 1s |
| Depth | 750ms |
| Open Interest | 2s |
| OI statistics | 30s |
| Taker ratio | 30s |
| ExchangeInfo | 12h |
| identical trade config | 5s |
| legacy recent account orders | 10s |
| Funding history strategy result | 30m |
| Kline | 1s ~ 10s + WS incremental |

Position/OpenOrders 的合法空结果也会作为有效 snapshot 缓存到 TTL 到期，避免空仓/无挂单时反复击穿 REST。账户 mutation 会 invalidate Position/OpenOrders cache，防止提交订单后继续使用 mutation 前 snapshot。

---

## 13. System Dashboard

V4-5 在 V4-4 Binance API Usage 基础上增加：

### Global API Budget

显示：

- product / environment；
- normal / warning / critical / exchange_throttled；
- Used Weight + Pending Weight / Limit；
- Order Count 10s / 1m；
- throttle until。

### Optimization Hits

按 source 显示：

- Local WS Hits；
- Cache Hits；
- Coalesced Requests；
- Prevented Duplicate Calls；
- Deferred Requests。

这些指标用于确认优化是否真正命中，但不会替代本报告的静态调用审计。

---

## 14. 明确保留 REST 的场景

以下接口不会为了 V4-5 强行改成 WS：

1. Create Order / Algo Order；
2. Cancel Order；
3. protection order mutation；
4. execution result uncertain 的 ClientOrderID reconcile；
5. Ownership 无健康 User Data mirror 时 mutation 前 target-symbol 安全确认；
6. historical bulk backfill；
7. Depth order book（除非未来实现完整 sequence-safe local book）；
8. User Data startup/reconnect/周期 authoritative full sync。

---

## 15. 未作为本阶段热点改造的路径

### `rate.ListenRateEat`

该旧任务在 `main.go` 当前没有启用，原设计按 enabled symbol 请求 `/income`，且 endpoint 权重较高。

它不是当前运行热点。由于每个配置有不同 `LastProfitTime`，直接合并成一次无 symbol 查询还需要完整处理分页和不同 cursor，不能简单改写后宣称等价。

因此 V4-5 只记录为后续可优化项，不为一个已禁用功能引入新的收益统计语义风险。

### Delivery

Delivery 当前不属于主要自动交易流程。它已经纳入统一 Transport/Budget/response-header 统计，但没有额外构建一套专用 cache/WS mirror。

---

## 16. 安全 Gate

V4-5 最终验证必须覆盖：

- 5 个候选同时满足开仓时，全账户 Position/OpenOrders 不按候选数线性增长；每个候选仍保留 target-symbol Position/OpenOrders 最终安全检查；
- User Data mirror reconnect 后，未完成当前 generation full sync 前不会被信任；
- Kline WS gap 自动 REST bootstrap；
- ticker WS stale 自动 REST fallback；
- stable active order 不再每 2 秒强制 GET order；
- uncertain order 不受 reconcile throttle；
- budget defer 的 mutation 不产生 exchange lookup，也不会误判成“已发送但结果未知”；
- 429/418 不产生 duplicate trade mutation；
- Background P2/P3 在 critical budget 下让路；
- 所有 mutation 仍保留 Ownership 与 deterministic ClientOrderID 安全语义。

---

## 17. 数据库与配置影响

- 无数据库 schema migration，不需要执行 `./go_binance_futures sync db`；
- `WsFuturesEnable` / `WsSpotEnable` / `WsDeliveryEnable` 已从 Go `Config` 模型和新库初始化 INSERT 删除，基础市场 WS 强制启用；
- **已有数据库可能继续物理保留** `ws_futures_enable` / `ws_spot_enable` / `ws_delivery_enable` 三个旧列，这是非破坏性升级留下的 orphan columns。运行时不再读取或写入它们，也不要求为此执行 DROP COLUMN；
- 新数据库不会创建/初始化这三个字段。若对一个保留旧列定义的异常旧库手工删除唯一 `config` 行后再复用初始化流程，应先确认旧列是否允许省略/有默认值；这不是正常升级路径；
- 不修改 `app.conf`；
- 新增优化均使用程序内存状态、现有 WS、现有 local tables 与现有 V4-4 observability。
