# Phase V3-5：真实交易安全与仓位/订单归属隔离

> 状态：✅ 已完成（2026-09-10）。实施结果见 [`v3-5-implementation-report.md`](./v3-5-implementation-report.md)。
> 人工验证：[`v3-5-agent-trade-ownership-testing-guide.md`](./v3-5-agent-trade-ownership-testing-guide.md)。

> 定位：P0。先解决真实资金下“谁创建、谁管理”的 ownership 边界，再补齐 Agent 受控交易的保护单、整仓平仓和重启恢复。
>
> **详细 ownership 设计真相源：** `doc/trade/合约自动交易仓位归属隔离改造计划.md`。本文件只定义 V3-5 的实施顺序、阶段边界和验收 Gate；如两份文档存在 ownership 细节差异，以详细设计文档为准。

## 1. 为什么 V3-5 必须先做 Ownership

项目目前存在多条真实 Binance 合约交易入口：

- 主自动策略 `StartTrade` → `auto_strategy`。
- 合约抢新 → `new_coin_rush`。
- 提醒触发自动下单 → `notice_auto_order`。
- 资金费率自动交易 → `funding_rate`。
- Agent 受控交易 → `agent_trade`。
- 用户本人或其它外部程序产生的仓位/挂单 → `unmanaged`。

Binance 返回的是账户级聚合仓位和账户级挂单，不能因为系统“看得到”就获得修改权限。

V3-5 的第一原则是：

```text
Observation 可以看全账户
Mutation 必须先确认 ownership
```

即：账户仓位/订单可以用于风险和冲突判断，但平仓、撤单、改单只能操作当前 owner 明确登记的数据。

## 2. 统一安全原则

### 2.1 谁创建，谁管理

```text
auto_strategy     只能管理 auto_strategy
new_coin_rush     只能管理 new_coin_rush
notice_auto_order 只能管理 notice_auto_order
funding_rate      只能管理 funding_rate
agent_trade       只能管理 agent_trade
unmanaged         所有模块均只读
```

主 `StartTrade` 不得替其它 owner 平仓或撤单，Agent 也不得接管主自动策略仓位。

### 2.2 Fail Closed

ownership 无法确定时：

- 不自动认领。
- 不自动平仓。
- 不自动撤单。
- 不自动改单。

升级前已经存在的 Binance 仓位/挂单，没有 managed record 就保持 `unmanaged`。

### 2.3 managed quantity，而不是 managed symbol

必须记录 `managed_qty`，不能只记录“BTCUSDT 属于系统”。

如果用户对同一 `(symbol, position_side)` 人工加仓：

```text
account_qty > managed_qty
→ managed_qty 不增加
→ 系统最多只能平 managed_qty
```

如果用户人工减仓：

```text
0 < account_qty < managed_qty
→ managed_qty 下调到 account_qty
→ 不补仓
```

如果账户仓位归零：

```text
account_qty == 0
→ managed position 关闭
```

### 2.4 Ownership 作为唯一仓位/订单修改边界

不再提供 `FutureExcludeSymbols` 人工排除列表。真实交易的修改权限只由 ownership 决定：

```text
managed by current owner = 允许该 owner 管理
unmanaged / other owner = 只读，不平仓、不撤单、不改单
```

手工仓位、其它 owner 仓位和来源不明仓位无需加入额外排除列表。

## 3. V3-5A：Ownership Foundation

第一步建立统一 ownership 数据模型和 Service。

核心对象：

- `futures_managed_positions`
- `futures_managed_orders`
- `OwnershipService`

至少表达：

- owner
- symbol / position_side
- managed_qty
- source_ref
- order intent
- client_order_id / exchange_order_id
- requested_qty / filled_qty
- status
- created_at / updated_at / closed_at

要求：

- 不把账户镜像表 `futures_positions` / `futures_orders` 当 ownership 真相源。
- 旧 `order` 表继续用于历史/统计兼容，不升级为唯一 ownership 账本。
- 同一 `(symbol, position_side)` 首版最多允许一个 active owner，避免 Binance 聚合仓位下产生归属歧义。

### V3-5A Gate

- 能明确区分 managed 与 unmanaged。
- ownership 缺失默认 fail closed。
- managed quantity 可独立于 account quantity 表达。
- Schema 通过 SQLite/MySQL sync Gate。

## 4. V3-5B：修复主 StartTrade 修改边界

这是当前真实资金风险最高的一步。

`StartTrade` 最终同时维护：

```text
accountPositions / accountOpenOrders
    → 全账户只读，用于风险、冲突和余额判断

managedPositions / managedOrders(owner=auto_strategy)
    → 主自动策略唯一可写集合
```

重点修改：

- `cancelTimeoutOrder()` 只能撤销 `auto_strategy` 自己登记的 managed order。
- 自动止损、止盈、AutoStopOrder、策略反转平仓只遍历 `auto_strategy` managed position。
- 平仓前重新获取账户数量，`close_qty = min(managed_qty, current_account_qty)`。
- 已存在 unmanaged 或其它 owner 同方向仓位时，新的 `auto_strategy` 开仓安全拒绝。
- 不再依赖人工排除列表；ownership 无法确认时一律 fail closed。

### V3-5B Gate

