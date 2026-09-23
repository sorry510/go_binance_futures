# Phase V4-1：Chat Workspace

## 1. 目标

把现有“一个消息选择一个 Skill”的 Chat 升级为真正可长期使用的个人 AI Workspace：

- 一个 Conversation 可以添加多个 Skill。
- 支持随时增删 Conversation 中的 Skill。
- 默认始终使用 `Auto` 模式，从当前 Conversation 已加入的 Skill 集合中选择当前消息的 Primary Skill。
- 一个 Conversation 可以通过 `/` 持续加入多个 Skill；加入新 Skill 不覆盖已有 Skill。单条消息仍只执行一个 Primary Skill，需要多 Agent 协作时继续复用现有 Team Runtime。
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

1. **Auto**：Web 默认且常态模式。Chat Router 只在已加入、enabled、chat-enabled 的 Skill 中确定一个 Primary Skill；无法可靠匹配时回退 `general_chat`。
2. **Add Skill**：用户输入 `/` 从全部 chat-enabled Skill 中选择，含义是把该 Skill 加入当前 Conversation 的 Skill 集合；再次选择其它 Skill 会继续累积，不覆盖已有 Skill。后端 `explicit` 请求模式仅为 API/兼容用途保留，不作为 Web 常规交互。

约束：

- `general_chat` 永远作为基础 fallback，不要求用户移除/添加。
- Auto Router 只能选择该 Conversation 已挂载、enabled 且 chat-enabled 的 Skill。
- Portable Skill 继续使用现有 Portable Runtime。
- Native Skill 保持自己已有的 Validator / Tool / Output Contract。
- 第一版不把多个 Skill 的 System Prompt 合并，也不在单条消息内自动串联多个 Skill，避免 Tool/Token 无界增长。
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
model_config_id
```

更推荐保存 Model Gateway 中稳定的 model config ID，而不是只保存自由字符串。

UI 提供模型下拉：

- 显示 enabled model 和 Router Candidate model。
- `Auto · 模型路由`（ID=0）继续使用现有 Model Gateway 与 fallback policy。
- 用户显式选择某个 model config 时严格使用该配置，不静默切换到其它模型；失败会明确报错。
- 切换后仅影响后续 Task，历史消息保持原模型事实。
- Task / Observation 继续记录实际 provider/model/model_config_id。

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

V4-1 最终采用接近 ChatGPT 的聊天交互，不在页面顶部或输入区常驻暴露 Skill 管理控件。

```text
┌──────────────────────────────────────────────┐
│ 输入消息……                                  │
│                                              │
│ 输入 / 选择 Skill     [合约]   [Model ▼] [发送] │
└──────────────────────────────────────────────┘
```

- 不显式选择 Skill 时天然就是 Auto，无需 Auto/Explicit 下拉。
- 输入 `/` 才显示 Skill 菜单。
- `/` 菜单展示全部 enabled、chat-enabled Skill，而不是只展示已挂载 Skill。
- 选择 Skill 的语义是**加入当前 Conversation 的 Skill 集合**，不会覆盖之前已经加入的 Skill。
- 已加入的业务 Skill 以多个可关闭标签同时显示；关闭标签才会从 Conversation 移除。
- 消息发送始终默认走 Auto，由 Router 在当前 Conversation 已加入的多个 Skill 中选择 Primary Skill；Web 不维护单值 `selectedSkill`。
- 合约选择会作为 Auto Router 的显式路由信号；如果最终回退 `general_chat`，所选合约仍会作为强制上下文传入，禁止被历史币种替换。
- Model 选择放在 composer 下方右侧，不占用 Chat 顶部。
- 禁用/删除的 Skill 不进入可选菜单。

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
- `/` 连续加入多个 Skill 后，所有 Skill 都保留在当前 Conversation，Web 发送仍保持 Auto。
- 单条消息只产生一个 Primary Skill Task；需要多 Agent 协作时继续复用已有 Team Runtime。
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

## 10. 实现结果

**V4-1 已完成（2026-09-19）。**

实现、API、Schema、安全边界、自动测试与人工测试步骤见 [v4-1-implementation-report.md](./v4-1-implementation-report.md)。
