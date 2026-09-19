# V4-1 Phase 审计报告：Chat Workspace

- **审计对象**：V4-1 实现**在工作区未提交**（`agent/app/chat_workspace.go`、`command/sql/version/18.sql`、`appversion/version.go` 及 `agent/`、`controllers/`、`llm/`、`models/`、`routers/`、`main.go` 共 20 个改动文件）。基线 `HEAD = ab21659`（`feat: ai agent v4-0`，分支 `feat/ai-agent-v4`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交/未跟踪文件；探针与副本审计后已删除，用户仓库未被改动）
- **审查类型**：初次只读审查；随后按审查结论完成修复与复验
- **结论**：**已完成**。§8 十条 Gate 当前 **10 条全部通过**。F1 已修复：`general_chat` 实现 `ChatAdapter`，可在 Explicit 模式中选择；F3/F4/F5 同步完成测试、入口收敛与文档修正。
- **审计日期**：2026-09-19

> **修复更新（2026-09-19）**：本报告第 2～3 节保留初次审计的取证过程；第 1、4、5 节已按修复后状态更新。

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-1 是否完成 | ✅ **已完成**：多 Skill 挂载 / Auto+Explicit 路由 / 会话内模型切换 / 删除聊天 / 迁移 v18 均已落地，十条 Gate 全部满足 |
| 未满足项 | 无。初次审计发现的 F1 已修复；`general_chat` 现已进入 chat-capable 目录并可被 Explicit 稳定选择 |
| 是否可进入 V4-2 | ✅ **可以** |
| DB 变更 | `agent_conversations.model_config_id`（新列）+ `agent_conversation_skills`（新表），Schema **v17 → v18**；`appversion.DatabaseSchemaVersion = 18` |
| 是否存在 P0/P1 安全缺陷 | ❌ 未发现（权限模型、Tool Permission、Provider 列表均未改动） |

---

## 2. 验收 Gate 逐项核对（§8 十条）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 同一 Chat 可挂载 3+ Skill | ✅ | `agent/conversation/chat.go` `AttachChatSkill`（无数量上限、幂等、`sort = count`）；`GET/POST/DELETE …/skills` 三路由（`routers/router.go:19-20`）；探针 P1 在同一库中建立 4 条绑定；前端 `index-C_MaSHll.js` 多选 diff 逻辑（`t=Array.from(new Set(["general_chat",...]))`） |
| 2 | Auto 只能从已挂载且 chat-enabled 的 Skill 中选择 | ✅ | `agent/app/chat_workspace.go:140-170`：候选集合 = `attachedSet`（挂载 ∪ general_chat），且必须命中 `ChatSkills()` 目录（要求 `enabled==1 && chat_enabled==1` 且实现 `ChatAdapter`，`agent/app/chat.go:59`）。**探针 P2 实证**：未挂载的 `symbol_analysis` 不被选中；`market_scan`（chat_enabled=0）与 `alert_analysis`（enabled=0）即使被"挂载"也不进目录、不可被 Auto 选中 |
| 3 | Explicit 可以稳定指定某个已挂载 Skill | ✅ | 对 chat-capable 的已挂载 Skill ✓（探针 P2：`symbol_analysis` 稳定返回）；未挂载 → 报 `not attached` ✓；已挂载但 chat-disabled / disabled → 报 `not available for chat` ✓。修复后 `general_chat` 实现 `ChatAdapter`，也可显式选择 |
| 4 | 单条消息只产生一个 Primary Skill Task | ✅ | `agent/app/chat.go:startChatMessage` 单次调用只 `manager.Start(...)` 或 `team.Runner.StartWithOptions(...)` 一次；无并行挂载执行、无 System Prompt 合并（`chat_workspace.go` 只 resolve 一个 skill 名） |
| 5 | 删除已挂载 Skill 后不能继续通过 Auto 或旧请求调用 | ✅ | **探针 P2 实证**：显式指定未挂载 Skill 直接被拒（`resolveChatSkill` explicit 分支先查 `attachedSet` 再查目录）；Auto 候选来自挂载集合，删除后自然消失 |
| 6 | Skill A/B 在同一 Conversation 共享历史，但各用各自执行契约 | ✅ | 历史仍由 conversation 维度提供（`ConversationHistory` 未改）；Skill 执行契约（Validator/Tool/Output Contract）未改动；`runtime/types.go` 仅新增 `ModelConfigID` 字段 |
| 7 | Model A → B 切换只影响新消息 | ✅ | `SetModelConfigID` 只更新会话偏好（`chat.go:SetModelConfigID`，带 `updated_at`）；每条消息解析出的 `modelConfigID` 写入该 Task；**resume 使用任务冻结值**（`manager.go:234 NewClientByID(item.ModelConfigID)`，既有测试 `TestManagerCancelAndResumeUsesFrozenModelConfig` 通过） |
| 8 | 实际 Task 能追踪最终 provider/model | ✅ | **探针 P3 实证**：显式配置 → 任务记录 `Provider=openai_compatible`、`Model=model-7`、`ModelConfigID=7`、`RouteReason="explicit model config id 7"`；`FinalModelConfigID` 保留 fallback 语义；Team 场景由既有测试断言 4 个角色同 `model_config_id`（`agent/team/runner_test.go`） |
| 9 | running Chat 不可删除 | ✅ | 既有测试 `TestDeleteChatRejectsConversationWithRunningTask` 通过；前端 `agentChat.message.deleteRunning` 文案 + 删除二次确认（`deleteConfirm/deleteTitle`）+ 运行中禁用输入（`disabled:!conversation||running`） |
| 10 | 删除 Chat 不删除 Task/Audit | ✅ | 既有测试 `TestDeleteChatRemovesConversationAndMessagesButKeepsTaskHistory` 通过；V4-1 额外在事务内清理 `agent_conversation_skills`（`agent/conversation/chat.go:DeleteChat`），不触碰 tasks/audits |

