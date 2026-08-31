# AI Agent V2 完善计划

## 1. 背景

V1 已完成统一 Agent Runtime、Skill、Tool Registry、Task/Conversation、Scheduler、Permission、Observability，以及 `market_regime`、`strategy_builder`、`symbol_analysis`、`alert_analysis` 四类核心能力。

V2 不重新实现一套 Agent，也不以“增加更多 Prompt”作为主要目标。V2 的重点是把当前可运行的 Agent 基础设施升级为可长期演进、可量化评估、可恢复、可路由、可治理的智能决策平台。

V1 文档已归档到 `../v1/`，V2 在兼容 V1 API 和现有业务链路的基础上渐进演进。

## 2. V2 总目标

V2 需要解决六个核心问题：

1. **推理过程可控**：从单一 ReAct 循环升级为具备计划、执行、校验和恢复能力的 Runtime。
2. **上下文质量可控**：统一管理行情证据、历史信息、Tool Result、Conversation 和长期 Memory，避免无限堆 Prompt。
3. **工具调用可靠**：Tool Schema、超时、缓存、并行、幂等、错误分类和结果裁剪统一由 Tool Runtime 管理。
4. **模型使用可治理**：根据任务能力、成本和健康度选择模型，支持 fallback，但不让业务 Skill 感知 Provider 差异。
5. **Agent 质量可评估**：建立 Eval、Replay、Prompt Version 和回归基线，让“效果变好/变差”可以量化。
6. **逐步具备执行能力**：先实现 Proposal 和 Risk Review，最后才考虑受控 write/trade Tool，禁止 Prompt 直接绕过风控。
## 3. V1 现状与 V2 需要补齐的缺口

根据当前代码，V1 已经具备稳定骨架，但仍有以下结构性缺口：

- `runtime.DefaultRunner.Run()` 同时承担生命周期、LLM 循环、Tool 执行、Repair、Validator 和进度推进，后续继续加能力会越来越难维护。
- Skill 主要通过静态 `SystemPrompt()`、`Tools()`、`BuildInput()` 表达行为，缺少 Execution Mode、Context Requirement、Output Contract 等显式元数据。
- Context 只做字节上限控制，没有统一 token 预算、摘要、裁剪优先级、证据新鲜度和历史选择策略。
- Tool 目前顺序执行；Metadata 已有 Schema/Timeout/Result Size，但没有统一 Schema Validator、cache、batch、parallel 和标准 Result Envelope。
- 当前仅选择一个启用的 LLM 配置，没有模型能力表、健康度、任务级路由和自动 fallback。
- Conversation 已持久化，但没有长期 Memory；历史分析主要由业务 Service 自行读取，没有统一记忆策略。
- Observability 已统计成功率、错误率、Token、P50/P95，但主要是进程内聚合，缺少长期趋势、按模型/Prompt Version 对比和质量指标。
- 当前 Validator 能保证结构和部分业务一致性，但还没有离线 Eval 数据集、Replay 和自动回归测试体系。
- write/trade 权限目前以禁止为主，尚未形成 Proposal -> Risk Review -> Approval -> Execution 的完整安全链。

因此 V2 的关键不是“让 LLM 调更多工具”，而是将推理、上下文、工具、模型、质量和风险六个层面分别产品化。
## 4. V2 目标架构

```text
Web / API / Scheduler / Event Bus
              |
        Agent Manager
              |
       Run Coordinator
        /     |      \
 Planner  Context   Policy
    |      Engine    Engine
    |         |         |
 Execution Engine ------+
    |          |
Model Gateway  Tool Runtime
    |          |
LLM Providers  Domain Services
               |
        Binance / DB / Scanner

横向能力：Task Store / Conversation / Memory / Eval / Observability / Security
```

V2 仍保持“一个统一 Runtime，多种 Skill”的原则，不默认引入多个互相聊天的 Agent。复杂任务优先使用单 Agent + 显式 Execution Plan，只有确认单 Agent 无法表达时才评估 Multi-Agent。

Runtime 不直接理解交易策略；交易领域知识继续存在 Skill、Validator、Domain Service 和 Risk Engine 中。
## 5. 核心设计原则

### 5.1 V1 兼容优先

现有 `runtime.Request` / `runtime.Result`、Task API、Conversation、四个 Skill 和前端任务中心不一次性推翻。V2 通过新接口和 Adapter 渐进替换内部实现。

### 5.2 确定性计算优先于 LLM

