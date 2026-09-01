# Phase V2-7：Model Gateway、Capability 与 Router

## 目标

从“启用一个模型”升级为模型能力、健康度和任务级路由，同时保持 Skill 与 Provider 解耦。

## Capability

系统维护模型 Profile：`structured_output`、`native_tool_calling`、`reasoning`、`long_context`、`json_reliability`、`max_context_tokens`、`cost_class`、`latency_class`。

Skill/Execution Profile 声明能力需求，不硬编码模型名。

## Router

选择顺序综合：主模型、能力匹配、健康度、延迟、成本和任务类型。Alert 可偏低延迟；Strategy Builder 可偏高推理能力。

Fallback 必须记录原因，且不能突破 Budget/Permission。

## Health

维护 Provider/Model 近期成功率、429、timeout、5xx、平均延迟；使用熔断/半开机制避免每个任务重复撞坏 Provider。

## Native Tool Calling

可以在 LLM Adapter 层使用厂商原生 Tool Calling，但上层仍转换为统一 ExecutionStep/Tool Runtime，不让 Skill 绑定厂商协议。

## 验收

- [ ] Router 可关闭并回退 V1 单模型行为。
- [ ] Task 保存候选、最终模型和路由原因。
- [ ] Provider 故障可按策略 fallback。
- [ ] 模型切换通过 Eval Gate。
