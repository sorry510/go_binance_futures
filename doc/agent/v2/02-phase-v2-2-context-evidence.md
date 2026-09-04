# Phase V2-2：Context Engine 与 Evidence Model

> 状态：✅ 已完成

## 1. 目标

V2-2 把“每轮给模型什么内容”从 Skill/Runtime 直接累加 `[]llm.Message`，升级为统一 Context Engine，并为 Tool 数据建立可审计的结构化 Evidence。

本阶段解决：

- 历史消息不断增长后只能触发 `context_too_large`，无法按价值裁剪；
- 当前任务、实时市场事实、历史消息没有统一优先级；
- Tool Result 只能证明“调用过某个 Tool”，无法识别具体数据版本、时间与内容 hash；
- stale 数据没有统一的 Runtime freshness 标记；
- Cancel/Resume 后需要保留 Context/Evidence 身份；
- 为 V2-6 标准 Agent Skills 的渐进式 Resource 加载预留统一协议。

## 2. 兼容边界

V2-2 不修改现有业务输出契约：

```text
symbol_analysis -> trading_plan_v1
alert_analysis  -> alert_v1
```

现有业务 JSON 中的：

```json
{"source":"get_symbol_analysis_context","finding":"..."}
```

继续保留，避免破坏前端、通知、历史记录和其它调用方。

Runtime 内部新增结构化 Evidence Registry。`symbol_analysis` / `alert_analysis` 在最终 Validator 中把 V1 `source` 映射回本轮 Runtime Evidence，验证该来源确实有对应的 Tool 数据、content hash 和 freshness 记录。

本阶段没有新增数据库表或字段；Context Trace 与 Evidence 审计信息复用已有 `agent_tasks.steps_json`。

## 3. Runtime 版本

V2-2 Runtime：

```text
runtime_version = 2.1.0
runtime_state_version = runtime_state_v2
```

V2-1 的 Runtime 2.0 checkpoint 不允许直接由 2.1 Runtime Resume。现有版本身份检查会拒绝跨 Runtime 版本恢复，避免用不同 Context/Evidence 语义继续执行旧 checkpoint。

V1 Skill 的 `skill_version / prompt_version / input_contract_version / output_contract_version` 不因本阶段改变。

## 4. Context Engine

新增目录：

```text
agent/contextengine/
├── types.go
├── estimator.go
├── freshness.go
├── hash.go
├── builder.go
├── messages.go
├── resources.go
├── tool.go
└── contextengine_test.go
```

### 4.1 ContextBlock

统一 Context 单元：

```go
type ContextBlock struct {
    ID              string
    Type            BlockType
    Source          string
    Role            string
    AsOf            string
    Priority        int
    EstimatedTokens int
    Freshness       Freshness
    Sensitive       bool
    Required        bool
    Content         string
    ContentHash     string
    EvidenceIDs     []string
    Order           int
}
```

支持类型：

```text
system
task
market
history
memory
tool
skill_instruction
mcp_resource
```

其中 `memory` 和 `mcp_resource` 已定义协议，本阶段不提前实现 V2-5/V2-9 的具体数据源。

### 4.2 默认优先级

```text
system             1000
task                900
market              800
skill_instruction   700
tool                 600
mcp_resource         500
memory               300
history              200
```

stale block 有额外优先级折减，missing block 折减更高。

当前任务可以标记 `Required=true`。预算不足时 Context Engine 先裁剪低优先级历史，而不是让旧对话挤掉当前任务或行情事实。

### 4.3 BuildInput 转换

现有 Skill 接口保持不变：

```go
BuildInput(...) ([]llm.Message, error)
```

Runtime 在 BuildInput 后转换为 ContextBlock：

- 最后一条初始消息：`task`，required；
- 前面的消息：`history`；
- `AGENT_FEEDBACK`：高优先级 `task`，required；
- Tool Result：`tool` / `market`；
- Skill Resource：`skill_instruction` 或声明的 Resource 类型。

因此 Strategy Builder 现有 conversation history 也自动进入统一预算，不需要重写 Skill API。

## 5. Token Budget

新增：

```go
Config.MaxContextTokens
```

默认：

```text
64 Ki tokens（启发式估算预算）
MaxContextBytes 继续作为第二道 byte 上限
```

Token estimator 使用保守启发式：

- CJK/非 ASCII：约 1 token / rune；
- ASCII/代码：约 4 char / token。