行情计算、技术指标、阈值、收益、手续费、Signal、风控规则继续使用 Go 实现。LLM 负责信息选择、假设生成、冲突解释、计划和结构化决策。

### 5.3 Evidence First

所有交易分析结论必须可以追溯到 Tool Result 或明确标记为推断。Evidence 需要携带来源、时间、新鲜度和关键数据摘要，不能只保存自然语言 finding。

### 5.4 全局预算继续统一管理

V2 不恢复每个 Skill 独立的最大 Token / 最大 Tool Call 配置。最大 Token、Tool Calls、并发、启动频率继续使用全局治理；Skill 只声明允许工具、能力需求和执行模式。

### 5.5 风控与执行严格分层

任何未来 write/trade 能力都必须经过独立 Risk Engine。LLM 只能产生 Proposal，不能自己批准自己的交易动作。
## 6. Phase V2-0：冻结 V1 基线与版本契约

### 目标

在开始重构 Runtime 前，把 V1 当前行为变成可重复验证的兼容基线。

### 工作项

- 固化四个现有 Skill 的输入、输出、Task/Event 状态和主要 API contract。
- 给 `TradingPlanV1`、Market Regime、Alert Result、Strategy Builder Result 建立版本化 fixture。
- 保存典型成功、Tool 失败、LLM 格式错误、Validator Repair、Timeout、Context Too Large 等运行样例。
- 为 Prompt 增加明确版本号，不再只靠 Go 常量内容判断版本。
- 定义 V2 Task Metadata：`runtime_version`、`skill_version`、`prompt_version`、`model_config_id`。
- 建立 V1 Replay 测试入口，后续 V2 必须可以重放同一输入和同一 Tool Fixture。

### 验收

- [ ] V1 四个 Skill 都有固定 Eval/Replay Case。
- [ ] 修改 Prompt、Validator 或 Runtime 后可以明确知道哪些 Case 发生变化。
- [ ] Task 可以追溯到 Runtime/Skill/Prompt/Model 版本。
- [ ] V2 开发不依赖生产实时行情才能完成核心回归测试。
## 7. Phase V2-1：Runtime V2 与执行状态机

### 目标

把当前 `DefaultRunner.Run()` 的大循环拆成可组合组件，并支持中断恢复、显式计划和不同执行模式。

### 建议组件

```text
runtime/coordinator     # Run 生命周期、状态推进、checkpoint
runtime/planner         # 可选计划生成与计划修订
runtime/executor        # 执行 LLM step / Tool step / Validate step
runtime/context         # 调用 Context Engine
runtime/recovery        # retry、resume、checkpoint restore
runtime/protocol        # Decision / Plan / Final 协议
```

### Execution Mode

至少支持：`react`、`workflow`、`plan_execute`。现有四个 Skill 默认先走 `react`，复杂分析逐步迁移到 `plan_execute`，确定性链路使用 `workflow`。

### 关键要求

- Plan 是可选能力，不要求每个简单任务先调用一次 LLM 生成计划。
- 每个 Step 有唯一 ID、类型、状态、输入摘要、输出摘要和错误。
- Tool 成功后可以 checkpoint；服务重启后允许从安全 checkpoint 恢复，而不是全部标记 interrupted。
- Planner 不能绕过 Skill Tool 白名单和 Permission。
- `max_rounds`、全局 Token/Tool 预算仍由 Runtime 强制执行。

### 验收

- [ ] V1 ReAct 行为通过兼容 Adapter 保持可用。
- [ ] Runtime 核心不再由一个函数同时承担所有职责。
- [ ] 可恢复任务能够从最近安全 checkpoint 继续。
- [ ] Plan/Step 全过程能在 Task Event 中查看。
## 8. Phase V2-2：Context Engine 与 Evidence Model

### 目标

把“给模型什么数据”从每个 Skill 自己拼 Prompt，升级为统一、可预算、可裁剪、可审计的 Context Engine。

### Context 分层

- `system`：协议、安全边界、输出契约。
- `task`：本次用户目标与 Skill 输入。
- `market`：实时行情和确定性指标。
- `history`：最近 Task、历史分析和策略结果。
- `memory`：经过筛选的长期记忆。
- `tool`：本轮 Tool Result。

每块 Context 都应携带 `source`、`as_of`、`priority`、`estimated_tokens`、`sensitive`、`freshness` 等元数据。

### 核心能力

