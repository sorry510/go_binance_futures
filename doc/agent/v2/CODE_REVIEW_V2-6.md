# Code Review 报告 — Phase V2-6：标准 Agent Skills 包、导入与运行

> 审查模式：**仅 review，未修改任何代码**，未提问。
> 对照文档：`doc/agent/v2/06-phase-v2-6-agent-skills.md`
> 范围锁定：`git status --porcelain` + `git diff --stat`

## 结论

**PASS（通过 Gate，可进入 V2-7）。**

- 后端 `go build ./...` 通过。
- `go test -count=1 ./...` 全量通过（所有包 `ok`）；`go test -count=1 -race` 重点包（`agent/portableskill`、`agent/app`、`agent/skillconfig`、`models`、`agent/tools/domain`）通过。
- 8 项验收全部满足，关键安全路径（ZIP Slip / symlink / zip bomb / 路径穿越 / 包哈希校验 / `allowed-tools` 不自动授权 / `scripts` 禁止执行）实现正确且有单元测试覆盖。
- 数据库迁移向后兼容（legacy SQLite 行在 additive sync 后仍可读）。

非阻塞观察 N1–N6 均为改进建议，不阻塞 Gate。

---

## 1. 审查范围与方式

**后端改动（git diff --stat 实测）**
- Modified：`agent/app/default.go`、`agent/skill/registry.go`、`agent/skill/version.go`、`agent/skillconfig/store.go`、`agent/tools/registry.go`、`conf/app.conf.example`、`controllers/agent_skill.go`、`main.go`、`models/agent_skill.go`、`models/agent_task_syncdb_test.go`、`routers/router.go`、`doc/agent/v2/06-phase-v2-6-agent-skills.md`、`doc/agent/v2/README.md`、`.gitignore`、`static/index.html`
- New：`agent/app/portable_skills.go`、`agent/portableskill/`（types/parser/adapter/importer/store + parser_test/runtime_test）、`controllers/agent_skill_portable.go`、`models/agent_skill_version.go`
- 构建产物：`static/static/js/*.js`、`static/static/css/skillManagement-*.css` 为前端重新构建输出（非手改源码）

**前端说明（范围限制）**：本仓库 `static/` 仅含构建产物，Vue 源码位于独立仓库 `go_binance_futrues_new_ui`，不在本仓库 review 范围内。本次对后端 API 契约（路由、请求/响应结构）做了审查，确认与实现一致；前端源码 review 建议在该仓库单独进行。

**执行**：构建 + 全量测试（见上）；逐文件通读 `portableskill` 全部文件、`agent/app/portable_skills.go`、`default.go`、`governance.go`、`runner.go`、`skillconfig/store.go`、`skill/registry.go`、`skill/version.go`、`tools/registry.go`、`controllers/agent_skill*.go`、`routers/router.go`、`main.go`、`models/agent_skill*.go`、`agent_task_syncdb_test.go`、`middlewares/auth.go`、phase doc。

---

## 2. 验收逐条评估

