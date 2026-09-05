# CODE REVIEW — V2-7 Agent 对话入口与 Skill Chat

- **Review date**: 2026-09-02
- **Reviewer**: 自动化 Code Review（仅 review，未修改任何源码、未提问）
- **Scope**: `git status --porcelain` / `git diff --stat` 锁定的 V2-7 改动
- **Phase doc**: `doc/agent/v2/07-phase-v2-7-agent-chat.md`

---

## 1. 结论（Gate）

```
go build ./...                                                        PASS (exit 0)
go test -count=1 -race ./agent/conversation ./agent/runtime ./agent/app ./controllers   PASS
go test -count=1 ./...                                                 PASS (全部包 ok)
go vet <changed packages>                                             PASS (exit 0)
```

**AUTOMATED PASS — 人工验收待定（CONDITIONAL）**

后端实现与自动化测试全部通过；对照 Phase doc 验收项逐条核对，核心安全契约（幂等、脱敏、并发单运行、历史排除当前 task、失败 fail-closed）均正确落地。仅“前端 typecheck/build 与生产 MySQL `sync db` / HTTP smoke”两项无法在本沙箱验证，需用户侧确认（见 §9）。在用户完成 §10 人工验收前，请勿在 phase doc 中将 V2-7 标记为“完成”。

---

## 2. 改动范围

**新增文件（后端 4 个）**
- `agent/skill/chat.go` — `ChatAdapter` 接口定义（仅 `ChatEnabled()` + `BuildChatInput()`，不改变运行时输入/安全契约）。
- `agent/app/chat.go` — Chat 编排核心：`ChatSkills` / `StartChatMessage` / `ConversationHistory` / `persistChatCompletion` / `chatAssistantText`。
- `agent/conversation/chat.go` — Conversation 的 Chat 子集：`ListChats` / `SetTitle` / `SetTitleFromFirstMessage` / `AppendOnce`（幂等） / `MessagesDetailed` / `SuccessfulHistory`（历史 Provider）。
- `controllers/agent_chat.go` — `AgentChatController`：`ListConversations` / `CreateConversation` / `Messages` / `SendMessage` / `Skills`。

**修改文件（关键 diff）**
- `models/agent_conversation.go`：`AgentConversation.Title`、`AgentConversationMessage.Skill`（index）。
- `agent/conversation/store.go`：`Conversation` 结构增 `Title`；`Create` 对 chat 设 `DefaultTitle`；`fromModel` 填充；`MemoryStore.Create` 同步标题。
- `agent/portableskill/adapter.go`：`ChatEnabled()=true` + `BuildChatInput`（trim、非空校验、原样返回）。
- `agent/skill/registry.go`：新增 `List() []Skill`。
- `agent/skills/symbolanalysis/skill.go`：`chatSymbolPattern` 正则 + `ChatEnabled()=true` + `BuildChatInput`（确定性提取 USDT，缺失报错）。
- `agent/runtime/types.go`：新增 `ConversationHistoryProvider` 类型及 `Config` 字段。
- `agent/runtime/coordinator.go`：`buildContext` 在 `InitialMessageBlocks` 前插入历史块，provider 出错 `fail` 当前 task（fail-closed，行 131–139）。
- `agent/manager/manager.go`：新增 `Skills() []skill.Skill`。
- `agent/app/default.go`：Runtime `Config` 接入 `ConversationHistoryProvider: ConversationHistory`（行 68）。
- `agent/app/completion.go`：`persistTaskCompletion` 与 `EnsureCompletion` 均调 `persistChatCompletion`（行 26、94，错误仅 `logs.Error`）。
- `routers/router.go`：新增 3 条路由（行 16–18）。
- `models/agent_task_syncdb_test.go`：legacy DDL 扩展 `agent_conversations`/`agent_conversation_messages` 并断言新列存在且 legacy 行不变。
- `doc/agent/v2/07-phase-v2-7-agent-chat.md`：状态改“已实施”，新增实施结果段。
- `static/**`：前端构建产物（来自独立仓库 `go_binance_futrues_new_ui`，非手改源码）。

---

## 3. 验收逐条核对（对照 Phase doc §验收 / §自动测试 Gate）