- token-aware budgeting，而不只是 `MaxContextBytes`。
- 按优先级裁剪：协议和当前行情不能被历史对话挤掉。
- 大 Tool Result 先结构化压缩，再决定是否进入模型上下文。
- 同一数据只保留一个 authoritative source，避免 snapshot/context/独立 Tool 重复注入。
- 对实时市场数据执行 freshness 校验；过期数据显式进入 `data_missing/stale_data`。
- Evidence 使用结构化 ID，Final Result 可以引用 Evidence ID，而不是只写 Tool 名称。
### Context Engine 验收

- [ ] 同一 Skill 在不同模型上下文窗口下可以稳定构建输入。
- [ ] Context 超预算时有可解释的裁剪记录，而不是直接失败。
- [ ] Tool 原始大结果不会无条件进入 Conversation。
- [ ] 每条关键交易结论能追溯到 Evidence ID 和数据时间。
- [ ] 历史信息不能覆盖更新鲜的当前行情事实。

## 9. Phase V2-3：Tool Runtime 2.0

### 目标

把 Tool 从“Registry + Execute”升级为独立执行层，提高数据获取效率和故障隔离能力。

### 工作项

- Runtime 在调用 Tool 前统一校验 `InputSchema`，不只依赖各 Tool 手写 `strictDecode`。
- 标准化 `ToolResultEnvelope`：`data`、`source`、`as_of`、`duration_ms`、`cache_hit`、`partial`、`warnings`、`error_type`。
- 增加 Tool error taxonomy：invalid_input、not_found、rate_limit、timeout、upstream、stale、partial、internal。
- 对只读、幂等 Tool 支持短 TTL cache；实时价格类 TTL 必须很短或禁止缓存。
- 支持同一 Step 中无依赖的只读 Tool 并行执行，减少单币分析串行等待时间。
- 支持 batch Tool，避免扫描 20 个 symbol 时由 LLM 发 20 次重复调用。
- Tool Result 统一做最大体积限制、摘要和敏感字段过滤。
- 记录 Tool source quality / freshness，为后续 Eval 提供依据。
### Tool Runtime 验收

- [ ] Tool 输入 Schema 在执行前统一校验。
- [ ] Tool 错误可以被 Runtime 按类型处理，而不是全部作为字符串返回给 LLM。
- [ ] 可并行的只读 Tool 能并行执行且不改变结果一致性。
- [ ] Cache 命中、数据时间和来源对 Agent 可见且可审计。
- [ ] 单个 Tool 故障不会破坏整个市场数据 Context；允许明确的 partial result。

## 10. Phase V2-4：Model Gateway、Capability 与 Router

### 目标

把“当前启用一个模型”升级为“模型配置 + 能力描述 + 健康度 + 任务路由”，同时保持 Skill 不依赖具体厂商。

### Model Capability

每个模型配置增加运行时能力描述，例如：`structured_output`、`native_tool_calling`、`reasoning`、`long_context`、`json_reliability`、`max_context_tokens`、`cost_class`。

这些字段优先由系统内置 Provider/Model Profile 给出，允许管理员覆盖；不要要求用户为每个模型手工填写大量参数。

### Router

- Skill 声明“需要的能力”，不声明固定模型名称。
- 默认优先使用当前主模型。
- 主模型不满足能力、连续失败或健康检查异常时，允许切换 fallback。
- Alert 等时效敏感任务可以选择低延迟模型；复杂策略生成可以选择推理能力更强的模型。
- 路由必须记录选择原因，Task 中保存实际 model config ID。
### Model Gateway 验收

- [ ] OpenAI Compatible / Anthropic / Ollama 等 Provider 继续使用统一上层接口。
- [ ] Task 可查看路由前候选模型、最终模型和 fallback 原因。
- [ ] 某个 Provider 临时故障时，允许符合策略的任务自动 fallback。
- [ ] fallback 不得突破全局 Token、并发和权限限制。
- [ ] Router 可以关闭，回退为 V1 的单一启用模型行为。

## 11. Phase V2-5：Memory 与历史知识管理

### 目标

Conversation 解决“本次连续对话”，Memory 解决“跨任务仍值得保留的信息”。两者必须分开。

### Memory 类型

- `user_preference`：用户明确要求长期遵循的分析偏好。
- `strategy_fact`：某策略的稳定配置、适用市场和已确认约束。
- `market_hypothesis`：有有效期的市场假设，不允许永久保留。
- `task_summary`：重要历史任务的压缩摘要。
- `lesson`：经过 Eval 或人工确认的失败原因/修正原则。

