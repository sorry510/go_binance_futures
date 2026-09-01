# Phase V2-5：标准 Agent Skills 包、导入与运行

## 目标

支持导入符合 Agent Skills 开放规范的 Skill，并与现有 Native Go Skill 共存。导入格式不能变成项目私有格式。

标准参考：https://agentskills.io/specification

## 标准 Skill 结构

最小结构：

```text
skill-name/
└── SKILL.md
```

标准还允许 `scripts/`、`references/`、`assets/` 和其他资源。`SKILL.md` 必须包含 YAML frontmatter + Markdown body。

Frontmatter 严格支持标准字段：

- 必填：`name`、`description`。
- 可选：`license`、`compatibility`、`metadata`、`allowed-tools`。

项目专用版本号应放在 `metadata.version`；包自身不得声明系统信任状态。现有本地 Skill 中若使用顶层 `version`、`trusted`，V2 标准导入器应提示迁移，而不是默默扩展标准。

## 支持的导入载体

### ZIP

ZIP 只是传输容器，不定义新的 Skill 格式。允许：

```text
skill-name/SKILL.md
skill-name/references/...
```

也允许 ZIP 根直接是 `SKILL.md`，导入器根据 frontmatter `name` 创建标准目录。

### 单文件

允许上传单个名为 `SKILL.md` 的 Markdown 文件。解析 `name` 后构建单文件 Skill Package。

### 服务器目录

管理员可从受允许的数据目录导入一个包含 `SKILL.md` 的目录；Web 普通用户不允许提供任意服务器绝对路径。

## Validation Pipeline

```text
Upload
 -> archive/file safety check
 -> unpack staging
 -> locate SKILL.md
 -> YAML/frontmatter parse
 -> Agent Skills spec validation
 -> file/reference validation
 -> security scan
 -> calculate package hash
 -> permission review
 -> install revision
 -> optional activation
```

应优先对齐 `skills-ref validate` 的标准校验语义，并在测试中放入官方标准的 valid/invalid fixture。

## 文件安全

- 防 Zip Slip/path traversal。
- 禁止归档内绝对路径、`..`、设备文件和危险 symlink。
- 限制压缩前/后大小、文件数和单文件大小，防 zip bomb。
- 计算 SHA-256 package hash。
- 上传进入 staging，校验通过前不进入 active store。
- 导入失败完整清理 staging。

## Package Store

标准 Skill 不适合只塞进当前 `agent_skills` 一行 DB。建议：

```text
<data_dir>/agent-skills/<name>/<package_hash>/
```

数据库保存 Registry/Revision 元数据，文件保存在持久化 volume。建议新增：

```text
agent_skills            logical skill / active revision / enabled
agent_skill_versions    hash / metadata / source / validation status
agent_skill_permissions requested/granted tool permissions
```

同名同 hash 重复导入为 no-op；同名新 hash 创建 revision，可切换 active revision 和 rollback。

## Native Skill 与 Portable Skill

保留现有 Go `skill.Skill` 作为 `native` 类型，它适合严格 Validator、Domain Service 和复杂输出契约。

新增 `portable` 类型，通过 Generic Skill Adapter 运行：

- metadata 参与 Skill Router。
- 激活后加载完整 `SKILL.md`。
- references/assets 按 Context Engine 需要加载。
- 运行仍走统一 Runtime/Tool Runtime。
- 可由系统侧 Execution Profile 绑定输出 Schema/Validator；这些配置不写回标准 Skill 包。

## `allowed-tools`

Agent Skills 标准把 `allowed-tools` 定义为可选且实验性字段。本项目解释为“作者请求预批准的工具集合”，绝不等于实际授权。

导入后执行映射：

```text
requested allowed-tools
 -> resolve Native/MCP Tool names
 -> system permission review
 -> granted tool allowlist
```

找不到、歧义或高风险 Tool 默认拒绝。MCP Tool 使用完整 namespace。

## Script 策略

V2 首版：`scripts/` 可被索引和读取，但**默认禁止执行**。

后续若支持执行，必须引入独立 Sandbox Executor、依赖白名单、资源限制、网络策略和审计；不能让 Skill 中一句“运行 scripts/x.sh”直接获得 shell 权限。

## Progressive Disclosure

遵循 Agent Skills 推荐模式：

1. Catalog 只加载 name/description 等 metadata。
2. Skill 被选择后加载 `SKILL.md` body。
3. references/scripts/assets 仅在需要时读取。

这与 V2 Context Engine 共用预算和 Trace。

## Web 管理

Skill 管理增加：导入 ZIP/SKILL.md、标准校验结果、来源、package hash、版本历史、文件浏览、requested/granted tools、active revision、rollback、删除。

可增加“导出 ZIP”，保持标准 Package 可移植性。

## 验收

- [ ] 官方规范有效 Skill 可导入。
- [ ] 非法 name/frontmatter/目录结构被明确拒绝。
- [ ] ZIP traversal、zip bomb、symlink Case 被拒绝。
- [ ] 同名 Skill 支持 revision 和 rollback。
- [ ] `allowed-tools` 不会自动升级权限。
- [ ] Imported Skill 不需要编译 Go 代码即可运行。
- [ ] Native Go Skill 行为保持兼容。
- [ ] Script 默认不可执行。
