# 07. Phase 6-7：调度、持久化、权限与可观测

## Phase 6：统一 Scheduler 与 Task 持久化

### Scheduler 目标

把散落在 `main.go` 的 AI 周期任务逐步迁移为统一调度定义，但非 AI 高频交易循环不强制迁移。

第一批只迁移：

```text
market_regime      every 60m
market_scan        configurable
```

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

敏感 Tool Result 不应无条件全量持久化；对可能包含账户/订单信息的数据使用白名单字段或脱敏摘要。

## Conversation 与 Task 分离

- Task = 一次 Agent Run。
- Conversation = 用户连续交互，可包含多个 Task。
- Memory = 未来可选的长期业务偏好，不属于第一版必须项。

Strategy Builder 的 `conversationId` 迁移时应映射为 Conversation，而不是继续复用 Task ID。

## Phase 6 验收

- 进程重启后可查看已完成 Task 历史。
- 运行中的任务重启后明确标记 interrupted/failed，不伪装为 running。
- Scheduler 不会重复执行同一 `skip_if_running` Job。
- UI 不再依赖进程内 map 才能查看历史结果。
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
- 每个 Skill 最大 Tool calls。
- 每分钟/每小时 Agent 调用数。
- Alert Agent 并发上限。
- 单 Task 最大 input/output token 或估算成本阈值。
- 相同 symbol + skill 的最短冷却时间。

### Observability

统一结构化日志字段：`task_id`、`conversation_id`、`skill`、`round`、`provider`、`model`、`tool`、`duration_ms`、`status`、`error_type`。

建议指标：任务成功率、LLM 错误率、Tool 错误率、Validator repair 次数、平均轮数、P50/P95 耗时、Token 使用、Alert Signal→Notify 转化率、Fallback 次数。

### 灰度与回滚

每个迁移 Skill 都有独立 enable flag。新旧路径切换应是配置行为；上线顺序固定为开发环境、手动任务、影子模式、少量请求、全量。

## Phase 7 验收

- 可快速关闭任一 AI Skill 而不影响基础行情/交易程序运行。
- API Key、Authorization、数据库密码不会进入 Agent Messages/Task Event/日志。
- LLM 服务异常不会拖死 WS、交易循环和 Web 服务。
- 所有写/交易 Tool 的权限行为有测试。
- 有可量化指标判断新 Agent 是否比旧实现更稳定。