Memory 必须有 `scope`、`source_task_id`、`created_at`、`expires_at`、`confidence`、`status`。市场类记忆默认带 TTL，避免旧行情观点污染当前判断。

### 原则

Memory 不自动把全部 Conversation 写进去。写入需经过规则筛选；高风险交易结论不能因为模型说“记住”就永久生效。
### Memory 验收

- [ ] Memory 与 Conversation 使用独立 Store 和生命周期。
- [ ] 可按 Skill / Symbol / Strategy / User Scope 查询。
- [ ] 市场观点过期后不会继续进入当前分析 Context。
- [ ] 用户可查看、禁用和删除 Memory。
- [ ] Memory 写入和读取过程进入审计事件。

## 12. Phase V2-6：Eval、Replay 与 Prompt Version

### 目标

建立 Agent 的自动化质量控制体系，使模型、Prompt、Runtime 和 Tool 改动都可以量化回归。

### Eval Case

每个 Case 至少保存：输入、固定 Tool Fixture、预期结构约束、关键事实、禁止事实、允许的方向范围、评分规则和标签。

首批数据集建议覆盖：

- `symbol_analysis`：趋势明确、震荡、数据缺失、极端 funding、OI 异常、强平异常、多周期冲突。
- `market_regime`：11 种 MarketCondition 的代表性快照与边界样本。
- `strategy_builder`：需求完整、需求冲突、缺少退出逻辑、复杂 AND/OR、非法 expr 修复。
- `alert_analysis`：应通知、应忽略、AI 故障 fallback、重复信号、低质量数据。

### 评分维度

结构正确率、事实一致性、Evidence 覆盖率、Data Missing 诚实度、Validator Repair 次数、Tool Calls、Token、耗时、方向稳定性和业务规则命中率。
### Eval 执行方式

- Offline Eval：固定 Tool Fixture，不访问实时 Binance，用于 CI 和 Prompt/Runtime 回归。
- Shadow Eval：生产输入复制给候选模型/Prompt，但候选结果不影响真实通知或交易。
- Replay：从历史 Task 重建输入和 Tool Result，对新版本重新执行。
- Human Review：只用于无法可靠自动评分的主观质量维度，不作为唯一 Gate。

### 验收

- [ ] 每个核心 Skill 至少有一组可自动执行的 Eval Cases。
- [ ] Prompt 改动必须产生可对比的 Eval Report。
- [ ] 模型切换前可以比较成功率、Repair、Token、耗时和业务评分。
- [ ] 关键 Case 退化时 CI 能阻止上线。
- [ ] Task 页面可以追溯使用的 Prompt Version。

## 13. Phase V2-7：业务 Workflow 与新 Skill

基础能力稳定后，再扩展真正有价值的 Agent 场景。优先增加“需要多步推理，但仍以只读分析为主”的能力。

### market_scan

确定性 Scanner 先筛候选，Agent 只分析少量候选并排序，输出 `MarketOpportunitySetV1`。禁止让 LLM 扫描全市场原始 Tick。

### strategy_review

读取 Strategy Template + 测试结果 + MarketCondition，分析策略在哪些市场环境有效、在哪些环境失败，并输出可执行的修改 Proposal；不直接修改模板。
### strategy_experiment

将“AI 提议策略修改”和“确定性测试”组合为 Workflow：Agent 生成候选参数/逻辑 -> Validator 检查 -> 测试引擎运行 -> Agent 对结果归纳。V2 初期只生成实验结果，不自动覆盖正式策略。

### alert_triage

把多个相关 Signal 在时间窗口内聚合为 Incident，再由 Agent 判断是否属于同一市场事件，减少 FastMove、Liquidation、OI 等多路重复报警。

### daily_market_brief

复用 MarketCondition、Scanner、重要 Signal 和历史变化，生成固定 Schema 的市场摘要。它应由 Scheduler 触发，并明确区分事实、推断和缺失数据。

### Workflow 验收

- [ ] 新 Skill 不复制 Runtime Loop、Task Store 或 Tool 调用框架。
- [ ] Scanner/Backtest 等大规模计算始终在确定性 Service 中完成。
- [ ] Agent 负责候选选择、解释和 Proposal，不替代测试引擎。
- [ ] 每个新 Skill 上线前必须补 Eval Cases。
- [ ] 所有新输出都使用版本化 Schema。

## 14. Phase V2-8：Proposal、Risk Engine 与受控执行

### 目标

为未来 AI 辅助交易建立安全执行边界，但本阶段默认仍不允许 Agent 自主真实下单。

推荐链路：