**其它文档要求核对**

| 要求 | 判定 | 证据 |
| --- | --- | --- |
| §3 新增轻量关系表 + 组合唯一约束 | ✅ | `models/agent_conversation.go`：`AgentConversationSkill` + `TableUnique{ConversationID, SkillName}` |
| §4 保存 Model Gateway 的 model config ID | ✅ | `AgentConversation.ModelConfigID`（`default(0)`）、`Conversation.ModelConfigID`、`PUT …/model` |
| §4 显式模型"严格使用、不静默切换、失败明确报错" | ✅ | **探针 P3 实证**：无效配置返回 `initialize LLM client: …` 且**不**回退路由（`router.calls` 不增） |
| §4 显式模型必须是 enabled 或 Router Candidate | ✅ | `chat_workspace.go:96-111 validateChatModelConfig`（`row.Enabled!=1 && row.RouterCandidate!=1` → 拒绝） |
| §5 删除 Chat 复用 V2 语义（不重复实现） | ✅ | 未新增删除接口，仅前端补齐 + DeleteChat 增加 bindings 清理 |
| §7 四条 API | ✅ | `routers/router.go:19-21`（GET/POST skills、DELETE skills/:skillName、PUT model）；`controllers/agent_chat.go` 请求体新增 `skill_mode/skill/model_id`，响应返回 `skill` 与 `model_config_id` |
| §7 发送消息 `model_id` 可省略 | ✅ | `ChatMessageOptions.ModelConfigID *int64`；为空时用会话偏好（`chat_workspace.go:86-89`） |
| §9 本阶段不做 | ✅ 未越界 | 无并行执行、无自动权限扩张、无新 Provider（`llm/*` 仅新增 `ModelName()` 方法）、未改 Tool Permission 模型（`permission.AllowWritesFor(nil)` 未动） |

---

## 3. 实证过的隐式契约（探针）

