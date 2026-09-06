# Code Review — V2-11 业务 Workflow 与新 Skill

> 评审模式：**review-only**（仅 review，未修改任何代码、未运行任何破坏性命令）。
> 评审时间：2026-09-05
> 评审范围：`git diff --stat`（工作树 vs HEAD）锁定的 V2-11 后端新增/修改文件（不含 `static/` 前端构建产物）。V2-7~V2-10 已在 HEAD。

---

## 0. 结论（Gate）

**AUTOMATED PASS**（构建 / 测试 / vet / race 全绿，八项验收中可自动验证项逐条满足，安全与 fallback 边界成立）。

**人工验收待定**（见 §7）：Eval/Replay Core Gate 为外部执行门（fixtures + 技能注册已就位）；前端 typecheck/build 已由 `static/` 同步产物佐证、需在 UI 仓库最终确认；真实 LLM 端到端无法在 review 环境运行。

阻塞级缺陷：**无**。条件级（非阻塞）建议：见 §6。

---

## 1. 构建 / 测试结果

| 项目 | 命令 | 结果 |
|------|------|------|
| 全量构建 | `go build ./...` | ✅ 通过（exit 0） |
| Race 测试 | `go test -count=1 -race ./service/alertpipeline/... ./service/workflow/... ./agent/skills/workflows/...` | ✅ ok（linker 警告为 macOS 噪声，非失败） |
| 全量测试 | `go test -count=1 ./...`（仅显示非 ok 行） | ✅ 无失败（无回归，60+ 包全绿） |
| 静态检查 | `go vet ./service/workflow/... ./service/alertpipeline/... ./agent/skills/workflows/... ./agent/app/... ./controllers/... ./models/... ./agent/eval/...` | ✅ 无告警 |
| 验证约束 | 全程 `go test -count=1`（race 单独 `-race`） | ✅ 已规避缓存 |

> 前端 `static/` 重建产物为独立 UI 仓库构建输出（`workflows-*.js/*.css` 已同步），非本仓手改，不计入 review 范围（既有 `src/views/permission/page/index.vue not found`(TS6053) 为历史遗留）。

---

## 2. 变更清单（范围锁定）

**新增后端文件（8 源 + fixtures）**
- `models/agent_workflow_run.go` — `AgentWorkflowRun`（表 `agent_workflow_runs`）：父 Run 生命周期（stages/status/result/child_task_ids）。
- `service/workflow/service.go` — `Service` 编排：5 类 Workflow 分发、`execute*`、`startAndWait`/`waitTask` 轮询、`complete`/`fail`/`update`、`validWorkflow`/`schemaVersion`/`validateRequest`。
- `service/workflow/store.go` — `Store`：`Save`（upsert）/Get/List、`toModel`/`fromModel`（落库脱敏）、`newRunID`。
- `service/workflow/data.go` — 确定性数据层：`build*Input`（scanner / 模板快照 / 手续费后测试统计 / synthetic env + expr 编译验证）、`buildAlertTriageInput`/`buildDailyBriefInput`、错误→`DataMissing` 降级。
- `service/workflow/data_test.go` / `store_test.go` — 单测。
- `agent/skills/workflows/workflows.go` — 6 个 `skill.Definition`（market_scan / strategy_review / strategy_experiment_propose / strategy_experiment_summary / alert_triage / daily_market_brief）：标准技能定义 + 版本化 Schema + 严格 Validator。
- `agent/app/workflows.go` — `DefaultWorkflowService()` 接线 `DefaultManager` + `Store{}`。
- `controllers/agent_workflow.go` — `AgentWorkflowController`：Start/Get/List。
- `command/sql/version/2.sql` — 版本标记（Schema 实际由 `orm.RunSyncdb` 应用）。
- `agent/eval/testdata/core/v2_11_*.json`（7）+ `agent/replay/testdata/*.json`（6）— Eval/Replay Core Gate fixtures。

