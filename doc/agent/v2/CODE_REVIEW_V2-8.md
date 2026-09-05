# CODE REVIEW — V2-8 Model Gateway、Capability 与 Router

- **Review date**: 2026-09-05
- **Reviewer**: 自动化 Code Review（仅 review，未修改任何源码、未提问）
- **Scope**: `git status --porcelain` / `git diff --stat` 锁定的 V2-8 改动
- **Phase doc**: `doc/agent/v2/08-phase-v2-8-model-gateway.md`

---

## 1. 结论（Gate）

```
go build ./...                                                        PASS (exit 0)
go test -count=1 -race ./agent/modelgateway ./llm ./agent/manager ./agent/runtime ./agent/task ./models   PASS
go test -count=1 ./...                                                 PASS (全部包 ok)
go vet <changed packages>                                             PASS (exit 0)
```

**AUTOMATED PASS — 人工验收待定（CONDITIONAL）**

后端实现与自动化测试全部通过；对照 Phase doc 四项验收逐条核对，核心契约（Router 可关闭回退单模型、Task 持久化候选/最终模型/路由原因、Provider 故障按策略 fallback、能力匹配、熔断半开）均正确落地。仅“前端构建、生产 `sync db`、`/llm/router` HTTP smoke、模型切换 Eval Gate”四项无法在本沙箱验证，需用户侧确认（见 §9）。在用户完成 §10 人工验收前，请勿在 phase doc / README 中将 V2-8 进一步标记（当前 phase doc 验收框已勾选，状态推进建议待人工验收闭环）。

---

## 2. 改动范围

**新增文件（后端 3 个包 / 1 控制器）**
- `agent/modelgateway/router.go` — `Router.Route`（能力匹配 + 评分排序 + fallback 决策）、`gatewayClient`（统一 `llm.Client`，内部按候选顺序 fallback，写 `RouteTrace`）、`classifyRouteError`（错误分类与可重试判定）。
- `agent/modelgateway/health.go` — `HealthRegistry`（滑动窗口观测、closed/open/half-open 状态机、单飞 half-open）。
- `agent/modelgateway/router_test.go` — 4 个单测覆盖关闭回退、能力匹配、429 fallback、熔断半开。
- `llm/routing.go` — 类型定义：`ModelProfile` / `ModelRequirements` / `RouteCandidate` / `RouteDecision` / `RouteTrace` / `RouteAttempt` / `RouterSettings` / `RoutingConfig` / `Router` 接口。
- `llm/router_store.go` — `RouterSettings` 读写、`RoutingConfigs` 装配、`validateRouterSettings` / `normalizeRouterSettings`。
- `controllers/llm_router.go` — `LLMRouterController`：`Get`（设置+健康快照）、`Put`（更新设置，带校验）。

