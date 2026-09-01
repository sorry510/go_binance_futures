# Phase V2-7：长期 Memory 与历史知识管理

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

- [ ] Conversation/Memory/Task 清晰分离。
- [ ] 市场记忆自动过期。
- [ ] Memory 读写进入 Trace。
- [ ] 用户可管理 Memory。
