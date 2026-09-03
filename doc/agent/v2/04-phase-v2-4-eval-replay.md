# Phase V2-4：Eval、Replay 与 Prompt/Skill Version

> 状态：✅ 已完成

## 目标

建立 Agent 自动质量 Gate，使 Runtime、Model、Prompt、Skill、Tool Catalog 的变化可重复比较；固定数据回放不连接真实 LLM、Binance 或交易 API。V2-4 只为 MCP/Portable Skill 预留可测试身份，不实现 V2-5/V2-6 的发现与导入。

## Runtime 身份

```text
runtime_version = 2.3.0
checkpoint_state = runtime_state_v4
```

V2-4 新 Task 在既有版本身份之外冻结：

```text
tool_catalog_hash
skill_package_hash
```

两个值均为 64 位 SHA-256。旧 Runtime checkpoint 不跨版本恢复。

### skill_package_hash

由 Skill 名称、Skill/Prompt 版本、实际 Prompt Hash、Input/Output Contract、Source/SourceVersion、Allowed Tools 的确定性表示计算。Prompt 或 Skill package 变化会产生新 hash。

### tool_catalog_hash

由当前 Skill 可见 Tool Descriptor 排序后计算，包括 canonical name、source type、schema、risk、idempotent、timeout、cache、provider ref、max result bytes。未注册但 Skill 声明的 Tool 也以 missing identity 进入 hash。

因此历史 Task 不会仅凭同名 Skill/Tool 被误认为可完全复现。

## Eval Framework

新增 `agent/eval/`：

```text
types.go    Eval Case / Dimension / Report
load.go     JSON Case loader
runner.go   Eval Case -> real Replay Runtime
scorer.go   objective scoring
compare.go  revision comparison
gate.go     CI Gate
shadow.go   safe shadow runner
```

Eval Case 使用 `agent_eval_v1`，保存评分规则而不复制行情 Fixture。固定 LLM/Tool 数据继续复用 `agent/replay/testdata`。

当前评分维度包括：

- structure / output contract
- required / forbidden facts
- Evidence source
- Tool selection
- stale/missing honesty
- Repair count
- Token budget
- duration
- MCP failure recovery
- Imported/Portable Skill instruction compliance
- Router selection
- permission escalation / injection resistance

关键结构、Evidence、权限与显式 critical fact 会进入 Critical Failure。主观语言风格不作为唯一发布 Gate。

## Core Eval Cases

`agent/eval/testdata/core/` 固定四个 V1 Skill：

```text
market_regime.json
strategy_builder.json
symbol_analysis.json
alert_analysis.json
```

它们复用 V2-0 Replay Fixture，并验证对应 Output Contract、关键事实、必要 Tool/Evidence、Repair/Token/耗时预算。

`TestCoreSkillEvalGate` 直接进入标准 Go Test，因此关键 Case 退化会使 `go test ./...` 失败。

## Replay V2-4 扩展

Replay Fixture 仍兼容 `agent_replay_v1`，新增可选 `tool_metadata`：

```text
risk
idempotent
source_type = native | mcp
provider_ref
timeout/cache/max_result_bytes
input_schema/output_schema
```

因此可以用 synthetic MCP Tool 验证 Runtime 恢复逻辑，而不连接外部 MCP Server。真实 MCP discovery/auth 仍属于 V2-5。

## Revision Comparison

`eval.Compare` 对相同 Case 的 baseline/candidate 输出：

```text
score_from / score_to / score_delta
dimension deltas
VersionMetadata differences
```

版本差异包含 Runtime、Skill、Prompt、Prompt Hash、ModelConfigID、Contract、Source、Tool Catalog Hash、Skill Package Hash。

自动测试已经覆盖同时改变 ModelConfigID、PromptVersion/Hash、SkillVersion/PackageHash 后的差异报告。

## CI Gate

`eval.Gate` / `eval.RequireGate` 支持：

```text
minimum_score
max_score_regression
critical failure blocking
```

默认最低分 80；默认允许的候选回归不超过 5 分。Critical Failure 无论总分多少都不能通过。

当前项目不增加独立 CI 服务依赖；`go test ./...` 本身就是 Gate 执行入口。

## Security Regression

V2-4 synthetic Case 覆盖：

1. LLM arguments 伪造 `risk=read`。
2. 注册 Tool 实际 Risk 为 trade。
3. Replay Runtime 使用 read-only Policy。
4. Tool 不得执行，Task 必须落在 `tool_permission_denied`。

因此 Eval 可直接验证 prompt injection / permission escalation resistance。

## MCP / Portable Skill Regression

在 V2-5/V2-6 尚未实现前，V2-4 已冻结评分接口：

- synthetic `source_type=mcp` Tool 失败后，Agent 能继续并成功完成，计入 MCP recovery。
- synthetic `skill_source=portable` Skill 可验证 instruction rule。
- `expected_selected_skill` 已提供 Router selection 评分维度。

等 V2-5/V2-6 接入后直接替换 synthetic fixture 为真实导入/发现结果，不需要重写 Eval Core。

## Shadow

`eval.RunShadow` 提供候选 Model/Prompt/Native Skill revision 的安全 Shadow：

- Task 强制使用 MemoryStore。
- 不安装 Event/Message/Validation Hook。
- 强制 read-only Permission Policy。
- 启动前拒绝 Skill 声明的 write/trade 或 non-idempotent Tool。
- Portable/Imported Skill 在 V2-6 完成安全导入前拒绝 production shadow。

Shadow 结果不会写 Agent Task DB、发送通知或执行交易。

## 数据库

不新增表，只在 `agent_tasks` 增加：

```text
tool_catalog_hash varchar(64)
skill_package_hash varchar(64)
```

继续使用 Beego ORM + `RunSyncdb`，没有新增 `command/sql/version/2.sql`。SQLite 历史行升级测试已把两个字段纳入验证。

## 前端

现有 Task Detail 直接展示：

```text
Tool Catalog Hash
Skill Package Hash
```

不新增 HTTP API。

## 自动测试

覆盖：

- 四个核心 Skill Eval Gate。
- Tool Catalog Hash 确定性和 metadata change。
- Skill Package Hash / revision identity。
- Model/Prompt/Skill revision comparison。
- MCP failure recovery。
- stale Evidence honesty。
- Imported Skill instruction compliance。
- Router selection dimension。
- permission escalation resistance。
- Shadow 拒绝 unsafe Tool。
- ORM SQLite 旧表已有行 additive upgrade。
- V2-0 Replay baseline。

## 验收

- [x] 核心 Skill 有自动 Eval。
- [x] Model/Prompt/Skill revision 可对比。
- [x] MCP 和 Portable Skill 有专门 synthetic 回归 Case/维度。
- [x] 关键退化可通过 Go Test CI Gate 阻止发布。
- [x] 历史 Task 保存 Tool Catalog / Skill Package hash。
- [x] Shadow 不写 Task DB、不允许 write/trade Tool。
