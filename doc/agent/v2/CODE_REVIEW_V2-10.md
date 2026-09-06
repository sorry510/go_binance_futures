# Code Review — V2-10 Observability、Trace 与运营页面

> 评审模式：**review-only**（仅 review，未修改任何代码、未运行任何破坏性命令）。
> 评审时间：2026-09-05
> 评审范围：`git diff --stat`（工作树 vs HEAD）锁定的 V2-10 后端新增/修改文件（不含 `static/` 前端构建产物）。V2-7/8/9 已在 HEAD，本仓未改动。

---

## 0. 结论（Gate）

**AUTOMATED PASS**（构建 / 测试 / vet 全绿，四项验收逐条满足，安全边界成立）。

**人工验收待定**（见 §7）：前端 `AI -> 可观测性` 页面在独立仓库 `go_binance_futrues_new_ui`，需用户侧确认与 `/agents/observability/{summary,traces,changes}` 接口字段对齐及 24h/7d/30d 概览联调。

阻塞级缺陷：**无**。条件级（非阻塞）建议：见 §6。

---

## 1. 构建 / 测试结果

| 项目 | 命令 | 结果 |
|------|------|------|
| 全量构建 | `go build ./...` | ✅ 通过（exit 0） |
| 影响包测试 | `go test -count=1 ./agent/observability/... ./agent/runtime/... ./agent/eval/... ./agent/mcpclient/... ./agent/portableskill/... ./controllers/... ./models/...` | ✅ 全部 ok |
| 全量测试 | `go test -count=1 ./...`（仅显示非 ok 行） | ✅ 无失败（无回归，60+ 包全绿） |
| 静态检查 | `go vet ./agent/observability/... ./agent/runtime/... ./agent/eval/... ./agent/mcpclient/... ./agent/portableskill/... ./controllers/... ./models/... ./agent/app/...` | ✅ 无告警 |
| 验证约束 | 全程使用 `go test -count=1` | ✅ 已规避缓存 |

> 前端 `static/` 重建产物为独立 UI 仓库构建输出，非本仓手改，不计入 review 范围（既有 `src/views/permission/page/index.vue not found`(TS6053) 为历史遗留，非本 Phase 引入）。

---

## 2. 变更清单（范围锁定）

**新增后端文件（6）**
- `models/agent_observation.go` — `AgentObservation`（表 `agent_observations`）：Task/Context/LLM/Tool/Validation/Repair/Eval 节点 Trace 全字段。
- `models/agent_change_event.go` — `AgentChangeEvent`（表 `agent_change_events`）：MCP Catalog/Protocol/Tool Schema、Skill import/activate/rollback/validation failure 历史。
- `agent/observability/store.go` — `Store`：InsertObservation / ListTraces / RecordChange / ListChanges + 持久化辅助（`persistObservation` 带 `recover()` 异步兜底）。
- `agent/observability/summary.go` — `Summary`：时间窗口聚合 Skill/Model/Prompt/SkillRevision 成功率与 Token 成本代理、Context 裁剪与 Memory 命中、Tool/MCP latency/cache/partial/timeout、Evidence/Repair/Eval、ChangeEvents。
- `agent/observability/store_test.go` — 2 个单测（Trace+Summary 聚合、Change 脱敏）。
- `controllers/agent_observability.go` — `AgentObservabilityController`：Summary / Traces / Changes。

**修改后端文件（V2-10 diff，13）**
- `agent/runtime/types.go`：`Observation` 结构体扩展 StepID/StepType/ToolSource/ProviderRef/ProtocolVersion/CatalogHash/SchemaHash/ContextTokens/ContextBlocks/TrimmedBlocks/MemorySelected/MemoryTrimmed/EvidenceCount/EvalCase/EvalScore。
- `agent/runtime/helpers.go`：`generateWithRetry` 的 llm_call 观察补齐 StepID/StepType。
- `agent/runtime/react_executor.go`：context_build / tool_call（串行+并行）/ validation 观察补齐 Step 维度与 ToolSource/ProviderRef/Protocol/Catalog/Schema/EvidenceCount。
- `agent/observability/collector.go`：`defaultCollector.persist=true`；`Observe` 在锁外调用 `persistObservation`（best-effort），故障不反向影响主任务。
- `agent/eval/runner.go`：`Evaluate` 末尾 `observability.RecordEval` 落 eval 观察。
- `agent/mcpclient/catalog.go`：catalog refresh / schema 变化记录 `ChangeEvent`（含 `review_required` 的 schema_changed）。
- `agent/portableskill/store.go`：Install（import）、Activate（activate/rollback，按 CreatedAt 判定）记录 `ChangeEvent`。
- `controllers/agent_skill_portable.go`：Import / ImportDirectory 校验失败记录 `import_validation_failed`。
- `main.go`：`orm.RegisterModel(new(AgentObservation), new(AgentChangeEvent))`。
- `models/agent_task_syncdb_test.go`：legacy DDL 增两表 + 列断言。
- `routers/router.go`：新增 3 条 `/agents/observability/*` 路由。
- `doc/agent/v2/10-phase-v2-10-observability.md` / `doc/agent/v2/README.md`：验收勾选 + V2-10 标 ✅。

