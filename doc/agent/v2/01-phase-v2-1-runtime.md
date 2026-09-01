# Phase V2-1：Runtime V2 与执行状态机

## 目标

拆分当前 `DefaultRunner.Run()` 的复合职责，使 Agent 支持显式 Step、不同执行模式、Checkpoint 和可控恢复，同时保留 V1 ReAct 兼容路径。

## 目标组件

```text
agent/runtime/coordinator   Run 生命周期
agent/runtime/protocol      Decision / Plan / Final 协议
agent/runtime/executor      LLM / Tool / Validate Step
agent/runtime/recovery      Retry / Resume / Checkpoint
agent/runtime/state         RunState / ExecutionStep
```

## Execution Mode

- `react`：兼容 V1，适合简单动态 Tool 调用。
- `workflow`：步骤由代码预定义，LLM 只处理指定节点。
- `plan_execute`：先形成受约束 Plan，再逐步执行和修订。

Skill 声明执行模式，但 Runtime 仍是唯一 Coordinator。

## ExecutionStep

每个 Step 至少包含：`step_id`、`type`、`status`、`attempt`、`depends_on`、输入摘要、输出摘要、开始/结束时间、错误类型、checkpoint 标记。

Step 类型至少支持：`build_context`、`llm`、`tool`、`parallel_tools`、`validate`、`approval`、`finalize`。

## Checkpoint 与恢复

- 只在确定性安全边界产生 checkpoint。
- 已成功且幂等的 Tool Step 默认不重复执行。
- write/trade Step 永远不能仅凭 Task 状态自动重放，必须使用幂等键和执行记录。
- 重启恢复时重新加载被冻结的 Skill/Prompt/Model 版本。

## 工作项

1. 抽 `RunState` 和 `ExecutionStep`。
2. 将现有 ReAct 循环搬到 Coordinator/Executor，不立即改变四个 Skill 行为。
3. Task Event 增加 step_id 和结构化状态。
4. 建 checkpoint store 接口并复用 Task Store。
5. 增加 cancel/resume/timeout 的一致状态转换。
6. Planner 后置为可选模块，不强迫所有 Skill 额外耗费一次 LLM。

## 验收

- [ ] V1 ReAct Case 全部通过。
- [ ] Runtime 核心职责已拆开，不再集中于单个大函数。
- [ ] 只读任务可从安全 checkpoint 恢复。
- [ ] Plan/Step 可以完整追踪。
- [ ] Planner 无法绕过 Tool 白名单、Permission 和 Budget。
