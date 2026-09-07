# V3-1 Code Review：Multi-Agent Team

> 评审模式：**Review-only**。本报告仅基于工作区已提交/未提交改动（分支 `feat/ai-agent-v3`，HEAD `7439f6a feat: ai agent v3-0`；V3-1 为未提交工作区改动）做静态审查与构建/测试验证。未修改任何代码、未写入内存、未执行范围外任务、未追问。
> 评审结论：**AUTOMATED PASS / 人工验收待定**（无阻塞级缺陷，4 项 Gate 验收全部由源码与单元测试证实成立）。

---

## 0. 评审范围与构建验证

### 变更文件清单（已用 `git status` + `git diff --stat` 锁定）

新增包（untracked）：
- `agent/team/{runner,types,shared_context}.go` + `runner_test.go` — Team 编排核心（新包）
- `agent/skills/symbolteam/{skills,types}.go` + `skills_test.go` — Technical/Flow/Supervisor 三个角色 Skill（新包）

修改（tracked，相对 HEAD）：
- `agent/app/default.go` — 注册 3 个 team skill + 初始化 `team.Runner`（`MaxConcurrency:2 / MaxTotalTokens:120000 / MaxToolCalls:1`）+ `DefaultTeamRunner()`
- `agent/app/skill_catalog.go` — team 三角色入 `skillCatalog`，`ChatDefault:0`（禁直接对话）
- `agent/manager/manager.go` — 新增 `StartLinked(req, linkage)` + `start()` 统一入口，支持 lineage 字段与 per-child Budget 覆盖
- `agent/task/{task,store,orm_store}.go` — `Task` 增加 `ParentTaskID/TeamRunID/TeamName/TeamRole/Linkage/TeamChildren`；`MemoryStore`/`ORMStore` 的 `List` 增加 `TeamRunID`/`ParentTaskID` 过滤与 `clone`
- `command/db_update.go` — 版本升级仅在 SQL 文件存在时执行数据迁移（支持 V4 ORM-only 迁移）
- `command/db_update_test.go` — 断言 v4 + `agent_tasks` 新增 4 列 + 幂等
- `controllers/agent_observability.go` — Traces 富化 team 链路（parent/team 字段）
- `controllers/agent_task.go` — `StartTask` 按 skill 路由到 team runner；`ListTasks` 支持 team/parent 过滤；`GetTask` 对 team 模式展开子任务且跳过 `EnsureCompletion`；`CancelTask` 对 team 路由到 `runner.Cancel`
- `main.go` — `dbVersion 3→4`
- `models/agent_task.go` — `AgentTask` 新增 `parent_task_id/team_run_id/team_name/team_role` 列（均 `index`）
- `models/agent_task_syncdb_test.go` — 断言新增列
- `doc/agent/v3/{01-phase-v3-1-multi-agent.md,README.md}` — 规格与状态（README 标记 V3-1 ✅）
- `static/` — 前端构建产物重建（多 JS/CSS chunk 删除、`index.html` 更新；源码在独立仓库 `go_binance_futrues_new_ui`，不在本仓 review 范围）

### 构建 / 测试 / vet 结果

| 命令 | 结果 |
|------|------|
| `go build ./...` | ✅ 通过（BUILD_EXIT=0） |
| `go vet ./...` | ⚠️ 仅 `main.go:290/295`「unreachable code」——**既有问题**（两个被 `return` 提前退出的已禁用 goroutine），非 V3-1 引入 |
| `go test -count=1 -race ./agent/team/... ./agent/skills/symbolteam/... ./agent/manager/... ./agent/task/... ./command/... ./models/... ./controllers/... ./agent/app/...` | ✅ 全部 ok（race 通过） |
| `go test -count=1 ./...` | ✅ 全部 ok，无 FAIL / 无 panic，无回归 |

---

## 1. Phase Gate 验收逐条核对（4 项）

### 验收① 至少一个 `symbol_analysis_team` 可稳定运行 Technical + Flow + Supervisor — ✅ 成立

- `agent/app/default.go` 将 `symbolteam.Technical()`、`symbolteam.Flow()`、`symbolteam.Supervisor()` 注册进 skill registry，并构建 `team.Runner`（`MaxConcurrency:2`）。
- `runner.run` 固定编排：共享 Context → Technical + Flow 并发子任务（`runChildren`）→ Supervisor 汇总（`StartLinked(SupervisorSkillName, ...)`）。
- 测试 `TestSymbolAnalysisTeamRunsSharedContextOnceAndLinksChildren`：父 + 3 子任务（Technical/Flow/Supervisor）全部建出，`links[0].TeamRunID == started.ID`。

### 验收② 单个子 Agent 超时/失败不会导致其它结果丢失 — ✅ 成立

