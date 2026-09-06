# Phase V2-12：Proposal、Risk Engine 与受控执行

## 目标

建立 AI 辅助执行边界。默认仍不允许 Agent 自主真实下单。

```text
Agent -> TradeProposal -> Deterministic Risk Engine
      -> Approval -> Execution Service -> Audit
```

## Proposal

至少包含 symbol、side、entry 条件、stop、take profit、Evidence、来源 Task、创建时间、有效期和失效条件。LLM 不决定最终可下单数量。

## Risk Engine

独立校验允许币种、MarketCondition、仓位、总暴露、杠杆、单笔风险、止损、价格 freshness、滑点、重复订单、cooldown、kill switch。

## Approval

首版人工确认。未来自动批准只能由明确 deterministic policy 完成，不能询问另一个 LLM“是否批准”。

## 外部 MCP

Risk/Trade 不允许直接调用任意 MCP Tool。即使外部 MCP 提供下单能力，也必须映射为 RiskTrade 且默认禁用；正式执行优先复用本项目 Execution Service。

## Imported Skill

Skill 包不能声明自己拥有交易权限；`allowed-tools` 中出现交易工具也只是权限请求，系统默认拒绝。

## 验收

- [x] Prompt Injection 不能绕过 Risk Engine。
- [x] write/trade 全链路有幂等和审计。
- [x] kill switch 可立即关闭 AI 执行。
- [x] 外部 MCP/Skill 不能自授交易权限。

## 实现摘要

- 新增 `agent_trade_proposals`、`agent_trade_executions`、`agent_trade_audits`，Proposal、Execution 与 Audit 独立持久化。
- Proposal 只能从成功的 `symbol_analysis` / `TradingPlanV1` 创建；neutral 或非结构化 Task 不能进入交易链路。
- Risk Engine 使用确定性 Go 代码检查白名单、方向、当前 MarketCondition 与漂移、行情 freshness、Entry Zone、Stop、滑点、仓位/挂单、重复订单、cooldown、杠杆、单笔风险、单笔名义金额、总暴露和 kill switch。
- 最终 quantity 根据最大风险、止损距离、最大名义金额及交易对 StepSize 确定性计算，LLM 不提供最终下单数量。
- 首版只支持人工 Approval；批准前和真实执行前都会重新执行完整 Risk Engine。
- Execution 使用数据库 CAS 抢占 `approved -> executing`，并使用确定性 Binance `client_order_id` 实现幂等。
- Binance 提交结果不确定时禁止自动重下单，只允许按 `client_order_id` Reconcile。
- Broker 提交前再次读取 kill switch，关闭后即使 Proposal 已批准也不能进入真实订单提交。
- Agent Runtime 的 `RiskTrade` 仍保持全局禁用；真实执行只通过独立 Controlled Execution Service。
- 外部 MCP 的交易能力会被映射为 `RiskTrade` 并强制禁用，不能启用或授权给 Skill；Portable Skill 同样不能获得 Trade Tool Grant。
- 前端新增 **AI → 受控交易**，提供 Proposal、Risk、人工批准/拒绝、执行、Reconcile 和 Audit；配置中心新增独立 kill switch、交易白名单及 Risk Policy 参数。
- DB Version 3 仅初始化 V2-12 的安全默认值：真实执行关闭、白名单为空；新增表和字段仍由 `orm.RunSyncdb` 管理。