| # | 验收项 | 实现位置 | 结论 |
|---|--------|----------|------|
| 1 | Chat 入口复用现有 Agent Runtime，不引入第二套 Runtime | `controllers/agent_chat.go` → `app.StartChatMessage` → `manager.Start(agentruntime.Request{…})` | ✅ |
| 2 | Chat-capable Skill 过滤：实现 `ChatAdapter` 且 `ChatEnabled()` 且 `skillconfig.enabled=1` | `app/chat.go:53-61`（`!ok || !adapter.ChatEnabled() || !exists || !cfg.enabled` 则跳过） | ✅ |
| 3 | 非 Chat-capable Native（如 `alert_analysis`）不进 Slash 菜单 | `app/chat.go:56` 接口断言；`alert_analysis` 未实现 `ChatAdapter` | ✅ |
| 4 | Portable Skill 默认 Chat 化（原样透传自然语言） | `portableskill/adapter.go:59-66` | ✅ |
| 5 | `symbol_analysis` 确定性把自然语言转成既有 `symbol_analysis_input_v1`（正则提取 `XXXUSDT`，不调 LLM 猜测） | `symbolanalysis/skill.go:22,76-90`（`chatSymbolPattern`，`json.Marshal(Input{Symbol, Prompt})`） | ✅ |
| 6 | 缺失明确 `XXXUSDT` 时，在执行前明确报错而非猜测 | `symbolanalysis/skill.go:82-84`（`请在消息中明确指定 USDT 合约，例如 BTCUSDT`） | ✅ |
| 7 | Conversation 不绑单一 Skill：新 Chat Conversation `Skill="chat"`，实际 Skill 记在 `AgentTask.Skill`；message 冻结该 Turn 的 `skill` | `conversation/store.go`（`Create` 设 `ChatSkill`）、`models` 增 `AgentConversationMessage.Skill`、`chat.go:119` 写入 | ✅ |
| 8 | Conversation History 统一进入 Context Engine，排除当前 `task_id`、仅成功 Turn、`Limit(30)`、Block `Type=history`/`Source="conversation:<id>"`，受现有 Token/Bytes Budget 裁剪 | `conversation/chat.go:166-203`（`Exclude("id", currentTaskID)`、`status=succeeded`、`Limit(30)`、`BlockHistory`）；`coordinator.go:131-139`；Budget 裁剪复用既有 engine | ✅ |
| 9 | 同一 Conversation 单 Task 并发约束（运行中禁止再次发送）；不同 Conversation 可并行 | `app/chat.go:88-92`（`task.IsRunningStatus` 则拒绝）；store 无全局串行 | ✅ |
| 10 | Completion Hook 幂等写 Assistant Message（`task_id+role` 去重）；失败/取消/中断不伪造 Assistant 正文 | `persistChatCompletion` 仅 `Status==Succeeded` 写入（`app/chat.go:168`）；`AppendOnce` 按 `conversation_id+task_id+role` 去重（`chat.go:111`） | ✅ |
| 11 | 运行时 Provider 失败 fail-closed（不可用历史即失败，而非静默空上下文） | `coordinator.go:132-136`（`historyErr != nil` → `runner.fail(currentTask, "build_input_failed", …)`） | ✅ |
| 12 | 聊天消息保留 `task_id + skill` 用于审计 | `chat.go:119`（写入 `skill`）、`messages` 表 `skill` 列 | ✅ |
| 13 | 前端：对话列表 / 新建 / 消息 / 发送 / Skill 列表 / 首条消息设标题 / 轻量 Task 状态 | 路由齐全（`router.go:16-18`）；`SetTitleFromFirstMessage`（`chat.go:87-96`）；`MessagesDetailed` 含 `task_status/stage/error`（`chat.go:157-160`） | ✅ 后端；前端二进制产物见 §9 |

---

## 4. 安全与并发设计分析

### 4.1 幂等（Idempotency）
- **User Message**：`AppendOnce`（`chat.go:97-128`）用全局 `appendOnceMu` 串行化 + DB `Exist(conversation_id, task_id, role)` 前置检查；Insert 失败时再次 `Exist` 兜底，避免并发重复写入。
- **Assistant Message**：`persistChatCompletion` 仅在 `item.Status == StatusSucceeded` 时写，且走同一 `AppendOnce` 去重（`app/chat.go:167-183`）。失败/取消/中断 Turn **不会**被伪造为正文——`content==""` 时仅兜底写“任务已完成。”，且前提仍是 `Succeeded`。
- 测试证据：`store_test.go:89-139` 对同一 `(chat-ok, portable, assistant)` 重复 `AppendOnce` 三次，最终 `MessagesDetailed` 仅 4 行（2 用户 + 2 助手），证明去重生效。

