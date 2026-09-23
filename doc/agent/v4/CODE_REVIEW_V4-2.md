# V4-2 Phase 审计报告：Web Skill Studio

- **审计对象**：V4-2 实现**在工作区未提交**（新增 `agent/portableskill/draft.go` 571 行 + `draft_test.go` 222 行、`controllers/agent_skill_draft.go` 223 行；改动 `agent/portableskill/parser_test.go`、`routers/router.go`、Phase 文档）。基线 `HEAD = b7cbc54`（`fix: ai agent v4-1`，分支 `feat/ai-agent-v4`）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝全部未提交文件；探针与副本审计后已删除，用户仓库未被改动）
- **审查类型**：review-only
- **结论**：**已完成，可进入 V4-3**。§8 七条 Gate **全部通过**；未发现 P0/P1 安全缺陷。风险集中在 1 项 P2（发布即等于"可改写 Agent 提示词"的管理员级操作，需在文档层面明确）与若干 P3（.gitignore 缺项、上传文件名净化、限额/非法 YAML 无永久回归测试）。
- **审计日期**：2026-09-23

> **修复更新（2026-09-23）**：F1～F4 已处理。F1 已在 Phase/README/实施报告明确 Skill Publish / Activate 属于管理员级 Agent System Prompt 变更；F2 已将 `/data/agent-skill-drafts/` 加入 `.gitignore`；F3 上传未显式指定 path 时会先将客户端文件名归一化为 basename，并新增 Windows `C:\\fakepath\\...` 等回归测试；F4 已把 broken YAML、文件数超限写入回滚、覆盖已有文件导致 Package 总量超限后恢复旧内容固化为永久测试。F5/F6 保持现状：损坏 Draft 不阻断其它 Draft 列表，部分发布错误继续使用明确的“已发布但后续步骤失败”消息。

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V4-2 是否完成 | ✅ **完成**（Web 新建/克隆草稿 → 编辑 → 校验 → 发布 immutable revision → 可选激活；复用 V2 Parser/Importer/Store，无第二套实现） |
| 是否可进入 V4-3 | ✅ **可以**（无阻塞项） |
| DB / 配置变更 | ❌ **无**：`models/`、`conf/`、`main.go` 均未被本阶段改动，Schema 仍为 v18，无需 `sync db` |
| 是否存在 P0/P1 | ❌ 未发现 |
| 安全边界 | ✅ 路径逃逸（`../`、绝对路径、符号链接、非法 draft id）、文件/包限额、二进制不作文本返回、`SKILL.md` 不可删、scripts 不执行、Tool 不自授 —— **均实证通过** |

> 附注：上一轮 V4-1 报告中 F1（`general_chat` 因未实现 `ChatAdapter` 而无法进入 Chat 目录）已由 `b7cbc54` 修复 —— `agent/skills/generalchat/skill.go:30-31` 现实现 `ChatEnabled()/BuildChatInput()`，并有永久回归测试 `agent/app/skill_catalog_test.go:40` 与 `agent/skills/generalchat/skill_test.go:17`。

---

