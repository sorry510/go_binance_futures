# Phase V2-7：Agent 对话入口与 Skill Chat

## 状态

P0，待实施。V2-6 完成人工验收后进入本 Phase。

## 目标

为现有 Agent 平台增加统一的人类交互入口，使 Native Skill 与 Portable Skill 可以在 Web 中被选择、执行、连续对话和审计。

本 Phase 不创建第二套 Agent Runtime。Chat 只负责 Conversation、用户消息、Skill 选择与现有 Task Runtime 之间的编排。

```text
AI / 对话
   ↓
Chat Controller / Service
   ↓
Conversation ─────→ Agent Task
                       ↓
                  Agent Manager
                       ↓
                     Runtime
                  /           \
              Native Skill  Portable Skill
                       ↓
                  Tool Runtime
```

## 当前缺口

现有 `agent_conversations` / `agent_conversation_messages` 已能持久化消息，但通用 `/agents/tasks` 没有普通 Chat API，也不会自动把 Conversation 历史注入 Runtime。
现有 `Conversation.Skill` 还把会话绑定到单个 Skill，不适合统一 Chat；另外不同 Native Skill 的 Input Contract 并不都是纯文本，不能把输入框文本直接原样传给所有 Skill。

## 范围

第一版必须完成：

- AI 菜单新增 `对话`，路径 `/ai/chat`，并作为 `/ai` 默认入口。
- 左侧分页 Conversation 历史与“新建对话”。
- 右上聊天消息区，展示 User / Assistant / Task 状态。
- 右下 Composer，输入 `/` 调出可用 Skill。
- 同一 Conversation 中允许切换不同 Skill。
- Conversation 历史进入统一 Context Engine，而不是由各 Skill 自行读取。
- Portable Skill 可直接以纯文本参与 Chat。
- Native Skill 只有明确实现 Chat Adapter 后才进入 `/` 菜单。
- Task 仍使用现有 Manager / Runtime / Tool / Permission / Budget / Trace。
- 第一版使用 Task polling，不增加 Token Streaming。

## 非目标

本 Phase 不实现 SSE/WebSocket Token 流、文件/图片上传、自动 Skill Router、Model 选择、多 Agent 协作、Message 编辑/Regenerate、AI 自动标题、长期 Memory 或新 Tool 权限模型。

未选择 Skill 时不自动猜测 Skill；用户必须通过 `/` 明确选择。

## UI 设计

```text
┌────────────────┬─────────────────────────────────────────┐
│ 对话历史        │ 对话消息                                │
│                │                                         │
│ + 新建对话      │ User                                    │
│                │ Assistant / Task Status                 │
│ 今天           │ Tool / Stage 简要状态                   │
│  BTC 分析       │                                         │
│  Skill 测试     ├─────────────────────────────────────────┤
│ 昨天           │ [ / selected-skill × ]                  │
│  新闻测试       │ 输入消息……                        [发送] │
└────────────────┴─────────────────────────────────────────┘
```

左侧只承担 Conversation 导航，不展示 Task 技术细节。标题第一版由首条用户消息确定性截断生成，不额外调用 LLM。

消息区只展示适合用户理解的运行状态，例如“思考中”“正在调用 xxx”“正在验证”；完整 Step、Evidence、Token 和错误仍通过 Task 详情查看。

发送期间同一 Conversation 禁止再次发送，避免同一会话多个 Task 并行导致 Message Sequence 和 Context 顺序不确定；不同 Conversation 允许并行。

## Slash Skill Selector

输入 `/` 打开 Skill 菜单；继续输入 `/sym`、`/单` 时按 `name / display_name / description` 本地过滤。

Slash 选择是 UI command，不作为 Prompt 文本发送给 LLM。选择后保存：

```text
selectedSkill = "symbol_analysis"
```

输入框上方显示可移除的 Skill Chip。Skill 选择在当前 Conversation 中保持粘性，直到用户切换或清除。

菜单只展示“当前 Runtime 可执行且 Chat-capable”的 Skill，而不是简单展示 `agent_skills.enabled=1` 的全部数据库行。

Portable Skill 默认 Chat-capable，输入契约为纯文本。Native Skill 必须显式实现 Chat Capability / Adapter，避免把自然语言错误传给结构化 Input Contract。

第一版至少保证 Portable Skill 与 `symbol_analysis` 可从 Chat 正常调用；其他 Native Skill 在具备明确、安全的 Chat Adapter 后再进入菜单。系统事件专用的 `alert_analysis` 不应为了菜单完整性强行伪造 Signal 输入。

## Chat Adapter

增加可选的 Chat Adapter 边界，只负责把自然语言消息转换成 Skill 已有 Input Contract；不能修改 Tool 权限、Validator、Budget、Execution Mode 或最终输出契约。
建议接口概念：

```go
type ChatInputAdapter interface {
    BuildChatInput(ctx context.Context, content string) (string, error)
}

type ChatCapabilityProvider interface {
    ChatEnabled() bool
}
```

