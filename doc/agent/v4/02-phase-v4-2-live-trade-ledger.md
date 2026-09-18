# Phase V4-2：Live Trade Ledger

## 1. 目标

建立真实交易的确定性收益账本，让每一笔 managed trade 最终都能回答：

```text
谁开的？
为什么开的？
实际成交多少？
手续费多少？
Funding 多少？
Realized PnL 多少？
最终 Net PnL 多少？
```

## 2. 当前基础

系统已有：

- `owner`
- `source_ref`
- Agent Proposal / Execution / Audit
- Managed Position / Managed Order
- Exchange Order ID / Client Order ID
- User Data 中的 Commission / Realized PnL
- Binance Income API

V4-2 的任务是把这些信息可靠归因，而不是重新计算交易所账本。

## 3. 账本边界

至少关联：

```text
Owner
SourceRef
ProposalID（存在时）
Symbol
PositionSide
Managed Order
Exchange Order
Fill
Commission
Realized PnL
Funding
```

最终支持：

```text
Gross Realized PnL
Commission
Funding
Net PnL
Holding Time
Entry VWAP
Exit VWAP
Closed Qty
```

## 4. 数据原则

- 交易所确定性数据优先于估算。
- 无法可靠归因时返回 unavailable，不猜测。
- Funding 必须按时间、Symbol、PositionSide 和仓位生命周期谨慎归因。
- manual/unmanaged 部分不能混入 managed trade PnL。
- 人工减仓导致 account qty 变化时，只记录系统能够可靠归因的 managed 部分。
- Ledger 必须可重建或可 Reconcile，不能只依赖单次内存事件。

## 5. Outcome Review

升级现有 Live Tab，支持真实展示：

- Net PnL。
- Gross PnL。
- Fees。
- Funding。
- Win / Loss。
- Holding Time。
- Owner / Source / Proposal。

Backtest/Paper/Live 仍保持完全分离。

## 6. Agent Context

Ledger 完成后，为后续 AI Position Manager 提供确定性字段：

- current managed qty。
- realized PnL。
- unrealized PnL。
- accumulated fee。
- accumulated funding。
- holding time。

LLM 不自行计算这些财务数据。

## 7. 验收 Gate

- Entry + Close 的 Realized PnL 与 Binance 数据可对账。
- Commission 可按 managed order/fill 归因。
- Funding 有明确归因规则并有 Fixture。
- Partial Fill 不重复记账。
- Reconcile 重放不重复累计。
- manual/unmanaged 数量不污染 managed Net PnL。
- Outcome Review 能从 Ledger 下钻回 Order / Proposal / Audit。

## 8. 本阶段不做

- 不做税务报表。
- 不做 Portfolio Accounting。
- 不做多账户总账。
- 不做 AI 盈亏计算。
