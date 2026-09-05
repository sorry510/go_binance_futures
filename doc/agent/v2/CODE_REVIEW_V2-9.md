# Code Review — V2-9 长期 Memory 与历史知识管理

> 评审模式：**review-only**（仅 review，未修改任何代码、未运行任何破坏性命令）。
> 评审时间：2026-09-02（续做）
> 评审范围：`git diff --stat` 锁定的 V2-9 后端新增/修改文件（不含 `static/` 前端构建产物）。

---

## 0. 结论（Gate）

**AUTOMATED PASS**（构建 / 测试 / vet 全绿，四项验收逐条满足，安全边界成立）。

**人工验收待定**（见 §7 待办）：前端 UI 未在本仓（在 `go_binance_futrues_new_ui` 独立仓库），需用户在另一端确认 Memory 管理页面与 `/agents/memories` 接口联调；其余后端项可进入下一 Phase。

阻塞级缺陷：**无**。条件级（非阻塞）建议：见 §6。

---

## 1. 构建 / 测试结果

| 项目 | 命令 | 结果 |
|------|------|------|
| 全量构建 | `go build ./...` | ✅ 通过（exit 0） |
| 影响包测试 | `go test -count=1 ./agent/memory/... ./agent/runtime/... ./agent/contextengine/... ./agent/app/... ./controllers/... ./models/...` | ✅ 全部 ok |
| 全量测试 | `go test -count=1 ./...` | ✅ 全部 ok（无回归，60+ 包通过） |
| 静态检查 | `go vet ./agent/memory/... ./agent/runtime/... ./agent/contextengine/... ./agent/app/... ./controllers/... ./models/...` | ✅ 无告警 |
| 验证约束 | 全程使用 `go test -count=1`（命中 `go test` 缓存已规避） | ✅ |

> 注：前端 `static/` 重建产物为前端仓库 `go_binance_futrues_new_ui` 的构建输出，非本仓手改，不计入 review 范围；其既有 `src/views/permission/page/index.vue not found`(TS6053) 为历史遗留、非本 Phase 引入。

---

## 2. 变更清单（范围锁定）

**新增后端文件（7）**
- `agent/memory/types.go` — Type/Status 常量、Scope、CreateInput/UpdateInput/ListOptions/ListResult/QueryScope。
- `agent/memory/scope.go` — `ScopeFromRequest` 从 req.Metadata + req.Input 提取 scope；符号大写、去空格归一。
- `agent/memory/service.go` — `Service`、`Context`、`PersistTaskSummary`、`extractSummary`、`ValidateAutomaticWrite`。
- `agent/memory/store.go` — `ORMStore`：CRUD、List、Query（交集匹配）、Expire、ContextBlocks。
- `agent/app/memory.go` — `defaultMemoryService` 接线 `MemoryContext` / `MemoryWrite`。
- `controllers/agent_memory.go` — `AgentMemoryController`：List/Create/Update/Delete/Disable/Enable/Approve。
- `models/agent_memory.go` — `AgentMemory`（表 `agent_memories`）。

**修改后端文件（13，含路由/集成/测试）**
- `agent/app/default.go`：`Config.MemoryContextProvider` / `MemoryWriter` 接线。
- `agent/runtime/types.go`：新增 `MemoryContextProvider` / `MemoryWriter` 类型与 Config 字段。
- `agent/runtime/helpers.go`：新增 `audit()`（写 `task.Event` + Save + EventHook）。
- `agent/runtime/react_executor.go`：首轮 `memory_read` 审计；`executeFinal` 末尾 `memory_write` 审计（错误非致命）。
- `agent/runtime/coordinator.go`：`buildContext` 注入 Memory 块（失败降级、仅 audit 不 fail）；step 含 `memoryCount`。
- `agent/contextengine/types.go`：`BuildTrace` 增 `SelectedMemoryIDs` / `TrimmedMemoryIDs`。
- `agent/contextengine/builder.go`：去重/裁剪时对 `BlockMemory` 分别计入 `TrimmedMemoryIDs` / `SelectedMemoryIDs`。
- `main.go`：`orm.RegisterModel(new(models.AgentMemory))`。
- `models/agent_task_syncdb_test.go`：legacy DDL 增 `AgentMemory` 断言。
- `controllers/strategy_template_ai_task.go`：strategy builder runner 接入 Memory 注入/写入。
- `controllers/strategy_template_ai_runtime_test.go`：测试中将 Memory 接入置 nil 隔离。
- `routers/router.go`：新增 6 条 `/agents/memories/*` 路由（见 §3）。
- `agent/runtime/runner_test.go`：新增 2 个 Memory 审计/降级测试（见 §5）。
- `agent/memory/store_test.go`：新增 3 个 Memory 单测（见 §5）。
- `doc/agent/v2/09-phase-v2-9-memory.md` / `doc/agent/v2/README.md`：验收勾选 + V2-9 标 ✅。

