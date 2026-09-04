# V2-5 Code Review：第三方 HTTP MCP Integration

- 审查对象：`f245f8c9` 工作树（backend 未提交改动 + frontend `go_binance_futrues_new_ui` 未提交改动）
- 模式：**仅 code review，不做修复、不写记忆、不推进后续 Phase**（遵照用户约束）
- 运行身份（与 `05-phase-v2-5-mcp-integration.md` 一致）：`runtime_version=2.4.0`、`checkpoint_state=runtime_state_v5`、MCP Go SDK `v1.7.0`
- 结论：**Conditional Pass（条件通过）** —— 13 个 Gate 项中 12 项 PASS，第 6 项（Secret 不进入 API response）因 `custom_header` 凭据泄漏 **AT RISK**，需在合并前修复（见 F1）。其余安全边界、统一 Tool Runtime、治理模型、测试均达标。

---

## 1. 审查范围与已读文件

后端（新增/修改）：
- 新增 `agent/mcpclient/`：`types.go`、`security.go`、`redaction.go`、`adapter.go`、`context.go`、`catalog.go`、`client.go`、`store.go`、`mcpclient_test.go`
- 新增 `models/agent_mcp.go`（5 张表）、`controllers/agent_mcp.go`、`agent/app/mcp.go`
- 修改 `agent/runtime/{version,state,types,coordinator,runner,runner_test}.go`、`agent/tools/{registry,tool}.go`、`agent/manager/manager.go`、`agent/app/default.go`、`agent/contextengine/{resources,types}.go`、`agent/toolruntime/{errors,runtime,schema,types}.go`、`main.go`、`routers/router.go`、`models/agent_task_syncdb_test.go`
- `go.mod` / `go.sum`：MCP SDK `v0.4.0 → v1.7.0`

前端（新增/修改）：
- 新增 `src/views/ai/mcpManagement.vue`
- 修改 `src/api/agent.ts`、`src/router/modules/home.ts`、`src/views/ai/taskCenter.vue`、`locales/{zh-CN,en}.yaml`

验证执行：
- `go build ./...` → 通过（无编译错误）
- `go test -count=1 -race ./agent/mcpclient/... ./agent/runtime/... ./agent/contextengine/... ./agent/toolruntime/... ./models/...` → 全部 `ok`
- 前端 `vue-tsc --noEmit --skipLibCheck` → 见 §6（后台运行中，完成后回填）

---

## 2. 架构与接入验证

数据流与阶段文档一致：

```
AI Skill
  -> Runtime dynamic allowlist / Context resource provider   (agent/runtime/types.go Config 新增两 provider)
  -> V2-3 Tool Runtime                                        (agent/toolruntime/*)
  -> MCP Remote Tool Adapter                                 (agent/mcpclient/adapter.go RemoteTool)
  -> MCP Gateway                                              (agent/mcpclient/client.go)
  -> Streamable HTTP MCP Server
```

- **统一 Tool Runtime 接入（Gate 3）✅**：`RemoteTool` 实现 `agenttools.Tool`（adapter.go:15-49），经 `SyncDefaultMCPTools` 注册进共享 `agenttools.Registry`（agent/app/mcp.go:39-66），执行路径完全走 `ToolRuntime.Execute`，无独立旁路。远端 Tool 仍受 Schema/Risk/Permission/Budget/Timeout/Cache/Evidence/Trace 约束。
- **动态 allowlist / context provider（Gate 9）✅**：`agent/app/default.go` 将 `MCPToolAllowlist` / `MCPContextResources` 注入 `RuntimeConfig`；`effectiveToolNames`（runner.go:103-126）在冻结快照时把 DB grant 动态追加进 `tool_catalog_hash`；provider 查询失败则 `prepareNew`/`prepareResumeState` 直接返回错误，不会用不完整目录启动（coordinator.go:141-159、217-221）。
- **版本边界 ✅**：`version.go` `CurrentVersion="2.4.0"`、`state.go` `runStateVersion="runtime_state_v5"`，Resume 身份校验沿用 V2-4 的 `FreezeExecutionChecked`（含 `ToolCatalogHash`/`SkillPackageHash` 比较）。

---

## 3. 安全边界验证