它不是 provider tokenizer 的精确替代，目标是在真正请求模型前提前做稳定裁剪。

以后 V2-8 Model Gateway 可以根据模型 capability 覆盖 `MaxContextTokens`，本阶段 Runtime 已支持不同预算配置。

### 5.1 裁剪规则

构建顺序：

1. 计算 system prompt 成本；
2. ContextBlock 标准化、hash、token estimate；
3. 相同 content hash 去重；
4. Required block 先进入预算；
5. Optional block 按 priority + freshness 排序；
6. 预算不足时记录 trim，不失败；
7. 只有 required context 本身无法装入时返回 `context_too_large`。

Build Trace 记录：

```text
budget_tokens
budget_bytes
system_tokens
selected_tokens
selected_bytes
input_blocks
selected_blocks
trimmed_blocks
selected_block_ids
trimmed[]
stale_evidence_ids
built_at
```

发生裁剪时 Task Event 增加：

```text
stage = context_trimmed
```

并记录裁剪数量和最终 estimated token。

## 6. Freshness Policy

统一状态：

```text
fresh
stale
missing
unknown
```

当前默认策略：

| Source | Max Age |
| --- | ---: |
| get_symbol_analysis_context | 3 min |
| get_symbol_snapshot | 3 min |
| get_features | 3 min |
| get_klines | 15 min |
| get_liquidations | 10 min |
| get_funding_rate | 2 h |
| get_market_condition | 2 h |

没有可靠 timestamp 且策略不强制 timestamp 的数据标记为 `unknown`，不会伪装成 fresh。

Timestamp 提取兼容项目当前数据格式，包括：

```text
as_of
updated_at_ms / updatedAtMs
update_time / updateTime
event_time / eventTime
timestamp
created_at / createdAt
closeTime / openTime
```

如果 Tool Result 的 `data_missing` 已包含 `stale` 标记，例如：

```text
symbol_snapshot_stale
```

则显式优先标记 Evidence 为 stale。

被选入模型上下文的 stale/missing block 会增加：

```text
CONTEXT_FRESHNESS source=... status=... as_of=...
```

模型不能把过期数据误认为正常实时数据。

## 7. Evidence Model

结构化 Evidence：

```go
type Evidence struct {
    ID           string
    SourceType   string
    Source       string
    AsOf         string
    ObservedAt   string
    ContentHash  string
    Freshness    Freshness
    FreshnessAge int64
    StaleReason  string
    KeyFields    map[string]string
    DataMissing  []string
}
```

Tool Result 转换过程：

```text
Tool success
    ↓
JSON canonical representation
    ↓
SHA-256 content hash
    ↓
extract timestamp / data_missing / key fields
    ↓
Freshness Policy
    ↓
Evidence ID + ContextBlock
```

Evidence ID 基于 source + content hash 确定性生成。同一 Tool 的同一份内容会得到相同 Evidence identity。

Tool Result 返回给 LLM 的 Envelope 现在同时包含：

```json
{
  "tool": "get_symbol_snapshot",
  "ok": true,
  "result": {},
  "evidence": []
}
```

因此模型可以看到证据身份和 freshness，而不是只有原始 Tool JSON。

## 8. Final Validator 与 Evidence

`symbol_analysis` 和 `alert_analysis` 新增可选接口实现：

```go
ValidatorForRunWithEvidence(
    req,
    toolResults,
    evidenceRegistry,
)
```

验证过程仍先执行原 V1 Validator，然后附加：

- final evidence source 必须是本轮真实成功 Tool；
- Runtime Evidence Registry 中必须存在相同 source；
- Evidence 必须有 content hash；
- `missing` Evidence 不能作为可用结构化证据。

stale Evidence 不会被偷偷升级为 fresh；它保留 stale 状态并进入 Context Trace / Tool Step / Validate Step 审计。

最终 Validate Step 会附带本轮结构化 Evidence 快照，因此任务完成清理 checkpoint 后仍可以追溯。

## 9. ExecutionStep 审计

ExecutionStep 新增：

```text
context_trace
evidence[]
```

典型分布：

```text
LLM Step
  -> context_trace

Tool Step
  -> evidence[]

Validate Step
  -> 本轮最终 Evidence 快照
```

这些数据通过原有 `steps_json` 持久化，不增加 schema。

## 10. Checkpoint / Resume

V2-2 checkpoint 保存：

```text
ContextBlocks
Evidence Registry
ToolResults
ExecutionSteps
其它 V2-1 Runtime State
```

