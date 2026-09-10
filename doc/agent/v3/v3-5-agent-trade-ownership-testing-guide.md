# V3-5 Agent Trade Ownership 人工测试指南

本文用于以后人工验证 `agent_trade` 是否正确创建 Ownership，并确认 Agent 受控交易只管理自己创建的仓位和订单。

> 建议仅使用极小仓位测试。本文涉及真实 Binance Futures 下单，不建议使用大额资金。

## 1. 测试目标

需要验证完整链路：

```text
Symbol Analysis Task
    ↓
Controlled Trade Proposal
    ↓
Risk PASS
    ↓
人工 Approve
    ↓
真实 Entry
    ↓
owner=agent_trade Managed Position
    ↓
Protective Stop
    ↓
可选一个 Take Profit
```

测试成功的核心标准是：Entry 成交后，系统必须建立明确的 `agent_trade` Ownership，而不是仅在 Binance 账户中出现一个无法区分来源的仓位。

## 2. 前置条件

测试前确认：

- 已执行 `./go_binance_futures sync db`，数据库版本为 10。
- `futures_managed_positions` 和 `futures_managed_orders` 已存在。
- Binance Futures API Key 可正常交易。
- `AgentTradeExecutionEnable = 1`。
- 测试 Symbol 在 Agent Trade Allowed Symbols 中。
- 单笔最大 Risk / Notional 设置为测试可接受的极小值。
- 测试 Symbol 当前没有手工或其它 owner 的同方向仓位/开仓挂单。

数据库可先检查：

```sql
SELECT version FROM config LIMIT 1;

SHOW TABLES LIKE 'futures_managed%';
```

预期数据库版本为 `10`，并看到：

```text
futures_managed_positions
futures_managed_orders
```

## 3. 创建 Symbol Analysis Task

进入：

```text
AI → 单币分析
```

选择一个用于测试的合约，完成一次正常单币分析。

要求分析结果：

- Task 状态为 `succeeded`。
- 方向必须是 `long` 或 `short`，不能是 `neutral`。
- Trading Plan 中必须存在 Entry Zone、Stop Loss、Take Profit 和 Evidence。

记录成功 Task ID，例如：

```text
task_xxxxxxxxx
```

后续 Controlled Trade Proposal 必须由这个 Task 创建。

## 4. 创建 Controlled Trade Proposal

进入：

```text
合约交易 → 仓位与受控交易
```

在“从 Task 创建 Proposal”中输入刚才的 Task ID，然后创建 Proposal。

创建成功后检查 Proposal：

```text
Risk Status = pass
Status = awaiting_approval
```

如果 Risk 未通过，不要为了测试绕过 Risk。先根据页面提示修正 Allowed Symbols、价格新鲜度、仓位冲突、Risk/Notional 等配置。

## 5. 人工批准

打开 Proposal 详情，点击：

```text
人工批准
```

预期：

```text
Status = approved
```

批准本身不会下单。真正执行时系统还会再次运行完整 deterministic Risk Engine。

## 6. 执行真实 Entry

确认仓位金额足够小后点击：

```text
执行真实交易
```

系统会执行：

```text
重新检查 Risk
→ 生成确定性 Client Order ID
→ 先登记 futures_managed_orders
→ 提交 Binance MARKET Entry
→ 根据实际成交数量更新 Ownership
```

Entry 成交后预期：

```text
Proposal Status = executed
```

如果 Entry 已成交但 Protective Stop 无法确认，则必须是：

```text
Proposal Status = protection_failed
```

此时不能把它当成普通执行成功，必须先处理保护单问题。

## 7. 页面检查 Ownership

仍在：

```text
合约交易 → 仓位与受控交易
```

查看“合约仓位 / 订单归属”，必要时点击“对账仓位归属”。

账户仓位应类似：

| Symbol | Side | Account Qty | Owner | Managed Qty | Source |
| --- | --- | ---: | --- | ---: | --- |
| BTCUSDT | LONG | 0.001 | `agent_trade` | 0.001 | Proposal ID |

必须满足：

- `owner = agent_trade`
- `managed_qty > 0`
- `source_ref = 当前 Proposal ID`
- `managed_qty` 来源于真实成交量，而不是简单复制请求数量

## 8. 检查 Managed Orders

Managed Orders 至少应看到 Entry 和 Protective Stop；如果 Trading Plan 有 Take Profit，还应看到一个 TP。

典型记录：

| Intent | Owner | Status | Client Order ID | Binance Order ID |
| --- | --- | --- | --- | --- |
| `open` | `agent_trade` | filled | `agt_...` | 有值 |
| `stop_loss` | `agent_trade` | submitted/new | `agt_sl_...` | 有值 |
| `take_profit` | `agent_trade` | submitted/new | `agt_tp_...` | 有值 |

每条保护单都应满足：

- `owner = agent_trade`
- `source_ref = 当前 Proposal ID`
- Client Order ID 非空
- Binance Order ID 非空
- Stop/TP 数量不得超过该 Proposal 的 managed quantity

