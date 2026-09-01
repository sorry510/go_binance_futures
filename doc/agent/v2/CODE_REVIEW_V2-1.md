# V2-1 Code Review 报告

> 审查范围：当前工作区（未提交）代码
> 后端：排除 `static/`（前端打包产物）
> 前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui`（master 分支，4 个改动文件）
> 对照文档：`doc/agent/v2/01-phase-v2-1-runtime.md`
> 审查日期：2026-09-01

---

## 一、结论摘要

| 维度 | 结论 |
|------|------|
| 后端架构 / Resume 安全规则 | ✅ 符合文档 |
| 后端 Cancel/Resume 接口 | ✅ 符合文档 |
| 后端单元测试（-race） | ⚠️ 5/6 包通过，`models` 包 **FAIL（阻塞）** |
| 前端类型 / API / 展示 | ✅ 与后端契约一致，无阻塞问题 |
| 前端 locale 枚举 | ✅ 与后端输出字符串一一对应 |
| **V2-1 Gate** | ❌ **未通过**：存在 1 个阻塞级 DB 升级缺陷，需修复后方可进入下一 Phase |

---

## 二、后端 Review（排除 static/）

### 2.1 架构与实现（符合文档要求）

| 审查项 | 文件 | 结论 |
|--------|------|------|
| Runtime 拆分为 Coordinator / Executor / RunState / ExecutionStep / Checkpoint | `agent/runtime/{coordinator,state,checkpoint,plan,plan_executor,react_executor}.go` | ✅ 结构清晰 |
| Resume 安全规则：仅 `Idempotent=true && Risk==read` 的工具可建立安全 Checkpoint；unsafe 工具执行前 `clearCheckpoint` | `react_executor.go:142-147` / `checkpoint.go` | ✅ 正确 |
| 版本 / 身份校验：`validateResumeIdentity` 校验 `runtime_version`、冻结 `model_config_id`、skill / contract / source 版本 | `coordinator.go` | ✅ 正确 |
| Planner 防绕过：`Planner` 接口仅 `Plan()`，无 Tool registry / Permission；Plan 步骤统一走 `react.executeTool` | `plan.go` / `plan_executor.go` | ✅ 正确 |
| 冻结 model config：Resume 使用 `NewClientByID(item.ModelConfigID)` 而非活跃配置 | `manager.go:153` / `llm/client.go` | ✅ 正确 |
| Cancel / Resume API：`POST /agents/tasks/:taskId/cancel`、`/resume`，异步返回 accepted | `controllers/agent_task.go` / `routers/router.go` | ✅ 正确 |
| `checkpoint_json` 三层 `json:"-"` 防泄漏（model / task 视图 / API 输出） | `models/agent_task.go:24` / `agent/task/task.go` / `agent/task/orm_store.go` | ✅ 正确 |
| `runtime_version` 写入 `CurrentVersion`（"2.0.0"），支撑前端版本门控 | `runtime/version.go:22` | ✅ 正确 |
| 超时失败判定 `Status==failed && Stage=="timeout"` | `coordinator.go:261` | ✅ 与前端一致 |

### 2.2 测试结论（-race）

```
ok   go_binance_futures/agent/runtime    2.959s
ok   go_binance_futures/agent/replay     4.789s
ok   go_binance_futures/agent/manager    5.039s
ok   go_binance_futures/agent/task       4.921s
ok   go_binance_futures/llm              5.150s
--- FAIL: go_binance_futures/models       (2 测试失败，见第三节)
```

---

## 三、阻塞缺陷：SQLite NOT NULL 升级失败（Gate 阻塞）

### 3.1 现象

`models` 包两个测试失败：

```
--- FAIL: TestAgentTaskSyncdbAddsNewColumnsToExistingTable
    agent_task_syncdb_test.go:141: RunSyncdb failed: Cannot add a NOT NULL column with default value NULL
--- FAIL: TestAgentTaskSyncdbPreservesExistingRows
    panic: <orm.RegisterModel> model `...models.AgentTask` repeat register, must be unique