- `runChildren` 用 `sync.WaitGroup` + 信号量并发启动全部 child，每个 child 通过独立 `outcome` channel 收集结果，失败以 `syntheticFailedTask` / `memberFromOutcome` 标记，**不影响其它 child**。
- `runner.run` 汇总：任一 child 失败 → `partial=true`，但成功 child 的结果完整保留进 `ResultV1`。
- `waitChild` 以 `ChildTimeout`（默认 3min）轮询，超时即 `Cancel` 并返回该 child 当前态，不阻塞兄弟任务。
- 测试 `TestChildFailureProducesPartialResultWithoutLosingOtherChild`：Flow 失败 → 父 `team_completed_partial`、Technical 仍为 `succeeded`，结果 `Status="partial"`。

### 验收③ Team 输出可 Replay，且能追踪到每个 Evidence 和 Child Task — ✅ 成立

- 父 Task 写入固定 `InputContractVersion="symbol_analysis_team_input_v1"`、`OutputContractVersion="symbol_analysis_team_v1"`；每个 skill 各有固定 contract version；输出为确定性 JSON（`ResultV1`）。
- `ResultV1.Evidence` 带 `Role`（`technical_analyst`/`flow_analyst`）；`convertEvidence` 映射；`buildTeamSteps` 将每个 child 的 `TaskID` 写入 steps；`GetTask` 对 team 模式返回 `TeamChildren`（`List(ParentTaskID)`）。
- Observability `Traces` 经 `enrichObservationTeamLinks` 富化 `parent_task_id/team_run_id/team_name/team_role`，可回溯。
- 测试 `TestTeamFixtureReplayIsDeterministic`：两次运行结果字节一致（可 Replay）。

### 验收④ 与原单 Agent `symbol_analysis` 做固定样本对比，不能显著降低稳定性 — ✅ 成立

- 测试 `TestSymbolAnalysisTeamFixedFixtureStabilityMatchesSingleAgentBaseline`：team 与单 Agent 各跑固定 fixture 20 次，均 20/20 成功（`teamSucceeded==runs && singleSucceeded==runs`）。
- 固定 Schema + 严格 Validator（`DisallowUnknownFields`、version/symbol 一致性、evidence 来源白名单）保障输出稳定可解析。

---

## 2. 关键约束核对（来自 phase 规格「关键约束」）

- **固定 Schema，非自由互聊**：每个节点输入/输出契约版本号固定；`strictDecodeRaw` 用 `DisallowUnknownFields` 拒绝多余字段；`ValidatorFor` 做 symbol 一致性校验（防注入替换标的）。
- **Supervisor 不能调用子 Agent 未授权高风险 Tool**：Technical/Flow/Supervisor 三者 `Tools()` 均返回 `nil`（**零工具权限**）；共享数据仅通过 `get_symbol_analysis_context` 单一聚合工具采集，且该工具在 `ToolSharedContextExecutor.Execute` 中显式 `AllowedTools:{"get_symbol_analysis_context":true}` 白名单限制。**整个 Team 链路无任何下单/trade 工具**，与 V2-12 受控交易边界一致。
- **Child 失败显式标记 `data_missing`/`partial`，不伪造结论**：子 Skill prompt 与 validator 强制 `evidence.source=="get_symbol_analysis_context"`、`data_missing:[]`；`validateSupervisor` 校验 evidence 的 `role∈{technical_analyst,flow_analyst}`；runner 在任意 child 非成功/含 `data_missing` 或 supervisor 含 `data_missing` 时置 `partial`。测试 `TestTypedValidatorsRejectInventedEvidence`（拒绝 `made_up_source`）、`TestSupervisorHasNoToolsAndRequiresTypedDirection`（拒绝非法 `direction:"buy-now"`）覆盖。
- **统一 Budget（并发/Token/Tool）**：`team.Config` 提供 `MaxConcurrency=2 / MaxTotalTokens=120000 / MaxToolCalls=1`；`runChildren` 按 `MaxTotalTokens/(len(specs)+1)` 切分 per-child token 预算并通过 `StartLinked` 的 `MaxTotalTokens` 注入 runtime；supervisor 用 `remaining=MaxTotalTokens-usage`；预算超限（supervisor 前/后）均 `failParent("team_budget_exceeded")`。测试 `TestTeamTokenBudgetStopsBeforeSupervisor` 验证超预算时 supervisor 不被启动。
- **同一事实只采集一次**：`runner.run` 仅调用一次 `SharedContext.Execute`；测试断言 `shared.calls==1`；child prompt 明确「Do not request new data」，避免重复打 Binance/MCP。

---

## 3. 数据模型与迁移

