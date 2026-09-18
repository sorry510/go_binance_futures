# Phase V4-1：Managed Position Guardian

## 1. 目标

建立统一的 Managed Position Guardian，让系统不仅在开仓时建立 Protection，而且在仓位整个生命周期中持续确认：**系统管理的真实仓位始终处于可解释、可恢复、受保护状态。**

## 2. 当前问题

V3 已有 Ownership 周期 Reconcile，但 Protection 修复能力并不统一：

- `notice_auto_order` 已有专门的 Protection Repair。
- `agent_trade` 主要在 Entry 后建立 Stop/TP。
- Stop/TP 被人工取消、失效、数量变化后，没有一个统一 Guardian 负责长期检查全部 owner。

## 3. Guardian 职责

周期检查所有 active managed position：

```text
Managed Position
  ↓
Account Qty / Managed Qty
  ↓
Managed Orders
  ↓
Exchange Orders
  ↓
Protection State
```

至少识别：

- Missing Stop。
- Stop Qty 与 managed qty 不一致。
- TP Qty 与 managed qty / 当前策略定义不一致。
- Stale Protection。
- Orphan Protection。
- Protection 进入 `reconcile_required`。
- Position 已关闭但 Protection 仍存活。
- 外部减仓后 Protection 需要缩量。

## 4. Repair 原则

- 先 Reconcile，再 Repair。
- 任何无法确认 Exchange 状态的订单都 fail-closed，不自动覆盖。
- 不允许 Guardian 扩大 `managed_qty`。
- 不认领 unmanaged/manual position。
- Repair 只能处理对应 owner 自己的订单。
- Stop Protection 优先级高于可选 TP。
- 修复动作必须有 Audit / 日志 / 可观测性记录。

## 5. 调度

复用现有低频 Ownership Reconcile 周期，不新建独立大型 Scheduler。

建议顺序：

```text
ReconcileAll
  ↓
Sync Closed Proposal
  ↓
Position Guardian Check
  ↓
Safe Repair
```

## 6. UI / Observability

在现有“仓位与受控交易”和系统看板中补充：

- protected / degraded / reconcile_required。
- Stop/TP 当前数量。
- 最近 Guardian 检查时间。
- 最近 Repair 结果。

不新建独立大型页面。

## 7. 验收 Gate

Fixture 至少覆盖：

- Stop 被取消后可安全重建。
- 人工减仓后 Stop/TP 数量缩小。
- Position 已关闭后 sibling protection 被清理。
- Exchange 状态不确定时不重复创建 Protection。
- manual/unmanaged position 不被 Guardian 处理。
- 不同 owner 不能互相 Repair。

Testnet 增加至少一次：

```text
Entry
→ 手工取消 Stop
→ Guardian 检测
→ 重建 Stop
→ Close
```

## 8. 本阶段不做

- 不做 AI 仓位判断。
- 不做 Multi-TP。
- 不做 Partial Reduce。
- 不做加仓。
- 不做 Hedge 双开。
