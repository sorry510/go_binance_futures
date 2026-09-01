# Phase V2-2：Context Engine 与 Evidence Model

## 目标

把“给模型什么”从 Skill 手工拼接升级为统一 Context Engine，并建立可验证 Evidence，解决上下文过长、重复数据、旧数据污染和结论不可追踪问题。

## Context Block

统一结构建议：

```text
id, type, source, as_of, priority
estimated_tokens, freshness, sensitive
content, content_hash, evidence_ids
```

类型至少包含 `system`、`task`、`market`、`history`、`memory`、`tool`、`skill_instruction`、`mcp_resource`。

## Evidence

Evidence 不是自然语言 finding，而是结构化事实引用：来源 Tool/MCP Resource、数据时间、关键字段摘要、content hash、freshness 状态。

Final Result 可引用 Evidence ID；Validator 可检查关键结论是否拥有有效证据。

## Token Budget

按优先级分配预算：协议/安全 > 当前任务 > 最新市场事实 > 必需 Skill 指令 > 当前 Tool 结果 > Memory > 历史对话。

超预算时先裁剪低优先级块，记录裁剪原因；不能让旧历史挤掉当前行情。

## Progressive Disclosure

本 Phase 同时为标准 Agent Skills 准备加载机制：启动时只索引 Skill metadata；激活 Skill 后加载 `SKILL.md`；`references/`、`assets/` 仅在需要时作为 Resource 加载。

## 工作项

1. ContextBlock/Evidence 类型。
2. token estimator 和预算分配器。
3. freshness policy。
4. Tool Result -> Evidence/Context 转换器。
5. Conversation/Memory/Skill Resource 的优先级策略。
6. Context Build Trace 和裁剪日志。

## 验收

- [ ] 不同上下文窗口模型都能稳定构造输入。
- [ ] 超预算有可解释裁剪，不是无条件失败。
- [ ] 实时数据过期会显式标记 stale/missing。
- [ ] 关键交易结论可追溯 Evidence。
- [ ] Skill/Resource 可渐进加载。
