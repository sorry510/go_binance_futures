# V4-2 Skill Studio Implementation Report

> 日期：2026-09-23  
> 状态：完成

## 1. 目标

V4-2 在现有“Skill 管理”页面中增加 Web Skill Studio，使 Portable Skill 可以直接在 Web 中创建、编辑、校验和发布，同时继续兼容项目现有标准 Agent Skills `SKILL.md` 格式。

核心原则没有改变：

```text
New / Clone Existing Version
  ↓
Draft
  ↓
Edit Files
  ↓
Validate
  ↓
Publish New Immutable Version
  ↓
Optional Activate
```

已发布或已激活的 revision 不允许原地修改。

## 2. 架构

V4-2 没有新增数据库表，也没有修改 `app.conf`。

Draft 默认目录固定为：

```text
data/agent-skill-drafts/<draft-id>/
├── draft.json
└── package/
    └── <skill-name>/
        ├── SKILL.md
        ├── references/
        ├── assets/
        └── scripts/
```

`draft.json` 只保存 Draft 元数据，不属于 Skill Package。

Draft 永远不会被 Agent Runtime 加载，也不会获得 Tool Permission。
正式发布仍复用现有：

```text
ParsePackage
  ↓
Importer.install
  ↓
AgentSkillVersion
  ↓
Permission Review
  ↓
Optional Activate
  ↓
SyncDefaultPortableSkills
```

因此 Package Hash、revision、rollback、permission review 和 Runtime Adapter 都继续使用 V2 既有实现。

## 3. Draft 能力

新增 `portableskill.DraftStore`：

- 创建新的标准 Skill Draft。
- 从已有 Portable Skill revision clone Draft。
- Draft 列表和详情。
- 读取文本文件。
- 新建/覆盖文本文件。
- 上传文本或二进制 Asset。
- 删除文件。
- 删除 Draft。
- 使用现有 `ParsePackage` 完整校验。
- 发布为新的 immutable revision。

新建 Draft 自动生成最小 `SKILL.md`：

```yaml
---
name: my-skill
description: Describe what this Skill does.
metadata:
  version: 0.1.0
---

# Instructions

Describe when this Skill should be used and how it should perform the task.
```
## 4. Web API

新增：

```text
GET    /agents/skills/drafts
POST   /agents/skills/drafts

GET    /agents/skills/drafts/:draftId
DELETE /agents/skills/drafts/:draftId

GET    /agents/skills/drafts/:draftId/file?path=...
PUT    /agents/skills/drafts/:draftId/file
DELETE /agents/skills/drafts/:draftId/file?path=...

POST   /agents/skills/drafts/:draftId/upload
POST   /agents/skills/drafts/:draftId/validate
POST   /agents/skills/drafts/:draftId/publish
```

新建 Skill：

```json
{
  "name": "my-skill"
}
```

Clone revision：

```json
{
  "source_version_id": 123
}
```

保存文本文件：

```json
{
  "path": "references/guide.md",
  "content": "# Guide\n..."
}
```

发布：

```json
{
  "activate": true,
  "delete_draft": false
}
```
## 5. 标准兼容与校验

V4-2 没有创建第二套 Skill Parser。

所有 Validate / Publish 最终都使用现有 `ParsePackage`，因此继续支持：

- `name`
- `description`
- `license`
- `compatibility`
- `metadata`
- `allowed-tools`

并继续执行现有规则：

- `SKILL.md` 必须以 YAML frontmatter 开始。
- 未知 top-level frontmatter 字段拒绝。
- `name` 必须与 package directory 一致。
- Markdown 本地引用必须存在。
- absolute path / `../` 拒绝。
- symlink / special file 拒绝。
- 单文件大小限制。
- Package 总大小限制。
- Package 文件数量限制。
- Draft 文件数或总大小超限时，本次写入会回滚；覆盖旧文件导致超限时恢复旧内容。
- 上传未显式指定目标 path 时，只使用客户端文件名的 basename，避免 `C:\\fakepath\\...` 形成无意义目录。
- `scripts/` 允许存储和读取，但产生 `script_execution_disabled` warning。

