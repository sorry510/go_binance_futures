# Phase V2-3：Tool Runtime 2.0

> 状态：✅ 已完成

## 目标

把本地 Tool 从 `Registry + Execute` 升级为独立执行层，并为后续 MCP Tool 提供同一入口。V2-3 不连接外部 MCP Server，只冻结本地/远端 Tool 共用的执行契约。

## Runtime 版本

```text
runtime_version = 2.2.0
checkpoint_state = runtime_state_v3
```

V2-2 及更早 checkpoint 不跨 Runtime 版本恢复；新任务统一使用 Tool Runtime 2.0。

## 目录

```text
agent/toolruntime/
  types.go       ToolDescriptor / Envelope / Trace
  errors.go      error taxonomy
  schema.go      JSON Schema compile + validation cache
  cache.go       task-scope TTL cache
  runtime.go     preflight / execute / envelope / trim / redact / evidence
  batch.go       safe batch + bounded parallel execution
```

Runtime 的 Native Tool 调用路径现在统一为：

```text
ReAct / Plan Execute
        ↓
agent/toolruntime.Check
        ↓
Descriptor + Skill allowlist + Permission
        ↓
checkpoint side-effect boundary
        ↓
agent/toolruntime.Execute / ExecuteBatch
        ↓
Schema → Cache → Timeout → Native Tool
        ↓
Output Schema → Redact → Trim → Evidence
        ↓
ToolResultEnvelope + ToolTrace
        ↓
Context Engine / Checkpoint / Validator
```

全仓生产代码中不再存在第二条直接调用 Native `Tool.Execute()` 的 Agent 路径。

## ToolDescriptor

统一 Descriptor 包含：

```text
canonical_name
source_type
source_type = native | mcp
description
input_schema
output_schema
risk
idempotent
timeout_ms
cache_policy
provider_ref
max_result_bytes
```

`cache_policy` 当前结构：

```json
{
  "enabled": true,
  "ttl_ms": 2000,
  "scope": "task"
}
```

Risk 永远来自系统注册的 `Tool.Risk()`，不会读取 LLM arguments 中的 risk，也不会把 Skill 请求当成授权。

V2-3 只预留 `source_type=mcp` 与 `provider_ref`。远端 MCP Tool 的发现、unclassified/disabled 状态和管理员分类在 V2-5 实现，Runtime 核心不需要再增加另一套 Tool 调用入口。

## JSON Schema

Tool Runtime 在底层 Tool 执行前统一验证 `input_schema`，执行成功后验证 `output_schema`。

使用：

```text
github.com/santhosh-tekuri/jsonschema/v5 v5.3.1
```

原因：现有 `github.com/google/jsonschema-go` 用于 Schema 生成，不提供本阶段需要的完整执行验证。V2-3 使用独立 Draft JSON Schema validator，避免只实现当前 Native Tool 能用的 Schema 子集，后续 MCP Tool 可直接复用。

invalid input 行为：

```text
Schema invalid
  ↓
底层 Tool 不执行
  ↓
ToolResultEnvelope.error_type = invalid_input
  ↓
错误返回给 Agent，由下一轮修复 arguments
```

因此 invalid input 不会无条件终止整个 Agent Task。

## ToolResultEnvelope

每次 Tool 调用统一产生：

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

`TOOL_RESULT` 中同时包含 Envelope 和 V2-2 Structured Evidence。

### Result 处理顺序

```text
Native result
  ↓
Output Schema validation
  ↓
JSON encode
  ↓
security.RedactPayload
  ↓
full-result content hash
  ↓
Context Engine Evidence conversion
  ↓
max_result_bytes trim
  ↓
ToolResultEnvelope
```

内部 `session.toolResults` 仍保留原始 concrete Go value，Validator 不会因为给 LLM 的结果被裁剪而失去精确数据。

Checkpoint 保存完整可恢复 Tool raw result；Task/DB 的既有安全脱敏边界继续生效。

## Result 裁剪

V2-1 的行为是 Tool Result 超出上限时直接让 Task 失败：

```text
result exceeds N bytes → task failed
```

V2-3 改为：

```text
result exceeds N bytes
  ↓
保留 raw_size + content_hash
  ↓
LLM data 使用安全 preview
  ↓
partial = true
error_type = partial
warnings = [trim reason]
  ↓
Agent 可以继续分析
```

这符合 V2-3 “统一裁剪而不是无条件失败”的目标。

## Error Taxonomy

统一类型：

```text
invalid_input
not_found
rate_limit
timeout
upstream
stale
partial
permission
internal
```

其中：

- `permission/not_found` 等执行边界错误可直接阻止任务继续。
- Tool 自身 `timeout/upstream/invalid_input` 作为 Envelope 返回 Agent，允许模型调整或降级。
- `stale/partial` 可以是成功 Tool Result 的质量状态，不等于 Tool 没有返回数据。
- parent Runtime context cancel 仍统一进入 V2-1 `cancelled/timeout` 生命周期，不会被伪装成普通 Tool timeout。

## Cache

Native read Tool 的公共 metadata 当前设置：

```text
source_type = native
idempotent = true
cache_ttl = 2s
```

缓存只在同时满足以下条件时使用：

```text
RiskRead
AND idempotent=true
AND cache TTL > 0
```

write/trade Tool 永远不会进入 cache。