为了避免 Tool Result 在 `Messages` 与 `ContextBlocks` 中重复保存，`RunState.Messages` 不再单独 JSON 序列化。

Resume 时根据 ContextBlocks 重建消息序列；ToolResults 继续通过 V2-1 `CheckpointCodec` 恢复具体 Go 类型。

这避免 V2-2 因 Evidence/Context 引入不必要的 checkpoint 体积翻倍。

## 11. Progressive Disclosure

Skill 新增可选接口：

```go
type ContextResourceProvider interface {
    ContextResources(req Request) []contextengine.Resource
}
```

Resource 支持：

```text
activation
on_demand
```

规则：

- activation Resource：Skill 激活时加载；
- on_demand Resource：只有请求指定 ID 时加载；
- 请求 metadata key：`context_resource_ids`。

这为 V2-6 Agent Skills 准备：

```text
Skill metadata -> 索引
SKILL.md       -> activation
references/*  -> on_demand
assets/*      -> on_demand
```

本阶段只完成 Runtime 协议，不提前实现 ZIP Agent Skill importer。

## 12. 前端

AI → 任务中心 → Task Detail → Execution Steps 支持展开审计信息。

LLM Step 展示：

```text
Token budget
selected/input blocks
trimmed blocks
stale evidence count
trim records
```

Tool / Validate Step 展示：

```text
Evidence ID
source
freshness
as_of
content hash
```

自动刷新继续使用 V2-1 的静默刷新，不增加 loading overlay。

## 13. 数据库

V2-2：

```text
无新增表
无新增字段
无 version/2.sql
```

继续复用：

```text
agent_tasks.steps_json
agent_tasks.checkpoint_json
```

## 14. 自动测试

Context Engine 覆盖：

- 超预算优先裁 History；
- 当前 Task/Market 保留；
- Required Context 装不下才失败；
- Evidence ID/hash 确定性；
- stale timestamp；
- `data_missing` stale marker；
- 项目真实 camelCase `updateTime` freshness；
- progressive resource disclosure。

Runtime 覆盖：

- Tool structured Evidence 注入下一轮 LLM；
- stale freshness header；
- Tool Step Evidence 持久化；
- LLM Step Context Trace 持久化；
- 超预算 Context 自动裁剪并继续执行；
- `context_trimmed` Event；
- Skill activation/on-demand Resource；
- V2-0 Replay 不变。

## 15. 手动验收

### 15.1 单币分析 Evidence

执行一次：

```text
AI -> 单币分析 -> BTCUSDT
```

然后进入：

```text
AI -> 任务中心 -> 对应 Task -> 查看
```

预期：

```text
runtime_version = 2.1.0
execution_mode = react
```

展开 Tool Step，可看到：

```text
Evidence ID
source = get_symbol_analysis_context
freshness
Data Time
Content Hash
```

展开 Validate Step，应看到本轮用于最终校验的 Evidence 快照。

### 15.2 Context Trace

展开 LLM Step，应看到：

```text
Token 预算
Context Block 数量
裁剪数量
stale Evidence 数量
```

普通短任务 `trimmed_blocks` 通常为 0，这是正常结果。

### 15.3 Freshness

正常实时运行时 `get_symbol_analysis_context` 应通常为 `fresh`。

如果行情快照确实超过 3 分钟，Runtime 会标记 stale；不要为了手测在生产环境故意停止行情服务，自动测试已经覆盖 stale 路径。

### 15.4 V1 输出兼容

单币分析最终结果仍应是：

```text
version = trading_plan_v1
```

报警分析仍应是：

```text
version = alert_v1
```

原有页面、通知和历史记录不应要求迁移到新的业务 JSON Schema。

## 16. Gate

- [x] ContextBlock / Evidence 类型
- [x] token estimator / budget allocator
- [x] explainable trim trace
- [x] freshness policy
- [x] Tool Result -> Evidence / Context
- [x] Final Validator 绑定 Runtime Evidence
- [x] Checkpoint 保存 Context/Evidence 并可 Resume
- [x] Progressive Disclosure Runtime 协议
- [x] Task Center Evidence / Context Trace 展示
- [x] V2-0 Replay Gate
- [x] 后端全量测试
- [x] 前端 TypeScript 检查与生产构建

V2-2 完成后，下一阶段严格进入 V2-3 Tool Runtime，不提前实现 MCP 或 Agent Skills importer。