Draft 写文件时也会即时扫描文件数和总大小；如果本次写入导致超限，会回滚本次文件写入，而不是等到 Publish 才发现。

## 6. Tool Permission

`allowed-tools` 继续只是作者请求。

Publish 后流程：

```text
allowed-tools
  ↓
AgentSkillPermission requested
  ↓
ReviewPortableSkillPermissions
  ↓
管理员 Granted
```

Skill Studio 不会自动授予任何 Tool。

现有 RiskTrade 限制仍保持，Portable Skill 不能通过 Studio 获得直接交易 Tool 权限。

### 管理权限边界

发布或激活 Skill 不只是普通内容编辑：`SKILL.md` 正文会作为 Portable Skill 的 Agent System Prompt，因此 **Publish / Activate 等同于管理员级 Agent 行为配置变更**。

当前系统是单用户自用部署，相关 API 已经过现有 JWT 鉴权，因此 V4-2 不再增加一套角色系统。如果未来扩展为多用户，必须把 Draft Publish / Version Activate 限制为管理员角色；不能让任意普通登录用户改写 Agent 提示词。
## 7. Immutable Revision

“编辑为新版本”不会修改原 revision。

流程：

```text
Portable Skill current revision
  ↓ clone files
Draft
  ↓ edit
Validate
  ↓
new Package Hash
  ↓
new AgentSkillVersion
```

已增加永久回归测试：

- 导入 v1。
- Clone v1 为 Draft。
- Draft 修改为 v2。
- Publish。
- 断言 v1 与 v2 ID 不同。
- 断言 Package Hash 不同。
- 重新读取 v1 `SKILL.md`，内容必须完全不变。
- v2 保存新的内容。
- Version History 同时存在两个 revision。

因此 Web Editor 不会破坏既有 rollback 语义。

## 8. Web UI

继续使用原 `AI → Skill 管理` 页面，没有增加单独 Developer Portal。

页面顶部新增：

```text
[草稿] [新建标准 Skill] [导入标准 Skill] [新增 Native Skill]
```

Portable Skill 行新增：

```text
[编辑为新版本] [版本] [编辑] [删除]
```

`编辑为新版本` 会 clone 当前 active revision；如果没有 active revision，则使用最新 revision。
Skill Studio 布局：

```text
┌──────────────┬──────────────────────────────┬──────────────┐
│ 文件树       │ CodeMirror / Raw Preview     │ Package 校验 │
│              │                              │              │
│ SKILL.md     │ 当前文件                      │ version      │
│ references/  │                              │ hash         │
│ assets/      │                              │ files/size   │
│ scripts/     │                              │ tools        │
│              │                              │ diagnostics  │
└──────────────┴──────────────────────────────┴──────────────┘
```

顶部操作：

```text
[新建文本文件]
[上传 Asset]
[保存文件]
[校验]
[发布新版本]
[发布并激活]
```

文件编辑行为：

- `SKILL.md` 和其它文本文件使用 CodeMirror。
- Raw Preview 可直接查看当前文本。
- 文件有未保存修改时切换其它文件会二次确认。
- `SKILL.md` 不允许删除。
- 二进制 Asset 支持上传/删除，但不进行在线文本编辑。
- Draft 可随时关闭 Studio，之后从“草稿”列表继续编辑。

## 9. Publish / Activate

`发布新版本`：

- 自动保存当前文件。
- 自动 Validate。
- 校验失败不允许 Publish。
- 创建 immutable revision。
- 执行 Permission Review。
- 不自动启用 Skill。

`发布并激活` 在上述流程后额外执行：

- Activate revision。
- `SyncDefaultPortableSkills`。
- 新 Skill 可直接进入现有 Portable Runtime / V4-1 Chat Skill 目录（仍受 enabled/chat-enabled 条件控制）。
## 10. 自动测试

本阶段新增/强化覆盖：