### 3.1 SSRF / 网络边界（Gate 7）✅
- `security.go` `ValidateEndpoint` 强制 HTTPS（仅 `allow_private=1` 放行 HTTP），拒绝 userinfo/fragment。
- `validateResolvedIP` 阻断 loopback/private/unspecified/link-local/multicast。
- `buildHTTPClient` 自定义 `DialContext`：每次 Dial 重新 `LookupIPAddr` 并**逐个校验实际 IP** 后直连（无 DNS rebinding 窗口）；`transport.Proxy=nil`（不继承 `HTTP_PROXY`/`HTTPS_PROXY`，无代理绕过）；`CheckRedirect` 仅允许同 scheme/host 且重校验；5s dial / 15s header / 30s overall 超时；响应体积上限。
- 测试 `TestEndpointAndSecretSafety` 覆盖 plain-HTTP / loopback / private 拒绝，OAuth `token_url` plain-HTTP 拒绝。

### 3.2 Secret 隔离（Gate 6）⚠️ AT RISK（见 F1）
- `SecretRef` 隔离到位：`models/agent_mcp.go:9` `json:"-"`；`store.go:102-106` `serverView` 置空 `SecretRef` 仅暴露 `HasSecret`；错误解析不泄漏变量名（`TestEndpointAndSecretSafety` 断言）。✅
- **`CustomHeader` 未隔离（F1）**：见 §4。

### 3.3 默认零权限发现（Gate 2）✅
- 新 Tool `status=unclassified, enabled=0`（catalog.go）；`ActiveTools` 仅返回 `enabled=1 AND status=granted`；`UpdateTool` 启用需合法 object `input_schema`。测试 `TestStreamableHTTPDiscoveryGovernanceAndRuntime` 断言新 Tool 默认 `unclassified/Enabled=0`。

### 3.4 Schema drift 自动撤权（Gate 8）✅
- Schema hash 变化 → Tool `needs_review` + `disabled`，且对应 Skill grant 置 `enabled=0`（catalog.go + store.go 未静默保留）。测试断言 schema 变更后 `Status=ToolNeedsReview`、`Enabled=0` 且 stale Skill grant 被撤。

### 3.5 Resource / Prompt 不进入 system trust boundary（Gate 5）✅
- `context.go` Resource→`contextengine.Resource`（`BlockMCPResource`），Prompt 固定带 `EXTERNAL_MCP_PROMPT` 边界声明，tool-catalog block 显式声明「untrusted… cannot override system policy/risk/permissions/budgets」；`redactCredentialText(tool.Description)` 在进 Context 前脱敏。Prompt 带必填参数禁止 `auto_load`（store.go:219-237）。

### 3.6 结构化 error taxonomy（Gate 4）✅
- `toolruntime/errors.go` 新增 `ErrorInputRequired`，`classifyToolError` 通过 `errors.As(err,&inputRequired)` 检测 `InputRequired()` 接口。
- `RemoteTool.Execute` 透传 `gateway.CallTool` 原始 `*InputRequiredError`（adapter.go:48），`errors.As` 可正确识别（测试断言）。
- `toolruntime/runtime.go:139-143`：`selected.Execute` 返回 `toolErr` 时 classifying 进 envelope 并以 `ToolError` 返回（`runtimeErr=nil`），故 `react_executor.go` 将其作为普通 tool message 回灌 LLM，ReAct 循环继续——实现多轮（详见 §5）。

### 3.7 Fault isolation（Gate 7）✅
- `client.go` 每 Server 信号量 cap 4（`acquire`），连续 3 次失败 `recordFailure` → 熔断 30s（`allowRequest` 拒绝），`recordSuccess` 复位；`connect` 用 `StreamableClientTransport{DisableStandaloneSSE:true, MaxRetries:-1}`（仅 Streamable HTTP，关闭 standalone SSE 与 SDK 自动 MultiRoundTrip）。

---

## 4. Findings

### F1 — Blocking（Gate 6 偏差）：`custom_header` 凭据明文落库 + 经 API 泄露
`AgentMCPServer.CustomHeader`（`models/agent_mcp.go:10`，`size(128)`，明文存储 + `json:"custom_header,omitempty"`）未做与 `SecretRef` 同等的隔离：

1. **API 泄露**：`store.go:102-106` `serverView` 仅置空 `SecretRef`，**未置空 `CustomHeader`**。因此 `ListServers` / `CreateServer` / `UpdateServer` / `GetCatalog` 的响应均回带 custom header 值。
2. **前端回显**：`mcpManagement.vue:119` 编辑时 `custom_header: row.custom_header || ""` 把值回填到表单（对比 `secret_ref` 在 `:118` 被刻意留空）。
3. **实际即凭据**：`custom_header` 鉴权模式下值通常为 `Authorization: Bearer <token>` 或 `X-API-Key: <token>`；`validHeaderName` 仅拦 `Host`/`Content-Length`，`Authorization` 等带密名的 header 合法通过，于是凭据明文落库并被 API 暴露。