```text
Agent -> TradeProposal -> Deterministic Risk Engine -> Approval -> Execution Service -> Audit
```
### TradeProposal 至少包含

- symbol、side、entry 条件、stop loss、take profit。
- proposal 来源 Task / Evidence。
- 生成时间、有效期、失效条件。
- 最大允许风险，而不是由 LLM 直接指定最终下单数量。

### Risk Engine 必须独立校验

- symbol 是否允许交易。
- MarketCondition 是否满足策略限制。
- 当前仓位、最大持仓数、总风险暴露。
- leverage 上限、单笔最大亏损、止损有效性。
- 价格是否过期、滑点是否异常、Proposal 是否过期。
- 是否命中 cooldown、重复订单和 kill switch。

### Approval

第一阶段只支持人工确认。后续若允许自动批准，也必须是明确的 deterministic policy，而不是再次询问 LLM“是否批准”。

### 验收

- [ ] Agent 无法直接调用真实交易 API。
- [ ] Proposal 即使被 Prompt Injection 操纵，也不能跳过 Risk Engine。
- [ ] Risk Reject 有结构化原因并写入审计记录。
- [ ] Execution 使用现有交易 Service，不在 Agent Tool 内重复实现交易逻辑。
- [ ] 全局 kill switch 可以立即关闭所有 AI write/trade 行为。
## 15. Phase V2-9：Observability、Trace 与运营页面

### 目标

从“看任务有没有成功”升级到“可以定位质量、成本和数据问题”。

### Trace

一个 Task Trace 至少串联：Request -> Context Build -> Plan -> LLM Call -> Tool Calls -> Evidence -> Validation -> Final -> Completion Hook。

每个节点统一记录：`task_id`、`step_id`、`skill`、`model_config_id`、`prompt_version`、`duration_ms`、`tokens`、`status`、`error_type`。

### 长期指标

除现有成功率和 P50/P95 外，增加：

- 按 Model / Prompt Version / Skill 的成功率和平均成本。
- Context tokens、裁剪率、Memory 命中率。
- Tool cache hit、partial result、stale data、timeout 比例。
- Evidence 覆盖率和 Validator Repair 分布。
- Eval Score 趋势和版本退化记录。
- Risk Proposal accept/reject 分布。

### Web

任务中心增加 Step Timeline、Context Summary、Tool/Evidence、模型路由、版本和 Eval 信息；默认不展示敏感 Tool 原始数据。
## 16. 推荐的数据模型演进

尽量复用现有 `agent_tasks`、`agent_task_events`、`agent_conversations`，不要为 V2 平行复制一套 Task 系统。

建议新增或扩展：

- `agent_tasks`：增加 runtime/skill/prompt/model config version、execution_mode、checkpoint 信息。
- `agent_task_events`：增加 step_id、event_type、structured metadata，继续作为主要 Trace Store。
- `agent_memories`：长期 Memory，带 scope、TTL、confidence、source task。
- `agent_prompt_versions`：Skill Prompt 版本、hash、状态和发布时间。
- `agent_eval_cases`：离线 Eval Case 元数据和 fixture 引用。
- `agent_eval_runs`：某 Runtime/Prompt/Model 组合的评测结果。

Tool cache 默认优先使用进程内 TTL cache，不急于落数据库；只有跨进程共享确实必要时再增加持久化 Cache。

所有新增表继续使用 ORM model + `orm.RunSyncdb` 自动同步数据结构；初始化数据通过现有初始化机制处理，不为普通加字段重复维护手写 migration SQL。

## 17. 推荐代码目录

```text
agent/
├── runtime/          # 保留公共类型，逐步拆 coordinator/executor/protocol
├── context/          # V2 Context Engine
├── evidence/         # Evidence model / registry / validation
├── modelgateway/     # capability、router、health、fallback
├── toolruntime/      # schema、parallel、cache、result envelope
├── memory/           # long-term memory
├── eval/             # cases、fixtures、replay、scoring
├── risk/             # proposal review / deterministic risk policy
└── skills/           # 业务 Skill，继续复用现有目录
```
## 18. V2 明确暂不做

以下能力不是 V2 前半程目标，除非前置基础已经通过 Eval 证明稳定：

