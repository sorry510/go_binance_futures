# Phase V4-5：AI Position Manager

## 1. 目标

让 Agent 不只负责“入场前分析”，还可以在真实 managed position 持有期间读取完整仓位 Context，并提出结构化管理建议。

AI 仍然不能直接操作 Binance。

## 2. Position Context

AI Review 至少读取：

- Symbol / LONG-SHORT。
- Entry Price / Current Price。
- Managed Qty / Account Qty。
- Leverage。
- Unrealized PnL / Leveraged ROI。
- Realized PnL。
- Fees / Funding。
- Holding Time。
- Stop / TP 状态。
- MarketCondition。
- 最新 Technical / Flow / Market Intelligence Evidence。
- 原始 Proposal / TradingPlan。
- Invalidation Conditions。

所有确定性数字由 Go/SQL/Exchange 数据提供。

## 3. AI 输出契约

第一版只允许：

```text
HOLD
REDUCE
TIGHTEN_STOP
CLOSE
```

结构化字段例如：

```json
{
  "action": "REDUCE",
  "ratio": 0.5,
  "new_stop": 0,
  "reason": "...",
  "evidence": []
}
```

## 4. 明确禁止的动作

AI 不允许输出或执行：

```text
ADD_POSITION
REVERSE
INCREASE_LEVERAGE
REMOVE_STOP
CLAIM_MANUAL_POSITION
```

## 5. Action Proposal

AI Position Review 只产生 `PositionActionProposal` 或等价结构。

执行链路：

```text
AI Position Review
  ↓
Structured Action Proposal
  ↓
Deterministic Validation
  ↓
用户确认
  ↓
Ownership-aware Executor
```

不允许 AI 自己批准自己的建议。

## 6. Trigger

第一版不需要高频运行。

可由以下方式触发：

- 用户手动点击“AI 复盘当前仓位”。
- 重要 Market Event / FastMove 后。
- MarketCondition 明显改变后。
- 固定低频 Scheduler（如需要，再复用现有 Scheduler）。

默认优先手动和事件触发，避免无意义 LLM 消耗。

## 7. Validation

Deterministic Validator 至少检查：

- Proposal 对应 active managed position。
- ratio 合法且不超过 managed qty。
- TIGHTEN_STOP 只能收紧风险，不能扩大止损距离。
- Close/Reduce 仍受 account qty 限制。
- `reconcile_required` 状态禁止自动 mutation。
- Position Guardian 当前状态必须健康或明确可执行。

## 8. UI

复用“仓位与受控交易”详情：

- `AI 分析当前仓位`。
- 展示建议、Evidence 和确定性指标。
- 用户可确认或拒绝 Action Proposal。
- 所有操作可回看 Audit。

## 9. 验收 Gate

- AI 无法直接访问 Binance Mutation Tool。
- 非结构化输出不能进入执行链路。
- REDUCE / TIGHTEN_STOP / CLOSE 都必须经过 deterministic validation。
- AI 建议与真实执行可以完整追踪。
- LLM 失败不会影响 Position Guardian、Reconcile 或保护单。

## 10. 本阶段不做

- 不自动批准。
- 不自动加仓。
- 不自动反手。
- 不自动取消 Stop。
- 不做高频 AI 仓位管理。
