# AI Agent V2 开发计划

## 1. 定位

V1 已完成统一 Runtime、Skill、Tool、Task/Conversation、Scheduler、Permission、Observability 以及四个核心业务 Skill。V2 不重做 V1，而是在兼容现有 API 和业务链路的前提下，将 Agent 升级为可恢复、可扩展、可接入外部能力、可量化评测、可治理的长期平台。

V2 新增两个明确目标：

1. **第三方 HTTP MCP Client 接入**：系统作为 MCP Client 连接外部标准 HTTP MCP Server，并把远端 Tool/Resource/Prompt 纳入统一 Runtime、Permission、Trace 和 Context；V2 不负责扩展本项目自身 MCP Server。
2. **标准 Agent Skills 包**：可以导入符合 Agent Skills 开放规范的目录、ZIP 压缩包或单个 `SKILL.md` 文件，支持校验、版本、启停、回滚、渐进加载和权限审批。

标准参考：

- MCP Specification: https://modelcontextprotocol.io/specification/2026-07-28
- MCP Go SDK: https://github.com/modelcontextprotocol/go-sdk
- Agent Skills Specification: https://agentskills.io/specification

## 2. V2 总体架构

```text
Web / API / Scheduler / Event Bus
              |
        Agent Manager
              |
       Run Coordinator
      /      |       \
 Planner  Context   Policy
    |       Engine    Engine
    |         |         |
 Execution Engine ----+
    |         |
Model Gateway Tool Runtime
    |         |----------------------+
LLM Providers |                      |
              |                MCP Client Gateway
              |                      |
        Domain Services        External MCP Servers
              |
       Binance / DB / Scanner

Skill Layer:
Native Go Skills + Imported Agent Skills Packages

Horizontal:
Task / Conversation / Memory / Eval / Trace / Security / Versioning
```

## 3. Phase 索引

V2 从现在开始严格按照 Phase 编号顺序开发，不再使用“编号顺序”和“实际开发顺序”两套顺序。前一 Phase 未通过 Gate 时，不进入下一 Phase。

| Phase | 优先级 | 目标 |
| --- | --- | --- |
| [V2-0](./00-phase-v2-0-baseline.md) | P0 ✅ | 冻结 V1 契约、Fixture、版本和 Replay 基线 |
| [V2-1](./01-phase-v2-1-runtime.md) | P0 ✅ | Runtime V2、ExecutionStep、Checkpoint、恢复 |
| [V2-2](./02-phase-v2-2-context-evidence.md) | P0 ✅ | Context Engine、Evidence、Token/Freshness 管理 |
| [V2-3](./03-phase-v2-3-tool-runtime.md) | P0 ✅ | Tool Runtime、Schema、Envelope、并行、缓存 |
| [V2-4](./04-phase-v2-4-eval-replay.md) | P0 ✅ | Eval、Replay、Prompt/Skill Version、回归 Gate |
| [V2-5](./05-phase-v2-5-mcp-integration.md) | P0 ✅ | 第三方 HTTP MCP Client、Interactive OAuth、远端能力发现与治理 |
| [V2-6](./06-phase-v2-6-agent-skills.md) | P0 ✅ | 标准 Agent Skills 导入、版本、加载与安全 |
| [V2-7](./07-phase-v2-7-agent-chat.md) | P0 ✅ | Agent 对话入口、Conversation Context、Slash Skill 调用 |
| [V2-8](./08-phase-v2-8-model-gateway.md) | P1 ✅ | Model Capability、Router、Health、Fallback |
| [V2-9](./09-phase-v2-9-memory.md) | P1 ✅ | 长期 Memory、TTL、Scope 与管理 |
| [V2-10](./10-phase-v2-10-observability.md) | P1 ✅ | Trace、长期指标、运营与管理页面 |
| [V2-11](./11-phase-v2-11-workflows.md) | P1 ✅ | market_scan、strategy_review、strategy_experiment、alert_triage、daily_market_brief |
| [V2-12](./12-phase-v2-12-risk-execution.md) | P2 | Proposal、Risk Engine、Approval、受控执行 |

## 4. 严格开发顺序

```text
V2-0 Baseline
  ↓
V2-1 Runtime
  ↓
V2-2 Context / Evidence
  ↓
V2-3 Tool Runtime
  ↓
V2-4 Eval / Replay
  ↓
V2-5 HTTP MCP Client
  ↓
V2-6 Agent Skills
  ↓
V2-7 Agent Chat
  ↓
V2-8 Model Gateway
  ↓
V2-9 Memory
  ↓
V2-10 Observability
  ↓
V2-11 Workflows
  ↓
V2-12 Risk / Execution
```

顺序约束：

- 每次只实施当前 Phase，不提前开发后续 Phase 的业务能力。
- 当前 Phase 可以预留接口，但不能以“后续需要”为理由提前落完整实现。
- 每个 Phase 完成后先执行该 Phase 的自动测试、Replay/Eval Gate 和必要的手动验收，再进入下一 Phase。
- V2-4 在外部 MCP、可导入 Skill、Model Router 和 Memory 之前落地，确保后续能力都能被量化回归。
- V2-5 MCP 必须建立在 V2-3 Tool Runtime 之上；V2-6 Imported Skill 必须使用已经完成的 Context/Tool/Permission 边界。
- V2-7 Chat 只是现有 Agent 平台的人类入口，必须复用 Conversation、Manager、Runtime、Context、Tool、Permission 和 Task，不允许形成第二套执行体系。
- V2-11 只在基础设施、治理和可观测性完成后增加新的复杂业务 Workflow。
- V2-12 最后实施真实执行边界，Agent 在此之前始终只产生分析结果或 Proposal。

## 5. 统一原则

- V1 API、现有四个 Skill 和任务中心优先兼容。
- 确定性计算继续由 Go Service 完成，LLM 不替代行情计算、风控和回测。
- MCP 外部 Tool 与本地 Tool 使用同一权限和审计体系。
- Agent Skills 的 `allowed-tools` 只表示 Skill 请求的能力，不自动代表系统授权。
- 导入 Skill 中的脚本默认不执行；包自身不能声明自己“可信”。
- Skill 指令、Tool 数据、Memory、用户输入分别标记来源，防止 Prompt Injection 跨边界升级权限。
- 全局 Token/Tool/并发预算继续统一管理，不恢复每个 Skill 重复配置。
- 真实交易始终经过独立 Risk Engine 和 Approval，LLM 不直接下单。
- 普通数据库结构变化继续使用 ORM Model + `orm.RunSyncdb`，不重复维护字段 migration SQL。

## 6. V2 Definition of Done

V2 完成时应满足：

- Runtime 具备 Step/Plan、Checkpoint 和安全恢复能力。
- Context 有统一预算、Freshness、Evidence 和渐进加载。
- Tool Runtime 能统一管理本地 Tool 和 MCP Tool。
- 系统可作为 MCP Client 安全连接第三方标准 HTTP MCP Server，并治理其 Tool/Resource/Prompt。
- 系统可严格导入 Agent Skills 标准 Skill，并支持版本、回滚和权限审批。
- 系统提供统一 Agent 对话入口，可在同一 Conversation 中显式选择并切换 Chat-capable Native/Portable Skill。
- Model、Prompt、Skill、Runtime 版本可追踪，核心能力有固定 Eval/Replay。
- Conversation、Memory、Task、Skill Package 职责清晰。
- 新 Skill 不复制 Runtime/Tool/Task/Permission 基础逻辑。
- AI 子系统故障不会阻塞 WebSocket、行情采集、基础策略与交易主循环。
- 任何真实交易动作都不能绕过 deterministic Risk Engine。