- 不默认引入 Multi-Agent 群聊、角色辩论或 Agent 自我创建。
- 不因为“长期记忆”就立即引入向量数据库；第一版 Memory 先使用结构化关系数据库和明确 scope。
- 不让 LLM 直接消费全市场 Tick、完整 OrderBook 或无限 Kline。
- 不把所有业务流程改成 LLM 驱动；确定性代码能解决的问题继续用 Go。
- 不允许 Agent 自己修改 Prompt、Skill 权限、Risk Policy 或全局预算。
- 不允许模型输出直接成为真实订单。
- 不为了使用某一家模型的原生 Tool Calling 而破坏统一 Runtime 协议。

## 19. 推荐实施顺序

| 阶段 | 优先级 | 主要产物 |
| --- | --- | --- |
| V2-0 Baseline | P0 | Fixture、Replay、版本元数据 |
| V2-1 Runtime | P0 | Coordinator、Executor、Step、Checkpoint |
| V2-2 Context | P0 | Context Engine、Evidence、Token Budget |
| V2-3 Tool Runtime | P0 | Schema、Envelope、Parallel、Cache |
| V2-4 Model Gateway | P1 | Capability、Router、Health、Fallback |
| V2-5 Memory | P1 | Memory Store、TTL、Scope、管理接口 |
| V2-6 Eval | P0 | Eval Case、Replay、Scoring、CI Gate |
| V2-7 Workflows | P1 | market_scan、strategy_review 等 |
| V2-8 Risk | P2 | TradeProposal、Risk Review、Approval |
| V2-9 Operations | P1 | Trace、长期指标、任务中心增强 |
实际开发顺序建议不是机械按编号，而是：

```text
V2-0 -> V2-1 -> V2-2 -> V2-3 -> V2-6
                              |
                              +-> V2-4 -> V2-5 -> V2-9 -> V2-7 -> V2-8
```

其中 Eval 必须在大量新业务 Skill 之前落地。Model Router 和 Memory 都必须接受 Eval 对比，不能上线后再补质量基线。

## 20. 第一轮开发建议

第一轮只做 P0 基础，不新增高风险 Skill：

1. 给现有 Task 增加 Runtime/Skill/Prompt/Model Version 元数据。
2. 建立 Prompt Version 机制和四个 V1 Skill 的初始版本。
3. 抽取 `ExecutionStep` / `RunState`，先让现有 ReAct 循环跑在新的 Coordinator 上。
4. 建立固定 Tool Fixture 和 Replay Runner。
5. 为 `symbol_analysis` 建第一批 10~20 个 Eval Case，作为 V2 的质量样板。
6. 实现 Context Block / Evidence 基础类型，但暂不一次性迁移所有 Skill。

这一轮完成后再决定 Runtime V2 的 Planner、并行 Tool 和 Model Router 的具体接口，避免过早设计。

## 21. 每个阶段统一 Gate

- **Compatibility Gate**：V1 API、已有页面和业务行为未被无意破坏。
- **Test Gate**：新增核心模块有单元测试、集成测试和失败路径测试。
- **Eval Gate**：涉及 Prompt/Model/Context 的变更必须有 Eval 对比。
- **Observability Gate**：新增 Step/Router/Memory/Tool 行为可在 Trace 中解释。
- **Security Gate**：敏感数据、write/trade 权限、Prompt Injection 边界有明确测试。
- **Rollback Gate**：新能力可按配置关闭并回退到 V1 兼容路径。
## 22. V2 Definition of Done

V2 完成的标准不是“Agent 会调用更多工具”，而是满足以下条件：

- Runtime 可以使用明确 Step/Plan 执行复杂任务，并支持安全 checkpoint/recovery。
- Context 有统一 token budget、freshness、裁剪和 Evidence 机制。
- Tool Runtime 具备统一 Schema、错误类型、并行、缓存和 partial result 能力。
- 模型可以按 Capability 和健康度路由，且 fallback 全程可追踪。
- Conversation、Memory、Task 三者职责清晰，旧市场观点不会无限污染新任务。
- 所有核心 Skill 都有固定 Eval/Replay 数据集；Prompt/Model 更新可以量化比较。
- 新增 Skill 不复制 Runtime、Context、Tool、Task、Permission 等基础逻辑。
- Agent 产生的交易动作只能是 Proposal；真实执行必须通过独立 Risk Engine 和 Approval。
- 任何 AI 子系统故障都不能阻塞 WebSocket、行情采集、基础策略和现有交易主循环。
- V2 任一关键能力都可以独立关闭，并回退到经过验证的 V1 兼容路径。

达到这些条件后，系统才适合继续讨论更高级的自动策略研究、受控自动交易或 Multi-Agent，而不是在基础设施尚不可评测时直接扩大 Agent 自主权。
