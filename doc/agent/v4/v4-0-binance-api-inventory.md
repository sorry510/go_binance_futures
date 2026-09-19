# V4-0 Binance API Inventory

> 日期：2026-09-19  
> 来源：当前仓库静态调用链 + go-binance/v2 v2.8.12 本地模块源码。  
> 说明：Weight 只记录代码中已明确写出的值。其它 endpoint 的真实 Weight、Order Count 和限额由 V4-4 从 Binance response headers 实测。

## 1. 分类

- P0：Trading Mutation。
- P1：Account / Risk Read。
- P2：Market Data。
- P3：Historical / Background。

该分类只用于 V4-0 inventory，不代表 V4-5 最终 Budget 数字。

## 2. USD-M Futures：账户与交易

| 本地入口 | Endpoint | 类别 | 当前调用来源 | 已知 Weight | 替代/优化方向 |
| --- | --- | --- | --- | --- | --- |
| GetFuturesAccount | GET /fapi/v2/account | P1 | Account Controller | 未静态确认 | 低优先 |
| GetPosition | GET /fapi/v2/positionRisk | P1 | StartTrade / Ownership / AgentTrade / Controller | 未静态确认 | User Data WS + snapshot |
| GetPositionV3 | GET /fapi/v3/positionRisk | P1 | 当前主业务未发现调用 | 未静态确认 | 低 |
| GetIncome | GET /fapi/v1/income | P3 | funding income 统计 | 未静态确认 | 增量持久化 |
| CreateOwnedOrder | POST /fapi/v1/order | P0 | Ownership Executor | Order Count类 | 不缓存、不盲重试 |
| GetOrderByClientOrderID | GET /fapi/v1/order | P1 | uncertain/reconcile | 未静态确认 | 必须保留 |
| GetOrderByOrderID | GET /fapi/v1/order | P1 | algo reconcile | 未静态确认 | 必须保留 |
| CancelOrder | DELETE /fapi/v1/order | P0 | managed cancel | Trade类 | 不缓存 |
| CreateOwnedAlgoOrder | POST /fapi/v1/algoOrder | P0 | Stop/TP | Order Count类 | 不缓存 |
| GetAlgoOrderByClientOrderID | GET /fapi/v1/algoOrder | P1 | protection reconcile | 未静态确认 | 必须保留 |
| CancelAlgoOrder | DELETE /fapi/v1/algoOrder | P0 | protection cancel | Trade类 | 不缓存 |
| SetLeverage | POST /fapi/v1/leverage | P0 | Auto/Notice/Rush/Agent open前 | 未静态确认 | 避免无变化重复设置 |
| SetMarginType | POST /fapi/v1/marginType | P0 | Auto/Notice/Rush open前 | 未静态确认 | 避免无变化重复设置 |
| GetOrders | GET /fapi/v1/allOrders | P1/P3 | TradeCoin1～4 cooldown | 未静态确认 | 本地 order 表可替代 |
| GetOpenOrder() | GET /fapi/v1/openOrders | P1 | StartTrade/UserData/AgentTrade/Controller | 代码注释40 | User Data WS + snapshot，最高优先 |
| GetExchangeInfo | GET /fapi/v1/exchangeInfo | P2/P3 | 12h precision + Rush/utility | 未静态确认 | 本地 symbol metadata + TTL |

## 3. USD-M Futures：Market Data

| 本地入口 | Endpoint | 类别 | 当前调用来源 | 本地/WS替代 | 优化候选 |
| --- | --- | --- | --- | --- | --- |
| GetDepth / GetDepthAvgPrice | GET /fapi/v1/depth | P2 | Auto/Agent/Symbol Analysis | 当前无完整 order-book mirror | short TTL / singleflight |
| GetTickerPrice | GET /fapi/v2/ticker/price | P2 | Rush/Notice | 全市场 WS symbols.close | 高可替代性 |
| GetKlineData | GET /fapi/v1/klines | P2 | Line Strategy/Listen/Symbol Analysis | 本地 historical + WS 可覆盖部分 | 同轮 cache |
| GetFundingRate | GET /fapi/v1/premiumIndex | P2 | 每120s bulk + Symbol Analysis | symbol_funding_rates / bulk snapshot | 复用 bulk |
| GetOpenInterest | GET /fapi/v1/openInterest | P2 | Symbol Analysis | 无完整本地镜像 | short TTL |
| GetOpenInterestStatistics | GET /futures/data/openInterestHist | P2 | Symbol Analysis | 可低频持久化 | short TTL |
| GetTakerLongShortRatio | GET /futures/data/takerlongshortRatio | P2 | Symbol Analysis | 可低频持久化 | short TTL |
| GetFundingRateHistory | GET /fapi/v1/fundingRate | P2/P3 | Strategy env / Controller | market_funding_rates/local | 中 |
## 4. USD-M Futures：Historical / User Stream