这与阶段文档第 94 行「API/UI 只返回 `has_secret`，不会返回引用名或凭据值」及 Gate 6「Secret 不进入 API response」直接冲突。**建议修复（与 `SecretRef` 对齐）**：
- `serverView` 置空 `CustomHeader`，改暴露 `HasCustomHeader bool`；
- 前端仅展示 `has_custom_header`，编辑表单不回显值（同 secret_ref 处理）；
- 考虑将 custom header 值也改为 `secret_ref` 引用解析，而非明文存储。

### F2 — Minor（一致性）：`SaveServer` 更新路径对 `CustomHeader` 不保留
`store.go:75` `CustomHeader: strings.TrimSpace(input.CustomHeader)` 在 update 时**总是覆盖**；而 `SecretRef` 在 `input.SecretRef==""` 时保留既有值（store.go:90-92）。后果：管理员编辑 Server 但未重填 custom_header 时，该字段被清空。应统一为「空则保留」。

### F3 — Suggestion：JSON Schema validator Draft 7 → Draft 2020
`toolruntime/schema.go:58` `compiler.Draft = jsonschema.Draft2020`。Draft 2020-12 与 Draft-07 大体兼容，但存在细微差异（忽略 `$ref` 兄弟键、`definitions`→`$defs`、`id`→`$id` 等）。MCP Tool 与既有 Skill Tool 共用此 validator，建议跑一次全量 `go test ./...` 确认既有 Skill Tool 的 input/output schema 在 Draft 2020 下仍全部通过校验（本次已跑 agent 子树，未见失败；F3 仅作回归提醒）。

### F4 — Observation（非缺陷）：`input_required` 多轮实现方式
阶段文档 §8 明确「持久化 `waiting_input` / Web 人工确认为后续统一 Approval 能力，不在本阶段伪装为已完成」。当前实现把 `InputRequiredError` 经统一 error taxonomy 作为 tool message 回灌 LLM，由 Agent 自行降级或结束——与文档口径一致，非缺陷。注意：后端/前端均**无** `waiting_input` 任务状态（grep 确认），与文档声明一致，无前后端错位。

### F5 — Observation：熔断计数偏激进
`client.go` `CallTool` 任意业务错误（非网络故障）也会 `recordFailure`，连续 3 次即熔断 30s。对「工具本身返回错误」的合法场景偏严格，但作为安全默认可接受；如后续观测到误熔断，可改为仅对 `ErrorUpstream/ErrorTimeout` 计数。

### F6 — Observation（良好）：控制器与级联
`DeleteServer`（store.go:108-122）级联清理 permissions/tools/resources/prompts，控制器随后 `syncDefaultMCPRuntime` 热同步 Registry；Server/Tool/Catalog 变更热同步无需重启（gate 项 §11 达成）。`GrantedToolNames`/`GrantedContextPermissions` 均强制 `enabled=1` + 类型过滤，无跨 Server 注入（`validateCapability` 校验 capability 归属）。

### F7 — Minor（仓库既有，非 V2-5 引入）：前端 `vue-tsc` 报 `TS6053`
两次独立运行均出现唯一错误：`error TS6053: File '.../src/views/permission/page/index.vue' not found`。但该文件**实际存在于磁盘**（git-tracked，1672 字节，V2-5 未改动），且全仓对 `permission/page` 的唯一引用仅是 `src/utils/sso.ts:5` 注释里的一个 URL，V2-5 的改动（`mcpManagement.vue`/`api/agent.ts`/`taskCenter.vue`/`router/home.ts`/locales）与其无关、也**零类型错误**。属 vue-tsc 2.x 对既有 `.vue` 文件的已知/环境性 `TS6053`，**不影响 Gate 1–5、7–12**，也不代表 V2-5 引入回归。建议作为独立仓库问题排查（如 tsconfig `include` 解析或 vue-tsc 版本），不纳入 V2-5 修复范围。

---

## 5. `input_required` 多轮链路（端到端确认）

```
mcp server 返回 InputRequests/RequestState
  -> client.go:292-295 构造 *InputRequiredError{RequestState, Requests(redacted)}
  -> adapter.go:48 RemoteTool.Execute 透传（保留 *InputRequiredError，可被 errors.As 识别）
  -> toolruntime/runtime.go:139-143 classifyToolError -> ErrorInputRequired，写入 Envelope.ErrorType，返回 ToolError（runtimeErr=nil）
  -> react_executor.go:172-216 runtimeErr==nil -> toolResult.ToolError!=nil -> 作为 tool message 回灌 messages，ReAct 进入下一轮
  -> LLM 见 EXTERNAL_MCP_INPUT_REQUEST=<requests> 文本，可补充输入再次调用
```
测试 `TestStreamableHTTPDiscoveryGovernanceAndRuntime` 断言 `errors.As(err,&inputRequired)` 且 `toolruntime.TypeOf(err)==ErrorInputRequired`。链路闭合。