---

## 3. 路由核对（routers/router.go diff 实读）

`git diff` 确认新增：
```
GET   /agents/memories            -> List
POST  /agents/memories            -> Create
PUT   /agents/memories/:id        -> Update
DELETE/agents/memories/:id        -> Delete
POST  /agents/memories/:id/disable-> Disable
POST  /agents/memories/:id/enable -> Enable
POST  /agents/memories/:id/approve-> Approve
```
`:id` 解析在 `controllers/agent_memory.go::memoryID()` 中做 `ParseInt` 且 `id <= 0` 报 400，路由与控制器一致，无遗漏/错挂。

---

## 4. Phase 验收逐条核对（对照 09-phase-v2-9-memory.md）

### 验收① Conversation / Memory / Task 三者生命周期清晰分离
- ✅ 独立表 `agent_memories` / `agent_conversations` / `agent_tasks`，模型各自独立（`models/agent_memory.go`）。
- ✅ Memory 不依赖 Conversation：scope 由 `user/skill/symbol/strategy` 驱动，与对话轮次解耦。
- ✅ Runtime 仅 `MemoryContextProvider`（读，注入上下文）与 `MemoryWriter`（写，终轮兜底）两个显式接入点，职责清晰。

### 验收② 市场记忆自动过期（market_hypothesis 强制 TTL）
- ✅ `DefaultMarketHypothesisTTL = 6h`（`types.go`）；`Create` 对 `market_hypothesis` 校验 `ExpiresAt` 必须为未来时间（`store.go` normalizeCreate）。
- ✅ `Expire()` 将到期 `active/candidate/disabled` 收敛为 `expired`；`List`/`Query`/`ContextBlocks` 访问时先 `Expire()`，默认排除 expired。
- ✅ 单测 `TestMarketHypothesisExpiresAndIsExcludedFromContext` 实证：过期后 `ContextBlocks` 返回 0 条，不泄漏。

### 验收③ Memory 读写进入 Trace
- ✅ `BuildTrace` 新增 `SelectedMemoryIDs` / `TrimmedMemoryIDs`（`contextengine/types.go` + `builder.go`）。
- ✅ `react_executor.go`：首轮选中 Memory 时 `audit("memory_read","success",...)`；终轮 `MemoryWriter` 调用后 `audit("memory_write",...)`；coordinator 读取失败 `audit("memory_read","error",...)` 但不 fail 任务。
- ✅ Task Event 新增 `memory_read` / `memory_write` stage；step 消息含 `memoryCount`。
- ✅ 单测 `TestRunnerTracesLongTermMemoryReadAndWrite` 实证 SelectedMemoryIDs / 两个 audit event 均落库；`TestRunnerDegradesWhenLongTermMemoryIsUnavailable` 实证 Memory 不可用时任务仍 Succeeded 且记 `memory_read/error`。

### 验收④ 用户可管理 Memory（增删改 + 审批 + 禁用/启用 + 查看过期）
- ✅ 控制器覆盖 List/Create/Update/Delete/Disable/Enable/Approve。
- ✅ `Create` 以 `candidate` 标志 → `StatusCandidate`（否则 `StatusActive`）；`Approve`/`Enable` → `StatusActive`；`Disable` → `StatusDisabled`；`List` 支持 `include_expired=1` 查看过期。
- ✅ `SetStatus` 守卫「过期不可启用」（`store.go`），避免复活已过期记忆。
- ⚠️ 详见 §6：控制器层与其它 `/agents/*` 一致，未新增端点级 RBAC（沿用现有鉴权中间件），属既定架构取舍，非缺陷。

---

## 5. 安全 / 权限 / 审计

### 5.1 自动写入权限边界（核心安全设计）— ✅ 成立
- `ValidateAutomaticWrite`（service.go:79）仅放行 `task_summary` / `lesson` 自动写；
  `strategy_fact` / `market_hypothesis` / `user_preference` 一律拒绝自动永久保存，强制走 candidate 或人工审批。
- `PersistTaskSummary`（MemoryWrite 后端）调用前置校验，且仅在 `task.Status == Succeeded` 时写入。
- **关键边界正确**：`ValidateAutomaticWrite` 限制的是 *Runtime 自动写* 路径；通过 `/agents/memories` API 由用户显式创建的 `active` 高风险记忆是合规的（人写 ≠ 机写）。设计与权限意图一致，未越权放行自动保存。
- 单测 `TestTaskSummaryWriteIsSafeAndIdempotent` 实证 `ValidateAutomaticWrite(strategy_fact)` / `(market_hypothesis)` 均返回 error。

