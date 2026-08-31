# 02. Phase 0-1：基线与 Agent Runtime

## Phase 0：固化现状与边界

### 目标

在动代码前建立可回归的行为基线，明确哪些代码属于 Runtime、业务 Skill、Domain Service 和入口层。

### 工作项

1. 为现有 AI 路径建立调用图：
   - `UpdateMarketCondition*`
   - `strategy_template_ai_task.go`
   - `strategy_template_ai_tools.go`
   - `llm/`
   - `mcpserver/`
2. 记录现有 API 路由、请求/响应结构和前端依赖，迁移期间保持兼容。
3. 给已有市场环境和策略生成补足关键回归测试，优先覆盖成功、LLM 失败、校验失败、Tool 失败、最大轮数。
4. 明确现有内存 Task Store 的行为和 30 分钟 retention，不在 Phase 1 顺手改 DB。
5. 形成 Tool 权限分类清单：read、write、trade。

### Phase 0 验收

- 不改业务行为。
- 可列出两条现有 AI 功能从 Controller 到 LLM/DB 的完整调用链。
- 有测试能证明迁移前输出契约。
- 明确旧 API 在 Phase 3 前不会删除。

## Phase 1：建立通用 Runtime

### 目标

从策略生成 Agent 中抽取“通用循环”，但此阶段不强制策略生成切换到新 Runtime。
### 新增公共对象

建议第一批只建立：

```text
agent/runtime      Runner、RunLoop、Decision Parser
agent/skill        Skill interface + Registry
agent/tools        Tool interface + Registry
agent/task         Task model + in-memory Store interface
agent/validator    FinalValidator interface
agent/permission   Tool risk level / policy
```

### Runner 最小执行流程

1. 接收 `AgentRequest`，生成 Task ID。
2. 从 Skill Registry 取得 Skill。
3. 调用 Skill 构造初始 messages。
4. 调用现有 `llm.Client.Generate`。
5. 解析统一 Decision JSON。
6. `action=tool`：检查 Tool 是否属于 Skill 白名单及权限，再执行。
7. 将结构化 Tool Result 追加到 messages，进入下一轮。
8. `action=final`：调用 Skill Validator。
9. Validator 失败时生成标准 Repair Feedback；未超轮数则继续。
10. 成功或不可恢复失败时结束 Task。

### 必须支持的运行控制

- `context.Context` 取消。
- 每次 Run 总超时。
- LLM 单次调用超时继续由 `llm` 配置控制。
- `MaxRounds`。
- 可配置 Retry Policy，仅重试网络/429/5xx 等可恢复错误。
- Max Context Bytes / Tool Result Bytes，防止上下文无限增长。
- Tool 调用次数限制，防止模型循环调用同一工具。
- Task Progress/Event hook，供 Controller/UI 订阅或轮询。
### Task 结构建议

```text
id, skill, status, stage, progress
input, result, error
round, max_rounds
provider, model, usage
created_at, started_at, updated_at, completed_at
```

状态统一为：`queued/running/waiting_llm/waiting_tool/validating/succeeded/failed/cancelled`。

### Phase 1 测试

- Fake LLM 返回 `final`，Runner 能正常结束。
- Fake LLM 先 `tool` 后 `final`，工具只被调用一次。
- 未注册 Tool、Skill 未授权 Tool、Tool 参数错误均被拒绝。
- Final Validator 失败后能 Repair 并在下一轮成功。
- 达到 MaxRounds 后失败，不形成无限循环。
- Context cancel 后 Runner 退出，Task 标记 cancelled/failed（最终统一一种语义）。
- 并发多个 Task 不相互污染 messages、progress、tool state。

### Phase 1 验收

- Runtime 测试完全不依赖具体 Binance 业务。
- `llm/` 无需知道 Skill/Tool。
- Controller 无需参与 Agent Loop。
- 旧策略生成和市场环境仍可继续沿旧路径工作。
- Runtime API 稳定后才进入 Phase 2。
## 实施结果（2026-08-28）

Phase 0-1 已按本文第一版范围完成：

- 新增 `agent/runtime`：统一 Run Loop、Decision Parser、Retry、Repair、Context/Tool Result 限制、取消。
- 新增 `agent/skill`：Skill interface/Definition/Registry。
- 新增 `agent/tools`：Tool interface/Func/Registry。
- 新增 `agent/task`：统一状态、事件、Usage、内存 Store。
- 新增 `agent/validator`：最终结果 Validator。
- 新增 `agent/permission`：read/write/trade 风险等级与静态 Policy；默认只允许 read。
- 新 Runtime 尚未接管旧 Controller，符合 Phase 1 的兼容要求。
- Runtime 单测覆盖 final、tool、权限、Tool 错误、Repair、MaxRounds、取消、并发隔离和 429 Retry。
- 旧市场环境和策略生成增加契约回归测试，详见 `00-phase-0-baseline.md`。