**修改后端文件（V2-11 diff，13）**
- `agent/app/default.go`：skill registry 注册 6 个 workflow 技能。
- `agent/app/scheduler.go`：新增 `daily_market_brief` Scheduler Job（默认关闭、`SkipIfRunning`、BuildInput 复用 `BuildDailyMarketBriefInput`）。
- `agent/app/skill_catalog.go`：skill 目录新增 6 条 workflow 实现描述。
- `controllers/index.go`：ServiceConfig 暴露 `AgentDailyMarketBriefScheduleEnable`/`IntervalMin`。
- `main.go`：`dbVersion 1→2` + `RegisterModel(AgentWorkflowRun)`。
- `models/tableStruct.go`：`Config` 增 `AgentDailyMarketBriefScheduleEnable`(默认0)/`AgentDailyMarketBriefIntervalMin`(默认1440)。
- `models/agent_task_syncdb_test.go`：legacy DDL 增 `agent_workflow_runs` 断言。
- `routers/router.go`：新增 `/agents/workflows`（List/Start）+ `/agents/workflows/:id`（Get）。
- `service/alertpipeline/pipeline.go`（+248）：`alert_triage` 集成——`incidentBucket`+定时器缓冲、per-symbol 聚合、`incidentWorker`、`processIncident`/`runTriageAI`/`applyTriageResult`/`notifyIncident`/`fallbackIncidentBatch`、Stats 增 4 项。
- `service/alertpipeline/types.go`：`Settings.TriageWindow`(默认3s) + Stats 4 项。
- `service/alertpipeline/pipeline_test.go`：+2 个 triage 测试（聚合为单通知 / AI 关闭保持确定性 fallback）。
- `agent/eval/eval_test.go`：`coreDefinition` 注册 6 个 workflow 技能。
- `doc/agent/v2/11-phase-v2-11-workflows.md` / `README.md`：验收勾选 + V2-11 标 ✅。

---

## 3. 路由核对（routers/router.go diff 实读）

```
POST /agents/workflows         -> Start   // 启动 Workflow（异步 accepted）
GET  /agents/workflows         -> List    // 按 workflow/status 分页
GET  /agents/workflows/:id     -> Get     // 查询单 Run（含 stage/status/result/child_task_ids）
```
控制器 `Start` 解析 `{workflow,input}`；`List` 经 `workflowservice.ListOptions`（`limit` 夹 [1,100]）；`Get` 对空/缺失 id 由 store 返回错误、控制器回 404。路由与方法一致。

---

## 4. Phase 验收逐条核对（对照 11-phase-v2-11-workflows.md）

### 验收① 新 Skill 不复制 Runtime/Task/Tool 基础设施 — ✅
- `agent/skills/workflows/workflows.go` 的 6 个技能均为标准 `skill.Definition`（`Name/SystemPrompt/ValidateInput/BuildInput/Validator/VersionInfo/Tools/MaxRounds/ModelRequirements`）。
- `Tools()` 返回 `nil`（纯 LLM + 结构化输出，无第二套 Tool 体系）；子步骤全部经既有 `Manager.Start` → Runtime → Task → Context → Memory → Tool/MCP → Permission → Observability（`service.go:startAndWait` 实证）。
- 提示词明确「复用既有确定性能力、不要自行扫描全市场/执行交易」，与「不建立第二套 Agent 执行体系」一致。

### 验收② 大规模计算由确定性 Service 完成 — ✅
- `market_scan`：`scanner.ScanTop30`（确定性强筛，Limit 30）+ 仅前 ≤10 候选交 Agent（`data.go:buildMarketScanInput`）。
- `strategy_experiment`：`testProposal` 用 `expr-lang/expr` 对候选规则做 `Compile`+`Run`（synthetic scenarios `{-8,0,8}`），产出 `ExperimentTestReport{Valid,...}`，Agent 无法篡改（见验收③）。
- `strategy_review`：手续费后净收益等由 `buildStrategyStats`（确定性 `CalculateTestTradeProfit`）计算。
- `alert_triage` / `daily_market_brief`：Signal 聚合、Scanner、MarketCondition 均为确定性 Service。