### 5.2 脱敏 + 完整性 — ✅ 成立（源码实读确认）
- `store.go:31` 与 `store.go:63`：`security.RedactText(input.Content)` 对 Create / Update 内容脱敏后再落库。
- `store.go:283` `hashContent`：`sha256.Sum256([]byte(strings.TrimSpace(content)))` 生成 `ContentHash`，Create/Update 均回填；`ContextBlocks` 也透出 `ContentHash`，便于下游校验完整性。

### 5.3 Scope 交集匹配 — ✅ 正确（单测实证）
- `Query`：Memory 某维度为空 → 匹配所有；非空 → 必须相等（`scope_user__in "",scope.User` 等）。
- 单测 `TestQueryMatchesAllNonEmptyScopes`：User+Skill+Symbol 组合正确命中 2 条、排除无关 symbol，交集语义成立。

### 5.4 注入上下文优先级 — ✅ 正确
- `coordinator.buildContext`：Memory 块位于 `ConversationHistory` 之后、`InitialMessageBlocks` 之前；`ContextBlocks` 构造 `BlockMemory` 时 `reference_only=true`，优先级低于当前事实 / Task / Skill 指令（与 09-phase doc 约定一致）。

### 5.5 审计非致命 — ✅ 健壮
- `react_executor.executeFinal`：`MemoryWriter` 出错仅 `audit("memory_write","error",...)`，不中断最终响应（非致命），避免记忆写入故障拖垮任务产出。

---

## 6. 非阻塞建议（Conditional / 可选改进）

1. **控制器鉴权一致性**：`AgentMemoryController` 未新增端点级 RBAC，与既有 `/agents/*` 控制器一致（依赖统一中间件）。如需对 `Approve`/`Disable` 等敏感操作加细粒度权限，建议在鉴权中间件层统一处理，而非在控制器内散点。非阻塞。
2. **`Create` 直接生成 `active` 高风险记忆**：API 允许 `candidate=false` 时直接落 `active` 的 `strategy_fact`/`market_hypothesis`。这是「人写即授权」的有意设计，但建议前端在创建高风险类型时默认勾选 `candidate`、引导走审批流，降低误永久化风险（前端侧事项，待 §7 联调确认）。
3. **`Approve` 与 `Enable` 同归 `StatusActive`**：行为正确；如需区分「首次审批」与「重新启用」的审计语义，可在 Event message 中带上来源动作，便于运维追溯。非阻塞。
4. **`Context` 上限 20 条**：`Service.Context` 固定 `Limit:20`，对高密度 scope 可能产生截断；`builder.go` 已将截断的 Memory 记入 `TrimmedMemoryIDs`，可追溯。若后续出现记忆覆盖不足，可考虑按 `confidence`/新鲜度排序后再截断。非阻塞。
5. **`Expire` 触发时机**：仅在 `List`/`Query`/`ContextBlocks` 访问时惰性收敛；若需保证 `expired` 及时可见（如管理列表实时性），可补充定时任务主动 `Expire`。非阻塞，当前实现已满足验收。

---

## 7. 未验证项 / 人工验收待办

- ❏ **前端联调**：Memory 管理页（列表/创建/审批/禁用启用/查看过期）在独立仓库 `go_binance_futrues_new_ui`，本仓 review 不含。需用户在 UI 仓库确认与 `/agents/memories/*` 接口字段对齐（请求体 `memoryCreateRequest`：`type/scope/confidence/content/expires_at/candidate`；列表查询参数 `type/status/user/skill/symbol/strategy/source_task_id/include_expired`）。
- ❏ **`SetStatus` 过期守卫**（store.go）建议补一条单测：对 `expired` 记忆调用 `Enable`/`Approve` 应返回 error；当前仅由代码逻辑保证，缺回归保护。
- ❏ **脱敏单测**：`security.RedactText` 是否覆盖预期敏感模式，建议补一条 `Create` 后断言 `Content` 已脱敏的测试（当前仅源码确认调用，无断言级覆盖）。
- ❏ **`Disable`/`Enable` 控制器路径**无独立单测（仅 store 层 `SetStatus` 间接覆盖）；建议补控制器层集成测试。

---

## 8. 评审签名

- 模式：review-only（零代码改动、零内存写入、零提问）
- 自动化结论：**AUTOMATED PASS**
- 人工验收：**待定**（见 §7，主要为前端联调 + 2 条可选补测）
- 阻塞项：**无** → 满足进入 V2-10 的 Gate 条件。
