# Phase V4-5：Binance API Budget & Optimization

## 1. 目标

基于 V4-4 的真实数据建立统一 Binance API Budget，减少重复请求并保护真实交易关键请求，重点解决“同时开很多单容易触发 Binance API 限制”。

核心原则：

**减少无效请求优先于 sleep；复用数据优先于重复查询；真实交易关键请求优先于后台任务。**

## 2. 已确认首要热点

当前 `StartTrade()` 每轮已经读取：

```text
positions
allOpenOrders
```

但每个新开仓进入：

```text
submitOwnedFeatureOpen
→ ensureAccountOpenSlotAvailable
→ GetTransformPositions
→ getTransformOpenOrders
```

可能再次读取相同账户数据。

在 `ws::futures_user_data != 1` 时，这会落到 REST；而当前无 Symbol 的 `GetOpenOrder()` 全账户请求代码标注权重 40。

同一轮同时开多个币，会把这个安全检查重复 N 次，是首个必须消除的调用放大点。

## 3. StartTrade Snapshot

V4-5 首先引入单轮不可变 Account Snapshot：

```text
StartTrade Cycle
  ↓
load positions once
load open orders once
  ↓
AccountSnapshot
  ↓
selector / risk / open-slot checks reuse
```

每成功创建一个订单后，在 snapshot 中确定性追加 pending/open state，防止同一轮后续候选使用旧视图。

需要绝对实时重新验证的最终 mutation，可以使用更低权重的 symbol-specific API 或 Ownership/WS 状态，不允许重新读取全账户高权重数据作为默认行为。

## 4. Request Coalescing / Cache

对读请求按 endpoint 特性处理：

### 短 TTL / singleflight

适用于同一瞬间重复：

- account position snapshot。
- open orders snapshot。
- premium index bulk snapshot。

使用 `singleflight` 或等价逻辑，同一 key 并发只打一次 Binance。

### 长 TTL

适用于：

- ExchangeInfo。
- Symbol precision / filters。

已有本地 symbols 数据优先使用，不必频繁重新请求完整 ExchangeInfo。

### 不缓存

Trade Mutation 不缓存：

- create order。
- cancel order。
- change leverage/margin（可避免重复设置，但不能把 mutation 响应当普通 cache）。

## 5. User Data WS 优先

当 `ws::futures_user_data=1` 且数据 freshness 合格时：

```text
Positions / Open Orders
→ local DB / WS mirror
```

REST 仅用于：

- startup/bootstrap。
- stale data fallback。
- reconcile uncertain state。
- mutation 前必要的低频确认。

必须有 freshness 判断，不能因为“本地有数据”就永久信任 stale snapshot。

## 6. Global API Budget

建立统一 Budget Coordinator，至少区分：

```text
P0 Trade Mutation / uncertain reconcile
P1 Account & Risk Read
P2 Market Intelligence / Selector Read
P3 Background / Historical / Maintenance
```

当 Used Weight 接近阈值：

- P3 延后。
- P2 降级/跳过。
- P1 保留必要额度。
- P0 优先，但仍遵守 Binance order-count 限制。

不要允许低优先级历史任务耗尽真实交易额度。

## 7. Header Feedback

Budget 不只靠固定 token bucket。

每次响应读取：

- actual Used Weight。
- Order Count。
- Retry-After。

动态校正本地预算。

避免 Binance 调整 endpoint 权重后本地常量失真。

## 8. 429 / 418 处理

### Read Request

- honor `Retry-After`。
- bounded backoff。
- 非关键任务可直接跳过本轮。

### Trade Mutation

- 绝不因为 429/timeout 直接盲目重复下单。
- 继续依赖 deterministic `client_order_id` + reconcile。
- uncertain result 进入已有 `reconcile_required`。

### 418

视为硬保护状态：

- 暂停非关键 REST。
- System Dashboard / notification 明确报警。
- 不循环重试加重封禁。

## 9. 其它重点优化检查

V4-4 报告完成后逐项确认：

- 全账户 Open Orders 是否可以换成 symbol-specific 请求。
- GetPosition 全账户调用是否可复用 User Data WS/local snapshot。
- Funding/Premium Index 是否存在逐币重复调用，可否 bulk fetch 后本地分发。
- ExchangeInfo 是否重复获取。
- Ownership Reconcile 与 StartTrade 是否在短时间重复拉相同账户数据。
- Market Intelligence / Agent Tool 是否可使用已有 local market data。
- Historical REST 是否统一进入低优先级预算，而不是独立抢占额度。

## 10. System Dashboard

V4-5 后增加：

- budget state：normal / warning / critical / exchange_throttled。
- deferred request count。
- coalesced request count。
- cache hit / local WS hit。
- prevented duplicate API calls。

## 11. Gate

重点压测场景：

```text
同一轮 5 个 Symbol 同时满足开仓条件
```

验证：

- Account Position / OpenOrders 不随 Symbol 数线性重复查询。
- 不再出现每个候选都调用全账户 weight-40 OpenOrders。
- Order Count 不越限。
- 429 时没有 duplicate order。
- Background task 在预算紧张时自动让路。
- User Data WS stale 时能安全 fallback REST。
- Ownership Safety 不因缓存/复用被削弱。

## 12. 本阶段不做

- 不通过无限增大 sleep 解决问题。
- 不降低 Ownership 安全检查。
- 不把 Trade Mutation 自动 retry 化。
- 不依赖用户手工计算 API 权重。
