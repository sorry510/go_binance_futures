# AI Agent V3 开发计划

## 1. 定位

V2 已经完成 Runtime、Task/Conversation、Context、Tool Runtime、MCP Client、Portable Skill、Chat、Model Gateway、Memory、Observability、业务 Workflow，以及受控真实交易链路。

V3 不再继续堆 Agent 基础设施，而是把现有能力用于建立一个 **可验证、可回测、可组合风控、可逐步自动化的交易决策闭环**。

本项目是单用户自用，因此 V3 明确简化：

- 不建设企业级 RBAC、审批人、多人工作流。
- Strategy Lab 只保留显式 Promote、版本和回滚，不做复杂审批流。
- Portfolio Risk 只做确定性风险控制，不做“谁有权 override”的权限系统。
- 真实自动执行只由 deterministic Policy 决定，LLM 不能自己批准交易。
- 所有高风险操作仍保留清晰的开关、状态和审计，防止程序错误和误操作。
## 2. V3 总体闭环

```text
Market Data / News / Announcement / Alpha
                    ↓
            Market Intelligence
                    ↓
            Multi-Agent Team
                    ↓
               Opportunity
                    ↓
        Strategy / Trade Proposal
                    ↓
     Backtest / Paper / Portfolio Risk
                    ↓
          Execution / Position Manager
                    ↓
                 Binance
                    ↓
          Outcome Attribution
                    ↓
             Continuous Eval
```

V3 的重点不是“让 AI 更自由”，而是让每个判断都更容易验证、复现和追踪。
## 3. Phase 索引

| Phase | 优先级 | 目标 |
| --- | --- | --- |
| [V3-0](./00-phase-v3-0-baseline.md) | P0 ✅ | 冻结 V2 行为、Benchmark、Replay/Eval 基线 |
| [V3-1](./01-phase-v3-1-multi-agent.md) | P0 | Multi-Agent Team：多个专业 Agent 以 Typed Output 协作 |
| [V3-2](./02-phase-v3-2-market-intelligence.md) | P0 | Market Intelligence：统一 Fact/Event，接入新闻、公告、Alpha 与资金面 |
| [V3-3](./03-phase-v3-3-backtest.md) | P0 | Historical Backtest Engine：正式、可复现的历史回测 |
| [V3-4](./04-phase-v3-4-strategy-lab.md) | P1 | Strategy Lab：候选、回测、模拟、Promote、回滚和退役 |
| [V3-5](./05-phase-v3-5-portfolio-risk.md) | P1 | Portfolio Risk：账户级风险预算、组合暴露和并发 Reservation |
| [V3-6](./06-phase-v3-6-execution-lifecycle.md) | P1 | Execution Lifecycle：保护单、部分成交、加减仓、平仓和恢复 |
| [V3-7](./07-phase-v3-7-opportunity-automation.md) | P1 | Opportunity Pipeline：自动发现机会、Shadow/Assisted/Auto 分级执行 |
| [V3-8](./08-phase-v3-8-outcome-eval.md) | P2 | Outcome Attribution：交易结果归因、漂移检测和持续评测 |
| [V3-9](./09-phase-v3-9-agent-studio.md) | P2 | Agent Studio：Web Skill 编辑、Team 编排、测试、版本和发布 |
## 4. 严格开发顺序

```text
V3-0 Baseline
  ↓
V3-1 Multi-Agent
  ↓
V3-2 Market Intelligence
  ↓
V3-3 Historical Backtest
  ↓
V3-4 Strategy Lab
  ↓
V3-5 Portfolio Risk
  ↓
V3-6 Execution Lifecycle
  ↓
V3-7 Opportunity Automation
  ↓
V3-8 Outcome Eval
  ↓
V3-9 Agent Studio
```

约束：前一 Phase 未通过 Gate，不进入下一 Phase。V3-3 未完成前不允许策略自动晋级；V3-5/6 未完成前不扩大真实自动执行范围。
## 5. 统一原则

- 继续复用 V2 Runtime、Task、Context、Tool、Permission、MCP、Memory、Observability，不新建第二套 Agent 平台。
- LLM 负责分析、解释、提出候选；行情计算、回测、风控、仓位和执行由 Go Service 确定性完成。
- Multi-Agent 通过 Typed Input/Output 和子 Task 协作，不允许无限自由对话。
- 新闻和外部信息必须保存来源、事件时间、观测时间和 freshness，避免把旧消息当实时信息。
- 策略修改永远先产生新版本；AI 不能直接覆盖 Active Strategy。
- Portfolio Risk 没有普通“强制绕过”按钮；失败时返回明确原因。
- 自动执行必须建立在明确的 Symbol/Strategy/Risk 白名单和 deterministic Policy 上。
- 真实订单测试使用 Fake Broker/Testnet/Replay，不在自动测试中调用生产 Binance 下单。

## 6. V3 Definition of Done

- 一个交易机会可以由多个专业 Agent 协作完成，并能追踪每个子判断。
- 新闻、公告、Alpha、资金面和本地 Signal 使用统一 MarketFact/MarketEvent 表达。
- 策略可以基于固定历史数据做可复现回测，并计算手续费、Funding、滑点和风险指标。
- 策略具备 Candidate → Backtest → Paper → Active → Retired 生命周期和版本回滚。
- 多个同时出现的 Proposal 受账户级 Portfolio Risk 统一约束。
- 真实交易具备 Entry、保护性 Stop、Take Profit、部分成交、平仓和重启恢复能力。
- 系统可以在 Shadow/Assisted/Auto 三种模式下自动发现机会，默认不直接扩大真实自动交易。
- 每笔交易结果能归因到 Signal、Evidence、Agent/Model/Skill、Strategy、Risk 和 Execution。
- Web 可以编辑标准 Skill 和 Team 配置，但仍受现有 Tool/Permission 安全边界约束。