探针写在隔离副本，跑完随副本删除；因 beego ORM `RegisterDataBase` 为进程级全局，探针以 `-run TestProbe` 单独执行，并另跑一次无探针的包测试确认原测试仍通过（均 PASS）。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **P1** 迁移 v17→v18 端到端（`command`） | 造一个"v17 库"（schema 就绪 + `config.version=17`），预置 4 个会话：`legacy-plain`(chat，无绑定)、`legacy-prebound`(chat，已绑 general_chat)、`legacy-other-skill`(chat，仅绑 symbol_analysis)、`non-chat`；随后调用真实 `SyncDatabase(18)` | ✅ `version=18`；`general_chat` 绑定数 = 1/1/1/0（**补齐、不重复、非 chat 会话不受影响**）；`agent_conversation_skills` 表与 `agent_conversations.model_config_id` 列存在；**第二次 sync 幂等**（总数不变）。该行为**没有永久回归测试保护**（既有 syncdb 测试只验证 schema，不验证 `18.sql` 的数据回填） |
| **P2** Skill 路由边界（`agent/app`） | 真实 `DefaultManager()` + 真实 skillconfig 行（general_chat 1/1、symbol_analysis 1/1、market_scan 1/0、alert_analysis 0/1） | ✅ 目录仅含 `symbol_analysis`；未挂载 → `not attached`；chat-disabled / disabled → `not available for chat`；未知 mode → `unsupported skill_mode`；Auto 忽略未挂载 Skill 且**无匹配时回退 general_chat** |
| **P2b** `general_chat` 可选性（`agent/app`） | 直接检查运行时 Skill 的 `ChatAdapter` 与目录、显式解析结果 | 初次审计发现 F1：`general_chat` 未实现 `ChatAdapter`。现已修复：它实现 `ChatEnabled` / `BuildChatInput`，进入 chat-capable 目录，可被 Explicit 选择；GET/POST 语义恢复一致 |
| **P3** 显式模型严格性（`agent/manager`） | 注入假路由器（ConfigID=999）与假 `NewClientByID`（仅 id=7 可用） | ✅ 显式有效：**路由不被调用**（calls=0），任务记录 `model_config_id=7 / model-7 / provider`，`RouteReason="explicit model config id 7"`；显式无效：返回 `initialize LLM client` 错误且**仍不回退路由**；未指定（0）：路由调用 1 次并记录 `999/routed-model`（既有行为未回归） |

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| **F1** | ✅ 已修复 | `general_chat` 原先未实现 `ChatAdapter`，导致 Explicit 模式不可选 | `agent/skills/generalchat/skill.go` 已实现 `ChatEnabled` / `BuildChatInput`，并增加接口与输入归一化测试；前端从后端目录动态生成选项，无需额外源码改动 |
| F2 | P3 | `AttachChatSkill` 的 `Sort = 当前绑定数`：先移除再新增会出现 sort 重复，排序退化为 `OrderBy("sort","id")` 兜底 | `chat.go:AttachChatSkill`。功能无感，仅在需要稳定自定义排序时需改为 `max(sort)+1` |
| F3 | ✅ 已修复 | Auto 打分阈值属于隐式调参点 | 已增加单关键词、阈值命中、Skill 名、显示名与 `/name` 的边界回归测试；算法与既有行为不变 |
| F4 | ✅ 已修复 | `StartChatMessage` 旧入口原先可绕过 attached Skill 校验 | 旧入口保留兼容性，但已委托 `StartChatMessageWithOptions`，统一走 Explicit 路由和挂载校验；空 Skill 仍兼容为 `general_chat`，模型仍保持旧入口的 Router 语义 |
| F5 | ✅ 已修复 | README 完成状态与 Phase 文档不一致 | `doc/agent/v4/README.md` 已将 V4-1 标为 `P0 ✅`，进度图同步完成状态 |
| F6 | 提示 | 显式模型每发一条消息都会 `llm.Store.Get(modelConfigID)` 校验一次（一次额外 DB 读） | 量级可忽略；若后续做高频聊天可加轻量缓存 |
| F7 | 提示 | 消息级 `model_id` 只作用于该条消息，不会写回会话偏好（需显式 `PUT …/model`） | 与文档 §4"允许发送消息时临时 override；选择'设为当前对话模型'再更新偏好"一致 ✓ |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **61 个包 ok，0 FAIL**（共 76 个含 no-test-files） |
| `go test -count=1 -race ./agent/conversation ./agent/app ./agent/manager ./agent/team ./models ./controllers` | ✅ 全部 ok（1.3s / 2.6s / 1.3s / 2.1s / 1.9s / 1.9s） |
| `gofmt -l`（20 个改动 Go 文件） | ✅ 无输出 |
| `git diff --check`（排除 static） | ✅ 无异常 |
| 隔离副本 `go build ./...` + 探针（P1/P2/P2b/P3） | ✅ 通过（F1 为观察结论，非测试失败） |
| 副本去掉探针后重跑受影响包 | ✅ command / agent/app / agent/manager / agent/conversation / agent/team 全 ok |
| 前端 `pnpm typecheck` / `pnpm build` / dist↔static | ⚠️ 无法直接复跑（前端源码在独立仓库）。已用构建产物交叉核对：`chatComposer-Cht59EYi.js`（Auto 选项 `__auto__`、显式标签可关闭回到 auto）、`index-C_MaSHll.js`（`skill_mode/selectedSkill/skills` 计算、`general_chat` 强制保留且不可移除、删除二次确认与 `deleteRunning` 文案）、`agent-Dh1pl1mC.js`（`GET/POST/DELETE …/skills`、`PUT …/model` 客户端） |

