# Phase V2-1：Runtime V2 与执行状态机

> 状态：✅ 已完成

## 目标

把 V1 集中在 `DefaultRunner.Run()` 的复合职责拆成显式 Coordinator、Executor、RunState、ExecutionStep 与 Checkpoint，同时保持 V1 四个 Skill 的 ReAct 行为兼容。V2-1 只建立 Runtime 执行骨架，不提前实现 V2-2 Context Engine 或 V2-3 Tool Runtime。

## 已落地架构

```text
DefaultRunner
    ↓
Coordinator
    ├── Build Context Step
    ├── Execution Mode
    │     ├── react → React Executor
    │     ├── plan_execute → Planner → Tool Executor → React Final
    │     └── workflow → 协议预留，V2-10 实现业务编排
    ├── RunState / ExecutionStep
    └── Checkpoint / Resume
```

主要代码：

```text
agent/runtime/coordinator.go
agent/runtime/react_executor.go
agent/runtime/plan_executor.go
agent/runtime/state.go
agent/runtime/checkpoint.go
agent/runtime/runner.go
```

`DefaultRunner.Run()` 现在只进入 Coordinator，不再承载完整 ReAct 状态机。

## Runtime Version

V2-1 Runtime 版本：

```text
runtime_version = 2.0.0
runtime_state_version = runtime_state_v1
```

旧 Task 可以继续读取，但 V1 Runtime checkpoint 不会被 V2 Runtime 自动 Resume。Resume 必须通过版本和 Skill/Contract 身份校验。

## Execution Mode

当前定义：

- `react`：默认模式，兼容 V1 四个 Skill。
- `plan_execute`：可选 Planner 先生成受约束 Tool Plan，再进入统一 Tool 执行路径和 ReAct Final。
- `workflow`：模式和 Step 类型已预留，但 V2-1 不实现业务 Workflow，留给 V2-10。

Skill 通过可选 `ExecutionModeProvider` 声明模式，不修改原有 `Skill` 核心接口。

## RunState / ExecutionStep

Task 现在可以追踪：

```text
execution_mode
plan
steps
resume_count
```

每个 `ExecutionStep` 包含：

```text
step_id
type
status
attempt
depends_on
input_summary
output_summary
started_at
completed_at
error_type
error
checkpoint
```

已定义 Step Type：

```text
plan
build_context
llm
tool
parallel_tools
validate
approval
finalize
```

其中 `parallel_tools` / `approval` 仅建立 Runtime 协议，本 Phase 不提前实现对应业务能力。

## Task Event

`agent_task_events` 增加并持久化：

```text
step_id
step_type
error_type
checkpoint
```

原有 `stage / progress / round / tool / status / duration_ms` 保持兼容，因此任务中心和现有 EventHook 不需要平行事件体系。

## Checkpoint

Checkpoint 继续存入现有 `agent_tasks`，不创建 V2 Task 表。

内部字段：

```text
checkpoint_json
```

该字段不通过普通 Task JSON API 暴露。Checkpoint 保存冻结 Prompt/Version、消息历史、Round、Tool Budget、成功 Tool、可恢复 Tool Result、Plan、Steps 和必要 Runtime metadata。

当前只持久化 Runtime 自己拥有且恢复后确实需要的 metadata，例如：

```text
scheduler_job
```

任意复杂 Skill metadata 不会自动写入 checkpoint。

## Resume 安全规则

自动恢复只接受：

```text
cancelled
interrupted
timeout
```

且必须存在安全 checkpoint。

安全 Tool checkpoint 必须同时满足：

```text
Idempotent = true
Risk = read
```

write / trade / 非幂等 Tool 在执行前会清除可恢复 checkpoint，避免进程在副作用发生后崩溃并自动重复执行。

成功的安全 Tool Result 会保存到 checkpoint；Resume 后不会再次调用该 Tool。Tool 如果依赖具体 Go 类型，可以实现 `CheckpointCodec` 恢复其类型。

Resume 会重新校验：

- Runtime version。
- Skill version。
- Input/Output Contract version。
- Skill source/source version。
- 冻结的 `model_config_id`。

Manager Resume 使用原 Task 的 `model_config_id` 加载 LLM 配置，不使用当前 active LLM。

## Cancel / Resume API

新增：

```text
POST /agents/tasks/:taskId/cancel
POST /agents/tasks/:taskId/resume
```

Cancel 使用 Manager 保存的运行中 `context.CancelFunc`，最终统一进入 Runtime `cancelled` 状态。

Resume 只在安全校验全部通过后异步重新进入同一 Task ID，并增加 `resume_count`。