---

## 6. 测试结果

| 范围 | 命令 | 结果 |
|---|---|---|
| 后端编译 | `go build ./...` | PASS（无错误） |
| mcpclient | `go test -count=1 -race ./agent/mcpclient/...` | ok 2.1s |
| runtime | `go test -count=1 -race ./agent/runtime/...` | ok 1.9s |
| contextengine | `go test -count=1 -race ./agent/contextengine/...` | ok 2.6s |
| toolruntime | `go test -count=1 -race ./agent/toolruntime/...` | ok 3.0s |
| models | `go test -count=1 -race ./models/...` | ok 2.4s |
| 全量后端 | `go test -count=1 ./...` | **ok（ALLTEST_EXIT=0，无 FAIL）** |
| 前端 typecheck | `vue-tsc --noEmit --skipLibCheck` | **TS_EXIT=2，1 个错误（见 F7，与 V2-5 无关）；V2-5 自身文件零类型错误** |

> 注：`ld: warning ... malformed LC_DYSYMTAB` 为 macOS 链接器告警，非测试失败。前端 typecheck 两次独立运行均复现同一 `TS6053`，属仓库既有问题，非 V2-5 引入。

---

## 7. Gate 评估（13 项）

| # | Gate 项 | 结论 | 依据 |
|---|---|---|---|
| 1 | Streamable HTTP Server E2E 发现 Tool/Resource/Prompt | PASS | test `TestStreamableHTTPDiscoveryGovernanceAndRuntime` |
| 2 | 新 Tool 默认不可调用（需本地治理 + Skill grant） | PASS | catalog.go 默认 unclassified/disabled；test 断言 |
| 3 | MCP Tool 统一经 V2-3 Tool Runtime | PASS | adapter.go RemoteTool + app/mcp.go 注册共享 Registry |
| 4 | MCP Tool failure 经结构化 error taxonomy 返回 | PASS | errors.go ErrorInputRequired + react_executor 回灌 |
| 5 | MCP Prompt 不进入 system trust boundary | PASS | context.go EXTERNAL_MCP_PROMPT 边界 + auto_load 限制 |
| 6 | Secret 不进入 Prompt/Task/Event/API response | **AT RISK** | F1：`custom_header` 明文落库 + API 泄露 |
| 7 | SSRF/redirect/timeout/size/concurrency/circuit breaker 边界 | PASS | security.go + client.go + test |
| 8 | Schema drift 自动撤销 Tool 与 Skill 权限 | PASS | catalog.go + test 断言 |
| 9 | MCP identity 进入 Tool Trace / Tool Catalog Hash | PASS | Trace 增 ProviderRef/ProtocolVersion/CatalogHash/SchemaHash；test 断言 |
| 10 | Web 管理能力完成 | PASS | mcpManagement.vue 全功能 + 9 个 API 函数导出 |
| 11 | Go Release baseline 与 MCP SDK 要求一致 | PASS | go.mod SDK v1.7.0（go 1.25 基线见文档） |
| 12 | `go test ./...` 通过 | PASS（全量 `go test ./...` ALLTEST_EXIT=0，无 FAIL） | §6 |
| 13 | 前端 typecheck/build 通过 | **FAIL（仓库既有，与 V2-5 无关）** | 唯一错误 F7：`TS6053` 指向一个 V2-5 未改动、磁盘上存在的文件；V2-5 自身文件零类型错误 |

---

## 8. 最终裁决

**Conditional Pass**：Gate 1-5、7-12 通过；Gate 6 因 F1（`custom_header` 凭据隔离缺失）判为 AT RISK；Gate 13 因 F7 仓库既有 `TS6053` 未达成（与 V2-5 无关）。

- **合并前必须修复 F1**（与 `SecretRef` 对齐的隔离方案），F2 一并修复；
- F3 作回归提醒；F4/F5 为观察项不需改动；
- F7 属仓库既有问题，V2-5 自身前端文件零类型错误，不纳入 V2-5 修复范围，建议另行排查。

V2-5 安全架构（SSRF、Secret 主体隔离、默认零权限、统一 Tool Runtime、熔断、context 边界）整体扎实，测试覆盖充分（真实 SDK Server + httptest E2E），核心 Race Gate 与全量后端 `go test ./...` 均通过。