Portable Adapter 默认返回原始 `content`。`symbol_analysis` Chat Adapter 使用确定性规则提取 `XXXUSDT`，生成现有 `symbol_analysis_input_v1` JSON；无法确定交易对时返回明确用户错误，不调用另一个 LLM 猜测。

## Conversation 语义

统一 Chat Conversation 不绑定单个业务 Skill。为兼容现有 Strategy Builder 等旧逻辑，保留 `AgentConversation.Skill` 字段；新 Chat 创建时固定写：

```text
skill = "chat"
```

真正执行的 Skill 继续记录在 `AgentTask.Skill`。同一 Conversation 可以依次执行不同 Skill。

建议给 `AgentConversation` 增加 `title TEXT/VARCHAR` 字段；新建时为“新对话”，首条用户消息成功创建 Task 后确定性更新标题。普通 schema 变化继续使用 ORM Model + `go_binance_futures sync db`。
建议 `AgentConversationMessage` 增加 `skill` 字段，直接冻结该 Turn 实际选择的 Skill，避免聊天历史展示依赖逐条 Task Join；`task_id` 继续保留作为审计关联。

消息写入原则：

- User Message：Task 成功创建后写入，带 `task_id + skill`。
- Assistant Message：Task succeeded 后由 Completion Hook 幂等写入。
- failed/cancelled/interrupted 不伪造普通 Assistant 正文，UI 根据 Task 状态展示错误卡片。
- 同一个 `task_id + role` 的 Chat Message 必须避免重复写入；Completion/EnsureCompletion 重放仍应幂等。

## Chat API

新增独立的薄 Chat Controller，不让前端自行拼接 Conversation Store 与 Task Manager：

```text
GET  /agents/chat/conversations?page=1&limit=30
POST /agents/chat/conversations
GET  /agents/chat/conversations/:id/messages
POST /agents/chat/conversations/:id/messages
GET  /agents/chat/skills
```

`GET /agents/chat/skills` 只提供当前实际可运行、Chat-capable 的 Skill 视图；不复制 Skill CRUD。返回 `name/display_name/description/type/version` 等 Slash Menu 所需字段。
发送消息请求：

```json
{
  "skill": "v26-test-skill",
  "content": "测试暗号是什么？"
}
```

响应只负责接受并返回现有 Task 身份：

```json
{
  "conversation_id": "conv_xxx",
  "task_id": "task_xxx",
  "status": "queued"
}
```

前端继续轮询现有 `GET /agents/tasks/:taskId`，不复制 Task 状态 API。Task 失败时 Chat API 不吞掉原始 Runtime `stage/error_type/error`。

## Conversation Context

V2-7 必须增加统一 Conversation History Provider。历史不能由每个 Skill 自行读 DB，也不能由前端拼进用户 Prompt。

Runtime 在 `BuildInput` 之后，把历史转换成 `contextengine.BlockHistory`，与 Skill Input、MCP Resource、Tool Result 一起交给现有 Context Engine 做 Token/Bytes Trim。
Provider 读取时必须排除当前 `task_id`，避免当前 User Message 既作为 `Skill Input` 又作为 Conversation History 重复进入模型。第一版最多读取最近 30 条消息，再由 Context Engine 做最终裁剪。

默认只把已成功完成 Turn 的 User/Assistant 消息作为后续模型历史；当前运行中或 failed/cancelled/interrupted Turn 仍在 UI 可见，但不自动污染下一轮模型 Context。

历史 Block 必须保留来源标记：

```text
Type   = history
Source = conversation:<conversation_id>
Role   = user / assistant
```

Conversation History 的优先级低于 System/Skill Instruction 和当前 Task Input，高于长期 Memory；后续 V2-9 Memory 不能反过来接管 Conversation 生命周期。

## Task / Completion 编排

发送消息的顺序固定为：校验 Conversation → 校验 Chat Skill → Chat Adapter 构造现有 Skill Input → `Manager.Start` 创建 Task → 持久化 User Message → 返回 task_id。

Runtime 与 Task Store 保持现状。Completion Hook 检测 `Conversation.Skill == chat` 后，把成功结果转换为用户可读 Assistant Message，并保留 `task_id + skill`；随后继续执行现有 symbol analysis 等业务 completion，不互相替代。
Assistant 文本转换第一版遵循确定性规则：形成用户实际可见的回答文本，优先包含 Runtime `Result.Summary`；Final result 是字符串时直接使用，结构化结果则附加有大小上限的紧凑 JSON/关键结果。Conversation 保存的是这份用户可见回答，完整原始结果仍以 Task 为唯一审计来源。这样后续 Turn 可以引用上一轮的关键结果，而不是只剩一句摘要。

## 前端结构

建议拆分：

```text
src/views/ai/chat/
├── index.vue
├── conversationList.vue
├── messageList.vue
├── messageItem.vue
├── chatComposer.vue
└── skillCommandMenu.vue
```

