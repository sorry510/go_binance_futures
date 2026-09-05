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

- [ ] Prompt Injection 不能绕过 Risk Engine。
- [ ] write/trade 全链路有幂等和审计。
- [ ] kill switch 可立即关闭 AI 执行。
- [ ] 外部 MCP/Skill 不能自授交易权限。