**修复后复验**

| 项目 | 结果 |
| --- | --- |
| `go test -count=1 ./...` | ✅ 全量通过（使用隔离的 `/tmp` Go build cache；需要本机临时端口的测试在允许绑定后通过） |
| `go test -count=1 -race ./agent/app ./agent/conversation ./agent/skills/generalchat` | ✅ 全部通过 |
| `go vet ./...` | ✅ 无输出 |
| `gofmt -l`（本次修改 Go 文件） | ✅ 无输出 |
| `git diff --check` | ✅ 无异常 |

---

## 6. 审计边界与未验证项

- **未验证**：前端源码与 `pnpm` 流水线（独立仓库）；`dist` 与后端 `static` 的逐文件 diff（仅做产物侧功能核对）。
- **未验证**：MySQL 上的 v17→v18 迁移（探针用 SQLite；`18.sql` 为纯 `INSERT … SELECT`，MySQL 兼容，但未实跑；实施报告称已在真实库执行 17→18）。
- **未验证**：真实 4 角色 Team 运行（既有单测断言了 model_config_id 传播；未跑真实 LLM 调用）。
- 未运行 AI 对话端到端人工测试（文档 §9 的 A–F 六组手工步骤仍建议由用户按页面执行，尤其"删除已挂载 Skill 后旧请求不可调用"与"模型切换只影响新消息"的体感确认）。

---

## 7. 附：V4-1 交付物清单

| 文件 | 变更 | 作用 |
| --- | --- | --- |
| `agent/app/chat_workspace.go` | 新增 209 行 | Skill 模式解析（Auto/Explicit）、打分路由、模型校验、消息入口 |
| `agent/app/chat_workspace_test.go` | 新增 39 行 | Auto 仅取挂载 + 回退 general_chat |
| `agent/conversation/chat.go` | +109 | `ChatSkillNames` / `AttachChatSkill` / `RemoveChatSkill` / `SetModelConfigID`；DeleteChat 清理绑定 |
| `agent/conversation/store.go` | +27/-7 | `Conversation.ModelConfigID`、新会话默认绑定 `general_chat`（失败回滚会话） |
| `agent/manager/manager.go` | +8/-1 | 显式 `ModelConfigID>0` 直连指定配置，不走路由；失败即报错 |
| `agent/team/{runner,types}.go` | +17 | 父/子/supervisor 全量传播 model_config_id |
| `llm/{client,openai,anthropic,bridge}.go`、`agent/modelgateway/router.go` | +14 | `ModelName()` 抽象，用于任务记录实际模型名 |
| `models/agent_conversation.go` | +28/-14 | `ModelConfigID` 列 + `AgentConversationSkill` 表（组合唯一） |
| `command/sql/version/18.sql` | 新增 11 行 | 为存量 chat 会话补 `general_chat` 绑定（幂等） |
| `appversion/version.go`、`main.go` | — | Schema v18；注册新模型 |
| `controllers/agent_chat.go`、`routers/router.go` | +70 / +3 | 四条 API + 消息请求/响应扩展 |