> 注：`agent/runtime/observe.go`（observe 方法）与 `agent/app/default.go:73 Observer: observability.Default()` 在本 Phase diff 之外已存在（属既有 Observer 机制），V2-10 启用 `persist=true` 完成落库闭环。

---

## 3. 路由核对（routers/router.go diff 实读）

```
GET /agents/observability/summary  -> Summary   // 长期运营指标
GET /agents/observability/traces   -> Traces    // 持久化节点 Trace
GET /agents/observability/changes   -> Changes   // MCP/Skill 变更历史
```
控制器方法名与路由动词一致；参数通过 `queryInt64` / `queryInt` 解析（`page` 默认 1、`limit` 默认 20）。store 层 `normalizePage` 进一步将 `limit` 夹在 [1,200]，查询安全。

---

## 4. Phase 验收逐条核对（对照 10-phase-v2-10-observability.md）

### 验收① 单个失败 Task 可定位到具体 Step/Provider/Tool/MCP — ✅
- `AgentObservation` 含 `task_id/skill/step_id/step_type/provider/model/tool/tool_source/provider_ref/error_type/error`；`Traces` 支持按 `task_id/skill/type/status/tool_source` + 时间窗过滤。
- 运行时在 `context_build`/`llm_call`/`tool_call`(串行+并行)/`validation` 各节点均发出带 `StepID`/`StepType` 的观察（react_executor.go + helpers.go 实读确认）。
- `providerRefID("mcp-server:<id>")` 与 `adapter.go:36` 的 `ProviderRef: "mcp-server:"+server.ID` 格式一致，MCP 调用可精确归因为具体 Server。

### 验收② MCP 与 Skill revision 变化有历史记录 — ✅
- `agent_change_events` 记录：MCP catalog_refresh/catalog_changed/schema_changed（mcpclient/catalog.go）、Skill import/activate/rollback（portableskill/store.go）、import_validation_failed（controller）。
- `schema_changed` 标记 `Status:"review_required"`，与「schema 变化需人工复核」的安全意图一致。
- `Changes` 端点支持按 `category/entity_type/entity_name/change_type/status` + 时间窗查询。

### 验收③ 运营页面不泄漏 Secret — ✅（源码实读 + 单测实证）
- `store.go` 落库时对 `Error`/`ProviderRef`/`DetailJSON`/`EntityName` 统一经 `security.RedactText` 脱敏（`observationModel` + `RecordChange`）。
- `eval` 观察的 `report.Error` 经 `security.RedactText(value.Error)` 脱敏。
- 仅持久化脱敏错误与元数据：**不保存 Tool 参数、原始 Tool 结果、Evidence 原文**（Observation 仅记 `EvidenceCount`，源码确认未存 Evidence body）。
- 单测 `TestChangeHistoryPersistsRedactedDetail` 实证 `DetailJSON` 中 `Bearer secret-value` 被脱敏，不落库明文。

### 验收④ AI 子系统故障与交易主循环指标分离 — ✅
- `agent_observations` / `agent_change_events` 为独立表，与交易主循环表（positions/orders 等）无耦合；`Summary` 只读 `agent_tasks`/`agent_observations`/`agent_change_events`/`agent_mcp_servers`（AI 子系统表）。
- 观察持久化经 `persistObservation`（带 `recover()`）且 `Collector.Observe` 在锁外 best-effort 调用，审计失败不反向影响 Agent 主任务（与 phase doc「审计失败采用 best-effort」一致）。

---

## 5. 安全 / 权限 / 审计

### 5.1 脱敏闭环 — ✅ 成立
- Observation.Error / ProviderRef / ChangeEntityName / ChangeDetailJSON 四处均经 `security.RedactText`，已在源码确认；变更事件脱敏有单测断言级覆盖。

### 5.2 MCP Server 归因正确性 — ✅ 成立
- `summary.go:providerRefID` 解析 `mcp-server:<id>`；`mcpclient/adapter.go:36` 写 `ProviderRef: "mcp-server:"+server.ID`；`portableskill` 写 `skill-version:<id>`（不被 MCP 聚合误吸）。格式两端对齐，归因精确。

