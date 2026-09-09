# Phase V3-5：真实交易安全与仓位生命周期

> 定位：P0。基于 V2 已完成的 Controlled Trade 补齐“开仓之后怎么办”，不建设大型 Portfolio Risk 系统。

## 1. 为什么需要这一阶段

V2 的 `service/agenttrade` 已经具备：

- Allowed Symbols / LONG-SHORT 开关。
- 最新价格 freshness 和预估滑点限制。
- 同 Symbol 已有仓位/挂单时禁止重复开仓。
- 最大持仓数、最大杠杆、单笔最大 Risk、单笔最大 Notional、总 Exposure。
- Symbol cooldown。
- 人工 Approve / Reject。
- MARKET Entry、clientOrderId 幂等和提交失败后的 Reconcile。
- 全局真实交易 Kill Switch。

因此 V3-5 **不重新实现 Portfolio Risk Engine**。真正缺少的是：开仓成功以后如何确保仓位始终可控，以及系统如何保证只操作自己创建的仓位。

## 2. 核心原则：只管理自己的仓位

系统必须能明确识别“Managed Position”。来源至少可以追踪到：

```text
AgentTradeProposal
      ↓
AgentTradeExecution
      ↓
clientOrderId / exchangeOrderId
      ↓
Binance Position
```

只有能够从本地 Execution 和 Binance 订单记录确认来源的仓位，才能由 Agent 交易链路执行 Stop、TP 或主动平仓。

以下仓位一律不主动管理：

- 用户手工下单产生的仓位。
- 其他机器人/脚本产生的仓位。
- 无法确认来源的历史仓位。

## 3. 首版生命周期

首版保持简单：**单次开仓、单仓位、整仓平仓**。

```text
approved
   ↓
entry_submitting
   ↓
entry_filled
   ↓
protecting
   ↓
protected
   ↓
closing
   ↓
closed
```

异常状态只保留真正必要的：

- `execution_uncertain`：提交结果未知，需要按 clientOrderId 查询。
- `protection_failed`：Entry 已成交，但保护单没有成功建立。
- `reconcile_required`：本地和 Binance 状态不一致。

不建设十几个细粒度状态。

## 4. 保护逻辑

Entry 成交后：

1. 根据 Proposal 的 `stop_loss` 创建保护性 Stop。
2. 可选创建一个整仓 Take Profit；首版不做多段 TP。
3. Stop/TP 必须使用 reduce-only / close-position 等不会反向开仓的语义。
4. 如果 Stop 创建失败，不能把交易显示为普通成功；按固定重试次数处理，仍失败则进入 `protection_failed` 并通知用户，必要时提供确定性安全平仓动作。

LLM 不参与保护单失败后的临场决策。

## 5. 平仓

新增受控整仓平仓能力：

- 只允许关闭 Managed Position。
- 平仓前重新确认 Symbol、方向、当前持仓数量和归属。
- 平仓订单使用确定性 clientOrderId，重复点击不能产生重复平仓。
- 平仓完成后清理仍存在的保护单。

首版不做：加仓、减仓、金字塔、反手、分批止盈。

## 6. Restart / Reconcile

程序启动或用户手动触发 Reconcile 时：

- 查询尚未结束的 AgentTradeExecution。
- 按 clientOrderId / exchangeOrderId 查询 Binance 真实订单。
- 检查对应 Managed Position 是否仍存在。
- 检查保护 Stop/TP 是否仍存在。
- 本地状态落后时，以 Binance 外部事实修正本地状态。
- 无法确定时进入 `reconcile_required`，不自动重复下单。

## 7. 风控只做必要补强

现有 V2 Risk 已覆盖大部分个人使用需求。本阶段最多补充两个可选 Kill Switch：

- 当日已实现亏损上限（默认 0 = 关闭）。
- 账户回撤上限（默认 0 = 关闭）。

不做 VaR、Monte Carlo、相关性风险、Sector Bucket、复杂 Reservation。单进程并发问题优先使用事务 / CAS / mutex 解决。

## 8. UI

优先扩展现有 `AI → 受控交易`，不新建大型交易终端：

- Proposal 详情显示 Entry、Stop、TP、Managed Position 状态。
- 显示 Binance order id / clientOrderId 和最后 Reconcile 时间。
- 提供“重新对账”和“关闭此系统仓位”按钮。
- 外部/手工仓位明确标注“非本系统管理”，不提供 Agent 平仓按钮。

## 9. 验收 Gate

- 用户手工仓位永远不会被 Agent 自动撤单、止损或平仓。
- Entry 成功后 Stop 建立成功，或明确进入 `protection_failed`，不会伪装成正常完成。
- 重复 Execute / Close / Reconcile 不制造重复订单。
- 网络超时通过 clientOrderId 查询，不直接再次提交。
- 重启后能恢复未结束 Managed Position 的基本状态。
- Fake Broker 覆盖 Entry 成功、提交超时、Stop 失败、Close、重启恢复。
- 自动测试不调用生产 Binance 下单。

## 10. 本阶段明确不做

- 不做机构级 Portfolio Risk Engine。
- 不做 Risk Reservation 系统。
- 不做复杂 LIMIT/算法订单路由。
- 不做加仓、减仓、反手、网格或高频交易。
- 不做多账户、多用户、审批角色。
