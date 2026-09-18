# AI Agent V4 开发计划：Real Trading Lifecycle & Position Intelligence

## 1. V4 定位

V4 不再扩展回测系统，也不继续堆叠 Agent 基础设施。V1～V3 已经完成 Runtime、Skill、Tool、Conversation、Memory、MCP、Workflow、Market Intelligence、Opportunity、受控交易、Ownership、Reconcile、Testnet、安全执行和交易复盘。

V4 聚焦当前系统最明显的真实交易缺口：**系统已经能够发现机会并安全开仓，但开仓后的真实仓位管理仍然较初级。**

V4 的目标是把现有链路从：

```text
Opportunity
  ↓
Symbol Analysis
  ↓
Trade Proposal
  ↓
Risk
  ↓
Entry
  ↓
Stop + first TP
  ↓
Full Close
```

升级为：

```text
Opportunity
  ↓
Symbol Analysis
  ↓
Trade Proposal
  ↓
Risk
  ↓
Entry
  ↓
Managed Position
  ├─ Position Guardian
  ├─ Live Trade Ledger
  ├─ Partial Reduce / Multi-TP
  └─ AI Position Review
           ↓
     HOLD / REDUCE / TIGHTEN_STOP / CLOSE
           ↓
   Deterministic Validation
           ↓
        用户确认
           ↓
   Ownership-safe Execution
```

V4 的核心不是让 AI 获得更多交易自由，而是让真实仓位从建立到结束都具备更完整的保护、归因、管理和可解释性。

## 2. V1～V3 已有基础

V4 直接复用以下能力，不重新建设：

- Agent Runtime / Task / Conversation / Memory。
- Native Skill / Portable Skill / MCP Client / Tool Runtime。
- Model Gateway、Trace、Observability、System Dashboard。
- Market Intelligence、Opportunity Watch、Symbol Analysis、TradingPlanV1。
- Trade Proposal、Deterministic Risk、人工 Approval。
- Futures Ownership、Managed Position、Managed Order、Restart/Reconcile。
- Binance Futures Hedge Mode、明确 `positionSide` 的下单模型。
- Testnet E2E、Stop/TP、人工减仓后 managed quantity 收缩。
- Outcome Review 与真实交易 Audit 链路。

## 3. 当前结构性缺口

### 3.1 仓位保护缺少统一 Guardian

当前 Ownership 会周期 Reconcile，但不同 owner 的 Protection 修复能力并不统一。Agent Trade 的 Stop/TP 主要在 Entry 后建立，缺少一个长期检查所有 managed position 的统一保护层。

### 3.2 Live 真实收益归因不完整

当前系统已经能拿到 Exchange Order、Commission、Realized PnL，并已有 Income API，但尚未完整归因到 `owner / source_ref / proposal / managed position`，因此 Live Outcome Review 无法可靠展示完整 Net PnL、Fee 和 Funding。

### 3.3 仓位管理动作过少

TradingPlanV1 支持多个 Take Profit，但真实 Agent 生命周期目前只使用第一个 TP；系统主要支持全量 Close，没有统一的 Partial Reduce、Multi-TP、Breakeven/Tighten Stop 管理路径。

### 3.4 Position Conflict 规则分散

Auto Strategy、Agent Risk、Ownership 分别有仓位冲突判断。目前为了安全，系统会阻止同币再开仓；这些规则应被整理成明确的 Position Policy，而不是继续散落在不同模块。

## 4. 关于多空双开与加仓

### 多空双开

底层已经使用 Binance Hedge Mode，并按 `symbol + positionSide` 管理 LONG/SHORT。当前无法同币多空双开的主要原因是上层 Policy 主动禁止，而不是 Binance 或 Ownership 基础结构不支持。

V4 不立即放开。它只在 V4-6 作为**默认关闭的可选能力**重新评估。

### 同方向加仓

V4 明确不做。

虽然底层 Fill 能更新 `managed_qty` 和加权 `entry_price`，但当前一条 managed position 只有一个 `source_ref`，直接支持多次独立加仓会引入 Position Leg / Lot、Leg-level Stop/TP、风险预算和收益归因复杂度，不符合当前个人项目的收益/复杂度比例。

## 5. Phase 索引

| Phase | 优先级 | 目标 |
| --- | --- | --- |
| [V4-0](./00-phase-v4-0-real-trading-baseline.md) | P0 | 冻结真实交易、Ownership、Protection、Reconcile 基线 |
| [V4-1](./01-phase-v4-1-position-guardian.md) | P0 | 建立统一 Managed Position Guardian，持续检查和修复仓位保护 |
| [V4-2](./02-phase-v4-2-live-trade-ledger.md) | P0 | 建立 Live Trade Ledger，可靠归因 PnL / Fee / Funding |
| [V4-3](./03-phase-v4-3-position-policy.md) | P1 | 统一 Position Conflict / Exposure Policy，不立即增加交易自由度 |
| [V4-4](./04-phase-v4-4-partial-reduce-multi-tp.md) | P1 | 支持 Partial Reduce 与 Multi-TP，补齐真实仓位管理动作 |
| [V4-5](./05-phase-v4-5-ai-position-manager.md) | P1 | 让 AI 在持仓阶段提出结构化仓位管理建议，但仍不能直接交易 |
| [V4-6](./06-phase-v4-6-optional-hedge-position.md) | P2 / 可选 | 在前述能力稳定后，再评估同币 LONG/SHORT 双开 |
| [V4-7](./07-phase-v4-7-finalization.md) | P2 | V4 收尾、长期运行检查、文档和最终 Testnet Gate |

## 6. 开发顺序

```text
V4-0 Real Trading Baseline
  ↓
V4-1 Position Guardian
  ↓
V4-2 Live Trade Ledger
  ↓
V4-3 Position Policy
  ↓
V4-4 Partial Reduce / Multi-TP
  ↓
V4-5 AI Position Manager
  ↓
V4-6 Optional Hedge Position（可选）
  ↓
V4-7 Finalization
```

V4-6 不作为 V4 完成的强制条件。如果 V4-5 完成后仍没有明确实际需求，可直接保持关闭并进入 V4-7。

## 7. V4 明确不做

- 不做任何新的 Backtest 功能、批量回测、稳健性分析或自动参数搜索。
- 不做 Portfolio Optimizer、VaR、复杂风险预算或自动调仓。
- 不做同方向多次加仓 / Position Leg / Lot Accounting。
- 不做 AI 自动反手。
- 不做 AI 自动批准真实交易。
- 不允许 LLM 绕过 Deterministic Risk / Position Policy / Ownership。
- 不增加新的大型 Agent Runtime、Multi-Agent、Memory、MCP 或 LLM Provider 平台能力。
- 不做多账户、Copy Trading、企业级审批或多租户。

## 8. V4 Definition of Done

V4 完成时应满足：

- 每个 managed position 都能被统一 Guardian 持续检查保护状态。
- Stop/TP 丢失、数量不匹配或 stale protection 可以被可靠发现并安全处理。
- Live 交易可以可靠归因到 Owner / Source / Proposal，并计算 Realized PnL、Fee、Funding 和 Net PnL。
- 系统支持安全的部分减仓和多个 Take Profit，不要求整仓一次性退出。
- Position Conflict 规则集中、可测试，默认仍维持 V3 的保守行为。
- AI 可以基于真实仓位 Context 提出 HOLD / REDUCE / TIGHTEN_STOP / CLOSE，但不能直接提交 Binance 订单。
- 所有真实 Mutation 继续经过 Ownership-aware Executor。
- V4 不引入同方向加仓复杂度。
- V4 不修改或依赖回测系统。
