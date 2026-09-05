# Phase V2-9：长期 Memory 与历史知识管理

## 目标

Conversation 负责一次连续对话，Memory 负责跨 Task 仍值得保留的信息；两者使用独立生命周期。

## Memory 类型

`user_preference`、`strategy_fact`、`market_hypothesis`、`task_summary`、`lesson`。

字段至少包含：scope、source task、created/updated/expire time、confidence、status、content hash。

## Scope

支持按 user、skill、symbol、strategy 查询。市场假设必须有 TTL；用户偏好可长期保存但必须可查看、删除和禁用。

## 写入

不自动把全部 Conversation 转 Memory。写入经过规则和权限；LLM 可以提出 memory candidate，但不能自己决定永久保存高风险交易事实。

## 读取

Context Engine 根据 task/scope/freshness 检索，预算不足时 Memory 优先级低于当前事实和 Skill 指令。

初版使用结构化关系数据库，不急于引入向量数据库。

## 验收

- [x] Conversation/Memory/Task 清晰分离。
- [x] 市场记忆自动过期。
- [x] Memory 读写进入 Trace。
- [x] 用户可管理 Memory。


## 实现摘要

- 新增 `agent_memories` 关系表与 `agent/memory` 独立包，Conversation、Task、Memory 使用独立表和生命周期。
- 支持 `user_preference`、`strategy_fact`、`market_hypothesis`、`task_summary`、`lesson`，字段包含 Scope、source task、confidence、status、content hash、created/updated/expires time。
- Scope 按 user / skill / symbol / strategy 的非空字段做交集匹配；Memory 以 `BlockMemory` 注入现有 Context Engine，优先级保持低于当前事实、Task 与 Skill 指令。
- `market_hypothesis` 强制 TTL，未显式指定时默认 6 小时；过期后不再进入 Context，并在访问时收敛为 `expired` 状态。
- Runtime 成功任务只自动持久化低风险 `task_summary`，默认 TTL 30 天并按 source task 幂等；`strategy_fact`、`market_hypothesis`、`user_preference` 不允许 Runtime 自动永久保存。
- 支持 `candidate` 状态及人工审批；仅 `active` Memory 可以被 Context 检索。
- Context Trace 新增 `selected_memory_ids` / `trimmed_memory_ids`；Runtime Task Event 新增 `memory_read` / `memory_write` 审计事件。
- 新增 `/agents/memories` 管理 API，以及前端 `AI -> Memory 管理` 页面，支持筛选、新增、编辑、审批、禁用、启用、删除和查看过期状态。
