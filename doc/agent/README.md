# AI Agent 落地计划

## 目标

在现有 `go_binance_futures` 后端中建立一套统一、可扩展、可观测的 AI Agent 架构，用同一套 Runtime 管理以下能力：

1. 定时判断当前合约市场大环境。
2. 根据用户自然语言生成自定义交易策略。
3. 根据最新数据分析指定合约并生成结构化交易计划。
4. 根据 WebSocket/行情事件触发检测、AI 判断与报警通知。

本计划采用渐进式重构。现有功能保持可用，每一阶段都必须能够独立验收，不要求一次性替换全部旧逻辑。

## 核心原则

- 不重复实现多套 Agent：建立一个 `Agent Runtime`，业务能力以 Skill/Workflow 形式接入。
- LLM 只负责推理、归纳、解释和计划，不负责高频数据流计算。
- 行情、指标、阈值、Signal 检测优先使用确定性 Go 代码。
- Agent 只能通过注册后的 Tool 获取数据或执行动作，不直接依赖 Controller。
- 内部 Agent Tool 直接调用 Go Service；MCP 与 Agent 共享同一业务 Service，不让内部 Agent 通过 HTTP 调自己。
- LLM 输出一律视为不可信输入，必须做结构化校验与业务校验。
- 交易类操作默认禁止 Agent 直接执行，先输出 Proposal，再由独立风控/执行层处理。
- 新架构优先复用现有 `llm/`、策略生成 Agent 雏形、市场环境分析、MCP、Scanner、Notify 和 Binance API 封装。

## 当前可复用基础

- `llm/`：已具备多 Provider 统一 Client。
- `controllers/strategy_template_ai_task.go`：已有多轮 Agent、Tool、Retry、Repair、Task Progress 雏形。
- `controllers/strategy_template_ai_tools.go`：已有局部 Tool 执行机制。
- `feature/market_condition.go`：已有算法兜底 + LLM 市场环境分类。
- `mcpserver/`：已有外部 MCP Tool 暴露能力。
- `feature/api/binance/`：已有 REST/WS 行情和交易封装。
- `scanner/`：已有确定性候选筛选。
- `notify/`：已有统一通知接口。
## 当前实施状态

- Phase 0：已完成基线记录与旧 AI 输出契约测试。
- Phase 1：已完成第一版通用 Runtime。
- Phase 2：已完成共享 Domain Service、8 个只读 Agent Tool，以及部分 MCP 进程内 Service 复用。
- Phase 3A：Market Regime 已接入统一 Runtime，旧 HTTP/Task 契约和算法 fallback 保持兼容。
- 下一步：Phase 3B，迁移 Strategy Builder。

## 分阶段路线

| 阶段 | 目标 | 主要结果 |
| --- | --- | --- |
| Phase 0 ✅ | 固化边界与基线 | 架构约束、接口草案、现有行为基线 |
| Phase 1 ✅ | 建立 Agent Runtime | Runner、Decision、Skill、Tool Registry、Validator、Task 抽象 |
| Phase 2 ✅ | 建立共享业务 Tool/Service 层 | Binance/DB/Scanner/Notify 能力以 Tool 复用，MCP 与 Agent 解耦 Controller |
| Phase 3 ◐ | 迁移已有 AI 能力 | Market Regime ✅；Strategy Builder 待迁移 |
| Phase 4 | 实现单币分析 | `symbol_analysis` Skill、市场数据 Context、结构化 TradingPlan |
| Phase 5 | 实现事件驱动报警 | Event Bus、Signal Engine、Alert Skill，WS 不直接调用 LLM |
| Phase 6 | 统一调度与持久化 | Scheduler、Task Store、Conversation、运行历史、配置管理 |
| Phase 7 | 风控、可观测与灰度 | Permission、预算、指标、审计、回归测试、逐步切流 |

## 推荐实施顺序

严格按照 Phase 0 → 7 推进。Phase 1/2 是公共基础，未稳定前不要直接开发复杂的 Symbol Agent 或 AI 自动交易。

每个阶段完成后都应满足三个 Gate：

1. **Compatibility Gate**：旧功能未被破坏，可继续运行。
2. **Test Gate**：新增核心接口有单元测试/集成测试，关键异常路径已覆盖。
3. **Observability Gate**：能看到 Task 状态、Tool 调用、失败原因和耗时，且日志不泄露密钥。

## 文档索引

- [00-phase-0-baseline.md](./00-phase-0-baseline.md)：重构前 AI 调用链、旧 API、Task Store、权限与回归基线。
- [01-architecture.md](./01-architecture.md)：目标架构、核心对象、目录边界。
- [02-phase-0-1-runtime.md](./02-phase-0-1-runtime.md)：基线与 Runtime 第一阶段落地。
- [03-phase-2-tools-services.md](./03-phase-2-tools-services.md)：Tool Registry 与共享 Service 层。
- [04-phase-3-migration.md](./04-phase-3-migration.md)：迁移市场环境与策略生成。
- [05-phase-4-symbol-analysis.md](./05-phase-4-symbol-analysis.md)：单币/合约 AI 分析。
- [06-phase-5-event-alert.md](./06-phase-5-event-alert.md)：WS、Event Bus、Signal、AI 报警。
- [07-phase-6-7-operations.md](./07-phase-6-7-operations.md)：Scheduler、Task 持久化、权限、观测与灰度。
- [08-acceptance-checklist.md](./08-acceptance-checklist.md)：整体验收清单与 Definition of Done。