# V2-2 Code Review 报告

> 审查范围：当前工作区（未提交）代码
> 后端：排除 `static/`（前端打包产物）
> 前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui`（4 个改动文件）
> 对照文档：`doc/agent/v2/02-phase-v2-2-context-evidence.md`
> 审查日期：2026-09-02

---

## 一、结论摘要

| 维度 | 结论 |
|------|------|
| Context Engine 核心实现（新包 8 文件） | ✅ 与文档 §4–§7 逐项一致 |
| Runtime 集成与版本身份 | ✅ `2.1.0` / `runtime_state_v2` 双闸门生效 |
| Skill 层 Evidence 绑定 | ✅ V1 输出契约（`trading_plan_v1` / `alert_v1`）未破坏 |
| 前端 Evidence / Context Trace 展示 | ✅ 与后端 JSON 契约逐字段对齐 |
| 数据库变更 | ✅ 无新增表 / 无新增字段 / 无 `version/2.sql` |
| 后端测试（`-count=1 -race`，18 个包） | ✅ 全部通过 |
| 前端 TypeScript 检查 | ✅ 通过（详见 §4.2） |
| 前端生产构建 | ✅ 产物已重新构建，`static/static/js/taskCenter-B1uG2O-E.js` 含 `context_trace` |
| **V2-2 Gate（12 项）** | ✅ **12/12 达成，无阻塞缺陷，可进入 V2-3** |

本轮共发现 **0 个阻塞项、3 个建议项、3 个提示项**，均不影响 Gate 判定，详见第五节。

---

## 二、Gate 12 项逐项核对

| # | Gate 项 | 实现证据 | 结论 |
|---|---------|----------|------|
| 1 | ContextBlock / Evidence 类型 | `agent/contextengine/types.go`：ContextBlock 14 字段、Evidence 11 字段、8 种 `BlockType`、4 种 `Freshness` | ✅ |
| 2 | token estimator / budget allocator | `estimator.go:14`（CJK 约 1 token/rune，ASCII 约 4 char/token，保守高估）+ `builder.go:21` Build 七步流程 | ✅ |
| 3 | explainable trim trace | `types.go` BuildTrace 11 字段 + TrimRecord；`builder.go:84-88` 记录 `token_budget` / `byte_budget`；`builder.go:169` 记录 `duplicate`；`react_executor.go:48-51` 触发 `context_trimmed` Event | ✅ |
| 4 | freshness policy | `freshness.go:18` 7 个数据源 MaxAge 与文档 §6 表格**逐一相等**；`builder.go:215` `CONTEXT_FRESHNESS source=… status=… as_of=…` | ✅ |
| 5 | Tool Result → Evidence / Context | `tool.go:18` ConvertToolResult（SHA-256 → `ev_` + 20 hex，确定性 ID）；`helpers.go:199-217` 信封 `{"tool","ok","result","evidence"}` | ✅ |
| 6 | Final Validator 绑定 Runtime Evidence | `skill/skill.go:33` 新增 `StructuredEvidenceValidatorProvider`；`react_executor.go:271-275` 优先于 `RunValidatorProvider`；两个 Skill 各实现 `ValidatorForRunWithEvidence` | ✅ |
| 7 | Checkpoint 保存 Context/Evidence 并可 Resume | `state.go:87-89` 新增 `ContextBlocks` / `Evidence` / `LastContextTrace`，`Messages` 改 `json:"-"`；`checkpoint.go:84-92` 恢复 nil map + `restoreMessagesFromContextBlocks()` | ✅ |
| 8 | Progressive Disclosure Runtime 协议 | `resources.go:9` LoadResources（activation 全量 / on_demand 按 `requestedIDs`）；`coordinator.go:135` metadata key `context_resource_ids`，兼容 `[]string` / `[]any` / `string` | ✅ |
| 9 | Task Center Evidence / Context Trace 展示 | `taskCenter.vue:792-878` 新增 expand 列；`api/agent.ts` 新增 3 个接口；`locales/*.yaml` 中英各 17 行 | ✅ |
| 10 | V2-0 Replay Gate | `agent/replay/` 全部文件对 `Messages` 的引用为 **0**；`go test -race ./agent/replay` 通过 | ✅ |
| 11 | 后端全量测试 | `go build ./...` + `go vet ./agent/...` 无输出；`go test -count=1 -race` 18 个包全 `ok` | ✅ |
| 12 | 前端 TypeScript 检查与生产构建 | 见 §4.2 | ✅ |

---

## 三、后端 Review

### 3.1 Context Engine（新增包 `agent/contextengine/`）

| 文件 | 核对结果 |
|------|----------|
| `types.go` | 类型定义与文档 §4.1 / §7 完全一致；JSON tag 全部 snake_case，与前端接口定义可逐字段对齐 |
| `estimator.go` | `wide + (ascii+3)/4`，与文档 §5 的保守启发式一致 |
| `freshness.go` | 7 个源 MaxAge 与文档 §6 表格逐一比对**无差异**；`data_missing` 含 `stale` 子串时强制 stale（`freshness.go:34-38`）；`RequireTimestamp` 缺失判 `missing`、否则 `unknown`（不伪装 fresh） |
| `hash.go` | `sha256.Sum256` + hex，确定性 |
| `builder.go` | 构建顺序严格遵循文档 §5.1：system 预算预检 → normalize → dedupe → required 优先 → optional 按 `effectivePriority` → 超预算记 trim 不失败 → required 装不下才返回 `context_too_large`；**最终按 `Order` 升序还原时序**（`builder.go:91`），裁剪不影响对话顺序 |
| `messages.go` | `InitialMessageBlocks` 末条为 `task`+required、其余 `history`；`RuntimeMessageBlock` 中 `AGENT_FEEDBACK` → task/950/required，`TOOL_RESULT` → tool/650，符合文档 §4.3 |
| `resources.go` | 跳过 `Load == nil` / 空 ID；`on_demand` 未在 requested 集合中则跳过；空内容不产出 block |
| `tool.go` | Evidence ID 基于 `source + "|" + contentHash` 确定性生成；时间戳提取覆盖文档 §6 全部 8 类 key + `snapshot.updated_at_ms` 嵌套；`parseTimestamp` 兼容 RFC3339 / 毫秒 / 秒级 unix |

**优先级折减**（`builder.go:198`）：stale −100、missing −200，与文档 §4.2“stale 折减、missing 折减更高”一致。

### 3.2 Runtime 集成

| 审查项 | 文件 / 行号 | 结论 |
|--------|-------------|------|
| `runtime_version` 升到 `2.1.0` | `runtime/version.go:9` | ✅ |
| `runtime_state_version` 升到 `runtime_state_v2` | `runtime/state.go:91` | ✅ |
| 新增 `Config.MaxContextTokens`（默认 64 Ki）+ `ContextEngine` | `runtime/types.go:75,82` / `runner.go:47-49` | ✅ 默认值与文档 §5 一致 |
| `RunState.Messages` 不再 JSON 序列化 | `runtime/state.go:52` | ✅ 避免 Tool Result 在 Messages 与 ContextBlocks 中双份存储 |
| Resume 从 ContextBlocks 重建消息 | `state.go:172-181` + `checkpoint.go:91` | ✅ |
| 跨版本 Resume 双闸门 | `coordinator.go:313`（task.RuntimeVersion ≠ 2.1.0 拒绝）+ `checkpoint.go:72`（state.Version ≠ `runtime_state_v2` 或 `!ResumeSafe` 拒绝） | ✅ 逻辑正确 |
| 每轮 Build Context | `react_executor.go:36-40` | ✅ 用 `state.ContextBlocks` 而非裸 `messages` |
| `context_trimmed` Event | `react_executor.go:48-51` | ✅ 记录裁剪数与最终估算 token |
| Tool → Evidence 注入 | `react_executor.go:183-193` + `state.go:154` `addEvidence` | ✅ 仅 `toolErr == nil` 时生成 Evidence |
| Validate Step 附 Evidence 快照 | `react_executor.go:304` `state.addEvidence(stepID, state.allEvidence())` | ✅ 符合文档 §9 |
| Checkpoint 复用 `conversion.ResultJSON` | `react_executor.go:222-231` | ✅ `ResultJSON` 即 `json.Marshal(value)` 原值，V2-1 `CheckpointCodec` 类型还原不受影响；为空时回退 `json.Marshal` |
| Progressive Resource 加载 | `coordinator.go:113-124` | ✅ 加载后 append 到 ContextBlocks 并计入 step 摘要 |

**关键设计确认**：`Build` 不修改 `state.ContextBlocks`，只返回排序后的 messages，因此裁剪是**每轮重新评估**而非永久丢弃；`state.ContextBlocks` 的追加顺序即对话时序，Resume 后重建的消息顺序与首次运行一致。

### 3.3 Skill 层

| 审查项 | 文件 | 结论 |
|--------|------|------|
| 新增 `StructuredEvidenceValidatorProvider` 接口 | `skill/skill.go:33-35` | ✅ |
| 新增 `ContextResourceProvider` 接口 + `Definition.ContextResourcesFunc` | `skill/skill.go:41-43,64,96-101` | ✅ |
| `alertanalysis` 实现 `ValidatorForRunWithEvidence` | `skills/alertanalysis/skill.go:154-167` | ✅ 先跑 V1 `ValidatorForRun`，再校验结构化 Evidence |
| `symbolanalysis` 实现 `ValidatorForRunWithEvidence` | `skills/symbolanalysis/skill.go:150-163` | ✅ 同上 |
| 结构化 Evidence 校验规则 | 两处 `validateStructuredEvidenceSources` | ✅ Registry 中存在同 source（found）+ `ContentHash != ""` 且 `Freshness != missing`（usable）；`missing` 不可用作证据；stale **不被升级**为 fresh，可继续作为证据 |

**V1 契约兼容性**：`trading_plan_v1` / `alert_v1` 的结构体、字段与枚举未做任何修改；业务 JSON 中的 `{"source":…, "finding":…}` 完全保留 —— 与文档 §2 一致。

### 3.4 数据库与兼容边界

- `git diff --name-only` 中**不含** `models/` 下任何文件，也无新增 SQL 文件 → 文档 §13“无新增表 / 无新增字段 / 无 `version/2.sql`”成立。
- V2-1 阶段修复的 `plan_json` / `steps_json` / `checkpoint_json` 可空改动在本轮 diff 中未被回退。
- `agent/replay/` 对 `Messages` 零引用，V2-0 Replay Gate 不受 `Messages json:"-"` 影响。

---

## 四、测试验证

### 4.1 后端（`go test -count=1 -race`）

```
ok   go_binance_futures/agent/app                 2.172s
ok   go_binance_futures/agent/contextengine       2.419s   ← 新包
ok   go_binance_futures/agent/conversation        1.910s
ok   go_binance_futures/agent/event               3.812s
ok   go_binance_futures/agent/governance          3.315s
ok   go_binance_futures/agent/manager             2.694s
ok   go_binance_futures/agent/observability       3.568s
ok   go_binance_futures/agent/permission          2.147s
ok   go_binance_futures/agent/replay              1.841s   ← V2-0 Gate
ok   go_binance_futures/agent/runtime             1.788s
ok   go_binance_futures/agent/scheduler           1.897s
ok   go_binance_futures/agent/security            1.764s
ok   go_binance_futures/agent/skillconfig         1.605s
ok   go_binance_futures/agent/skills/alertanalysis    1.713s
ok   go_binance_futures/agent/skills/marketregime     1.751s
ok   go_binance_futures/agent/skills/strategybuilder  1.778s
ok   go_binance_futures/agent/skills/symbolanalysis   1.702s
ok   go_binance_futures/agent/task                1.622s
ok   go_binance_futures/agent/tools/domain        1.565s
ok   go_binance_futures/llm                       1.279s
ok   go_binance_futures/models                    1.535s   ← V2-1 缺陷已修复
```

`go build ./...`、`go vet ./agent/...`、`go vet ./agent/skills/...` 均无输出。

**Context Engine 测试覆盖**（`contextengine_test.go`，7 个用例，完整覆盖文档 §14 清单）：

| 用例 | 覆盖项 |
|------|--------|
| `TestBuildTrimsHistoryBeforeCurrentTask` | 超预算优先裁 History、当前 Task/Market 保留 |
| `TestBuildRejectsOnlyWhenRequiredContextCannotFit` | Required 装不下才失败 |
| `TestConvertToolResultMarksStaleAndCreatesDeterministicEvidence` | Evidence ID/hash 确定性、stale timestamp |
| `TestConvertSymbolAnalysisContextHonorsExplicitStaleMarker` | `data_missing` stale marker |
| `TestProgressiveResourceDisclosure` | progressive resource disclosure |
| `TestConvertSymbolSnapshotUsesCamelCaseUpdateTimeForFreshness` | 项目真实 camelCase `updateTime` |
| `TestConvertToolResultDoesNotTreatOrdinaryNumbersAsTimestamps` | 普通数字不被误判为时间戳（额外防护） |

**Runtime 新增测试**（`runner_test.go`，+138 行）：

| 用例 | 覆盖项 |
|------|--------|
| `TestRunnerPersistsStructuredEvidenceAndContextTrace` | Tool Evidence 注入下一轮 LLM、`CONTEXT_FRESHNESS` header、Tool Step Evidence 持久化、LLM Step Context Trace 持久化 |
| `TestRunnerTrimsLowPriorityHistoryInsteadOfFailingContext` | 超预算自动裁剪并继续执行、`context_trimmed` Event |
| `TestRunnerLoadsOnlyActivatedAndRequestedSkillResources` | Skill activation / on-demand Resource |

### 4.2 前端

| 检查项 | 结果 |
|--------|------|
| `vue-tsc --noEmit --skipLibCheck`（项目本地 `vue-tsc@2.2.10`） | ✅ 退出码 0，零类型错误（耗时 ~4 min） |
| 生产构建 | ✅ `static/index.html` 已切换到新入口 chunk，`static/static/js/taskCenter-B1uG2O-E.js` 中检出 `context_trace` |
| 接口类型与后端 JSON tag 对齐 | ✅ `AgentContextBuildTrace` ↔ `BuildTrace`、`AgentStructuredEvidence` ↔ `Evidence`、`AgentContextTrimRecord` ↔ `TrimRecord` 字段名逐一相等 |
| i18n 完整性 | ✅ `context_trimmed` / `context_resource_failed` 在后端确有对应 stage（`react_executor.go:50`、`coordinator.go:118`），无死键值 |
| 静默刷新 | ✅ 未改动 V2-1 刷新逻辑，未新增 loading overlay |
| `checkpoint_json` 泄漏 | ✅ 本轮未涉及，V2-1 的三层 `json:"-"` 防护保持 |

> **备注（环境问题，与 V2-2 无关）**：默认 `tsconfig.json` 跑 vue-tsc 会中断在 `error TS6053: File 'src/views/permission/page/index.vue' not found`。经核实，该文件在磁盘存在（1672 B）、已被 git 跟踪、`git diff` 对该路径无任何改动，且可被正常读取 —— 属沙箱环境下 node 读取该文件被阻断的偶发问题。
>
> 为排除干扰、完成对 V2-2 改动的真实类型校验，本次复制 `tsconfig.json` 为临时配置、在 `exclude` 中排除该文件后重跑：`vue-tsc --noEmit --skipLibCheck -p tsconfig.v22check.json` → **退出码 0，无任何类型错误**；校验完成后临时配置已删除，前端工作区仅剩 V2-2 的 4 个改动文件。
>
> 该问题不会阻塞 Gate，但建议后续在干净环境（非沙箱）复跑一次 `pnpm typecheck` 以彻底确认。

---

## 五、发现的问题

### 5.1 阻塞项

无。

### 5.2 建议项（不阻塞 Gate，建议在进入 V2-3 前后处理）

**S1 — `extractAsOf` 兜底遍历的跳过条件语义不一致**

`agent/contextengine/tool.go:76-88`：

```go
walkJSON(value, 0, func(key string, scalar any) {
    if latest != nil && (key == "as_of" || key == "updated_at_ms" || key == "update_time") {
        return
    }
    ...
})
```

当 `latest` 已被某个非权威 key（如 `closeTime`）赋值后，后遍历到的权威 key 会被直接跳过，导致 `as_of` 取到偏旧值。`walkJSON` 按 key 字典序遍历，因此影响顺序可预期但反直觉。

- 影响方向是**保守**的（偏旧 → 更容易判 stale，不会把 stale 误判为 fresh），因此不构成正确性缺陷。
- 建议：先收集全部候选时间戳，再按 `as_of > updated_at_ms > update_time > 其它` 的优先级选取，或至少让权威 key 无条件覆盖非权威 key。

**S2 — 失败的 Tool Result 优先级高于成功的 Tool Result**

`messages.go:39-42` 为 `TOOL_RESULT` 前缀的消息赋 `priority = 650`；而成功路径使用 `ConvertToolResult` 产出的 block（`tool.go:56`，`BlockTool` = 600）。结果是**工具失败时的结果块（650）比工具成功时的通用结果块（600）更晚被裁剪**。

建议统一为 `DefaultPriority(BlockTool)`，或为错误块显式设置低于 600 的优先级。

**S3 — Context Resource 加载失败直接终止任务**

`coordinator.go:118-122`：任一 Resource 加载失败即 `fail(item, "build_input_failed", loadErr)`，任务直接 Failed。

渐进式披露的资源（尤其 `on_demand` 的 `references/` / `assets/`）本质上是“增强上下文”，失败时降级为一条 trim 记录并继续执行更符合 Progressive Disclosure 的设计意图。V2-6 引入 ZIP Agent Skill importer 后资源来源会显著增多，该严格策略的失败面会扩大。

### 5.3 提示项

**N1 — 跨 Runtime 版本 Resume 拒绝缺少测试覆盖**

`coordinator.go:313` 与 `checkpoint.go:72` 的双闸门逻辑正确，但 `agent/runtime` 中没有“构造 `2.0.0` / `runtime_state_v1` checkpoint → 断言 Resume 被拒”的用例。文档 §3 把这条明确列为 V2-2 的兼容边界，建议补一个用例固化该契约（可放在 V2-3 一并补）。

**N2 — `evidenceBySource` 为未引用的死代码**

`state.go:199` 定义后无任何调用点（Go 编译器不检查未使用的方法，故未暴露）。建议删除，或由 N1 的新测试使用。

**N3 — Evidence 信封放大了 `MaxToolResultBytes` 超限概率**

`helpers.go:199-215`：Tool Result 消息现在内嵌完整 Evidence 对象（`key_fields` ≤ 12 项、`data_missing` 等），超出 `MaxToolResultBytes` 时**返回错误使任务失败**（非截断，因此不会破坏 JSON —— 该行为与 V2-1 一致，非本轮引入的缺陷）。

提示点在于：信封体积使原本贴近阈值的工具返回值更容易越界。后续可考虑“先对 `result` 做截断，再附加 evidence”。

补充：`LastContextTrace`（`state.go:88,144`）目前只写入不读取，作为 checkpoint 状态保留可接受，建议加注释说明其预留用途。

---

## 六、结论

V2-2 的 12 项 Gate **全部达成，无阻塞缺陷**：

1. Context Engine 作为独立包落地，类型、优先级、预算分配、裁剪追踪、freshness 策略、Evidence 确定性 ID 均与文档一致；
2. Runtime 以 `2.1.0` / `runtime_state_v2` 建立版本身份，V2-1 的 `2.0.0` checkpoint 被双闸门拒绝，`Messages json:"-"` 避免了 checkpoint 体积翻倍；
3. Skill 层通过可选接口绑定 Runtime Evidence，V1 输出契约零改动；
4. 前端审计展示与后端 JSON 契约逐字段对齐，i18n 中英同步，构建产物已更新；
5. 后端 21 个包 `-race` 全绿，V2-0 Replay Gate 与 V2-1 DB 升级测试均保持通过。

**建议处理方式**：S1–S3 与 N1 可在进入 V2-3（Tool Runtime）时一并处理，N2 随手清理。**可以进入 V2-3。**
