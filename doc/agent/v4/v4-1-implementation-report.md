# V4-1 Chat Workspace Implementation Report

> 日期：2026-09-19  
> 状态：实现完成，最终 Gate 见本文末尾。

## 1. 已实现能力

- Conversation 可挂载多个 chat-capable Skill。
- 新 Chat 默认挂载 `general_chat`，且不能移除。
- Chat 默认就是 Auto Skill 模式，不提供额外的 Auto/Explicit 常驻下拉。
- 输入 `/` 才打开 Skill 菜单；菜单展示全部 chat-enabled Skill。
- 通过 `/` 选择 Skill 时只把它加入当前 Conversation 的 Skill 集合，不覆盖已有 Skill；已加入的业务 Skill 会以多个标签同时显示。
- 每条消息仍按 Auto 模式发送，由 Router 在已加入的多个 Skill 中选择 Primary Skill；关闭某个标签才会把该 Skill 从 Conversation 移除。
- Conversation 因此可以逐步累积多个 Skill，用户不需要理解额外的“挂载”管理页面。
- Conversation 可保存 `model_config_id` 作为后续消息的模型偏好。
- 模型选择位于输入框下方右侧，可切换 `Auto · 模型路由` 或具体 LLM Config。
- 消息 API 支持 message-level `model_id` override。
- Chat 删除继续复用 V2 已有安全语义。

## 2. 数据模型

数据库版本由 17 升级到 18。

`agent_conversations` 新增：

```text
model_config_id BIGINT default 0
```

`0` 表示继续使用现有 Model Gateway；正数表示固定使用对应 LLM Config。
新增关系表：

```text
agent_conversation_skills
- id
- conversation_id
- skill_name
- sort
- created_at
```

`conversation_id + skill_name` 使用组合唯一约束。

version 18 migration 会为升级前已有的 Chat Conversation 补 `general_chat` 绑定。

Schema 仍只通过：

```bash
./go_binance_futures sync db
```

升级已实际执行成功：17 → 18。

## 3. Skill 路由

### Explicit

用户显式选择 Skill 时：

- Skill 必须已经挂载到当前 Conversation。
- Skill 必须当前 enabled 且 chat-enabled。
- `general_chat` 作为不可移除的基础 Skill，也可以被显式选择。
- 不允许通过旧请求绕过 attached Skill 边界。

### Auto

Auto 只在当前 Conversation 已挂载且仍可用的 Skill 中选择一个 Primary Skill。

第一版采用确定性轻量 Router：

- `@name` / `/name` 明确引用优先。
- Skill name / display name 命中。
- Native 常用 Skill 使用少量领域关键词辅助。
- 无可靠匹配时回退 `general_chat`。
- Portable Skill 不会因为模糊语义被任意自动选中；名称/显示名明确匹配才有较高机会。

单条消息只执行一个 Primary Skill，不合并多个 System Prompt，也不自动串联多个 Skill。

需要多 Agent 协作时继续使用 V3 Team Runtime。

## 4. 模型切换

模型选择分两种：

1. `model_config_id = 0`：继续使用现有 Model Gateway，包括原有路由与 fallback。
2. `model_config_id > 0`：严格使用用户指定的 LLM Config。

显式模型选择不会静默 fallback 到其它模型；指定模型失败时返回明确错误，避免用户以为正在使用 A 实际却用了 B。

普通 Skill Task 与 Symbol Analysis Team 都会传播该模型配置。
Team 场景中：

- technical child
- flow child
- news child
- supervisor

都会收到相同 `model_config_id`。

Task 继续记录实际：

- provider
- model
- model_config_id
- final_model_config_id

因此历史任务不会因为后续切换模型而改变。

## 5. API

新增：

```text
GET    /agents/chat/conversations/:id/skills
POST   /agents/chat/conversations/:id/skills
DELETE /agents/chat/conversations/:id/skills/:skillName
PUT    /agents/chat/conversations/:id/model
```

发送消息扩展：
```json
{
  "skill_mode": "auto",
  "skill": "",
  "content": "分析 BTCUSDT",
  "symbol": "BTCUSDT",
  "model_id": 0
}
```

`model_id` 可省略；省略时使用 Conversation 的模型偏好。

Explicit 示例：

```json
{
  "skill_mode": "explicit",
  "skill": "symbol_analysis",
  "content": "分析 BTCUSDT",
  "symbol": "BTCUSDT"
}
```

响应会返回实际启动 Task 的 `skill` 和 `model_config_id`。

## 6. Web UI

V4-1 最终采用接近 ChatGPT 的输入交互，不在页面顶部或输入区常驻暴露 Skill 管理控件。

正常状态：

```text
┌──────────────────────────────────────────────┐
│ 输入消息……                                  │
│                                              │
│ 输入 / 选择 Skill     [合约]   [Model ▼] [发送] │
└──────────────────────────────────────────────┘
```