同时到 Binance Futures 页面确认真实 Stop 已存在。

V3-5 首版 Take Profit 只使用 Trading Plan 的第一个 TP，不做多段止盈。

## 9. 数据库核对

先查 Managed Position：

```sql
SELECT id, owner, symbol, position_side, managed_qty,
       entry_price, status, source_ref, last_reconciled_at
FROM futures_managed_positions
WHERE owner = 'agent_trade'
ORDER BY id DESC
LIMIT 10;
```

再查 Managed Orders：

```sql
SELECT id, owner, symbol, position_side, intent,
       client_order_id, exchange_order_id,
       requested_qty, filled_qty, order_type,
       status, source_ref, last_reconciled_at
FROM futures_managed_orders
WHERE owner = 'agent_trade'
ORDER BY id DESC
LIMIT 20;
```

重点核对当前 Proposal：

```text
position.owner        = agent_trade
position.source_ref   = Proposal ID
order.owner           = agent_trade
order.source_ref      = Proposal ID
open.filled_qty       > 0
position.managed_qty  = 已归属的实际成交数量
```

如果 Binance 已有仓位，但数据库没有对应 `agent_trade` managed record，系统必须把它视为 unmanaged，不能自动认领。

## 10. 测试整仓 Close

在 Proposal 详情中，只有仍存在 `owner=agent_trade` Managed Position 时才应出现：

```text
关闭此系统仓位
```

点击后确认提示，再执行 Close。

预期行为：

- Close 数量最多为 `min(managed_qty, Binance 当前 account_qty)`。
- 手工额外加仓不会被一起平掉。
- Close 前先撤该 Proposal 的 TP，Stop 保留作为最后保护。
- Close 确认成功后再清理剩余 Stop/TP。
- Close 结果不确定时不能盲目重发，Stop 不能提前删除。
- 成功后 Proposal 状态变成 `closed`。

数据库中对应 Managed Position 最终应为 closed/归零状态，保护单应为 filled/canceled 等终态。

## 11. 人工加仓隔离验证（强烈建议至少测试一次）

假设 Agent Entry 后：

```text
managed_qty = 0.001 BTC
```

随后在 Binance 手工同方向增加：

```text
0.002 BTC
```

此时账户总量为 `0.003 BTC`，对账后必须保持：

```text
account_qty = 0.003
managed_qty = 0.001
```

然后执行“关闭此系统仓位”，最多只能平 `0.001 BTC`，手工增加的 `0.002 BTC` 必须保留。

## 12. 人工减仓验证

假设：

```text
managed_qty = 0.003 BTC
```

手工在 Binance 平掉 `0.002 BTC` 后，只剩 `0.001 BTC`。

执行 Ownership Reconcile 后应变成：

```text
account_qty = 0.001
managed_qty = 0.001
```

系统不得补仓，也不得继续按旧的 `0.003 BTC` 平仓。

## 13. 常见失败状态

### `risk_rejected`

Proposal 没通过 deterministic Risk。根据 Risk Checks 修复配置或账户冲突，不要绕过检查。

### `execution_uncertain`

Entry 请求结果不确定。使用“按 Client Order ID 对账”，禁止再次提交 Entry。

### `protection_failed`

Entry 已真实成交，但 Protective Stop 尚未确认成功。这是需要立即关注的安全状态。

优先操作：

```text
重新对账
```

如果仍无法恢复保护，不要重复 Entry；根据页面 Ownership 和 Binance 实际仓位决定是否使用“关闭此系统仓位”。

### `closed`

该 Proposal 对应的 Agent Managed Position 已关闭。重复 Close 不应产生新订单。

## 14. 最终通过检查表

一次完整人工测试至少确认以下项目：

- [ ] Symbol Analysis Task 成功且方向不是 neutral。
- [ ] Proposal 创建后 Risk PASS。
- [ ] 必须人工 Approve 后才能 Execute。
- [ ] Entry 只提交一次，存在确定性 Client Order ID。
- [ ] `futures_managed_positions` 出现 `owner=agent_trade`。
- [ ] `source_ref` 正确关联当前 Proposal ID。
- [ ] `managed_qty` 使用实际归属成交量。
- [ ] Protective Stop 已真实存在于 Binance。
- [ ] Stop 记录存在于 `futures_managed_orders`。
- [ ] 如创建 TP，只创建首个 TP，且属于当前 Proposal。
- [ ] 页面能展示 Managed Position、Stop/TP、Client Order ID 和 Binance Order ID。
- [ ] 手工/其它 owner 仓位没有 Agent Close 按钮。
- [ ] Agent Close 不超过 managed quantity。
- [ ] Close 成功后清理 sibling protection orders。
- [ ] 重启/Reconcile 后 Ownership 不丢失，也不会认领手工仓位。

以上全部通过后，可以认为 `Agent Trade → Ownership → Protection → Close` 人工验证完成。