**修改文件（关键 diff）**
- `llm/types.go`：`Response` 增 `Provider` / `ConfigID` / `RouteTrace`（均 `json:"-"`，不污染对外 JSON）。
- `llm/store.go`：`ConfigInput` / `PublicConfig` 增路由画像字段；`normalizeConfigInput` 增加画像校验；新增 `applyRoutingProfileDefaults` / `normalizeClass` / `intValueOr`；`configFromModel` / `modelFromInput` / `toPublicConfig` 透传画像（含 `cfg.ID = row.ID`）。
- `models/llm_config.go`：`LLMConfig` 增 `RouterCandidate` / `StructuredOutput` / `NativeToolCalling` / `Reasoning` / `LongContext` / `JSONReliability` / `MaxContextTokens` / `CostClass` / `LatencyClass`；新增 `LLMRouterSetting` 表。
- `main.go`：`orm.RegisterModel(new(models.LLMRouterSetting))`。
- `models/agent_task.go`：`AgentTask` 增 `FinalModelConfigID` / `RouteCandidatesJSON` / `RouteReason` / `RouteFallbackJSON`（均 nullable `null`）。
- `agent/task/task.go`：`Task` 增对应运行时字段。
- `agent/task/orm_store.go`：`toModel` / `fromModel` 双向映射上述字段（经 `sanitizePayload` / `sanitizeText` / `stringPointer` 脱敏与 nullable 安全）。
- `agent/skill/skill.go`：新增 `ModelRequirementProvider` 接口与 `Definition.ModelRequirementsValue` / `ModelRequirements()` 实现。
- `agent/manager/manager.go`：`Config.ModelRouter` 字段；`Start` 在 `ModelRouter != nil` 时调 `Route` 并落库候选/选中/原因；`Resume` 用 `NewClientByID(item.ModelConfigID)`。
- `agent/app/default.go`：Runtime `Config.ModelRouter = modelgateway.Default()`。
- `agent/runtime/react_executor.go`：`response.ConfigID` → `FinalModelConfigID`；`response.RouteTrace` → `RouteReason` / `RouteFallback`。
- `routers/router.go`：新增 `/llm/router`（`get:Get;put:Put`）。
- 5 个 Skill 声明 `ModelRequirements`：`alertanalysis`（StructuredOutput+MinJSONReliability:70+PreferLowLatency）、`marketregime`（StructuredOutput+MinJSONReliability:65）、`strategybuilder`（StructuredOutput+Reasoning+MinJSONReliability:75）、`symbolanalysis`（StructuredOutput+MinJSONReliability:70）、`portableskill`（StructuredOutput+MinJSONReliability:55）。
- `models/agent_task_syncdb_test.go`：legacy SQLite DDL 扩展 `agent_tasks`（4 新列）与 `llm_configs`（9 画像列）、`llm_router_settings` 表，断言新列存在且 nullable、legacy 行不变。
- `doc/...`：`08-phase-v2-8` 验收框勾选、`README.md` V2-7/V2-8 标 ✅、`07-phase-v2-7` 人工验收全勾、`doc/TODO.md` 增补条目。
- `static/**`：前端构建产物（来自独立仓库 `go_binance_futrues_new_ui`，非手改源码）。

---

## 3. 验收逐条核对（对照 Phase doc §验收）

| # | 验收项 | 实现位置 | 结论 |
|---|--------|----------|------|
| 1 | Router 可关闭并回退 V1 单模型行为 | `router.go:72-74`（`settings.Enabled != 1` → `single()`，仅用 enabled primary，无 fallback 候选）；`DefaultRouterSettings().Enabled=0`（默认关闭，向后兼容 V1） | ✅ |
| 2 | Task 保存候选、最终模型和路由原因 | `models/agent_task.go`（RouteCandidatesJSON / FinalModelConfigID / RouteReason / RouteFallbackJSON）；`manager.go:109-116`（落库候选+选中+原因）；`react_executor.go:84-92`（执行后覆盖 FinalModelConfigID + RouteTrace） | ✅ |
| 3 | Provider 故障可按策略 fallback | `router.go:201-242`（`Generate` 顺序尝试候选；`classifyRouteError` 判定可重试）；测试 `TestGatewayFallbackOn429` 验证 429 切到 secondary 且 `RouteTrace.Attempts==2` | ✅ |
| 4 | 模型切换通过 Eval Gate | 路由能力匹配 + 评分已实现；**Eval 实跑需用户侧确认**（本沙箱无 real LLM，见 §9） | ⚠️ 待用户侧 Eval |

补充契约（来自 §Capability / §Router / §Health / §Native Tool Calling）：

