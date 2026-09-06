# Phase V3-7：Opportunity Pipeline 与分级自动化

## 目标

让系统从“用户主动问某个币”升级为“持续发现机会”，但默认先 Shadow，再逐步扩大自动化。

## Pipeline

```text
Market Event / Scanner
        ↓
Candidate Symbols
        ↓
Multi-Agent Team
        ↓
Opportunity
        ↓
Trade Proposal
        ↓
Trade Risk + Portfolio Risk
        ↓
Execution Mode
```

Opportunity 必须保存来源 Signal/Event、Evidence、Team Run、Strategy Version 和失效条件。
## 三种模式

### Shadow

自动发现、分析、生成 Proposal、计算 Risk，但绝不创建真实订单；记录“如果执行会怎样”。这是默认模式。

### Assisted

自动发现并生成通过 Risk 的 Proposal，由用户一次确认后进入受控执行。不建设审批流，只有单用户确认。

### Auto

只有满足明确 deterministic Policy 的 Proposal 才能自动执行。Policy 至少绑定 Allowed Symbols、Strategy Version、最大 Risk、Portfolio Risk、MarketCondition、时间窗口、每日次数和 Kill Switch。

LLM 的“强烈建议执行”不是 Auto 条件。

## 验收 Gate

- Shadow 模式运行时 Broker Submit 次数始终为 0。
- Assisted 模式没有用户确认不能进入执行。
- Auto 模式只接受固定 Policy，Prompt/MCP/Skill 无法修改 Policy 结果。
- 同一 Opportunity 不能重复生成多个真实 Execution。
- Kill Switch 能立即阻止新的自动执行。
- Shadow/Assisted/Auto 的结果可在同一页面对比。

## 本阶段不做

- 不做复杂审批权限。
- 不做 LLM 自我批准。
- 不默认开启 Auto；升级后仍保持保守默认值。
