# Phase V4-2：Web Skill Studio

## 1. 目标

让用户不需要在服务器文件系统手工创建 Skill，可以直接在 Web 中编写、校验、预览并发布标准 Agent Skills Package。

必须继续兼容 V2 已实现的 Agent Skills 标准解析器和安全边界。

## 2. 编辑模型

禁止直接修改已经发布/激活的 immutable Skill Version。

统一流程：

```text
New / Clone Existing Version
  ↓
Draft
  ↓
Edit Files
  ↓
Validate
  ↓
Publish New Version
  ↓
Optional Activate
```

这样现有 Package Hash / Version / Rollback 语义不被破坏。

## 3. 支持的标准结构

```text
my-skill/
├── SKILL.md
├── references/      optional
├── assets/          optional
└── scripts/         optional
```

`SKILL.md` 继续使用现有 frontmatter：

```yaml
---
name: my-skill
description: ...
license: ...
compatibility: ...
metadata:
  version: 1.0.0
allowed-tools: tool-a tool-b
---
```

V4-2 以项目当前 parser 支持范围作为最低兼容标准；如果标准新增字段，先更新 parser fixture 后再暴露给 Web。

## 4. Web Editor

复用前端现有 CodeMirror 能力，提供：

- 文件树。
- `SKILL.md` Markdown/YAML 编辑。
- 新建 reference / asset 文本文件。
- 删除文件。
- Raw Preview。
- Frontmatter Validation。
- Link/reference Validation。
- requested tools / permissions 预览。
- Package size / file count。

对于二进制 asset 首版可只支持上传/删除，不要求在线编辑。

## 5. Draft 存储

Draft 与正式版本隔离。

建议默认目录：

```text
data/agent-skill-drafts/<draft-id>/
```

不新增 `app.conf` 配置项。

Draft 不能被 Runtime 加载，也不能获得 Tool Permission。

Publish 时：

1. 使用现有 `ParsePackage` 完整校验。
2. 使用现有 Importer/Store 安装新 immutable version。
3. 生成 package hash。
4. 用户选择是否 activate。
5. Draft 可保留或删除。

## 6. 安全边界

- 路径必须经过现有 `safeJoin` 类逻辑。
- 禁止 `../`、absolute path、symlink。
- 文件数/单文件大小/Package 总大小继续使用 V2 限制。
- `scripts/` 首版仍然**不执行**。
- Web Editor 不能访问 Skill Draft 根目录以外的服务器文件。
- `allowed-tools` 仍然只是权限请求，不能自授 Tool Grant。
- **发布/激活 Skill 属于管理员级配置变更**：`SKILL.md` 正文会进入 Agent 的 System Prompt，等价于修改 Agent 行为。当前项目为单用户自用部署，由现有 JWT 保护即可；如果未来支持多用户，Draft Publish / Activate 必须增加管理员角色权限，不能仅依赖“已登录”。

## 7. UI

现有“Skill 管理”页增加：

```text
[新建 Skill]
[编辑为新版本]
[查看文件]
[校验]
[发布]
[发布并激活]
```

不新建独立 Developer Portal。

## 8. Gate

- Web 创建最小标准 Skill 并成功发布。
- Clone 已有版本 → 修改 → 发布新版本，旧版本保持不变。
- 非法 YAML / 未知字段 / name-directory mismatch 被阻止。
- 引用不存在文件被阻止。
- path traversal 被阻止。
- scripts 可存储但不会执行。
- 发布后的 Skill 可直接加入 V4-1 Chat。

## 9. 本阶段不做

- 不在线执行 Skill scripts。
- 不做 Git IDE。
- 不做多人协同编辑。
- 不自动让 LLM 无审查修改已激活 Skill。

## 10. 实现结果

**V4-2 已完成。**

实现继续复用 V2 Portable Skill Parser / Importer / Store，没有新增数据库 Schema，也没有新增 `app.conf` 配置。Draft 仅存储在 `data/agent-skill-drafts/<draft-id>/`，正式发布仍生成 immutable Portable Skill revision。

完整实现、API、安全边界、自动测试和人工测试步骤见 [v4-2-implementation-report.md](./v4-2-implementation-report.md)。