| # | 契约 | 实现位置 | 结论 |
|---|------|----------|------|
| 5 | Skill 声明能力需求，不硬编码模型名 | `skill.go:37-39`（`ModelRequirementProvider`）；5 个 Skill 声明 `ModelRequirements` | ✅ |
| 6 | 能力匹配（structured/native_tool/reasoning/long_context/json_reliability/max_context_tokens） | `router.go:128-148` `matches()`；`RoutingConfigs` 装配 `ModelProfile`（`router_store.go:118-120`） | ✅ |
| 7 | 选择顺序综合主模型/能力/健康/延迟/成本/任务类型 | `scoreCandidate`（`router.go:150-174`：primary +35、reasoning +30、long_context +20、低延迟/低成本 class 加分、健康成功率 +30、超高延迟 -10）；`routed` 按 score 降序 | ✅ |
| 8 | 熔断/半开避免重复撞坏 Provider | `health.go`：`Allow`（half-open 单飞）/ `Record`（阈值开闸 + cooldown）/ `Snapshot`；测试 `TestHealthCircuitOpensAndHalfOpens` | ✅ |
| 9 | Fallback 记录原因且不突破 Budget/Permission | `RouteDecision.Reason` / `RouteTrace.Attempts` 全程记录；fallback 仅在同一组 `RoutingConfigs`（用户已配 LLM）间切换 LLM 客户端，不改变 Skill 的 Tool/Permission/Budget 策略（策略在 runtime 层独立强制） | ✅ |
| 10 | Fallback 不突破能力要求 | fallback 候选均经 `matches(requirements)` 预筛（`router.go:91-93`），不会切到不满足能力的模型 | ✅ |

---

## 4. 设计分析

### 4.1 关闭回退（向后兼容 V1）
`DefaultRouterSettings().Enabled=0`，即默认关闭 Router，行为与 V1 单模型一致（`single()` 仅构造 primary 的 `gatewayClient`，`candidates` 仅 1 个，`Generate` 不再进入 fallback 分支）。`RouterSettings` 读空行时回落 `DefaultRouterSettings()`（`router_store.go:25-27`），避免新部署无设置即报错。`Route` 要求至少一个 `Primary`（enabled）配置，否则报错——保证关闭状态下仍有可用模型。

### 4.2 Fallback 与错误分类
`classifyRouteError`（`router.go:256-282`）将错误分为可重试与不可重试：
- 可重试（触发 fallback）：`context.DeadlineExceeded`(timeout)、`io.EOF`/`ECONNRESET`/`EPIPE`(network)、`net.Error` timeout/temporary(timeout)、HTTP `429`(429)、HTTP `>=500`(5xx)。
- 不可重试（立即失败、不 fallback）：HTTP 4xx（如 400/401/403，属客户端/鉴权错误，换模型无意义）、其他 `request` 类。

`Generate` 循环：`index>0 && FallbackEnabled!=1 → break`（关闭 fallback 时即便多候选也只尝试首个）；`!health.Allow → circuit_open 跳过`；成功则写 `RouteTrace` 并返回；失败则 `Record` 健康并 `appendAttempt`，可重试且 `FallbackEnabled==1` 才继续下一候选。逻辑闭环、无越界。

### 4.3 熔断状态机
`HealthRegistry`（进程内全局单例 `defaultHealth`，符合“避免每个任务重复撞坏 Provider”的全局视角）：
- `Record` 成功 → `closed`；失败 → `consecutiveFailures++`，当 `half_open` 或 `>=threshold` 时 `open` 并设 `openUntil=now+cooldown`。
- `Allow`：`open` 且未过 cooldown → 拒绝；过 cooldown → 转 `half_open` 并允许单飞（`halfOpenInFlight` 防并发多发）；`half_open` 单飞中再次 `Allow` → 拒绝；`half_open` 成功 → `closed`。
- 窗口 `window`（默认 20）滑动截断观测，计算 `SuccessRate` / `Rate429` / `Timeouts` / `ServerErrors` / `AverageLatencyMs`，供 `scoreCandidate` 加权。
- 单飞锁 `halfOpenInFlight` 为进程内；测试 `TestHealthCircuitOpensAndHalfOpens` 覆盖 open→half-open→closed 全转换。

> 注：健康状态为进程内存态，重启后清零（非持久化）。对单进程部署可接受；若未来多实例部署需外部共享健康（非本 Phase 范围，见 §8）。