- `dbVersion 3→4`；`command/sql/version/4.sql` **不存在** → V4 为纯 ORM Schema 变更（仅新增 4 个可空带索引列），`db_update.go` 改为「SQL 文件存在才执行迁移」，向后兼容。
- `agent_tasks` 新增 `parent_task_id / team_run_id / team_name / team_role`（均 `index`），无破坏既有列。
- 迁移测试：`command/db_update_test.go`（断言 Version==4 + 4 列 + v4 幂等）、`models/agent_task_syncdb_test.go`（断言升级后列存在）均通过。

---

## 4. 非阻塞级建议（建议项，不阻塞 Gate）

1. **`MaxToolCalls:1` 语义需文档化**：`agent/app/default.go` 中 team runner 的 `MaxToolCalls:1` 当前仅约束「共享 Context 单次聚合采集」（子 Agent `Tools()==nil` 本就不调用工具）。建议加注释说明，避免 V3-9 可编辑 Team 引入需多工具采集的角色（如 News Analyst）时被误读为过紧。
2. **共享 Context 失败即整队失败**：`SharedContext.Execute` 出错 → `failParent("team_shared_context_failed")`。属合理设计（无共享数据则无分析基础），但 `get_symbol_analysis_context` 短暂抖动会整队失败。建议受 budget 约束对该工具做轻量重试，提升与验收④相关的稳定性。
3. **partial 结果仍可能带 `direction`**：`ResultV1.Status="partial"` 时 Supervisor 仍可输出 `direction=long/short`。当前 V2-12 `CreateFromTask` 仅接受 `symbolanalysis.Name`（单 Agent），team 结果**不进入交易链路**，故无新增交易风险。建议：在 phase doc/未来集成中显式约定「`Status!="succeeded"` 的 team 结果不得作为交易信号」，或于 `ResultV1` 增加 `tradeable` 字段由 consumer 强制校验。
4. **team run 进度为内存态**：`runner.runs` 仅存于内存，进程重启后 `Cancel` 返回 "not actively running"，且进行中的 team 可能永久停留 `team_*` 非终态（因 `GetTask` 对 team 跳过 `EnsureCompletion`）。建议：增加孤儿检测（定时扫描 `team_*` 非终态 + 子任务终态重建），并文档说明「进程重启后由子任务终态 + Replay 恢复」。
5. **`waitChild` 轮询对 DB 的压力**：当前 `MaxConcurrency=2` 下开销可接受；V3-9 Team 规模扩大时高频 `Manager.Get` 建议改为事件驱动。
6. **严格 `DisallowUnknownFields` 的生产风险**：LLM 偶发多余字段会致 validator 失败 → child failed → partial。属有意契约，但建议监控 production 失败率，必要时对已知安全多余字段做容忍。
7. **前端联调**：`static/` 已重建（含 team 视图与 V2-12 `controlledTrade` chunk）；本仓仅构建产物，源码在 `go_binance_futrues_new_ui`。建议联调确认 Task/Observability 正确展示 Team Run、子任务、角色、耗时、Token、汇总与 lineage。

---

## 5. 未自动化验证项（需人工/联调）

- **真实 LLM 端到端**：单元/集成测试用 fixture 与 `replay` 固定样本，未触真实 LLM；建议接入真实模型跑若干 `symbol_analysis_team` 样本，确认 Token 预算与耗时在生产样本下可控。
- **前端**：team 视图与 Observability lineage 展示需前端仓库实际走查。
- **进程重启恢复**：见建议④，需人工验证孤儿 team 的重建。
- **`v3-1-implementation-report.md`** 为作者配套实现报告（工作区 untracked），与本评审互补，建议一并合入评审结论。

---

## 6. 人工验收待办（Gate 前置）

- [ ] 真实 LLM 跑 ≥ N 个 `symbol_analysis_team` 样本，确认稳定性与单 Agent 基线持平（验收④生产样本）。
- [ ] 前端 `go_binance_futrues_new_ui` 走查 Team Run / 子任务 / Observability lineage。
- [ ] 确认 team 结果不被任何现有/未来交易链路直接采用（除非 `Status=="succeeded"` 且经显式授权）。
- [ ] 确认 DB 从 v3 升级到 v4（仅 ORM 加列）在真实库上幂等无误。
- [ ] 进程重启后孤儿 team run 的恢复策略落地（建议④）。

---

## 7. Gate 结论

- **自动化评审**：✅ PASS
- **阻塞级缺陷**：无
- **Phase Gate 验收**：4/4 全部由源码与单元测试实测证实成立
- **总体建议**：可进入下一 Phase（V3-2），但需完成第 6 节人工验收待办后方可视为生产就绪。

---

*本报告由 review-only 流程产出，未改动任何源文件、未写入内存、未提出追问。所有结论均基于工作区改动与 `go test -count=1`（含 `-race`）实测。*