不要把 Conversation、polling、Slash Menu、Message Render 全部堆进一个大 Vue 文件。

Chat Composer 支持 Enter 发送、Shift+Enter 换行；运行中禁用发送但允许切换左侧其它 Conversation。页面刷新后根据 URL/local state 恢复最近选中的 Conversation，消息以数据库为准重新加载。

## 运行状态显示

第一版不展示私有 Chain-of-Thought。只使用 Task/Event 的公开运行阶段生成状态，例如 waiting_llm、waiting_tool、validating、Tool canonical name、失败 stage 和错误。

## 实施顺序

### V2-7A：Phase/Schema/Store

- 重排 V2-7～V2-12 文档编号。
- `AgentConversation` 增加 title。
- `AgentConversationMessage` 增加 skill。
- Conversation Store 增加分页 List、按 task/role 幂等写入和 Chat History 查询。

### V2-7B：Chat Capability 与 Input Adapter

- 定义可选 Chat Capability / Adapter。
- Portable Skill 默认支持文本 Chat。
- `symbol_analysis` 增加确定性 Chat Input Adapter。
- 非 Chat-capable Native Skill 不进入 Slash Menu。

### V2-7C：Conversation Context

- 增加统一 Conversation History Provider。
- 排除当前 Task 和未成功 Turn。
- 进入 Context Engine 的历史 Block 必须受现有 Token/Bytes Budget 控制。

### V2-7D：Chat API 与 Completion

- Conversation List/Create/Messages/Send/Skills API。
- 同一 Conversation 单 Task 并发约束。
- 成功 Task 幂等写 Assistant Message。
- 失败 Task 保留 Runtime 错误并由 UI 渲染错误卡片。

### V2-7E：Web Chat

- AI → 对话菜单与 `/ai/chat`。
- Conversation List / Message List / Composer。
- `/` Skill Selector、搜索、粘性选择。
- 复用 Task polling 展示轻量运行状态。

### V2-7F：Gate 与人工验收

- Native/Portable Chat 回归。
- 多轮 Context 回归。
- Skill 切换、失败、刷新、并发和权限回归。
- 前端生产构建、静态部署和真实 Portable Skill 人工测试。

## 自动测试 Gate

至少覆盖：Conversation List/分页、首消息标题、AppendOnce 幂等、当前 Task 排除、失败 Turn 不进入 Context、30 条历史上限和 Context Trim。
还要覆盖：Portable 默认 Chat、Native 非 Chat-capable 不曝光、`symbol_analysis` 自然语言转结构化输入、Skill disabled/removed 后拒绝启动、Tool/Permission/Budget 不因 Chat 绕过，以及 Completion 重放不重复写 Assistant Message。

全量 Gate：

```text
go test ./...
go test -race ./agent/conversation ./agent/runtime ./agent/app ./controllers
frontend npm run typecheck
frontend npm run build
git diff --check
```

## 人工验收

- [ ] AI 菜单出现“对话”，刷新页面后 Conversation 历史仍存在。
- [ ] 新建 Conversation，输入 `/` 能看到当前可运行的 Chat-capable Native + Portable Skill。
- [ ] 选择 V2-6 导入的 Portable Skill，可以正常调用 `skill.<name>.read-resource` 并返回结果。
- [ ] Portable Skill 已审批 MCP Tool 时可正常调用；未审批时仍被 Permission 拒绝。
- [ ] `symbol_analysis` 可以从自然语言中确定交易对并执行现有结构化 Input Contract。
- [ ] 同一 Conversation 第二轮能看到第一轮成功消息历史。
- [ ] 同一 Conversation 可以从 Portable Skill 切换到另一个 Chat-capable Skill，Task 分别记录实际 Skill。
- [ ] failed/cancelled Task 在 UI 显示错误，但不会作为成功 Assistant History 注入后续模型。
- [ ] 同一 Conversation 运行中不能再次发送；不同 Conversation 可以并行。
- [ ] 点击“查看任务详情”可以回到现有 Task 审计信息。

## 版本边界

本 Phase 预期只新增 Chat orchestration、Conversation schema/store 能力和 Runtime Context Provider，不改变已有 Checkpoint/Resume state schema。除非实施时确实修改 RunState 持久化结构，否则不因为新增 Web Chat 强行提升 `CurrentVersion` / `runStateVersion`。

Chat Conversation 是人类交互入口，不是长期 Memory。后续 V2-9 Memory 只能作为 Context 的独立低优先级来源，不能把 Conversation 表重新解释为 Memory Store。

## Definition of Done

V2-7 完成后，用户不修改 Go 代码即可在 `/ai/chat` 中选择并测试一个 V2-6 Portable Skill；多轮消息会持久化、受 Context Budget 管理，并通过同一 Agent Runtime、Tool Runtime、MCP Permission、Task/Trace 体系执行。