| # | 验收项 | 结论 | 证据 |
|---|--------|------|------|
| 1 | 官方规范有效 Skill 可导入 | ✅ | `ParsePackage` 解析 frontmatter+body；`testdata/valid-minimal`、`valid-resources` fixture；`TestParseStandardFixtures` |
| 2 | 非法 name/frontmatter/目录结构被拒绝 | ✅ | `validName`=`^[a-z0-9]+(?:-[a-z0-9]+)*$`、长度 1–64、`--` 禁止、name==dirname；顶层 `version`/`trusted`/`unknown` 拒绝并提示迁移 `metadata.*`；`TestStrictFrontmatterRejectsPrivateTopLevelFields`、`TestParseRejectsNameDirectoryMismatch` |
| 3 | ZIP traversal / zip bomb / symlink 被拒绝 | ✅ | `unpackZIP` 多重防御（见 §3）；`TestZIPSecurityRejectsTraversalAndSymlink`（traversal / `C:` 盘符 / symlink）、`TestZIPSecurityRejectsTooManyEntries` |
| 4 | 同名 Skill 支持 revision 与 rollback | ✅ | `Store.Install` 同 hash 幂等（Duplicate）、新 hash 建 revision；`Store.Activate` 切换 `active_version_id`；`TestInstallRevisionRollbackAndAllowedToolsRequireReview` |
| 5 | `allowed-tools` 不会自动升级权限 | ✅ | 导入只写 `requested`/`granted=0`；`ReviewPortableSkillPermissions` 解析（0→unresolved / 1→resolved / 多→ambiguous，全 denied）；`SetPermissionGrant` 要求 `ResolvedName` 非空才允许 grant；测试断言 `Granted==0` |
| 6 | Imported Skill 不需编译 Go 代码即可运行 | ✅ | `Adapter` 实现 `skill.Skill`；`LoadAdapter` 运行时解析 `SKILL.md` + hash 校验后注册进共享 Registry |
| 7 | Native Go Skill 行为保持兼容 | ✅ | `skillconfig.Store` 共用 `agent_skills` 表，`Type` 区分；`EnsureDefaults` 防 native default 覆盖 portable；`AdmitSkill` 经共享表 gate；`DefaultManager` 仍注册 3 个 native definition |
| 8 | Script 默认不可执行 | ✅ | `scanPackageFiles` 对 `scripts/` 仅发 warning；`systemPrompt` 明确禁止执行；`read-resource` tool 仅读文本（<256KB，拒绝二进制）；全代码无 shell 执行入口 |

---

## 3. 安全分析（重点）

**Zip Slip / 路径穿越**
- `unpackZIP`：拒绝 `/` 前缀、`C:` 盘符段、`../` 段、`..`、绝对路径（`importer.go:197`）。
- `safeJoin`（`parser.go:195`）：`filepath.Clean` 后二次校验结果仍落在 root 内，作为 zip 解包与所有文件读取的统一防线。
- `ImportDirectory`：`safeJoin` + `rejectSymlinkPath` 双重约束服务器目录导入（`importer.go:97,352`）。

**Symlink**
- zip 内条目 `os.ModeSymlink` 拒绝（`importer.go:200`）。
- 解包后 `copyPackage` 拒绝 symlink（`importer.go:329`）。
- 服务器目录导入 `rejectSymlinkPath` 逐段 `Lstat` 拒绝（`importer.go:352`）。

**Zip bomb**
- 声明层：`maxFiles=256`、`maxSingleFileBytes=4MB`、`maxUnpackedBytes=32MB`（按 `UncompressedSize64` 累加）。
- 实际写入：`writeLimited` / `writeLimitedExisting` 用 `io.LimitReader(r, max+1)` 硬限单文件实际字节，杜绝压缩膨胀。
- 解包后 `scanPackageFiles`（`parser.go:114`）用**真实文件大小**二次校验（总字节 32MB、单文件 4MB）。
- 失败清理：导入失败 `defer os.RemoveAll(staging)`（`ImportFile`），DB 提交失败 `defer os.RemoveAll(finalPath)`（`install`），无半成品落盘。

**包完整性 / 防篡改**
- 安装时计算 SHA-256 package hash（`hashPackage`，文件排序后 `rel\x00content\x00` 拼接，确定性）。
- `LoadAdapter`（`adapter.go:29`）每次加载重新解析已安装包并比对 `pkg.PackageHash == version.PackageHash`，**drift 即报错**，检测磁盘篡改。

**权限边界（`allowed-tools`）**
- 导入仅产生 `requested` 权限（`granted=0`），绝不自动授权。
- `resolveRequestedTool`（`portable_skills.go:168`）：精确名匹配 → 单候选；否则按别名（去非字母数字）匹配；排除 `portable_skill` 源工具自身；0 候选 `unresolved`、≥2 候选 `ambiguous`，均默认拒绝。
- `SetPermissionGrant`（`store.go:188`）：授予前强制 `ResolvedName` 非空；高风险（非 Read）需管理员显式审批后才进入动态 allowlist。
- 运行时 `effectiveToolNames`（`runner.go:101`）将 `skill.Tools()`（仅 `read-resource`）与 `EffectiveToolAllowlist`（MCP + 已授权 portable 工具）并集进冻结 catalog；未授权工具不进入 catalog，沿用 V2-5 catalog 执行约束。

