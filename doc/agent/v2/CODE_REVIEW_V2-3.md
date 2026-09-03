# V2-3 Code Review 报告

> 审查范围：当前工作区（未提交）代码
> 后端：排除 `static/`（前端打包产物）
> 前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui`（4 个改动文件）
> 对照文档：`doc/agent/v2/03-phase-v2-3-tool-runtime.md`
> 审查日期：2026-09-02

---

## 一、结论摘要

| 维度 | 结论 |
|------|------|
| Tool Runtime 2.0 核心实现（新包 `agent/toolruntime/`，7 文件） | ✅ 与文档 §ToolDescriptor / §Envelope / §JSON Schema / §Error Taxonomy / §Cache / §Batch 逐项一致 |
| Runtime 集成与版本身份 | ✅ `2.2.0` / `runtime_state_v3` 双闸门生效，旧 checkpoint 被拒 |
| 前端 Tool Trace 展示 | ✅ `AgentToolTrace` 与 `toolruntime.Trace` 逐字段对齐，i18n 完整 |
| 数据库变更 | ✅ 无新增表 / 无新增字段 / 无新迁移 SQL，复用 `steps_json`/`checkpoint_json` |
| 后端测试（`-count=1 -race`） | ✅ `agent/...` 全 `ok`（含 `toolruntime`/`runtime`/`replay`）；`models`/`llm`/`notify`/`service` 全 `ok` |
| 前端 TypeScript 检查 | ✅ `vue-tsc --noEmit --skipLibCheck` EXIT=0 |
| **V2-3 Gate（5 项）** | ✅ **5/5 达成，无阻塞缺陷，可进入 V2-4** |

本轮共发现 **0 个阻塞项、0 个重要项、3 个建议项、2 个提示项**，均不影响 Gate 判定，详见第五节。

---

## 二、Gate 5 项逐项核对

| # | Gate 项（文档 §验收） | 实现证据 | 结论 |
|---|----------------------|----------|------|
| 1 | Native Tool 全部通过统一执行层 | `agent/toolruntime/runtime.go` `Check()`/`Execute()` 为唯一执行入口；`react_executor.go:136 executeTool` 不再直调 `Tool.Execute()`，而是构造 `toolruntime.ExecuteRequest` 经 `ToolRuntime.Check`→`ToolRuntime.Execute`；`register.go:76` 系统 Tool `SourceType:"native"` | ✅ |
| 2 | Tool Schema 在执行前校验 | `schema.go:61 compiler.Draft = jsonschema.Draft7`；`runtime.go` `Execute` 顺序为 `Check→normalize→input-schema→cache→timeout→tool→output-schema→cache-set`，**input schema 在底层 execute 之前拦截**（见 `runtime_test.go` input-schema reject 用例） | ✅ |
| 3 | 错误类型结构化 | `errors.go` 9 类 `ErrorType`（invalid_input/not_found/rate_limit/timeout/upstream/stale/partial/permission/internal）；`classifyToolError` 将 `context.DeadlineExceeded`/`Canceled`→timeout、`net.Error`→timeout/upstream、消息子串→rate_limit/not_found/timeout/upstream/internal；`Envelope.ErrorType` 全链路透传 | ✅ |
| 4 | 并行、缓存和 partial result 有测试 | `runtime_test.go` 10 用例覆盖：input-reject-before-exec、output-reject、cache concrete-type restore、timeout、partial、stale、permission(注册 Risk)、batch parallel-read / serial-write、not-found；`batch.go` `ExecuteBatch` 并行条件 `len>1 && 全无 DependsOn && read+idempotent` | ✅ |
| 5 | 后续 MCP Tool 无需修改 Runtime 核心即可接入 | `ToolDescriptor` 用 `CanonicalName`/`SourceType`(`native`|`mcp`)/`ProviderRef`/`CachePolicy` 抽象；`Runtime.Execute` 仅依赖 `registry` 出 `tools.Tool` 接口，不耦合具体 provider；MCP 接入只需新增 `SourceType:"mcp"` 的注册与 `ProviderRef` 解析 | ✅ |

---

## 三、后端 Review

### 3.1 Tool Runtime 新包 `agent/toolruntime/`

| 文件 | 核对结果 |
|------|----------|
| `types.go` | `ToolDescriptor`（CanonicalName/SourceType/Risk/Idempotent/TimeoutMs/CachePolicy/ProviderRef 等）、`CachePolicy{Enabled,TTLms,Scope}`（`NewCachePolicy` 设 `Scope="task"`）、`ToolResultEnvelope`（Data/Source/AsOf/DurationMs/CacheHit/Partial/Warnings/ErrorType/RawSize/ContentHash）、`Trace`、`ExecuteRequest`、`ExecuteResult` 字段与文档 §ToolDescriptor / §Envelope 一致；JSON tag 全 snake_case |
| `errors.go` | 9 类 `ErrorType` 常量；`Error()`/`Unwrap()` 实现；`TypeOf(err)` 安全提取；`classifyToolError` 分支覆盖 context 取消/超时、网络错误、子串匹配；`ErrorType` 直接落 `Envelope.ErrorType` |
| `schema.go` | `schemaValidator`（mutex + `items map` 编译缓存，sha256 作 key）；`validate()` 跳过空 schema，使用 `decoder.UseNumber()` 保留数字精度，并校验 trailing-JSON（拒绝 `{"a":1} garbage`）；`compiler.Draft = jsonschema.Draft7` |
| `cache.go` | `memoryCache`（mutex + map）；`get`/`set` 带 `expiresAt`；`get` 返回 raw 字节副本，避免调用方篡改缓存内容；`Execute()` 仅在 `read && idempotent && ttl>0` 时读写缓存（task 作用域） |
| `runtime.go` | `Runtime` 持有 registry/policy/contextEngine/cache/schemas/`defaultMaxResultBytes`/`now`；`Execute` 顺序严格遵循文档：Check→normalize→input-schema→cache hit 短路→timeout ctx→tool→output-schema→cache-set→`successResult`；`successResult`：`security.RedactPayload`→`contextEngine.ConvertToolResult`→`trimResult`→`data_missing`/`stale` 标记→envelope；`errorResult` 用 `security.RedactText` 红脱 warning；`restoreValue` 经 `CheckpointCodec` 还原具体类型，保证缓存命中后值与首次执行类型一致 |
| `batch.go` | `ExecuteBatch(ctx, []BatchRequest, maxParallel)`：并行条件 `len(requests)>1` 且**每一项** `len(DependsOn)==0 && descriptor.Risk==permission.RiskRead && descriptor.Idempotent`，否则按输入顺序串行；semaphore 限并发 + `ctx` 取消传播；每个子项仍经 `Execute`（含 `Check` 权限门），并行路径不绕过权限 |
| `runtime_test.go` | 10 用例：descriptor metadata / input-schema 执行前拒绝 / output-schema 拒绝 / cache 具体类型恢复 / timeout 结构化 / partial / stale / permission 用注册 Risk / batch 并行-read / batch 串行-write / not-found 分类。全部 `-race` 通过 |

### 3.2 Runtime 集成

| 审查项 | 文件 / 行号 | 结论 |
|--------|-------------|------|
| `runtime_version` 升到 `2.2.0` | `runtime/version.go:9` | ✅ |
| `run_state_version` 升到 `runtime_state_v3` | `runtime/state.go:93` | ✅ |
| `ExecutionStep.ToolTrace *toolruntime.Trace` | `runtime/state.go` | ✅ 与 V2-2 `ContextTrace`/`Evidence` 并列持久化 |
| `setToolTrace()` 辅助 | `runtime/state.go` | ✅ |
| `cfg.ToolRuntime` 注入（nil 时 `toolruntime.New`） | `runtime/runner.go` | ✅ `MaxParallelToolCalls` 默认 4 |
| `Config.ToolRuntime *toolruntime.Runtime` / `Config.MaxParallelToolCalls int` | `runtime/types.go` | ✅ |
| `Observation` 新增 `CacheHit`/`Partial`/`RawSize`/`ContentHash` | `runtime/types.go` | ✅ |
| `parallel_tools` 协议解析（≥2 且各非空） | `runtime/decision.go` | ✅ `toolDecision{Tool,Arguments}`、`decision.Tools []toolDecision` |
| checkpoint 辅助内联 | `runtime/checkpoint.go` | ✅ 移除 `toolCreatesSafeCheckpoint`，逻辑并入 `react_executor.go` |
| `executeTool` 重构为统一层 | `react_executor.go:136` | ✅ `ToolRuntime.Check` 将 `ErrorNotFound`/`ErrorPermission` 映射到 fail 阶段；checkpoint 安全 = `descriptor.Idempotent && descriptor.Risk == permission.RiskRead`（`:165`） |
| `executeParallelTools` 真并发 | `react_executor.go:244` | ✅ 父 `StepParallelTools`；每 child 校验 `read+idempotent+无重名`；`state.ToolCalls += count`（`:303`）在批前计入；`ExecuteBatch`；**整批成功后才 `saveCheckpoint`**（`:372`） |
| Tool Result 消息信封 | `runtime/helpers.go` `buildToolResultMessage` | ✅ payload 改为 `{"tool":envelope.Source,"ok":...,"result":envelope,"evidence":...}`，`envelope` 直接嵌为 `result`；移除 `maxBytes` 入参（裁剪已下沉 toolruntime） |
| 系统 Tool metadata 扩展 | `agent/tools/tool.go:17-19` | ✅ `SourceType`(omitempty)/`ProviderRef`(omitempty)/`CacheTTL`(`json:"-"`，不序列化) |
| 系统 Tool 注册 | `agent/tools/domain/register.go:76` | ✅ `SourceType:"native"`、`CacheTTL:2*time.Second`、`Idempotent:true`、`Timeout`/`MaxResultBytes` 由既有参数传入 |
| 新依赖 `jsonschema/v5 v5.3.1` | `go.mod`/`go.sum` | ✅ Draft7 校验器，符合文档 §JSON Schema |
| 缓存隔离（per-Task） | `agent/manager/manager.go:80` `NewRunner(runtimeConfig)` 每任务调用一次 → `toolruntime.New` 在 `NewRunner` 内创建 → cache 随 Runner 生命周期即 per-Task | ✅ 任务间不串缓存，满足 task-scope TTL 语义 |

### 3.3 版本身份双闸门（Resume 安全）

- `coordinator.go:313`：`item.RuntimeVersion != CurrentVersion || state.Snapshot.Version.RuntimeVersion != CurrentVersion` → 拒绝 resume。
- `checkpoint.go:71`：`state.Version != runStateVersion || !state.ResumeSafe` → 拒绝 resume。
- 旧版 checkpoint（V2-1 `2.1.0`/`runtime_state_v2`、V2-2 同）被 `2.2.0`/`runtime_state_v3` 正确拒绝，符合「升级即不可回放旧状态」预期，无静默兼容风险。

### 3.4 数据库 / Checkpoint 安全

- 无新增表、无新增字段、无 `command/version/2.sql` 之类迁移；复用 `steps_json`/`checkpoint_json`。
- `ToolTrace` 写入 `ExecutionStep`，与 V2-2 `ContextTrace`/`Evidence` 同通道，前端按需展开。
- 并行批 checkpoint 仅在整批成功时保存；单工具安全判定 `Idempotent && Risk==Read`，与 V2-1/V2-2 既有 checkpoint 安全策略一致。

---

## 四、前端 Review

| 文件 | 核对结果 |
|------|----------|
| `src/api/agent.ts` | 新增 `AgentToolTrace` 接口（canonical_name/source_type/risk/idempotent/timeout_ms/cache_ttl_ms/arguments_hash/call_index/call_budget/duration_ms/cache_hit/partial/error_type/raw_size/content_hash/as_of/warnings）；`AgentExecutionStep.tool_trace?` 可选字段。字段名与 `toolruntime.Trace` JSON tag **逐一对齐** |
| `src/views/ai/taskCenter.vue` | 展开列新增 `<template v-if="row.tool_trace">` 块，展示 canonical_name/source_type/risk/call_index-call_budget/cache_hit(partial)/error_type/raw_size + warnings `<pre>`；空态 `v-if` 已并入 `!row.tool_trace`，旧证据块仍正常显示 |
| `locales/zh-CN.yaml` | 新增 8 key：`toolRuntimeTrace`/`toolSourceType`/`toolRisk`/`toolBudget`/`cacheHit`/`partial`/`errorType`/`rawSize`（行 408–415）；既有 `detail.tool: Tool` 存在（行 395），`taskCenter.vue` 引用的 `t('agentTaskCenter.detail.tool')` 有效 |
| `locales/en.yaml` | 同上 8 key（行 408–415）+ 既有 `detail.tool: Tool`（行 395），中英文逐行对齐 |

前端 `vue-tsc --noEmit --skipLibCheck` 退出码 **0**，无类型错误。

---

## 五、发现项

### 阻塞项（0）
无。

### 重要项（0）
无。

### 建议项（3，均不阻塞 Gate）
- **S1（Schema 数字精度）**：`schema.go` 使用 `decoder.UseNumber()`，校验后 `Envelope.Data` 中数字以 `json.Number` 形式存在。下游若直接类型断言 `float64` 会失败。当前 Evidence 转换由 `contextEngine.ConvertToolResult` 处理（其入参为原始 `value`，非 `Data`），风险低；建议确认 Skill 层读取 `envelope.Data` 时不假设 `float64`，或在 `runtime.go` 落 `Data` 前统一 `json.Number→interface` 解码。
- **S2（批路径权限）**：`batch.go` 并行判定只看 `descriptor.Risk==Read`，未显式校验运行时 `AllowedTools` 权限。已确认 `ExecuteBatch` 每个子项仍经 `Runtime.Execute`→`Check()`，权限门不被绕过，故无安全缺口；建议后续在 `ExecuteBatch` 入口处显式 `Check` 一次以提升可读性/防御深度。
- **S3（TTL 来源）**：`register.go:76` 当前硬编码 `CacheTTL:2*time.Second`，与文档 §Cache 一致。若未来不同 Tool 需差异化 TTL，建议从 `Metadata.CacheTTL` 读取（该字段已存在，`json:"-"` 不影响序列化），而非常量化。当前阶段无问题。

### 提示项（2）
- **N1（macOS 链接告警）**：后端 `-race` 测试出现 `ld: warning: ... malformed LC_DYSYMTAB`，为本地 macOS 链接器告警，非测试失败（全部包 `ok`）。CI（Linux）不受影响。
- **N2（前端生产构建）**：本次仅执行 `vue-tsc` 类型检查，未重跑 `vite build` 生成 `static/` 产物（V2-2 已验证构建产物含对应字段）。如需在本次一并刷新产物，可补 `pnpm build`，但不影响 Gate 判定。

---

## 六、测试结论

**后端**
- `go build ./...` → 退出码 0（全模块编译通过）。
- `go test -count=1 -race ./agent/...` → 全部 `ok`，含关键包：
  - `agent/toolruntime`（新包，2.438s）
  - `agent/runtime`（集成，2.325s）
  - `agent/replay`（V2-0 Replay Gate，2.047s，未受影响）
  - `agent/manager`、`agent/scheduler`、`agent/tools/domain` 等
- `go test -count=1 ./models/... ./llm/... ./notify/... ./service/...` → 全部 `ok`。

**前端**
- `vue-tsc --noEmit --skipLibCheck` → 退出码 0。

---

## 七、Gate 判定

V2-3 五项 Gate 全部达成（5/5），无阻塞项、无重要项。新包 `agent/toolruntime` 与文档逐节一致，Runtime 集成、版本身份双闸门、前端 Tool Trace 展示均验证通过，测试全绿。

**结论：V2-3 满足 Gate，可进入 V2-4（Eval/Replay 增强）。**