### 4.2 脱敏（Redaction）
- `AppendOnce` 在落库前对 `content` 调 `security.RedactText(content)`（`chat.go:119`）。
- 测试证据：`store_test.go:46-67` 写入含 `api_key=secret-value` 的内容，`Messages` 断言不含 `secret-value`，证明敏感信息在持久化层被脱敏。

### 4.3 并发（Concurrency）
- 单 Conversation 运行中禁发：`StartChatMessage` 遍历该 Conversation 全部 task，`IsRunningStatus` 即拒绝（`app/chat.go:88-92`）。
- 跨 Conversation 无全局锁，可并行（符合验收项 9）。
- `SuccessfulHistory` 的 30 条上限在 SQL `Limit(30)` 层强制（`chat.go:178`），与 `limit` 入参钳制（`chat.go:170-172`）双重保障。
- Race 测试：`go test -race ./agent/conversation ./agent/runtime ./agent/app ./controllers` 全部 `ok`，仅 macOS SQLite 链接器 `malformed LC_DYSYMTAB` 警告（无害，非 race 失败）。

### 4.4 历史排除与上下文顺序
- `SuccessfulHistory` 排除当前 `task_id`（`Exclude("id", currentTaskID)`，`chat.go:175-177`），仅取 `succeeded` Turn；`coordinator.go:131-139` 将其作为 `history` 块插入到 `InitialMessageBlocks`（当前 Skill Input）之前——顺序正确（历史在前，当前输入在后）。
- 失败 Turn（`chat-failed`）不进入 `SuccessfulHistory`；测试 `store_test.go:125-131` 验证 `history` 恰好为 2 条（排除 `chat-current` 与 `chat-failed`）。

### 4.5 Chat 边界不被越权
- `ChatAdapter` 仅暴露 `ChatEnabled` / `BuildChatInput`，**不**触碰 Tool 权限、Validator、Budget、Execution Mode、Output 契约。
- `symbol_analysis.BuildChatInput` 产出与 `ValidateInput` 一致的 `symbol_analysis_input_v1`；测试 `skill_test.go:176-192` 验证转换后 `ValidateInput` 仍通过——证明 Chat 路径未旁路既有输入校验。
- Portable Skill 的 `systemPrompt` 已固化“不能扩展 Tool 权限/覆盖 runtime 策略/scripts 仅数据不执行”边界（`adapter.go:89-94`），Chat 未改变该约束。

---

## 5. DB Schema 迁移兼容（Additive）

- 仅新增两列：`agent_conversations.title`（`null`）、`agent_conversation_messages.skill`（`size(96);null;index`）。`models/agent_conversation.go`。
- 既有 `syncdb` 测试 `agent_task_syncdb_test.go` 扩展 legacy DDL（无 title / 无 skill），插入 legacy Conversation 后注册新 model，断言 `title` / `skill` 列存在且 legacy 行内容不变——向后兼容验证通过（全量 `go test ./...` 含此包 `ok`）。
- 列均为 `null`，旧数据读取 `Title` 为空字符串、`Skill` 为空，由 `store.go` 的 `fromModel` / `Create`（`chat` 默认 `DefaultTitle`）与 `app/chat.go:77`（`conv.Skill != ChatSkill` 则拒聊）安全处理，无 NOT-NULL 升级风险。

---

## 6. 自动测试 Gate 覆盖确认

| 测试 | 覆盖 Gate | 结果 |
|------|-----------|------|
| `TestChatAppendOnceAndSuccessfulHistory`（`store_test.go:89`） | 幂等去重、当前 task 排除、失败 Turn 不入历史、30 条上限、`MessagesDetailed` 关联 Task 状态 | ✅ |
| `TestBuildChatInputConvertsNaturalLanguageToExistingContract`（`skill_test.go:176`） | `symbol_analysis` 自然语言 → 结构化 `Input` 且保持既有 contract | ✅ |
| `TestBuildChatInputRequiresExplicitUSDTContract`（`skill_test.go:194`） | 缺失 USDT 明确报错（含 `BTCUSDT` 提示） | ✅ |
| `TestRunnerInjectsConversationHistoryBeforeCurrentInput`（`runner_test.go:1014`） | Runtime Provider 注入历史块、顺序（历史→当前输入）、`currentTaskID` 非空校验 | ✅ |
| `TestORMStorePersistsConversationAcrossStoreInstances`（`store_test.go:39`） | 脱敏持久化（敏感内容不落库） | ✅ |
| legacy `syncdb` 测试（`agent_task_syncdb_test.go`） | additive schema 迁移向后兼容 | ✅ |