## 2. 验收 Gate 逐项核对（§8 七条）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | Web 创建最小标准 Skill 并成功发布 | ✅ | `DraftStore.Create` 生成最小 `SKILL.md`（`draft.go:551-563`）→ `Validate`（`draft.go:359-369`，直接调 `ParsePackage`）→ `Publish`（`controllers/agent_skill_draft.go:161-219`：PackageRoot → Validate → `ImportDraft` → 权限 Review → 可选 Activate）。永久测试 `TestDraftCreateEditValidate`；**探针 P3 实证**：从草稿成功发布并落库为 1 个 revision |
| 2 | Clone 已有版本 → 修改 → 发布新版本，旧版本保持不变 | ✅ | `CloneVersion`（`draft.go:108-154`，仅 portable 且 `copyPackage` 拒绝符号链接）→ `Importer.ImportDraft`（`draft.go:565-571`）→ `install` 内容寻址 `data_dir/<name>/<hash>`。永久回归测试 `TestDraftClonePublishPreservesImmutableRevision`：断言新旧 ID/Hash 不同、旧 `SKILL.md` 逐字节不变、版本历史 2 条 |
| 3 | 非法 YAML / 未知字段 / name-directory mismatch 被阻止 | ✅ | **探针 P4 实证**：broken YAML → `yaml: line 1: did not find expected ',' or ']'`；未知字段 → `unsupported Agent Skills frontmatter field "private-key"`；缺 frontmatter → `must start with YAML frontmatter delimiter ---`；name 不匹配 → `must match parent directory "fm-draft"`；缺 `SKILL.md` → 拒绝。永久测试另有 `TestStrictFrontmatterRejectsPrivateTopLevelFields`、`TestParseRejectsNameDirectoryMismatch` |
| 4 | 引用不存在文件被阻止 | ✅ | 永久测试 `TestDraftValidationRejectsMissingReferenceAndNameMismatch`（`does not exist`） |
| 5 | path traversal 被阻止 | ✅ | **探针 P1 实证 9 类输入全部拒绝**：`../`、`../../`、`..\..\`、`a/../../`、`references/../../../`、`/etc/passwd`、`/tmp/escape.md`、`filepath.Join(outsideDir,...)`、以及"包根 → 外部文件"的相对路径；写入/读取/删除三个入口均拒绝；`SKILL.md`（含 `./`、`references/../` 归一化变体）删除被拒；非法 draft id（`../1`、`/tmp`、31 位、大写、超长、空）全部拒绝；**外部文件内容与目录均未被改动** |
| 6 | scripts 可存储但不会执行 | ✅ | 永久测试 `TestDraftScriptsAreStoredButValidationWarns`（`script_execution_disabled` warning）；全仓检索 `agent/` 内**无任何 `os/exec`/`exec.Command`** → scripts 无执行路径 |
| 7 | 发布后的 Skill 可直接加入 V4-1 Chat | ✅ | `agent/portableskill/adapter.go:62` `ChatEnabled() == true` → Portable Adapter 实现 `skill.ChatAdapter`；`Store.Activate` 置 `Enabled=1`（`store.go:271`）+ 写 change event（activate/rollback）；`agentapp.SyncDefaultPortableSkills` 同步进运行时注册表（`agent/app/portable_skills.go:31-78`，只遍历 `ActiveVersions`）→ 之后受 V4-1 既有的 `enabled && chat_enabled` 条件控制（报告 §9 已如实声明该前提） |

**§5/§6 安全与边界要求核对**

| 要求 | 判定 | 证据 |
| --- | --- | --- |
| Draft 与正式版本隔离，`data/agent-skill-drafts/<draft-id>/` | ✅ | `draft.go:18` `defaultDraftRoot`；`draft.json` 仅元数据（`draft.go:430-443`） |
| Draft 不能被 Runtime 加载、不能获得 Tool Permission | ✅ | **探针 P3 实证**：发布前 `GetSkillByName` 与 `ActiveVersions` 均无该 Skill；`SyncDefaultPortableSkills` 只读 DB 的 active versions，永不扫描 drafts 目录 |
| 不新增 `app.conf` 配置项 | ✅ | 工作区无 `conf/` 改动（`DraftStore.Root` 仅在测试中注入） |
| 路径必须经过现有 `safeJoin` 类逻辑 | ✅ | `draftDir`/`packageRootFor`/`draftFilePath` 均经 `safeJoin`（`parser.go:195-207`，显式拒绝 `filepath.IsAbs`）＋ `ensureNoSymlinkComponents`（`draft.go:497-538`） |
| 文件数/单文件大小/Package 总大小沿用 V2 限制 | ✅ | `maxFiles=256`、`maxSingleFileBytes=4MiB`、`maxUnpackedBytes=32MiB`（`importer.go:23-25`）；`WriteFile` 写入后调 `scanPackageFiles` 复核并**回滚**（`draft.go:295-303`）；**探针 P2 实证**：第 256 个文件之后写入失败、被拒文件无残留、草稿仍可校验通过 |
| `allowed-tools` 只是权限请求，不能自授 Tool Grant | ✅ | `ReviewPortableSkillPermissions`（`agent/app/portable_skills.go:107-167`）只写 `review_required`/`high_risk`/`unresolved`/`ambiguous`，**从不置 `Granted=1`**；解析时跳过 portable 来源的 Tool（防自引用）。**探针 P3 实证**：发布后权限行存在但 `granted != 1`，`GrantedToolNames` 为空 |
| 复用现有 `ParsePackage`，不建第二套 parser | ✅ | `Validate`/`install` 均调用 `ParsePackage`（`draft.go:364`、`importer.go:111`），本阶段无新增 parser 代码 |
| Publish 不自动激活；URL 安全 | ✅ | `ImportDraft` → `install(..., activate=false)`；**探针 P3 实证**：首次发布后 `Enabled!=1` 且 `ActiveVersionID` 未指向该 revision |
| API 与文档 §4 一致 | ✅ | `routers/router.go` 七条路由对应文档 10 个端点（`GET/POST drafts`、`GET/DELETE drafts/:id`、`GET/PUT/DELETE drafts/:id/file`、`POST upload|validate|publish`）；控制器请求体与文档示例一致 |
| §9 本阶段不做 | ✅ 未越界 | 无 Git IDE、无协同编辑、无 scripts 执行、无 LLM 自动改激活版本 |

---

## 3. 实证过的隐式契约（探针）

探针写在隔离副本（`/tmp/phase_audit`、`/tmp/pa2`），跑完随副本删除；探针以 `-run TestProbe` 单独执行，并另跑一次去掉探针的包测试确认原测试仍通过（均 PASS）。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **P1** 路径与文件安全矩阵 | 9 类逃逸输入 × 读/写/删三入口、二进制不作文本返回、符号链接文件/目录、`SKILL.md` 删除、非法 draft id、外部文件完整性 | ✅ 全部拒绝；无逃逸产物；外部文件未被修改（且 `WriteFile` 的 `ensureNoSymlinkComponents` 在 `MkdirAll` **前后各校验一次**） |
| **P2** 限额与回滚 | 单文件 > 4 MiB（写入前拒绝）、超 `maxFiles` 的第 257 个文件 | ✅ 单文件未落盘；文件数溢出被回滚（磁盘无残留、`files` 计数 = 256 上限、草稿仍 Valid）。**该回滚行为无永久回归测试保护**；「覆盖已存在文件导致总量超限」的分支本轮未构造（需 ~32 MiB 写入），亦无永久测试 |
| **P3** 草稿隔离 / 不可变 / 不自授权限 | 发布前 Store 与 ActiveVersions 均无该 Skill；发布后 1 个 revision、未激活；权限行 `granted=0`；重复发布相同内容 | ✅ 隔离成立；`Duplicate=true` 且**复用同一 revision**（版本数仍为 1，不产生冗余 immutable 版本）；`GrantedToolNames` 为空 |
| **P4** frontmatter 严格性（经 Draft Validate 路径） | broken YAML / 未知顶层字段 / 缺 frontmatter / name 不匹配 / 缺 `SKILL.md` | ✅ 五类全部拒绝且错误信息明确（见 Gate #3 证据） |

---

## 4. 风险与待改进

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| F1 | ✅ 已修复 | 「发布（并激活）」等价于**可改写 Agent 系统提示词**的管理员级动作；当前 `/agents/skills/drafts/*` 已受 `JwtMiddleware` 保护。 | 已在 Phase、V4 README 和实施报告明确“Skill Publish / Activate = 管理员级 Agent 行为配置变更”；当前单用户部署维持 JWT 即可，未来多用户必须增加管理员角色权限。 |
| F2 | ✅ 已修复 | 草稿目录原先未加入 `.gitignore`。 | 已加入 `/data/agent-skill-drafts/`，Web 草稿不会污染 Git 工作区。 |
| F3 | ✅ 已修复 | `UploadFile` 原先在未传 path 时直接采用客户端文件名。 | 新增 `draftUploadDefaultPath`：先统一斜杠再取 basename，`C:\\fakepath\\a.sh`、绝对路径和相对路径都只保留文件名；并新增永久回归测试。 |
| F4 | ✅ 已修复 | 原先限额回滚和 Draft 级非法 YAML 主要依赖审计探针。 | 已新增永久测试：broken YAML 拒绝、文件数超限新增文件回滚、覆盖已有文件导致 Package 总量超限后恢复旧内容。 |
| F5 | P3（可诊断性） | `DraftStore.List` 对解析失败的 `draft.json` 直接 `continue`（`draft.go:172-175`），损坏草稿会**静默消失** | 建议记录一次 warning 或在响应里返回 `skipped` 计数 |
| F6 | P3（响应语义） | `Publish` 在"已发布但权限解析/激活失败"时返回 400（消息已明确写"Skill 已发布但…"），前端可能展示为整体失败 | 可接受（消息自解释）；若前端做乐观刷新，建议返回 200 + 部分成功字段 |
| F7 | 提示 | 草稿根目录为**相对路径** `./data/agent-skill-drafts`，随进程 CWD 解析（与既有 `DataDir()` 同一约定），从不同 CWD 启动会落到不同位置 | 与现有部署约定一致，非缺陷；如需固定可在未来加配置项（本阶段明确不新增配置 ✓） |

---

## 5. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **61 个包 ok，0 FAIL** |
| `go test -count=1 -race ./agent/portableskill ./controllers` | ✅ portableskill 1.828s / controllers 2.192s |
| `gofmt -l`（V4-2 改动 Go 文件） | ✅ 无输出 |
| `git diff --check`（排除 static） | ✅ 无异常 |
| 隔离副本 `go build ./...` + 探针（P1–P4） | ✅ 全部通过 |
| 副本去掉探针后重跑 `./agent/portableskill`、`./controllers` | ✅ 全 ok |
| 前端 `pnpm typecheck` / `pnpm build` / dist↔static | ⚠️ 无法直接复跑（源码在独立仓库）；**产物侧交叉核对**：`agent-BX-vcwOa.js` 含 10 处 `skills/drafts` 与 `/upload`/`/validate`/`/publish`；`skillStudioDialog-Cfyh1Eht.js` 为独立 Studio 对话框 chunk，含 `agentSkillStudio.button.{newFile,save,deleteFile,publish,publishActivate,drafts,newSkill,editNewVersion,resume}` 与 `confirm.{title,deleteDraft}`、`package_hash`、`requested_tools`、`SKILL.md`、`upload`、`unsaved` 等（与报告 §8 的草稿/新建/编辑为新版本/版本/发布/发布并激活/文件树/未保存二次确认一致） |

---

## 6. 审计边界与未验证项

- **未验证**：前端源码与 pnpm 流水线（仅做构建产物交叉核对）；`dist` 与后端 `static` 的逐文件 diff。
- **未验证**：`WriteFile` 的「覆盖已有文件 → 总量超限 → 回滚旧内容」分支（需构造 ≈32 MiB 写入，本轮未跑）；`draft.json` 损坏时 `List` 的静默跳过行为（仅代码审阅，未构造损坏样本）。
- **未验证**：真实浏览器端到端人工测试（文档 §11 的 A–H 八组步骤，尤其 CodeMirror 交互、未保存二次确认、二进制 Asset 上传/删除的手感）。
- 未在 MySQL 上验证（本阶段无 schema 变更，风险极低）。

---

## 7. 附：V4-2 交付物清单

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `agent/portableskill/draft.go` | 571 行（新） | `DraftStore`：创建/克隆/列/详情/读写删文件/删除/校验/发布；路径与符号链接防护、限额即时校验与回滚 |
| `agent/portableskill/draft_test.go` | 222 行（新） | 创建编辑校验、缺失引用与 name 不匹配、`../` 与绝对路径、`SKILL.md` 删除、scripts warning、多文件累积删除、Clone→Publish 不可变性 |
| `controllers/agent_skill_draft.go` | 223 行（新） | 10 个端点的薄控制器；上传限额与流式读取；Publish 编排（校验→导入→权限 Review→可选激活+运行时同步） |
| `agent/portableskill/parser_test.go` | +10/-3 | 测试 DB 改为**独立 alias**（避免多 revision 测试互相污染），`RegisterModel` 收敛为一次 |
| `routers/router.go` | +7 行 | `/agents/skills/drafts*` 七条路由（含 upload/validate/publish） |
| 文档 | — | `02-phase-v4-2-skill-studio.md` §10 实现结果；新增 `v4-2-implementation-report.md` |
