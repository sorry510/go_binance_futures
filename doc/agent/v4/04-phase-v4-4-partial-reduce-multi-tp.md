# Phase V4-4：Partial Reduce & Multi-TP

## 1. 目标

补齐当前真实仓位只能“一个 TP + 全量 Close”的限制，使系统能够在不引入 Position Leg 的前提下进行简单、安全的分批退出。

## 2. 新增核心动作

### Partial Reduce

统一支持：

```text
REDUCE 25%
REDUCE 50%
CLOSE 100%
```

执行时始终：

```text
requested_qty <= managed_qty
requested_qty <= current account qty
```

只允许减少，不允许借此增加仓位。

### Multi-TP

TradingPlanV1 已支持多个 Take Profit。V4-4 让真实生命周期能够使用多个目标。

第一版保持简单，可采用确定性分配，例如：

```text
TP1 50%
TP2 30%
TP3 20%
```

若目标只有两个，则按明确规则重新分配；不允许隐式超配总数量。

## 3. Protection Resize

发生以下事件后，Guardian 必须重新校验 Protection：

- TP 部分成交。
- Partial Reduce。
- 人工减仓。
- Close 部分成交。

剩余 Stop/TP 数量必须与新的 `managed_qty` 一致。

## 4. Position 模型

继续维持：

```text
一个 owner + symbol + positionSide
→ 一个 active managed position
```

不创建 Position Leg。

多个 TP 只是同一 managed position 的多个保护/退出订单。

## 5. 手工操作

现有“关闭此系统仓位”扩展为：

- 减仓 25%。
- 减仓 50%。
- 全部关闭。

实际下单前继续重新读取 account qty 和 managed qty。

## 6. Audit / Ledger

每次 Reduce / TP Fill 都必须写入：

- Source。
- Requested Qty。
- Filled Qty。
- Remaining Managed Qty。
- Realized PnL / Fee（由 Ledger 归因）。

## 7. 验收 Gate

至少覆盖：

- 50% Reduce 后 managed qty 正确下降。
- Stop 自动缩量。
- TP1 成交后 TP2/TP3 和 Stop 不 over-protect。
- 部分成交可重复 Reconcile 且不重复减少数量。
- Close 与 Reduce 并发不会越过 managed qty。
- 人工减仓后 Multi-TP 重新缩量。
- 最终归零后清理全部 sibling protection。

Testnet 必须真实验证 Partial Reduce + Multi-TP 生命周期。

## 8. 本阶段不做

- 不加仓。
- 不 Position Leg。
- 不动态改变每个 TP 的策略逻辑。
- 不自动反手。
