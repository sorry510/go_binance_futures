# 07. Phase 6-7：调度、持久化、权限与可观测

## Phase 6：统一 Scheduler 与 Task 持久化

### Scheduler 目标

把散落在 `main.go` 的 AI 周期任务逐步迁移为统一调度定义，但非 AI 高频交易循环不强制迁移。

第一批实际迁移：

```text
market_regime      default every 60m, configurable
```

`market_scan` 当前项目尚无对应 Skill，因此 Phase 6 不注册空 Job；后续实现该 Skill 后直接挂入统一 Scheduler。

Scheduler 只负责“何时触发哪个 Skill”，不包含 Prompt、LLM 或业务分析逻辑。

### 调度任务字段

建议配置至少包含：`name`、`skill`、`enable`、`interval/cron`、`timeout`、`concurrency_policy`、`input`。

`concurrency_policy` 至少支持 `skip_if_running`，用于 Market Regime 避免任务重叠。

### Task Store 持久化

Phase 1 先使用内存 Store；到 Phase 6 再增加数据库 Store，避免 Runtime 初期同时处理 ORM 迁移复杂度。
建议拆分三类数据：

1. `agent_tasks`：单次执行状态和最终结果摘要。
2. `agent_task_events`：阶段、Tool 调用、错误、耗时等审计事件。
3. `agent_conversations` / message storage：只有需要续聊的 Skill 才保存，不把所有定时任务都当会话。

大体字段：

```text
agent_tasks:
id, skill, status, input_json, result_json, error
round, max_rounds, provider, model
input_tokens, output_tokens, total_tokens
created_at, started_at, updated_at, completed_at
```

当前实现使用 `agent_tasks`、`agent_task_events`、`agent_conversations`、`agent_conversation_messages` 四张表。
Task 每次阶段更新直接写数据库；服务启动时把遗留的 queued/running/waiting 状态标记为 `interrupted`。
Task Input/Result/Error/Event 及 Conversation Message 在持久化前对 API Key、Authorization、Token、Password、Secret 等字段进行脱敏。

敏感 Tool Result 不应无条件全量持久化；对可能包含账户/订单信息的数据继续使用白名单字段或脱敏摘要。

## Conversation 与 Task 分离

- Task = 一次 Agent Run。
- Conversation = 用户连续交互，可包含多个 Task。
- Memory = 未来可选的长期业务偏好，不属于第一版必须项。

Strategy Builder 的 `conversationId` 已映射为独立 Conversation，而不是继续复用 Task ID。每次续聊创建新的 Task，并复用同一个 Conversation；消息历史写入数据库，浏览器保存最近的 Conversation ID / Task ID 以支持页面刷新后的恢复。

## Phase 6 验收

- [x] 进程重启后可查看已完成 Task 历史。
- [x] 运行中的任务重启后明确标记 `interrupted`，不伪装为 running。
- [x] Scheduler 不会重复执行同一 `skip_if_running` Job。
- [x] Scheduler interval 动态修改后会重新计算 next run。
- [x] AI → 任务中心可查看持久化 Task、Events 和 Scheduler 状态。
- [x] Strategy Builder 的 Conversation 与 Task 分离并持久化消息。
## Phase 7：权限、预算、监控与灰度

### Permission

每个 Tool 按 `read/write/trade` 分类：

- read：默认允许给明确授权的 Skill。
- write：必须配置 Skill + Tool 双重白名单；重要写操作可要求用户确认。
- trade：第一版 Runtime 全局禁止。

未来交易自动化也必须额外经过独立 Risk Engine，不允许 Prompt 绕过仓位、杠杆、止损、总风险限制。

### 调用预算

建议增加：

- 每个 Task 最大 LLM rounds。
- 全局单 Task 最大 Tool calls；不再为单个 Skill 单独配置 Tool/Token 预算。
- 每分钟/每小时 Agent 调用数。
- Alert Agent 并发上限。
- 单 Task 最大 input/output token 或估算成本阈值。
- 相同 symbol + skill 的最短冷却时间。

### Observability

统一结构化日志字段：`task_id`、`conversation_id`、`skill`、`round`、`provider`、`model`、`tool`、`duration_ms`、`status`、`error_type`。

建议指标：任务成功率、LLM 错误率、Tool 错误率、Validator repair 次数、平均轮数、P50/P95 耗时、Token 使用、Alert Signal→Notify 转化率、Fallback 次数。

### 灰度与回滚

每个迁移 Skill 都由 `agent_skills` 表独立管理启用状态；Token/Tool 调用预算统一使用全局配置。新旧路径切换应是配置行为；上线顺序固定为开发环境、手动任务、影子模式、少量请求、全量。

### 当前实施状态（已完成）

- `agent_skills` 持久化 Skill 注册、显示信息和启用状态，Web 支持创建、编辑、软删除和恢复；Runtime admission 每次启动均读取数据库状态。
- Tool Permission 在 Runtime 强制执行：read 需 Skill Tool 白名单，write 需 Skill Tool 白名单 + Permission 双授权，trade 第一版全局拒绝。
- 全局治理配置统一限制每分钟/每小时 Agent 启动数、单 Task Token 和 Tool 调用数；Alert 额外保留并发和每分钟 AI 预算。
- Runtime Collector 统计全局和 per-Skill 的 Task 成功率、LLM/Tool 错误率、Validator/Repair、Token、平均轮数及 P50/P95；Alert 统计有效 Signal→Notify 转化率与 Fallback 率。
- Task、Conversation 和结构化错误日志共用 `agent/security` 脱敏；覆盖 API Key、Authorization、Token、Password、Secret、数据库密码、DSN 和 URI 凭据。
- Alert AI 启动失败或 Task 失败自动规则 fallback；Scheduler Agent 启动失败调用 fallback hook，故障与 WS/交易主循环隔离。

## Phase 7 验收

- [x] 可快速关闭任一 AI Skill 而不影响基础行情/交易程序运行。
- [x] API Key、Authorization、数据库密码不会进入持久化 Agent Messages/Task Event/结构化日志。
- [x] LLM/Tool 服务异常不会拖死 WS、交易循环和 Web 服务，Alert/Scheduler 有明确 fallback。
- [x] 所有 write/trade Tool 的权限行为有 Runtime/Policy 测试。
- [x] 有 Task 成功率、LLM/Tool 错误率、P50/P95、Token、Validator/Repair、Alert 转化/Fallback 等量化指标。