```

第二个测试的 panic 是第一个测试失败后在**同一进程内** ORM model 重复注册导致的连锁污染，根因同一下。

### 3.2 根因

`models/agent_task.go` 在 V2-1 为 `AgentTask` 与 `AgentTaskEvent` 新增了若干列，但**非 text 列未声明 `default`**。

beego ORM 对未声明 `default` 的列生成 `NOT NULL` 且无默认值的 DDL；SQLite 在**表已存在行**时拒绝执行 `ALTER TABLE ADD COLUMN ... NOT NULL`（无默认值）。`RunSyncdb(alias, false, false)` 的升级逻辑因此中断。

> 代码注释（line 18-21）已识别出 text 列的同类风险并为 `plan_json` / `steps_json` / `checkpoint_json` 加了 `default('')`，但**遗漏了 string / int / bool 类型的非 text 新增列**。

`models` 包的 `RunSyncdb` 错误在 `main.go` 中被**忽略**，意味着旧部署升级时这些列会**静默缺失**，运行时 ORM 读写将报错——这是一个线上升级隐患，而非仅测试问题。

### 3.3 修复清单（精确字段）

**`AgentTask` 结构体（需补 `default`）：**

| 字段 | 类型 | 当前 tag | 建议修改 |
|------|------|----------|----------|
| `ExecutionMode` | string | `column(execution_mode);size(32)` | 加 `;default('')` |
| `ResumeCount` | int | `column(resume_count)` | 加 `;default(0)` |
| `RuntimeVersion` | string | `column(runtime_version);size(64)` | 加 `;default('')` |
| `SkillVersion` | string | `column(skill_version);size(64)` | 加 `;default('')` |
| `PromptVersion` | string | `column(prompt_version);size(64)` | 加 `;default('')` |
| `PromptHash` | string | `column(prompt_hash);size(64)` | 加 `;default('')` |
| `ModelConfigID` | int64 | `column(model_config_id);index` | 加 `;default(0)` |
| `InputContractVersion` | string | `column(input_contract_version);size(96)` | 加 `;default('')` |
| `OutputContractVersion` | string | `column(output_contract_version);size(96)` | 加 `;default('')` |
| `SkillSourceVersion` | string | `column(skill_source_version);size(128)` | 加 `;default('')` |
| `SkillSource` | string | `column(skill_source);size(32);default(native)` | `default(native)` 改为 `default('native')`（无引号隐患，SQLite 会把 `native` 当列名） |

> 已正确处理的列：`plan_json` / `steps_json` / `checkpoint_json`（`default('')`）、`SkillSource`（需补引号）。

**`AgentTaskEvent` 结构体（需补 `default`）：**

| 字段 | 类型 | 当前 tag | 建议修改 |
|------|------|----------|----------|
| `StepID` | string | `column(step_id);size(64);index` | 加 `;default('')` |
| `StepType` | string | `column(step_type);size(32);index` | 加 `;default('')` |
| `ErrorType` | string | `column(error_type);size(64)` | 加 `;default('')` |
| `Checkpoint` | bool | `column(checkpoint)` | 加 `;default(false)` |

### 3.4 用户实际修复方案（2026-09-01 已修复并验证）

用户未采用 3.3 的逐列 `default(...)` 方案，而是抓住了真正的根因：**beego v2.1.0 对 `type(text)` 列会忽略 `default(...)`**，因此 text 列必须用 `null` 才能允许 SQLite 在已有行表上 `ALTER ADD`；非 text 列（string/int/bool）beego 在 `RunSyncdb` 的 `ADD COLUMN` 阶段会自动补齐默认值（`''` / `0` / `false`），无需显式 `default`。

实际改动（`models/agent_task.go`）：
- `plan_json` / `steps_json` / `checkpoint_json`：`default('')` → `null`（关键修复，使 text 列升级可空）
- 非 text 新增列保持原样（beego 自动兜底默认值，已验证）
- 升级契约测试重写为 `TestAgentTaskSyncdbUpgradesExistingSQLiteRows`，真实模拟 legacy 表 + 1 行数据 → `RunSyncdb` → 校验新增列存在、text 列 nullable、legacy 行可读、V2 字段 round-trip

> 关于 `SkillSource` 的 `default(native)`：beego 对 varchar 列的 default 值会自动加引号生成 `DEFAULT 'native'`（见 `cmd_utils.go:136` 的 `DEFAULT '%s'` 分支），SQLite 正确接受，**无跨库引号隐患**，当前实现安全，无需改为 `default('native')`。

### 3.5 修复后验证（已执行）

```bash
export PATH=/usr/local/go/bin:$PATH
go test -count=1 -race ./agent/runtime ./agent/replay ./agent/manager ./agent/task ./llm ./models
```

结果：6 个包全部 `ok`（含 `models` 之前失败的两个升级契约测试）。确认 V2-1 新增列在已有行表上的 SQLite 升级不再失败，legacy 数据保留，V2 字段读写正确。

---

## 四、前端 Review

### 4.1 改动文件

- `src/api/agent.ts`（+44 行）
- `src/views/ai/taskCenter.vue`（+190 行）
- `locales/en.yaml` / `locales/zh-CN.yaml`（各 +29 行）

### 4.2 一致性核对（✅ 全部通过）

| 审查项 | 结论 |
|--------|------|
| API 调用：`cancelAgentTask` / `resumeAgentTask` → `POST /agents/tasks/:id/cancel`、`/resume` | ✅ 与 `routers/router.go` 完全一致 |
| `AgentTask` 类型新增 `execution_mode` / `plan` / `steps` / `resume_count` / `runtime_version` / `skill_version` / `prompt_version` / `prompt_hash` / `model_config_id` / `input_contract_version` / `output_contract_version` / `skill_source` / `skill_source_version` | ✅ 全部对应后端 `task.go` 字段 |
| `AgentExecutionStep` 接口（step_id/type/status/attempt/depends_on/input_summary/output_summary/started_at/completed_at/error_type/error/checkpoint） | ✅ 对应后端 `ExecutionStep` |
| `AgentTaskEvent` 新增 `step_id` / `step_type` / `error_type` / `checkpoint` | ✅ 对应后端 event 新字段 |
| **`checkpoint_json` 未暴露** | ✅ 前端类型无此字段，后端 `json:"-"` 屏蔽，无泄漏 |
| Cancel / Resume 按钮可见性逻辑与后端 `isResumableTask` / `coordinator.go:261` 语义一致 | ✅ `canCancelTask`: queued/running/waiting_llm/waiting_tool/validating；`canResumeTask`: `runtime_version` 以 `2.` 开头 且 (cancelled / interrupted / (failed && stage==timeout)) |
| locale 新增 stage / failure 枚举键（input_built / planning / plan_ready / resuming / tool_checkpoint / checkpoint_failed / execution_mode_unavailable / execution_mode_invalid / planning_failed / invalid_plan 等） | ✅ 与后端 `runtime/*.go` 实际输出字符串一一对应 |
| `state.yes` / `state.no` | ✅ locale 已定义（zh-CN:386-388 / en:386-388） |
| 防重复提交（`:loading="actionTaskId === row.id"`） | ✅ 良好实践 |
| Cancel 二次确认 `ElMessageBox.confirm` | ✅ 危险操作已保护 |

### 4.3 建议（非阻塞）

1. **Resume 无二次确认**：`cancelTask` 有 `ElMessageBox.confirm`，`resumeTask` 直接调用。Resume 会基于冻结 `model_config_id` 重新执行，误操作风险较低，但建议对"已 failed 且将重跑"的任务加确认，避免误触发重复执行。
2. **`canResumeTask` 未校验 checkpoint 存在**：后端 `isResumableTask` 还要求"存在非空的 checkpoint"，前端仅按 status/stage 显示按钮。极端情况下会出现"按钮可点但后端拒绝"（返回 `resumeFailed`）。后端为权威校验，可接受；如需更好 UX，可在前端也对 `detail.checkpoint` 或 steps 中是否存在 `checkpoint=true` 的步骤做前置判断。
3. 与 3.3 同：后端 `SkillSource` 的 `default(native)` 无引号隐患建议一并修复。

---

## 五、V2-1 Gate 评估

- **架构 / 安全 / 接口**：满足文档要求。
- **前端**：满足，与后端契约一致，无阻塞问题。
- **测试**：`agent/runtime`、`agent/replay`、`agent/manager`、`agent/task`、`llm`、`models` 在 `-race` 下全部通过（2026-09-01 复测）。
- **结论（2026-09-01 更新）**：V2-1 代码实现完整且正确；原阻塞级 DB 升级缺陷已由用户修复并经测试验证通过（见 §3.4 / §3.5）。**V2-1 现已满足 Gate 条件，可进入 V2-2。**

---

## 六、后续动作（缺陷已修复）

1. ✅ 阻塞缺陷已修复并验证（`models` 升级契约测试通过，全量 V2-1 相关包 `-race` 通过），V2-1 Gate 放行。
2. （可选，非阻塞）前端 `resumeTask` 增加二次确认；前端 `canResumeTask` 增加 checkpoint 存在性前置判断（见 §4.3）。
3. 建议：在 `main.go` 中至少记录 `RunSyncdb` 返回的错误（当前被忽略），避免未来类似升级失败被静默吞掉。