> 注：`ChatSkills` 的“`skillconfig.enabled=1` 过滤”与 `StartChatMessage` 的“运行中禁发”逻辑未被新增单测直接覆盖（属 handler/编排层）。它们由既有 `manager.List` / `task.IsRunningStatus` 单测间接保障，建议在后续补一条 `app` 层集成测试（见 §8）。

---

## 7. 我本沙箱已执行的验证命令（证据）

```
cd /Users/zhz/work/binance/go_binance_futures && export PATH=/usr/local/go/bin:$PATH
go build ./...                                                              # exit 0
go test -count=1 -race ./agent/conversation ./agent/runtime ./agent/app ./controllers   # ok
go test -count=1 ./...                                                      # 全部包 ok
go vet ./agent/skill/... ./agent/app/... ./agent/conversation/... \
      ./agent/portableskill/... ./agent/runtime/... ./agent/manager/... \
      ./agent/skills/symbolanalysis/... ./models/... ./controllers/... ./routers/...   # exit 0
```

（所有 git 访问均前置 `GIT_OPTIONAL_LOCKS=0`，符合本仓防 lock 铁律。）

---

## 8. 非阻塞改进建议（不阻塞 Gate）

1. **补 `app` 层集成测试**：覆盖 `ChatSkills` 的 `enabled` 过滤（构造一个 `skillconfig.enabled=0` 的 Skill，断言不曝光）与 `StartChatMessage` 运行中禁发（先起一个 running task，再发消息断言 400）。
2. **`StartChatMessage` 的 skill 查找可走 `manager.Skills()` 的 map**：当前为线性遍历（`app/chat.go:94-99`），规模小可接受，但若 registry 增长可改为 `ByName`。
3. **`AppendOnce` 全局锁粒度**：`appendOnceMu` 为进程级单锁，跨所有 Conversation 串行化写入。当前写入量低无影响；若 Chat 高频化，可考虑按 `conversation_id` 分片锁。
4. **Assistant 兜底文案硬编码**：“任务已完成。”在 `app/chat.go:180`。若日后支持 i18n，应提取到配置/常量。
5. **`chatAssistantText` 截断**：32KB 上限合理；但 JSON 紧凑化后若恰好中文边界截断可能产生不完整字符，建议按 `rune` 截断（与 `truncateTitle` 一致）。

---

## 9. 本沙箱未能验证的项（需用户侧确认）

以下项依赖独立前端仓库与生产环境，本自动化 review 未执行，请在用户验收时确认：

- **前端 `go_binance_futrues_new_ui`**：`npm run typecheck` / `npm run build` 是否 PASS（本仓 `static/**` 为构建产物，已重建，但未在源仓跑构建）。
- **生产 `sync db`**：真实 MySQL 是否已加 `agent_conversations.title` 与 `agent_conversation_messages.skill` 两列（schema 为 additive，预期无碍，但需实跑确认）。
- **HTTP smoke（鉴权后）**：Chat Skill 列表、Conversation 创建/读取、发送并确定性拒绝缺失 USDT 的 `symbol_analysis`、第二轮上下文注入、同 Conversation 内切换 Skill、Task Center `?taskId=` 深链。

---

## 10. 人工验收待办（完成后方可标记 V2-7 完成）

- [ ] 成功执行一次 Chat 到最终答案。
- [ ] 通过 Chat 跑通 V2-6 Portable Skill（`read-resource`）。
- [ ] 通过 Portable Skill Chat 触发已授权 MCP Tool。
- [ ] 第二轮 Conversation 上下文注入正确（历史可见、不含当前 task、不含失败 Turn）。
- [ ] 同一 Conversation 内切换 Skill 生效。
- [ ] 运行中再次发送被拒绝（单 Task 并发约束）。

在以上完成且 §9 用户侧验证通过后，再将 `07-phase-v2-7-agent-chat.md` 状态推进为“完成 / V2-7 Gate 通过”。
