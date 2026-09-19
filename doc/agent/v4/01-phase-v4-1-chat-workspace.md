# Phase V4-1：Chat Workspace

## 1. 目标

把现有“一个消息选择一个 Skill”的 Chat 升级为真正可长期使用的个人 AI Workspace：

- 一个 Conversation 可以添加多个 Skill。
- 支持随时增删 Conversation 中的 Skill。
- 支持 `Auto` 模式从已挂载 Skill 中选择当前消息需要的 Skill，也支持用户显式指定。
- 一条消息可以在受控范围内顺序使用多个相关 Skill，但不把所有 Skill 无条件并行执行。
- 支持 Conversation 内切换模型。
- Web 支持删除聊天。

不建立第二套 Runtime。

## 2. 多 Skill 模型

一个 Conversation 保存 `attached skills`，例如：

```text
Conversation
  ├─ general_chat
  ├─ symbol_analysis
  ├─ market_scan
  └─ my-portable-skill
```

### 执行原则

多个 Skill “挂载在同一个 Conversation”不等于把所有 Skill 的 System Prompt 拼接在一起，也不等于每条消息并行运行全部 Skill。

V4-1 提供两种使用方式：

1. **Auto**：默认模式。Chat Router 只读取已挂载 Skill 的轻量 metadata（name / description / chat capability），根据当前用户消息决定使用 `general_chat`、一个 Skill，或在确有必要时顺序调用少量相关 Skill。
2. **Explicit**：用户在输入框使用下拉或 `@skill` 明确指定一个已挂载 Skill，本轮直接使用该 Skill，不再自动路由到其它 Skill。

约束：

- `general_chat` 永远作为基础 fallback，不要求用户移除/添加。
- Auto Router 只能选择该 Conversation 已挂载、enabled 且 chat-enabled 的 Skill。
- Portable Skill 继续使用现有 Portable Runtime。
- Native Skill 保持自己已有的 Validator / Tool / Output Contract。
- 多 Skill 顺序使用时，每个 Skill 独立执行并保留 Task/Trace，不把多个互相冲突的 System Prompt 强行合并成一个 Prompt。
- 第一版限制单条消息最多触发少量 Skill（建议 2～3 个），避免 Tool/Token 无界增长。
- 涉及真实交易 Mutation 的 Skill 不允许仅凭 Auto Router 获得额外权限，仍受原 Risk/Permission 边界约束。

如果需求属于已有 Team Runtime 的多 Agent 协作场景，继续复用 V3 Team Runtime；V4-1 不重新实现 Multi-Agent。

## 3. Conversation Skill 数据

建议新增轻量关系表而不是 JSON 字段：

```text
agent_conversation_skills
- conversation_id
- skill_name
- sort
- created_at
```

理由：

- Skill 可增删。
- 便于唯一约束。
- Portable Skill 删除/禁用时可校验。
- 不把 Conversation Model 继续膨胀成 JSON 配置。

创建新 Chat 时默认挂载 `general_chat`。

## 4. Model 切换

Conversation 增加可选：

```text
model_preference
```

更推荐保存 Model Gateway 中稳定的 model config ID，而不是只保存自由字符串。

UI 提供模型下拉：

- 只显示 enabled model。
- 切换后仅影响后续 Task。
- 历史消息保持原模型事实。
- Task / Observation 继续记录实际 provider/model。
- Model 不健康时仍可按现有 Gateway fallback policy 处理，并在 Task 中显示实际最终模型。

允许发送消息时临时 override；如果用户选择“设为当前对话模型”，再更新 Conversation preference。

## 5. 删除 Chat

后端已有 `DeleteChat`，V4-1 不重复实现。

前端补齐：

- 对话列表删除按钮。
- 二次确认。
- running task 时展示明确错误，不能强删。
- 删除后自动切换到其它 Chat / 新建 Chat。

删除范围维持现有安全语义：

- 删除 Conversation。
- 删除 Conversation Message。
- Task / Trace / Audit 保留，避免破坏运行审计。

## 6. UI

Chat 顶部：

```text
[Model ▼]   [Skills: general_chat, symbol_analysis, xxx +]
```

输入框附近：

```text
Skill: [Auto ▼]
       ├─ Auto
       ├─ general_chat
       ├─ symbol_analysis
       └─ my-portable-skill
```

也支持 `@skill-name` 快速显式指定。

Skill 管理弹层：

- 搜索 chat-capable Skill。
- 添加。
- 移除。
- 禁用/删除的 Skill 明确显示 unavailable。

## 7. API

在现有 `/agents/chat` 下补齐：

```text
GET    /agents/chat/conversations/:id/skills
POST   /agents/chat/conversations/:id/skills
DELETE /agents/chat/conversations/:id/skills/:skillName
PUT    /agents/chat/conversations/:id/model
```

发送消息请求增加：

```json
{
  "skill_mode": "auto",
  "skill": "",
  "model_id": "optional-message-override"
}
```

显式模式：

```json
{
  "skill_mode": "explicit",
  "skill": "symbol_analysis",
  "model_id": "optional-message-override"
}
```

## 8. Gate

- 同一 Chat 可挂载 3+ Skill。
- Auto 只能从已挂载且 chat-enabled 的 Skill 中选择。
- Explicit 可以稳定指定某个已挂载 Skill。
- Auto 在确有需要时可顺序使用 2 个以上 Skill，并为每次执行保留独立 Task/Trace。
- 删除已挂载 Skill 后不能继续通过 Auto 或旧请求调用。
- Skill A / Skill B 在同一 Conversation 中共享聊天历史，但仍使用各自执行契约。
- Model A → Model B 切换只影响新消息。
- 实际 Task 能追踪最终 provider/model。
- running Chat 不可删除。
- 删除 Chat 不删除 Task/Audit。

## 9. 本阶段不做

- 不并行执行所有 attached Skills。
- 不自动让 LLM 决定交易类 Skill 权限。
- 不新增 Model Provider。
- 不修改 Tool Permission 安全模型。