| 本地入口 | Endpoint | 类别 | 当前行为 | 备注 |
| --- | --- | --- | --- | --- |
| GetEarliestHistoricalKline | GET /fapi/v1/klines | P3 | listing boundary | 只在历史缺失时 |
| GetHistoricalKlines* | GET /fapi/v1/klines | P3 | historical分页 | 已有 waitHistoricalREST |
| GetHistoricalFundingRates | GET /fapi/v1/fundingRate | P3 | historical funding分页 | 纳入未来低优先Budget |
| GetListenKey | POST /fapi/v1/listenKey | P1 | User Data WS启动 | 低频 |
| UpdateListenKey | PUT /fapi/v1/listenKey | P1 | 每20m | 低频 |
| timestamp sync | GET /fapi/v1/time | P1 | -1021时校时 | midpoint sync + exactly-one retry |

## 5. Spot REST

| 本地入口 | 典型 Endpoint | 类别 | 主要场景 | 可替代方向 |
| --- | --- | --- | --- | --- |
| Spot GetFuturesAccount | GET /api/v3/account | P1 | Account/余额 | 后续统一观测 |
| Spot GetExchangeInfo | GET /api/v3/exchangeInfo | P2/P3 | precision/metadata | spot_symbols |
| Spot GetKlineData | GET /api/v3/klines | P2 | Spot strategy/listen | local historical |
| Spot GetTickerPrice | GET /api/v3/ticker/price | P2 | Rush/price lookup | Spot WS spot_symbols.close |
| Spot create order | POST /api/v3/order | P0 | Buy/Sell | 不缓存 |
| Spot GetOrders | GET /api/v3/allOrders | P1/P3 | order history | 本地订单 |
| Spot GetOrder | GET /api/v3/order | P1 | reconcile/query | 局部本地状态 |

Spot TryRush 虽每100ms调度，但只有内部命中任务才会真正发 REST；V4-4 必须用真实 endpoint count 判断。
## 6. COIN-M Delivery

当前主要 REST 只有 Account 和 ExchangeInfo，市场 ticker 已使用 WebSocket 写入 delivery_symbols。Delivery 不是当前 API 超限的首要来源。

## 7. 调用放大链路

### 7.1 StartTrade

当前结构：

    every 2s StartTrade
      -> GetTransformPositions
      -> getTransformOpenOrders
      -> for each candidate
           -> submitOwnedFeatureOpen
             -> ensureAccountOpenSlotAvailable
               -> GetTransformPositions
               -> getTransformOpenOrders

WS 开启时重复的是本地 DB；WS 关闭时重复会变成 Binance REST。一轮多个候选开仓时，该安全检查会随候选数量放大。

V4-5 目标：单轮共享 AccountSnapshot，并在 mutation 成功后把 pending/open state 写回 snapshot。

### 7.2 ReconcileAll

每分钟按 5 个 owner 执行 ReconcileOwner。每个存在 active managed position 的 owner 都可能独立 GetPosition(all)。

V4-5 目标：ReconcileAll 只加载一次 account position snapshot，再共享给各 owner。

### 7.3 AgentTrade Risk

一次 risk evaluation 当前可读取 GetPosition(all) + GetOpenOrder(all) + GetDepthAvgPrice(symbol)。连续 Proposal 有短 TTL/singleflight 复用空间。
### 7.4 Selector cooldown

TradeCoin1～4 -> GetOrders(StartTime=最近5/10m)。
TradeCoin5～6 -> 本地 order 表。

这是已经有现成替代实现的 REST，V4-3 可统一本地化。

### 7.5 Symbol Analysis / Market Tools

一次分析可能读取 premiumIndex、openInterest、openInterestHist、takerlongshortRatio、depth、klines。按需调用本身合理，但同 symbol 短时间重复 Task 时适合 short TTL + singleflight。

## 8. 当前已有保护

- Futures User Data WS + 本地 Position/OpenOrder mirror。
- Historical REST 独立 throttling。
- signed recvWindow。
- -1021 midpoint server time sync + exactly one retry。
- deterministic client order ID。
- uncertain submit reconcile，不 blind retry。
- 全市场行情 WS。
- ExchangeInfo 主刷新 12h。

## 9. V4-4 必须实测

静态代码不能可靠回答：

- endpoint 当前真实 Weight。
- X-MBX-USED-WEIGHT-1M。
- X-MBX-ORDER-COUNT-10S / 1M。
- 哪些 scheduler 实际经常 early-return。
- 429/418 的真实来源。
- Testnet/Mainnet header差异。

因此 V4-4 的 Transport 级观测是必要的，不能仅凭静态表设置固定限流数字。
## 10. V4-5 优化优先级

按当前静态证据：

1. P0：StartTrade 重复 Position/OpenOrders snapshot。
2. P0：all-account GetOpenOrder() 高权重调用。
3. P0：ReconcileAll 多 Owner 重复 Position snapshot。
4. P1：AgentTrade Risk account snapshot 复用。
5. P1：TradeCoin1～4 cooldown 改本地。
6. P1：Kline 同 symbol/interval 同轮重复请求。
7. P1：ExchangeInfo 多入口统一 TTL。
8. P2：PremiumIndex bulk snapshot 复用。
9. P2：Symbol Analysis endpoints short TTL/singleflight。
10. P2：Historical REST 接入统一低优先 Budget。
11. P2：Spot REST 进入同一套观测和预算。

V4-0 不修改上述调用，仅冻结现状和优先级。
