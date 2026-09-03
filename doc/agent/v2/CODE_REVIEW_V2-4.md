# V2-4 Code Review 报告

> 审查范围：当前工作区（未提交）代码
> 后端：排除 `static/`（前端打包产物）
> 前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui`（4 个改动文件）
> 对照文档：`doc/agent/v2/04-phase-v2-4-eval-replay.md`
> 审查日期：2026-09-03

---

## 一、结论摘要

| 维度 | 结论 |
|------|------|
| Eval Framework 新包 `agent/eval/`（9 文件） | ✅ 与文档 §Eval Framework / §Core Eval Cases / §Revision Comparison / §CI Gate / §Shadow 逐项一致 |
| Tool Catalog / Skill Package Hash 冻结与落库 | ✅ `catalog.go` + `skill/version.go` 确定性 SHA-256，经 `ApplyVersionMetadata`→`orm_store` 完整持久化 |
| Runtime 身份双闸门升级 | ✅ `2.3.0` / `runtime_state_v4`；resume 新增 `SkillPackageHash`/`ToolCatalogHash` 比对，变更即拒绝 |
| Replay V2-4 扩展 | ✅ `tool_metadata` 可选字段支持 synthetic MCP/portable Tool，复用 `agent_replay_v1` |
| 数据库变更 | ✅ **additive**：`agent_tasks` 新增两列 `size(64)`，Beego `RunSyncdb` 自动 ALTER，无新迁移 SQL；SQLite 升级测试已含两列 |
| 前端展示 | ✅ 复用既有 Task 序列化，无新 API；`tool_catalog_hash`/`skill_package_hash` 展示 + i18n 完整 |
| 后端测试（`-count=1 -race`） | ✅ `agent/eval`、`agent/replay`、`agent/runtime`、`agent/manager`、`agent/task`、`agent/toolruntime`、`agent/models`、`agent/llm` 全 `ok` |
| 前端 TypeScript 检查 | ✅ `vue-tsc --noEmit --skipLibCheck` EXIT=0 |
| **V2-4 Gate（6 项）** | ✅ **6/6 达成，无阻塞缺陷，可进入 V2-5** |

本轮共发现 **0 个阻塞项、0 个重要项、4 个建议项、3 个提示项**，均不影响 Gate 判定，详见第四节。

---

## 二、Gate 6 项逐项核对

| # | Gate 项（文档 §验收） | 实现证据 | 结论 |
|---|----------------------|----------|------|
| 1 | 核心 Skill 有自动 Eval | `agent/eval/eval_test.go:35 TestCoreSkillEvalGate` 直接进 `go test`；`testdata/core/` 4 个 `agent_eval_v1` Case（market_regime/strategy_builder/symbol_analysis/alert_analysis）复用 V2-0 Replay Fixture | ✅ |
| 2 | Model/Prompt/Skill revision 可对比 | `compare.go` `Compare` 输出 `score_from/to/delta` + `VersionDifferences`；`diff.go` `CompareVersions` 已含 `tool_catalog_hash`/`skill_package_hash`；`eval_test.go:105 TestRevisionComparisonTracksSkillAndPackageIdentity` 验证 ModelConfigID+Prompt/Skill 变更后 diff≥5 且含 `skill_package_hash` | ✅ |
| 3 | MCP 和 Portable Skill 有专门 synthetic 回归 Case/维度 | `eval_test.go:56 TestMCPFailureRecoveryAndPermissionEscalationDimensions`（MCP 失败恢复）、`:72 TestPortableInstructionAndRouterDimensions`（portable instruction + router）；scorer `mcp_failure_recovery`/`imported_skill_compliance`/`router_selection` 维度 | ✅ |
| 4 | 关键退化可通过 Go Test CI Gate 阻止发布 | `gate.go` `Gate`/`RequireGate`：`MinimumScore` 默认 80、`MaxScoreRegression` 默认 5、Critical Failure 无论总分阻断；`eval_test.go:82 TestGateRejectsCriticalRegression` 验证临界回归与 >5 分回归均被拒 | ✅ |
| 5 | 历史 Task 保存 Tool Catalog / Skill Package hash | `models/agent_task.go` 两列 `size(64)`；`task/task.go` `ApplyVersionMetadata`/`VersionMetadata()` 往返；`orm_store.go` `toModel`/`fromModel` 落库；`manager.go:92` / `coordinator.go:98` 写入；`manager_test.go` 断言 64 位 | ✅ |
| 6 | Shadow 不写 Task DB、不允许 write/trade Tool | `shadow.go RunShadow`：强制 `task.NewMemoryStore()`（不写真实 DB）、`permission.AllowReadOnly()`、`config.ToolRuntime=nil`、清空 hooks；启动前遍历 `definition.Tools()` 拒绝 `Risk!=Read || !Idempotent`；`eval_test.go:132 TestShadowRejectsUnsafeToolsAndUsesMemoryTaskStore` | ✅ |

---

## 三、后端 Review

### 3.1 Eval Framework 新包 `agent/eval/`

| 文件 | 核对结果 |
|------|----------|
| `types.go` | `CaseVersion="agent_eval_v1"`；`FactRule`（Path/Equals/Contains/Exists/Critical）、`Expectations`（status/contract/required+forbidden facts/tools/evidence/repair/token/duration/freshness/mcp/instruction/router/permission）、`Dimension`、`Report` 字段与文档 §Eval Case / §维度一致 |
| `load.go` | `Load` 校验 `version==agent_eval_v1` 且 name/skill 非空；`LoadDir` 按文件名排序保证可重复；`FixturePath` 相对 SourcePath 解析 |
| `runner.go` | `Run` 经 `replay.Load`→`Evaluate`；`Evaluate` 校验 case/fixture/definition 三者 skill 一致，否则返回 error Report；复用 `replay.Run`（固定 LLM/Tool，不连真实服务） |
| `scorer.go` | 13 维度 `weights`（structure15/facts20/evidence15/tool_selection15/freshness10/repair5/token5/duration5/mcp5/imported5/router5/security10，合计 115，归一化 `got*100/max`）；`report.Passed = CriticalFailures==0 && Score>=80`；critical 维度或 critical fact 失败即进 `CriticalFailures`；`taskAudit` 从 `item.Steps` 解析 `toolruntime.Trace` 与 `contextengine.Evidence`；`matchesRule`/`lookupPath`（支持 `.` 与 `[i]` 路径）/ `outputMentions` 实现事实/证据/诚实度匹配 |
| `compare.go` | `Comparison` 含 `score_delta` + `VersionDifferences`（`replay.CompareVersions`）；维度 delta 仅列变化项 |
| `gate.go` | `Gate` 默认 `MinimumScore=80`/`MaxScoreRegression=5`；单报告未过/低于最低分/有 critical 失败，或任一 comparison 回归超阈值 → `Passed=false`；`RequireGate` 返回 error 阻断 CI |
| `shadow.go` | `RunShadow` 拒绝 `client==nil`/portable-imported skill（V2-6 前）/unsafe tool；强制 MemoryStore + 只读 Policy + nil ToolRuntime + 清空 hooks；`req.TaskID=""` 强制新 ID，结果不落真实 DB |
| `time.go` | `var now = time.Now`（可注入，便于测试） |
| `eval_test.go` | 8 个测试覆盖 4 核心 Case + MCP recovery + 权限提升 + portable/router + 临界回归 + freshness + revision 对比 + shadow 拒绝/接受 |

**Scoring 关键一致性**：`tool_permission_denied` 阶段字符串（`react_executor.go:157/279`）与 scorer `security` 维度比对一致；`SourceMCP="mcp"`（`toolruntime/types.go:15`）与 `mcp_failure_recovery` 维度比对一致；`Task.Stage`(task.go:64)/`SkillSource`(task.go:85)/`VersionMetadata()`(task.go:114) 均存在，scorer 引用全部可解析。

### 3.2 Tool Catalog / Skill Package Hash

| 审查项 | 文件 / 行号 | 结论 |
|--------|-------------|------|
| `ToolCatalogHash` 计算 | `toolruntime/catalog.go:17 CatalogHash` | ✅ 去重+排序 names；命中 `Descriptor` 取副本，未注册则 `Missing:true`；整体 `json.Marshal`→SHA-256 hex。符合文档「未注册但 Skill 声明的 Tool 也以 missing identity 进入 hash」 |
| `SkillPackageHash` 计算 | `skill/version.go:39 PackageHash` | ✅ 确定性 SHA-256，输入含 skill_name/version/prompt_version/prompt_hash/input+output_contract/source/source_version/tools(排序)；符合文档 §skill_package_hash |
| 冻结入口 | `runtime/version.go:28 FreezeExecution` 设 `SkillPackageHash`；`runtime/runner.go:80 FreezeExecution` 方法（在 `NewRunner` 之后，能拿到 `cfg.ToolRuntime`）设 `ToolCatalogHash` | ✅ 顺序正确：先 `NewRunner` 注入 `ToolRuntime`，再算 catalog hash |
| 落库链路 | `task/task.go:107 ApplyVersionMetadata`（coordinator.go:98 / manager.go:92 调用）→ `Task.ToolCatalogHash/SkillPackageHash` → `orm_store.go toModel/fromModel` → `models.AgentTask` 两列 | ✅ 全链路往返，DB 持久化成立 |
| Resume 一致性 | `coordinator.go:329` `validateResumeIdentity` 新增 `stored.SkillPackageHash != actual.SkillPackageHash \|\| stored.ToolCatalogHash != actual.ToolCatalogHash` → 拒绝 unsafe resume | ✅ 历史 Task 不会仅凭同名 Skill/Tool 被误判可复现 |
| 旧快照回填 | `coordinator.go:67`（`snapshot!=nil` 分支）若 `SkillPackageHash`/`ToolCatalogHash` 为空则回填当前值 | ✅ 兼容旧序列化快照 |

### 3.3 Runtime 身份双闸门

- `runtime/version.go:9` `CurrentVersion="2.3.0"`；`runtime/state.go:93` `runStateVersion="runtime_state_v4"`。
- `coordinator.go:315` 仍拒绝 `item.RuntimeVersion != CurrentVersion || state.Snapshot.Version.RuntimeVersion != CurrentVersion`。
- `checkpoint.go`（沿用 V2-3）拒绝 `state.Version != runStateVersion || !ResumeSafe`。
- V2-1/V2-2/V2-3 checkpoint（≤`2.2.0`/`runtime_state_v3`）被 `2.3.0`/`runtime_state_v4` 正确拒绝，符合预期。

### 3.4 Replay V2-4 扩展

| 审查项 | 文件 / 行号 | 结论 |
|--------|-------------|------|
| `ToolMetadata` 结构 | `replay/fixture.go:30` | ✅ 含 risk/idempotent/source_type/provider_ref/timeout/cache/schemas，复用 `agent_replay_v1` 兼容 |
| `fixtureTool` 安全默认 | `replay/tools.go` `Risk()` 默认 `RiskRead`；`Metadata()` 透传 schema/timeout/max_result_bytes/idempotent/source/provider/cache，idempotent 默认 true | ✅ 默认只读，synthetic `risk=trade`/`source=mcp` 可声明以验证恢复/拒绝 |
| 只读策略 | `replay/runner.go:43` `Policy: permission.AllowReadOnly()` | ✅ 安全回归测试基础：LLM 伪造 `risk=read` 不被信任，实际 `RiskTrade` 工具被拒绝，任务落 `tool_permission_denied` |
| MemoryStore | `replay/runner.go:36` | ✅ 回放不写真实 DB / 不连真实 LLM·Binance·交易 |
| 版本 diff 扩展 | `replay/diff.go:27` 新增 `tool_catalog_hash`/`skill_package_hash` 比较 | ✅ |

### 3.5 数据库变更（additive）

- `models/agent_task.go` 新增 `ToolCatalogHash`/`SkillPackageHash`（`orm:"column(...);size(64)"`，JSON tag 同），无其它字段变更。
- 无新增 `command/sql/version/2.sql`；依赖 Beego `RunSyncdb` 自动 `ALTER TABLE ADD COLUMN`（additive，存量行新列为 NULL 不破坏）。
- `models/agent_task_syncdb_test.go`：`TestAgentTaskSyncdbUpgradesExistingSQLiteRows` 已把两列纳入 `requireAgentColumns` 期望集合，并插入/读回 64 位 hash 断言 round-trip，验证升级兼容。

### 3.6 前端 Review

| 文件 | 核对结果 |
|------|----------|
| `src/api/agent.ts` | `AgentTask` 接口加 `tool_catalog_hash?`/`skill_package_hash?`；复用既有 Task 详情序列化（无新 API，符合文档 §前端） |
| `src/views/ai/taskCenter.vue` | 详情 `el-descriptions` 新增两项展示 `detail.tool_catalog_hash`/`detail.skill_package_hash`（空显 `-`） |
| `locales/zh-CN.yaml` / `locales/en.yaml` | 各 +2 key：`detail.toolCatalogHash`/`detail.skillPackageHash`；vue 引用的 `t('agentTaskCenter.detail.toolCatalogHash'/*skillPackageHash)` 均存在 |

前端 `vue-tsc --noEmit --skipLibCheck` 退出码 **0**，无类型错误。

---

## 四、发现项

### 阻塞项（0）
无。

### 重要项（0）
无。

### 建议项（4，均不阻塞 Gate）
- **S1（评分权重非 100 基准）**：`scorer.go weights` 合计 115，报告分数经 `got*100/max` 归一化到 100。行为正确，但单看 `weights` 易被误读为百分比。建议在 `weights` 上方注释「权重为相对值，最终分数归一化到 100」。
- **S2（Evidence 取自输出 JSON）**：`outputEvidenceSources(root)` 从**最终输出 JSON** 的 `evidence` 数组读取 source，而非 runtime 追踪的 `step.Evidence`。core Case `alert_analysis`/`symbol_analysis` 已设 `required_evidence_sources` 且测试通过，说明对应 Skill 输出内嵌了 `evidence`。该契约合理，但建议在 Eval 文档中明确「`required_evidence_sources` 校验的是 Skill 最终输出中的 evidence 数组」，以便后续 Case 正确编写。
- **S3（CatalogHash 含 Descriptor 副本）**：`catalog.go:35 copy := descriptor` 取指针副本再 `&copy`，避免后续 mutation 影响 registry 中原始 descriptor。实现正确；仅提示确认 `ToolDescriptor` 未来若增可变字段需同步副本语义。
- **S4（Shadow 显式置 nil ToolRuntime）**：`shadow.go:47 config.ToolRuntime = nil`，但 `NewRunner` 对 nil `ToolRuntime` 会重建默认实例（`runtime/runner.go` V2-3 逻辑），故 `=nil` 为冗余。可简化为不设置（或显式注释意图为「禁用缓存」）。

### 提示项（3）
- **N1（macOS 链接告警）**：后端 `-race` 出现 `ld: warning: ... malformed LC_DYSYMTAB`，为本地 macOS 链接器告警，非测试失败（全部包 `ok`）。CI（Linux）不受影响。
- **N2（生产部署需触发一次 Syncdb）**：additive 迁移依赖 Beego `RunSyncdb` 自动 ALTER；上线部署时应确保 `RunSyncdb` 被执行一次，使存量 `agent_tasks` 获得两新列。测试已验证列存在与 round-trip。
- **N3（前端仅展示、无新 API）**：两 hash 经既有 Task 序列化下发，无需新增接口，符合文档约定。

---

## 五、测试结论

**后端**
- `go test -count=1 -race` 以下包全部 `ok`（含关键包）：
  - `agent/eval`（2.587s，含 `TestCoreSkillEvalGate` 直接进 CI Gate）
  - `agent/replay`（1.809s，V2-0 Replay baseline + 版本 diff 扩展）
  - `agent/runtime`（2.433s）、`agent/manager`（2.056s，断言两 hash 冻结）
  - `agent/task`（2.368s）、`agent/toolruntime`（2.737s）
  - `agent/models`（2.896s，SQLite 升级测试含两列 round-trip）
  - `agent/llm`（3.131s）
- 全模块 `go build ./...` 编译通过（测试运行即隐含）。

**前端**
- `vue-tsc --noEmit --skipLibCheck` → 退出码 0。

---

## 六、Gate 判定

V2-4 六项 Gate 全部达成（6/6），无阻塞项、无重要项。Eval Framework、`tool_catalog_hash`/`skill_package_hash` 冻结与落库、Runtime 身份双闸门升级、Replay synthetic MCP/portable 扩展、additive DB 变更、Shadow 安全隔离、前端展示均验证通过，测试全绿。

**结论：V2-4 满足 Gate，可进入 V2-5（MCP 集成）。**