- 创建最小 Draft。
- 编辑 `SKILL.md`。
- 创建 `references/` 文件。
- 标准 Validate 成功。
- requested tools 正确解析。
- 缺失 Markdown reference 被拒绝。
- `name` / directory mismatch 被拒绝。
- `../` path traversal 被拒绝。
- absolute path 被拒绝。
- `SKILL.md` 删除被拒绝。
- scripts 可以保存，但必须产生 execution-disabled warning。
- 多文件累积与删除。
- Draft Clone → Publish 生成新 immutable revision。
- 旧 revision 内容和 hash 保持不变。
- broken YAML 经 Draft Validate 被拒绝。
- 文件数超限时拒绝并删除本次新增文件，Draft 保持 Valid。
- 覆盖已有文件导致 Package 总大小超限时拒绝并恢复覆盖前内容。
- 上传默认文件名会净化 `C:\\fakepath\\helper.sh`、绝对路径和相对路径，只保留 basename。
- Beego Portable Skill 测试 DB 改为独立 alias，避免多个 revision 测试互相污染。

Gate：

```text
go test -count=1 ./...                                   PASS
go test -count=1 -race ./agent/portableskill ./controllers PASS
go build ./...                                           PASS
git diff --check                                         PASS

pnpm typecheck                                           PASS
pnpm build                                               PASS
```

race 只有既有 macOS linker `LC_DYSYMTAB` warning；前端只有 Browserslist/caniuse-lite 过旧提示，不影响结果。

V4-2 没有数据库 Schema 变化，因此数据库版本仍保持 V4-1 的 v18，本阶段没有执行 `sync db`。
## 11. 人工测试

### A. Web 新建标准 Skill

1. 打开 `AI → Skill 管理`。
2. 点击“新建标准 Skill”。
3. 输入 `web-skill-test`。
4. Studio 应打开，并自动包含 `SKILL.md`。
5. 把 `metadata.version` 改为 `1.0.0`，修改 description 和 Instructions。
6. 点击“保存文件”。
7. 点击“校验”，右侧应显示校验通过、version、hash、文件数和大小。

### B. Reference 校验

1. 点击“新建文本文件”。
2. 创建 `references/guide.md`。
3. 在 `SKILL.md` body 添加 `[guide](references/guide.md)`。
4. 校验应通过。
5. 删除 `references/guide.md` 后重新校验，应明确提示 reference 不存在。

### C. 路径安全

尝试创建：

```text
../escape.md
/tmp/escape.md
```

两者都必须被后端拒绝。

### D. 发布

1. 恢复为合法 Package。
2. 点击“发布新版本”。
3. Portable Skill 列表应出现该 Skill，但未激活时 Runtime 不应加载它。
4. 打开“版本”，应看到新 revision、Package Hash、Files 和 Permission Review。

### E. 发布并激活

1. 从草稿继续编辑，或新建另一个合法 Draft。
2. 点击“发布并激活”。
3. Portable Skill 应变为 enabled，active revision 指向新版本。
4. 如果 `chat_enabled=1`，V4-1 Chat 输入 `/` 后应可以找到该 Skill。

### F. 编辑为新版本

1. 找一个已有 Portable Skill。
2. 点击“编辑为新版本”。
3. Studio 顶部应显示来源 Revision。
4. 修改 `metadata.version` 和正文。
5. 发布新版本。
6. 打开版本历史，应同时看到旧 revision 和新 revision。
7. 查看旧 revision 的 `SKILL.md`，内容必须仍是旧版本。

### G. allowed-tools

在 `SKILL.md` 添加：

```yaml
allowed-tools: get_symbol_snapshot get_market_condition
```

Validate 右侧应展示两个 Requested Tools。

发布后 Permission 表中应为 Requested/Review 状态，不能自动 Granted。

### H. scripts

上传：

```text
scripts/helper.sh
```

Validate 应成功，但 Diagnostics 必须显示 scripts execution disabled。系统不应执行该脚本。

## 12. 本阶段未做

- 不执行 `scripts/`。
- 不做 Git IDE / Git commit。
- 不做多人协作编辑。
- 不用 LLM 自动修改已激活 revision。
- 不允许 Draft 绕过 Portable Skill Permission 模型。
- 不复制第二套 Agent Skills parser。