### 验收③ 每个输出有版本 Schema 和 Eval — ✅（源码实读 + 严格 Validator）
- 六类输出均有版本化结构体：`opportunity_set_v1` / `strategy_review_v1` / `strategy_experiment_proposal_v1` / `strategy_experiment_result_v1` / `incident_set_v1` / `daily_market_brief_v1`。
- 每技能 `Validator()` 为 `validator.Func`：先 `strictDecode`（`json.Decoder.DisallowUnknownFields()` 拒绝未知字段），再逐字段校验；并**交叉比对 output 与 input**：
  - `opportunity_set_v1`：候选 Symbol 必须 ∈ 输入 candidates 且唯一、Confidence∈[0,1]、direction 枚举、market_condition 一致。
  - `strategy_review_v1`：verdict 枚举、必填数组、`template_id`/`market_condition` 一致。
  - `strategy_experiment_result_v1`：**`reflect.DeepEqual(out.Test, in.Test)`**（deterministic test report 原样保留）、`TechnologyJSON`/`StrategyJSON` 必须与 proposal 完全一致（候选 JSON 不被改写）、verdict 枚举。
  - `incident_set_v1`：incident action 枚举(notify/suppress/monitor)、severity 枚举、每 signal_id 必须 ∈ 输入且**不重复归属**，且输入全部 signal 必须被 triage（不漏报）。
  - `daily_market_brief_v1`：版本/as_of/必填字段 + market_condition 一致。
- 该设计满足 phase doc「输出固定 Schema、原样保留候选 JSON 与 deterministic test report、不会自动覆盖正式策略」。

### 验收④ 外部 MCP/Skill 故障有 fallback 或明确 failure — ✅
- **Workflow 层**：child task 失败 → `waitTask` 返回 error → `s.fail(run, err)`，Run 置 `failed`+error，可经 `GET /agents/workflows/:id` 查询（明确 failure，可观测）。
- **Alert pipeline 层**：`processIncident` 在 AI 关闭 / 预算超限 / 并发满 / triage task 失败时一律 `fallbackIncidentBatch` → `notifyFallback`（确定性逐条告警，**不吞报警**）；单条 Signal 仍走既有 `alert_analysis`，多条约进入 `alert_triage`。
- **数据层**：`build*Input` 对缺失依赖（market_condition / signal summary）写入 `DataMissing` 数组而非崩溃，Agent 据此降级。
- 单测 `TestPipelineAIDisabledKeepsDeterministicPerSignalFallback` 实证：AI 关闭时 2 条 Signal 各自确定性 fallback、不进入 triage。

### 验收⑤ `go test ./...` 通过 — ✅（见 §1，全量无失败）
### 验收⑥ V2-11 相关 race test 通过 — ✅（`-race` 跑 `alertpipeline`+`workflow`，ok；并发缓冲/定时器/锁路径覆盖）
### 验收⑦ Replay/Eval Core Gate 通过 — ⚠️ 代码侧已就位，外部门需执行（见 §7）
- `agent/eval/eval_test.go` 的 `coreDefinition` 已注册 6 个 workflow 技能；`agent/eval/testdata/core/v2_11_*.json`（7）+ `agent/replay/testdata/*.json`（6）fixtures 已落地。Eval 运行器（V2-10 `RecordEval`）会落 eval 观察。Core Gate 本身是外部执行门，review 环境未运行真实 LLM 评估。
### 验收⑧ 前端 typecheck/build 通过并已同步到 backend/static — ⚠️ 由 `static/` 产物佐证（见 §7）
- `static/static/js/workflows-*.js` 与 `static/static/css/workflows-*.css` 已被新构建同步，`static/index.html` 引用更新。typecheck/build 需在 UI 仓库最终确认。

---

## 5. 安全 / 权限 / 审计 / 降级

### 5.1 Workflow 持久化脱敏 — ✅ 成立（源码实读）
- `store.go:toModel` 对 `Input`/`Result`/`Error` 统一经 `security.RedactText` 后再入库；`fromModel` 回读的是已脱敏文本。与 `agent_observations`/`agent_change_events` 一致，符合项目全局脱敏策略。
- **提示（非阻塞）**：因脱敏在落库时不可逆，若 workflow 输入/结果含命中脱敏启发式的非秘密文本，存储与 API 返回值将被掩码。当前 6 类输出均为结构化 Schema（symbol/分数/incident 等），几乎不含敏感串，风险低。

### 5.2 报警不漏报 — ✅ 成立（逻辑 + 单测）
- `validateIncidentSet` 强制「输入全部 signal 必须被某 incident 覆盖」；pipeline `fallbackIncidentBatch` 在 triage 任何环节失败时回退为逐条确定性告警。`TestPipelineAIEnabledCoalescesCrossTypeSignalsIntoOneIncidentNotification` 实证跨类型 Signal（fast+liquidation）聚为单 incident 通知且不漏。

### 5.3 子步骤复用既有基础设施 — ✅ 成立
- 全部子 Agent 步骤走统一 `Manager`，自动继承 V2-7~V2-10 的 Memory 注入、Observability Trace、权限与预算；无并行执行体系。