## Planner 安全边界

Planner 是可选能力，不强制普通 ReAct Skill额外调用一次 LLM。

Planner 只能返回受约束 `Plan`：

- 只能使用 Skill allowlist 中的 Tool。
- Tool 数不能超过剩余 Tool Budget。
- Step ID 必须唯一。
- 依赖必须引用之前的 Step，禁止 self dependency。
- V2-1 Planner 只允许 Tool Step。

Planner 本身不能直接执行 Tool。所有 Plan Tool 仍走统一 Tool 执行入口，因此不能绕过：

```text
Skill allowlist
Permission
Risk
Tool Budget
Timeout
Checkpoint safety
```

Planner cancel / timeout 也统一进入 Runtime 的 `cancelled / timeout` 终态。

## V1 兼容 Gate

V2-0 的 Replay Fixture 已全部在 Runtime 2.0 上通过，保持：

- 四个 V1 Skill output contract。
- malformed Decision → Repair。
- Tool Error 返回 Agent 后可继续。
- Timeout → `timeout`。
- Cancel → `cancelled`。
- Context Too Large → `context_too_large`。
- 最大轮次 → `max_rounds`。

## 自动测试

已通过：

```bash
go test ./...
```

核心并发与恢复链路通过：

```bash
go test -race ./agent/runtime ./agent/replay ./agent/manager ./agent/task
```

覆盖重点：

- ReAct 兼容。
- Step/Event 持久化。
- 安全 Tool checkpoint Resume 后不重复执行。
- unsafe Tool 清除 checkpoint。
- ORM checkpoint Save/Load/Clear。
- Planner Permission / Budget 防绕过。
- Planner self dependency 拒绝。
- Planner cancel / timeout 状态一致。
- Runtime-owned resume metadata 保留。

## 手动验收建议

### 1. 正常任务 Step 追踪

在 Web 执行一次 `AI -> 单币分析`，完成后请求：

```text
GET /agents/tasks/:taskId
```

预期：

```text
runtime_version = 2.0.0
execution_mode = react
steps != []
```

`steps` 应至少看到部分：

```text
build_context
llm
tool
validate
finalize
```

具体 Tool/Repair 次数取决于模型本次输出。

Task `events` 中的新事件应包含 `step_id / step_type`，安全 checkpoint 事件会出现：

```text
checkpoint = true
```

### 2. Cancel

选择一个正在 `waiting_llm` 或 `waiting_tool` 的 Agent Task，调用：

```text
POST /agents/tasks/:taskId/cancel
```

然后查询 Task，预期：

```text
status = cancelled
stage = cancelled
```

如果任务已经完成，Cancel 返回“not actively running”属于正常行为。

### 3. Resume

对一个有安全 checkpoint 的 `cancelled` Task 调用：

```text
POST /agents/tasks/:taskId/resume
```

预期：

```text
Task ID 不变
resume_count + 1
最终继续 succeeded 或进入正常 Runtime 错误状态
```

当前默认 Domain Tool 均为只读幂等 Tool，因此正常单币分析通常具备安全恢复点。

### 4. 冻结模型配置恢复

先创建 Task A 并在运行中 Cancel，记录：

```text
Task A.model_config_id
```

然后把系统 active LLM 切换到配置 B，再 Resume Task A。

预期 Task A 仍使用原来的冻结配置 ID，不应变成 B 的配置 ID。

### 5. 旧任务兼容

V2-0/V1 历史 Task 仍应能正常 List/Get。旧任务没有 `steps/plan` 时允许为空，不应导致任务中心或 API 500。

## 验收

- [x] V1 ReAct Case 全部通过。
- [x] Runtime 核心职责拆开，不再集中于单个大函数。
- [x] 只读任务可从安全 checkpoint 恢复。
- [x] Plan/Step 可以完整追踪。
- [x] Planner 无法绕过 Tool 白名单、Permission 和 Budget。
- [x] Cancel/Resume/Timeout 使用一致的终态转换。
- [x] Resume 使用被冻结的模型配置身份。
- [x] 全量测试与核心 Race 测试通过。

## Phase V2-2 Gate

进入 Context Engine 开发前必须保持：

1. V2-0 Replay 全部通过。
2. V2-1 Runtime/Resume/Planner 测试全部通过。
3. Context Engine 只能替换 `build_context` 的构造与预算机制，不得绕过 Coordinator / RunState / Step / Checkpoint。
4. Context 重构不得让旧 Task、旧 Skill 或 Resume checkpoint 失去可解释的失败路径。