- 不选 Skill 时天然就是 Auto，不需要用户操作“Auto”下拉。
- 输入 `/` 才弹出 Skill 菜单。
- 菜单展示全部 chat-enabled Skill，并标识已加入当前 Conversation 的 Skill。
- 选择未加入的 Skill 时后台自动加入；同一 Conversation 可以逐步使用多个 Skill。
- 已加入的业务 Skill 以多个持久标签同时显示；关闭某个标签才从 Conversation 移除，发送消息不会清空标签。
- 消息本身始终以 Auto 模式发送，由 Router 在这些已加入 Skill 中选择 Primary Skill。
- Model 位于 composer 下方右侧，选择 Auto Router 或具体 LLM Config。
- 运行中的 Task 期间禁用模型和发送相关操作，避免中途改变执行上下文。

原有删除聊天能力保留：

- 二次确认。
- running task 时禁止删除。
- 删除消息和 Conversation。
- Task / Trace / Audit 历史保留。

## 7. 安全边界

- Attached Skill 不等于自动获得 Tool Permission。
- Auto Router 不会扩大 Skill 权限。
- Disabled / non-chat-enabled Skill 不能被新 Task 调用。
- `general_chat` 是 Auto 无可靠匹配时的 fallback，也可作为无 Tool 的 Explicit Skill。
- 显式模型必须是 enabled 或 Router Candidate 配置。
- 多 Skill 不合并 System Prompt。
- 不重新实现 Multi-Agent。

## 8. 自动测试

新增/强化覆盖：

- 新 Chat 默认 `general_chat`。
- Skill attach 幂等。
- Skill remove。
- `general_chat` 不可移除。
- Conversation model preference round-trip。
- Auto 只选择 attached Skill。
- Auto 无匹配回退 `general_chat`。
- Explicit 可选择 `general_chat`。
- 旧消息入口已收敛到同一 attached Skill 校验路径。
- Auto 路由覆盖单关键词、阈值、名称、显示名和 `/name` 边界。
- Team 所有 child/supervisor 继承指定 model config。
- SQLite additive schema upgrade 包含新表/字段。

## 9. 人工测试

### A. 默认 Auto

1. 打开 `AI → 对话` 并新建 Chat。
2. 页面顶部不应出现 Skill 或 Model 下拉，输入区也不应出现 Auto/Explicit Skill 下拉。
3. 不输入 `/`，直接发送普通消息；系统应按 Auto 模式执行。
4. 当前消息没有显式 Skill 时，不显示 Skill 标签。

### B. `/` 选择 Skill 与多 Skill

1. 在输入框输入 `/`，应弹出全部 chat-enabled Skill。
2. 选择 `symbol_analysis`；若它此前未加入当前 Conversation，应自动加入，不额外弹出管理流程。
3. 输入区出现 `/ 单币分析` 标签，并持续保留。
4. 再输入 `/` 选择另一个 Skill，第一个标签必须仍然存在，两个 Skill 同时属于当前 Conversation。
5. 选择 BTCUSDT 并发送“分析这个币”，消息仍以 Auto 模式发送；Task 应根据已加入 Skill 和所选合约路由到 `symbol_analysis`。
6. 发送完成后两个 Skill 标签仍保留；只有点击某个标签的关闭按钮才移除对应 Skill。

### C. Auto Skill

1. 先通过 `/` 使用一次 `symbol_analysis`，让它成为当前 Conversation 的 attached Skill。
2. 下一条消息不要再选择 Skill，直接输入“分析 BTCUSDT 的技术走势和支撑阻力”。
3. Task Detail 应看到 Auto 路由到 `symbol_analysis`。
4. 输入普通文本问题，无法可靠匹配业务 Skill 时应回退 `general_chat`。

### D. 模型切换

1. 在输入框下方右侧 Model 选择具体 LLM Config A。
2. 发送普通消息，Task Detail 检查 model/model_config_id 对应 A。
3. 切换为 Config B 再发送，新 Task 应使用 B；旧 Task 保持 A。
4. 切回 `Auto · 模型路由`，后续 Task 应恢复 Gateway 路由。

### E. Team 模型

1. 输入 `/` 并选择 `symbol_analysis_team`。
2. 在输入框下方右侧指定一个具体模型。
3. 发起 BTCUSDT Team Analysis。
4. Task Center 查看 parent 和 child Tasks。
5. technical / flow / news / supervisor 应使用相同 model config。

### F. 删除 Chat

1. 空闲 Chat 删除应成功。
2. 正在运行 Task 的 Chat 删除应被拒绝。
3. Task 完成后删除 Chat。
4. Chat 消息消失，但 Task Center 中历史 Task 仍可查看。

## 10. 本阶段未做

- 单消息自动串联多个 Skill。
- 新 Model Provider。
- Tool Permission 模型修改。
- Skill Web Editor（V4-2）。

## 11. Gate 结果

- `go test ./...`：PASS。
- `go test -count=1 ./command ./models`：PASS。
- `go test -race ./agent/conversation ./agent/app ./agent/manager ./agent/team ./models ./controllers`：PASS。
- `go build ./...`：PASS。
- `git diff --check`：PASS。
- 前端 `pnpm typecheck`：PASS。
- 前端 `pnpm build`：PASS。
- 前端 dist 已同步到后端 static。
- `./go_binance_futures sync db`：PASS，数据库 Schema version=18。

race 阶段只有 macOS linker 的 `LC_DYSYMTAB` warning，测试本身全部通过；前端 build 只有 Browserslist/caniuse-lite 过旧提示，不影响构建。