### 5.4 DB 迁移兼容 — ✅ 成立
- `main.go` dbVersion 1→2，`RegisterModel(AgentWorkflowRun)`；`syncdb_test.go` 断言 `agent_workflow_runs` 表存在、legacy 行不变；`command/sql/version/2.sql` 保留版本标记（实际 DDL 由 `RunSyncdb` 应用）。`go test ./models/...` 通过。

---

## 6. 非阻塞建议（Conditional / 可选改进）

1. **Workflow Run 无自动续跑**：`Start` 以 `go s.execute(context.Background(), id)` 异步执行；若进程在子任务等待期间重启，Run 会停留在 `running`（`waitTask` 10min 超时后转 `failed`）。当前 phase 未要求 Run 级 resume；如需韧性，可加启动期扫描 `status='running'` 的 Run 重新 `execute` 或标记 `interrupted`。非阻塞。
2. **子任务结果未二次校验**：`runSingle`/`complete` 信任子 task 的 `Status==succeeded` 并直接存 `item.Result`，未再跑该 skill 的 `FinalValidator`。这依赖 Runtime 内 validator 已将失败转 task-failed（通常成立）；若担心「task 成功但结果非法」，可在 `complete` 前重放 validator。非阻塞。
3. **`waitTask` 轮询开销**：每 250ms 轮询 `Manager.Get` 直到终态或 10min 超时；高并发 workflow 下对 task 表有读压。可后续改 task 完成事件订阅。非阻塞。
4. **`incidentBucket.timer` 未显式 Stop**：`flushIncident` 由 `time.AfterFunc` 触发后从 map 删除；进程退出时若 timer 未触发无清理（一次性 timer，GC 可回收），无泄漏风险，仅非阻塞提示。
5. **`summary.go`/`data.go` 全量加载模式**（沿用 V2-10 观察）：`buildStrategyStats` 取窗口内最多 2000 行测试统计在内存聚合，30d 大账户下内存峰值较高；当前可接受，后续可分批/SQL 聚合。非阻塞。

---

## 7. 未验证项 / 人工验收待办

- ❏ **Eval/Replay Core Gate（验收⑦）**：fixtures（`agent/eval/testdata/core/v2_11_*.json`、`agent/replay/testdata/*.json`）与 `coreDefinition` 技能注册已就位；但 Core Gate 是外部执行门（需跑真实 LLM 评估），review 环境未执行。请在 CI/本地运行 Eval/Replay Gate 确认 v2_11 六个 case 全绿。
- ❏ **前端 typecheck/build（验收⑧）**：`static/` 已含 `workflows` 构建产物且 `index.html` 已更新引用，表明 UI 已构建并同步；请在 UI 仓库 `go_binance_futrues_new_ui` 最终确认 `npm run typecheck` / `build` 通过。
- ❏ **前端与接口字段对齐**：需确认 `AI → 业务 Workflow` 页面与 `POST /agents/workflows`（`{workflow,input}`）、`GET /agents/workflows`（List，`ListOptions{workflow,status,page,limit}`）、`GET /agents/workflows/:id`（Run 含 `stage/status/result/child_task_ids`）字段一致；Dashboard 的 Daily Market Brief 开关/周期映射到 `AgentDailyMarketBriefScheduleEnable`/`IntervalMin`（默认关 / 1440min）。
- ❏ **真实 LLM 端到端**：review 无法调用真实 Model Gateway；建议手动触发 `market_scan`/`alert_triage`（开启 AI 报警）验证 incident 聚合与异步 Run 状态流转。
- ❏ **多实例竞争**：`DefaultWorkflowService` 为进程内单例 + `execute` goroutine；若未来多副本部署，Run 状态由 DB upsert 保证，但并发 `execute` 同一 run 不会发生（run 由 Start 创建后仅本 goroutine 推进）。如需多副本，建议加 `status='queued'→'running'` 的原子 claim。非阻塞。

---

## 8. 评审签名

- 模式：review-only（零代码改动、零内存写入、零提问）
- 自动化结论：**AUTOMATED PASS**
- 人工验收：**待定**（见 §7：Eval/Replay Core Gate 执行、前端 typecheck 最终确认、接口字段对齐、真实 LLM 端到端）
- 阻塞项：**无** → 满足进入 V2-12 的 Gate 条件。