### 4.4 候选与最终模型的审计完整性
- 任务创建即落：`RouteCandidates`（候选全量 JSON）、`ModelConfigID`+`Model`（Router 选中）、`RouteReason`（选中原因）。
- 执行成功覆盖：`FinalModelConfigID`（实际最终，可能因 fallback 不同于选中）、`RouteReason`（trace reason）、`RouteFallback`（attempts JSON，含每候选 status/error/duration）。
- 因此 Task 同时保留「初始候选/选中」与「实际最终/尝试链」，满足“保存候选、最终模型和路由原因”。

### 4.5 Native Tool Calling 范围
本 Phase 将 `NativeToolCalling` 作为**能力画像字段**接入 `ModelProfile` 与 `ModelRequirements`，并在 `matches()` 中作为匹配条件使用（需要原生 tool calling 的 Skill 可在路由时筛掉不支持的模型）。但**实际“在 LLM Adapter 层使用厂商原生 Function Calling、再转换为统一 ExecutionStep”的适配实现不在本 diff 内**——executor 仍走既有统一 decision 协议。Phase doc 该节为设计原则，非显式验收项；能力字段已就位，待后续 Phase 落地原生适配。非阻塞。

---

## 5. DB Schema 迁移兼容（Additive）

新增列/表均为 nullable 或带默认值，向后兼容：
- `agent_tasks`：新增 `final_model_config_id`(int64,index)、`route_candidates_json`(text,null)、`route_reason`(text,null)、`route_fallback_json`(text,null)。`RouteReason` 用 `*string` + `sanitizeText`，`RouteCandidatesJSON`/`RouteFallbackJSON` 用 `*string` + `sanitizePayload`——旧行 NULL 可读、可显式写入。
- `llm_configs`：新增 9 个画像列（均带 `default`，如 `structured_output default 1`、`json_reliability default 80`、`cost_class/latency_class default medium`）。
- 新增表 `llm_router_settings`（pk=1 单行设置）。
- 既有 `syncdb` 测试 `models/agent_task_syncdb_test.go` 扩展 legacy DDL，断言新列存在且 **nullable**（避免 SQLite ALTER 含行表时 NOT NULL 失败），并新增 `llm_configs`/`llm_router_settings` 表存在性校验、legacy 行不变——全量 `go test ./...` 含此包 `ok`，向后兼容验证通过。

---

## 6. 自动测试覆盖确认

| 测试 | 覆盖 Gate | 结果 |
|------|-----------|------|
| `TestRouterDisabledUsesPrimaryOnly`（`router_test.go:55`） | 关闭→仅 primary、fallback 不发生、候选=1 | ✅ |
| `TestRouterCapabilitySelectsMatchingCandidate`（`router_test.go:75`） | 能力匹配（Reasoning）选中正确候选、不匹配者排除 | ✅ |
| `TestGatewayFallbackOn429`（`router_test.go:91`） | 429 切 secondary、`RouteTrace.Attempts==2`、响应带 `ConfigID`/`Provider` | ✅ |
| `TestHealthCircuitOpensAndHalfOpens`（`router_test.go:114`） | 连续失败开闸→cooldown 后 half-open 单飞→成功闭合 | ✅ |
| legacy `syncdb` 测试（`models/agent_task_syncdb_test.go`） | additive 迁移向后兼容（新列 nullable、新表存在、legacy 行不变） | ✅ |

> 注：`manager.Start` 的“Route 失败回退 NewClient”、“Resume 用 ModelConfigID”、以及 `Put` 设置校验（`validateRouterSettings`）由既有 `manager` 单测与 `router_store` 逻辑间接保障；建议后续补 `app`/集成层测试覆盖 `Route` 在 `manager.Start` 的落库（见 §8）。

---

## 7. 本沙箱已执行的验证命令（证据）

```
cd /Users/zhz/work/binance/go_binance_futures && export PATH=/usr/local/go/bin:$PATH
go build ./...                                                              # exit 0
go test -count=1 -race ./agent/modelgateway ./llm ./agent/manager ./agent/runtime ./agent/task ./models   # ok
go test -count=1 ./...                                                      # 全部包 ok
go vet ./agent/modelgateway/... ./llm/... ./agent/manager/... ./agent/runtime/... \
      ./agent/task/... ./models/... ./controllers/... ./agent/skill/... ./agent/app/... \
      ./agent/portableskill/... ./routers/... ./agent/skills/...            # exit 0
```
（所有 git 访问均前置 `GIT_OPTIONAL_LOCKS=0`，符合本仓防 lock 铁律。）