- 手工仓位不会被 `StartTrade` 平掉。
- 手工 LIMIT 挂单不会被 timeout cancel 撤销。
- `auto_strategy` 自己创建的仓位仍能正常退出。
- 人工加仓后只平 managed quantity。
- 人工减仓后 managed quantity 自动收缩，不补回。

## 5. V3-5C：其它真实交易模块 Owner 化

在主 `StartTrade` 安全边界稳定后，逐个接入：

1. `new_coin_rush`
2. `notice_auto_order`
3. `funding_rate`
4. `agent_trade`

统一原则：

```text
生成 clientOrderId
→ 先登记 owner / source_ref
→ 提交 Binance
→ 根据真实成交更新 filled_qty / managed_qty
→ 后续只能由原 owner 管理
```

部分成交必须按实际成交数量归属，不能把请求数量直接当成 managed quantity。

### V3-5C Gate

- 不同 owner 之间不能互相平仓、撤单或改单。
- Agent 创建的仓位不能被 `auto_strategy` 处理。
- 同方向存在其它 owner 或 unmanaged 仓位时，新 owner 默认拒绝开仓。

## 6. V3-5D：Restart / Reconcile

程序重启或用户手动对账时：

- 先读取 Binance 当前账户真实状态。
- 再读取本地 managed positions / managed orders。
- 通过 clientOrderId / exchangeOrderId / source_ref 对账。
- 本地落后时，以 Binance 外部事实修正数量和状态。
- 不根据账户当前仓位自动创建 ownership。
- 不根据旧 `order` 表自动认领历史仓位。
- 无法确定时进入 `reconcile_required`，不重复提交订单。

重点覆盖：

- 人工部分平仓。
- 人工额外加仓。
- LIMIT 部分成交。
- WS 断线后 API 补偿。
- 下单请求超时但交易所实际已接单。

### V3-5D Gate

- 重启后 ownership 不丢失。
- 历史手工仓位不会因为启动 Reconcile 被认领。
- 网络超时优先查询 clientOrderId，不盲目重发。
- WS 模式和直接 API 模式 ownership 结果一致。

## 7. V3-5E：Agent 受控交易仓位生命周期

Ownership 稳定后，再补齐 V2 `service/agenttrade` 的“开仓之后怎么办”。

首版只做：**单次开仓、单仓位、整仓平仓**。

```text
Approved Proposal
      ↓
Entry
      ↓
owner=agent_trade Managed Position
      ↓
Protective Stop
      ↓
可选一个整仓 Take Profit
      ↓
整仓 Close
      ↓
清理剩余保护单
```

要求：

- Stop/TP 只能针对 `owner=agent_trade` 的 managed quantity。
- 使用 reduce-only / close-position 等不会反向开仓的语义。
- Entry 成功但 Stop 建立失败时，不能显示为普通成功。
- 固定重试仍失败则进入 `protection_failed` 并通知用户。
- 提供“重新对账”和“关闭此系统仓位”。
- LLM 不参与 Stop 创建失败、紧急平仓等确定性安全动作。

### V3-5E Gate

- Entry 成功后 Stop 成功，或明确进入 `protection_failed`。
- 重复 Execute / Close / Reconcile 不产生重复订单。
- 外部/手工仓位不提供 Agent 平仓能力。
- Fake Broker 覆盖 Entry、超时、Stop 失败、Close 和 Restart Recovery。

## 8. 风控边界

V2 Controlled Trade 已具备：

- Allowed Symbols / LONG-SHORT 开关。
- Price freshness / slippage。
- 同 Symbol 仓位/挂单检查。
- 最大持仓数、最大杠杆。
- 单笔 Risk / Notional / Total Exposure。
- Cooldown。
- 人工 Approve / Reject。
- Kill Switch。
- MARKET Entry + clientOrderId Reconcile。

因此 V3-5 不重新建设大型 Portfolio Risk Engine。

如实际使用确有需要，只允许增加简单可选保护，例如：

- 当日已实现亏损上限，默认 0 = 关闭。
- 账户回撤上限，默认 0 = 关闭。

## 9. UI

不建设大型交易终端，优先复用现有页面：

- 配置中心保留合约交易总开关，但移除 `FutureExcludeSymbols`。
- 合约交易总开关切换后直接保存，不再弹出二次确认框；交易安全由 ownership 边界保证。
- `合约交易 → 仓位与受控交易` 展示 Agent Managed Position、Stop/TP、order id、clientOrderId 和最近 Reconcile 时间。
- 对 unmanaged / 其它 owner 仓位明确显示归属，但不提供越权操作按钮。

## 10. 本阶段明确不做

- 不做 VaR、Monte Carlo、相关性矩阵或机构级 Portfolio Risk。
- 不做复杂 Risk Reservation 平台。
- 不做多 owner 共持同一方向聚合仓位。
- 不做多账户、多用户、审批角色。
- 不做网格、高频、复杂算法订单路由。
- Agent 首版不做加仓、减仓、金字塔、反手和多段 TP。

## 11. V3-5 Definition of Done

V3-5 完成的标准不是“Agent 能下单”，而是：

> **整个项目所有真实合约交易模块都具备明确 ownership；任何模块只能修改自己登记的订单和 managed quantity；无法确认来源的仓位/订单始终 fail closed；在此基础上 Agent 受控交易具备基本 Stop、整仓 Close 和 Restart/Reconcile 能力。**