### 5.3 故障隔离（best-effort）— ✅ 成立
- `persistObservation` / `RecordChange`（包级函数）均 `defer recover()`，DB 异常不会 panic 主流程。
- `Collector.Observe` 把 `persistObservation(context.Background(), value)` 放在 `mu.Unlock()` 之后，避免持锁期间 DB I/O 阻塞聚合。

### 5.4 写入链路贯通 — ✅ 成立（实读确认）
- `default.go:73 Observer: observability.Default()`（persist=true）；`controllers/strategy_template_ai_task.go:264` 的 strategy builder runner 同样 `Observer: observability.Default()`。
- 两条 Agent 运行时入口均接通 Observer → Collector.Observe → persistObservation → `agent_observations`，**无 Trace 覆盖盲区**。

### 5.5 DB 迁移兼容 — ✅ 成立
- `main.go` 注册两新模型；`syncdb_test.go` 断言 `agent_observations`/`agent_change_events` 表存在且列齐备、legacy 行不变；`go test ./models/...` 通过。

---

## 6. 非阻塞建议（Conditional / 可选改进）

1. **`Summary` 全量加载**：`summary.go` 对窗口内 `agent_tasks` 与 `agent_observations` 均用 `All()` 全量载入内存后聚合。30d 窗口 + 高频任务下内存峰值较高；若后续数据量大，建议改为按维度 `GROUP BY` 的 SQL 聚合或游标分批。当前规模可接受，非阻塞。
2. **每步同步落库**：每个 observation 在 `Collector.Observe` 内同步 `Insert`（带 recover）。极高吞吐下增加主库写压力；可考虑批量/异步 flush。当前 best-effort 设计已满足可用性，非阻塞。
3. **聚合锁竞争**：`Collector.Observe` 在 `mu` 下做 map 累加，高频观察会有互斥开销；若需更高并发，可对维度分片锁。非阻塞。
4. **路由注释残留（纯 cosmetic）**：`routers/router.go` 中 `/agents/observability/changes` 行尾残留了原 governance 行的注释 `// 查询 Agent 权限、预算和运行指标`（被拼接到本行注释后）。不影响编译/路由，建议顺手清理。
5. **`Summary` 缺 `errors` 类型外的失败细分**：`EvalAggregate`/`EvidenceAggregate`/`Repairs` 已覆盖，但 `BySkill` 等维度的 `Failed` 用 `default` 分支兜底（非 succeeded/cancelled 即计 failed），若后续新增中间态需注意归类。非阻塞。

---

## 7. 未验证项 / 人工验收待办

- ❏ **前端联调**：`AI -> 可观测性` 页面（24h/7d/30d 概览、Task Trace、MCP/Skill 变更历史）在独立仓库 `go_binance_futrues_new_ui`，本仓 review 不含。需确认与接口字段对齐：
  - `GET /agents/observability/summary?start_time=&end_time=` → `Summary{Global,BySkill,ByModel,ByPrompt,BySkillRevision,Context,Tools,MCPServers,Repairs,Errors,Evidence,Eval,ChangeEvents}`。
  - `GET /agents/observability/traces?task_id=&skill=&type=&status=&tool_source=&start_time=&end_time=&page=&limit=` → `TraceListResult{List[]AgentObservation}`。
  - `GET /agents/observability/changes?category=&entity_type=&entity_name=&change_type=&status=&start_time=&end_time=&page=&limit=` → `ChangeListResult{List[]AgentChangeEvent}`。
- ❏ **Change 脱敏单测覆盖**：`RecordChange` 的 `EntityName`/`ProviderRef` 脱敏仅源码确认、缺断言级单测（DetailJSON 已有 `TestChangeHistoryPersistsRedactedDetail`）。建议补一条断言 `EntityName`/敏感 `ProviderRef` 脱敏。
- ❏ **MCP Server 聚合联调**：`providerRefID` 与 `adapter.go` 格式对齐已源码确认，但缺一条集成测试实证 `tool_call(ProviderRef=mcp-server:N)` 在 `Summary.MCPServers[N]` 正确累加 calls/errors/availability。
- ❏ **`Summary` 边界**：`start_time > end_time` 返回错误（validateWindow）；但 `end_time` 仅 `<=0` 时回退 now，若传超未来时间不影响正确性，仅空结果。建议前端约束。

---

## 8. 评审签名

- 模式：review-only（零代码改动、零内存写入、零提问）
- 自动化结论：**AUTOMATED PASS**
- 人工验收：**待定**（见 §7，主要为前端联调 + 2 条可选补测）
- 阻塞项：**无** → 满足进入 V2-11 的 Gate 条件。