---

## 8. 非阻塞改进建议（不阻塞 Gate）

1. **`routed()` 中 `NewClient` 失败的候选被静默 `continue`**（`router.go:100-102`）：若 primary 客户端构造失败（如坏 APIURL），它会被丢弃、可能让低优先级 secondary 被当作“选中”，且路由原因不体现 primary 不可用。建议：primary 构造失败时返回明确错误（或至少在 `RouteDecision.Reason` 标注 dropped primary），避免掩盖配置错误。
2. **健康态未持久化**：`defaultHealth` 为进程内存，重启清零。单进程可接受；若后续多实例部署，需引入共享健康（Redis/DB 外部存储），否则各实例独立熔断、全局视角失效。
3. **`Resume` 不走 Router**：`manager.Resume` 用 `NewClientByID(item.ModelConfigID)` 直连初始选中配置，不经 `gatewayClient`（无 fallback/健康）。若初始选中配置已故障，resume 会直连失败。可考虑 resume 也经 `Route` 以获得 fallback，但需注意与已冻结 ExecutionSnapshot 的一致性（非阻塞）。
4. **`single()` 路径仍包 `gatewayClient`**：关闭 Router 时也会写健康观测，无害；但若希望关闭时完全零开销，可直连 `llm.Client`。当前实现无功能问题。
5. **`classifyRouteError` 对 4xx 直接失败**：符合“不浪费 fallback”设计，但若某些 4xx（如 429 已被单独处理）需细分，当前已覆盖主要可重试类，无需改动。
6. **补充集成测试**：建议加 `manager`/`app` 层测试，校验 `Start` 在 `ModelRouter` 下将 `RouteCandidates`/`RouteReason`/`FinalModelConfigID` 正确落库，以及 `Put` 设置校验（`failure_threshold` 越界、`cooldown_seconds` 越界等返回 400）。

---

## 9. 本沙箱未能验证的项（需用户侧确认）

- **前端 `go_binance_futrues_new_ui`**：`npm run typecheck` / `npm run build` 是否 PASS（`static/**` 为构建产物，已重建，但未在源仓跑构建）；`/llm/router` 设置页 UI 是否可用。
- **生产 `sync db`**：真实 MySQL 是否已加 `agent_tasks` 4 新列、`llm_configs` 9 画像列、新建 `llm_router_settings` 表（schema 为 additive + nullable/默认值，预期无碍，但需实跑确认）。
- **HTTP smoke（鉴权后）**：`GET /llm/router` 返回设置+健康；`PUT /llm/router` 校验后更新；开启 Router 后真实任务按能力/健康选模型并 fallback；关闭 Router 后回退单模型。
- **模型切换 Eval Gate**：真实 LLM 下多模型路由对 Skill 产出的 Eval 对比（验收项 4），需用户在 Eval 环境实跑。

---

## 10. 人工验收待办（完成后方可将 V2-8 视为完全闭环）

- [ ] 生产 `sync db` 成功，新列/新表就位，旧数据可读。
- [ ] 前端 `/llm/router` 设置页可用，`Put` 校验生效（阈值/冷却越界报错）。
- [ ] 开启 Router：多 LLM 配置下，按 Skill 能力/健康/延迟/成本正确选模型。
- [ ] 手动制造 primary 故障（如 429/5xx/超时），验证按策略 fallback 且 `RouteTrace` 记录原因、不突破 Budget/Permission。
- [ ] 关闭 Router：回退单模型行为，任务仍正常完成。
- [ ] 熔断验证：连续失败触发 open，cooldown 后 half-open 单飞，成功闭合。
- [ ] 模型切换通过 Eval Gate（真实 LLM 多模型对比）。

以上完成后，V2-8 可标记为完全通过。