**鉴权**
- 新路由 `/agents/skills/import`、`/import-directory`、`/versions/*`、`/permissions/:id`、`/activate` 全部在 `JwtMiddleware` 下（`middlewares/auth.go`）；仅 `/agents/mcp/oauth/client-metadata`、`/callback`、`/login` 等白名单豁免。导入为管理员鉴权操作。

**Script 执行**
- 全代码无 `os/exec`、无执行 `scripts/` 的入口；`read-resource` tool 仅 `os.ReadFile` 文本并拒绝二进制（`null` 字节）。符合 phase doc "默认禁止执行"。

---

## 4. 集成与运行时

- `initializeDefaultPortableSkills`（`portable_skills.go:24`）在 `DefaultManager`（`default.go:45`）启动时把 active portable revision 载入**共享** `Skill Registry` 与 `Tool Registry`；`SyncDefaultPortableSkills` 在导入/激活/回滚/删除/启用时增量同步（受 `portableMu` 保护，注册表内部锁保护并发）。
- 运行时 `RuntimeConfig.ToolAllowlistProvider = EffectiveToolAllowlist`（`default.go:66`），与 MCP 共用同一动态 allowlist 合并逻辑，认知一致。
- 版本/回滚执行身份用 `package_hash` 冻结（`Adapter.VersionInfo` / `SkillPackageHash`），与 V2-5 `tool_catalog_hash` 机制对齐。
- `AdmitSkill`（`governance.go:31`）经共享 `agent_skills` 表 gate：portable active（`enabled=1`）即通过，未激活（`enabled=0`）即 "disabled"，符合预期。

---

## 5. 数据库迁移

- `models.AgentSkill` 新增 `type` / `active_version_id`（additive 列）；新增 `agent_skill_versions`、`agent_skill_permissions`；`main.go` 注册三者模型。
- `models/agent_task_syncdb_test.go` 扩展回归（`TestAgentTaskSyncdbUpgradesExistingSQLiteRows`）：legacy SQLite 行在 `RunSyncdb` 后仍可读；V2-6 表/列存在；native 默认 `type=native`/`active_version_id=0` 保持；V2 task 字段 round-trip。测试通过。

---

## 6. 非阻塞观察（建议，非 Gate 条件）

- **N1（低）**：`parseSkillMD`（`parser.go:68`）要求 frontmatter 闭合为 `\n---\n`；若 `---` 恰为文件末行且无尾换行会被拒。建议同时接受 EOF 处 `\n---$`。
- **N2（低）**：`Store.ReadFile`（`store.go:318`）上限 4MB，而 `adapter.read-resource`（`adapter.go:134`）上限 256KB，二者不一致（均拒绝二进制）。建议统一或显式区分意图（管理端浏览 vs 运行时上下文）。
- **N3（极低）**：`adapter.go:114-118` `filepath.Abs(path)` + `_ = info` 为死代码，可删。
- **N4（低）**：软删除 portable skill 后磁盘包文件（`DataDir/<name>/<hash>/`）不清理，长期可能堆积；如需可加孤儿清理（保留历史以支撑 undelete 亦可接受）。
- **N5（信息）**：`unpackZIP` 的 `total` 按声明 `UncompressedSize64` 累加，实际写入已按单文件 4MB 限流；恶意包最坏瞬时占 ~1GB staging，但解包后 `scanPackageFiles` 实际 32MB 校验失败即拒绝并清理 staging，不会持久化。建议在解压循环内改为按**实际累计字节**限流以更紧。
- **N6（信息/范围）**：前端仅见构建产物（`static/`），Vue 源码在 `go_binance_futrues_new_ui`，本次未纳入；后端 API 契约（路由、请求/响应）一致。

---

## 7. Gate 判定

| 维度 | 结果 |
|------|------|
| 构建 | ✅ `go build ./...` 通过 |
| 测试 | ✅ 全量 `go test ./...` 通过；race 重点包通过 |
| 验收 1–8 | ✅ 全部满足且有测试覆盖 |
| 安全关键路径 | ✅ Zip Slip / symlink / bomb / 穿越 / hash 校验 / 权限不自动升级 / script 禁执行 均正确 |
| DB 迁移 | ✅ 向后兼容，回归通过 |
| 阻塞项 | 无 |

**Gate 结论：PASS。可进入 V2-7。**