缓存 scope 当前为 `task`：每个 Runner/Task 独立，不跨 Agent Task 共享短时行情。这可以消除单个 Agent 的无意义重复调用，同时避免不同任务之间共享可能已经变化的市场数据。

Cache key：

```text
canonical tool name + canonical JSON arguments hash
```

Cache hit 后通过 Tool `CheckpointCodec` 恢复 concrete Go type；例如 `get_symbol_analysis_context` 不会因为缓存命中变成 `map[string]any`，现有 RunValidator 继续正常工作。

## Batch / Parallel

Tool Runtime 提供：

```go
ExecuteBatch(ctx, []BatchRequest, maxParallel)
```

`BatchRequest` 显式包含 `DependsOn`。

只有整批调用都满足：

```text
depends_on 为空
RiskRead
idempotent=true
```

才会并行；任何一个调用不满足条件，整批自动按输入顺序串行执行。

因此 write/trade 或存在依赖的 Tool 不会因为调用者选择 Batch API 而获得并行执行权限。

### ReAct 可选协议

Runtime 现在额外接受：

```json
{
  "action": "parallel_tools",
  "tools": [
    {"tool": "tool_a", "arguments": {}},
    {"tool": "tool_b", "arguments": {}}
  ]
}
```

约束：

- 至少两个 Tool。
- 全部 Tool 必须通过 Skill allowlist + Permission。
- 全部必须 `read + idempotent`。
- 同一批禁止重复 canonical tool name，避免 `toolResults[name]` 产生歧义。
- 并行结果按请求顺序确定性写回 LLM Context。
- 产生一个 `parallel_tools` 父 Step 和独立 Tool 子 Step。
- 整批成功完成后建立一个 safe checkpoint。

现有 V1 Skill Prompt 没有被强制改成 parallel protocol，因此 Replay 行为保持兼容；后续 Skill/Workflow 可选择使用。

## Tool Trace / Budget

`ExecutionStep` 新增：

```text
tool_trace
```

内容包括：

```text
canonical_name
source_type
risk
idempotent
timeout_ms
cache_ttl_ms
arguments_hash
call_index
call_budget
duration_ms
cache_hit
partial
error_type
raw_size
content_hash
as_of
warnings
```

Tool budget 仍由 V2 Runtime 的全局 Task budget 控制，单次和 parallel batch 都在执行前检查总调用次数。

Task Center 的 Step 展开区已展示 Tool Runtime Trace；不新增 HTTP API，也不新增数据库列，继续存入现有 `steps_json`。

## Evidence / Freshness

V2-2 Context Engine 仍是 Evidence 唯一转换器。Tool Runtime 成功结果统一送入：

```text
ContextEngine.ConvertToolResult
```

因此：

- Tool Envelope 与 Evidence 使用同一 source。
- stale/missing freshness 会映射成 `error_type=stale` + warnings。
- `data_missing` 会把 Envelope 标成 partial。
- full content hash 可从 ToolTrace、Envelope、Evidence 互相追踪。

## Checkpoint 安全

V2-1 原安全规则保持不变：

```text
read + idempotent → 可以建立 safe checkpoint
write/trade/non-idempotent → 执行前清 checkpoint
```

Permission 预检发生在清 checkpoint 之前，所以一个本来就没有授权的 write Tool 不会仅因为被 LLM 请求就破坏当前安全恢复点。

Parallel batch 只允许 safe read/idempotent Tool，因此可以在整批结果确定性合并后保存 checkpoint。

## 数据库

V2-3 没有新增表或字段。

继续使用现有：

```text
agent_tasks.steps_json
agent_tasks.checkpoint_json
agent_task_events
```

没有新增 `command/sql/version/2.sql`。

## 自动测试

覆盖：

- ToolDescriptor 由系统 Tool metadata 生成。
- `source_type=native`、cache policy、timeout、Risk。
- Input JSON Schema 在底层 Execute 前拦截。
- Output JSON Schema 执行后验证。
- Permission 使用注册 Risk，不信任 arguments。
- timeout taxonomy。
- stale taxonomy。
- partial/data_missing。
- oversize result trim。
- TTL cache hit/miss。
- cache concrete type 恢复。
- safe read batch 并行。
- write batch 自动串行。
- ReAct `parallel_tools` 真并发执行。
- ToolTrace / Budget 持久化到 Step。
- V2-0 Replay 回归。

## 验收

- [x] Native Tool 全部通过统一执行层。
- [x] Tool Schema 在执行前校验。
- [x] 错误类型结构化。
- [x] 并行、缓存和 partial result 有测试。
- [x] 后续 MCP Tool 无需修改 Runtime 核心即可接入。

## 手动验收建议

1. 执行一次 `symbol_analysis`。
2. 任务详情确认：

```text
runtime_version = 2.2.0
```

3. 展开 `get_symbol_analysis_context` Tool Step，确认存在：

```text
tool_trace.canonical_name
tool_trace.source_type = native
tool_trace.risk = read
tool_trace.call_index / call_budget
tool_trace.raw_size
tool_trace.content_hash
```

4. 若同一 Task 在 2 秒内重复请求同 Tool + 同 arguments，可观察第二次：

```text
cache_hit = true
```

5. Evidence/Context Trace 仍应与 V2-2 一致。
6. 原单币分析、报警分析、Market Regime、Strategy Builder 最终业务 JSON 不应改变。
