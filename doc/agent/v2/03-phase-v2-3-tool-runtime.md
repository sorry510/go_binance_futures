# Phase V2-3：Tool Runtime 2.0

## 目标

把本地 Tool 从 Registry + Execute 升级为独立执行层，并为后续 MCP Tool 提供同一入口。

## 统一 ToolDescriptor

建议包含：`canonical_name`、`source_type`、`description`、`input_schema`、`output_schema`、`risk`、`idempotent`、`timeout`、`cache_policy`、`provider_ref`。

`source_type` 至少支持 `native` 和 `mcp`。

## ToolResultEnvelope

```text
data
source
as_of
duration_ms
cache_hit
partial
warnings
error_type
raw_size
content_hash
```

## 关键能力

- JSON Schema 输入统一校验。
- error taxonomy：invalid_input、not_found、rate_limit、timeout、upstream、stale、partial、permission、internal。
- 幂等只读 Tool 短 TTL cache。
- 无依赖只读 Tool 并行。
- batch Tool，避免 LLM 对 N 个 symbol 发 N 次调用。
- Tool Result 统一裁剪、脱敏和 Evidence 转换。
- Tool 调用全程记录 Budget 与 Trace。

## 权限

Risk/Permission 根据系统 ToolDescriptor 决定，不能信任 LLM 参数、Skill 包或远端 MCP Server 自报权限。

外部 MCP Tool 默认状态为 `unclassified/disabled`，管理员明确分类并授权后才能进入 Skill 白名单。

## 验收

- [ ] Native Tool 全部通过统一执行层。
- [ ] Tool Schema 在执行前校验。
- [ ] 错误类型结构化。
- [ ] 并行、缓存和 partial result 有测试。
- [ ] 后续 MCP Tool 无需修改 Runtime 核心即可接入。
