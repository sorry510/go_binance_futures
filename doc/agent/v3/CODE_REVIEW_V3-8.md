# V3-8 Phase 审计报告：系统看板、个人运维与 V3 收尾

- **审计对象**：V3-8 实现**在工作区未提交**（`appversion/`、`service/systemhealth/`、`service/logcleanup/`、`command/operations.go`、`controllers/system_dashboard.go`、`main.go`、`routers/router.go`、文档）。基线 `HEAD = b1188ae`（Merge PR #54，已含 V3-7）
- **审计方式**：原地只读审计 + 隔离副本写临时探针（`git archive HEAD` + 拷贝未提交文件；探针与副本审计后已删除，用户仓库未被改动）+ 实跑 CLI
- **审查类型**：review-only
- **结论**：**已完成，V3 可收尾**。Gate §6 的 6 条全部满足（前端 `pnpm build`/`vue-tsc` 与 dist 一致性因前端源码在独立仓库而无法直接复跑，但构建产物侧证据充分）。未发现 P0/P1 缺陷；发现 4 项非阻塞项。
- **审计日期**：2026-09-15

---

## 1. 结论先行

| 判定项 | 结论 |
| --- | --- |
| V3-8 是否完成 | ✅ **完成**（范围与文档 §1 四部分一致：系统看板 / doctor / cleanup logs / 收尾） |
| 是否可进入 V3 收尾 | ✅ **可以**（V3-8 是 V3 最后一个 Phase，无阻塞项） |
| 是否新增 DB 表/字段 | ❌ 未新增；`appversion.DatabaseSchemaVersion = 14`，SQLite/MySQL 均无需新表 |
| 是否存在 P0/P1 | ❌ 未发现 |
| 新增依赖 | 无（`go.mod` 未改） |

---

## 2. 验收 Gate 逐项核对（§6 + §8）

| # | Gate | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | 系统看板位于顶级菜单最上方，原 AI 菜单不再出现"可观测性" | ✅ | 构建产物 `static/static/js/index-BvGtTSqI.js`：路由 `{path:"/system-dashboard", meta:{title:"menus.systemDashboard", rank:0.1}}`（Home 为 `rank:0, showLink:!1` 隐藏；config-center 为 `rank:1`）→ **首个可见顶级菜单**；菜单/页面 i18n 中已无"可观测性"条目，唯一残留是页面副标题文案"…Agent/LLM/Tool/MCP 可观测性。"（说明性文字，非菜单项）。看板复用原 Observability 页面 chunk（`observability-Bp9u_blf.js`）✓ 与"复用原 Observability"一致 |
| 2 | 用户可快速判断 8 个维度状态 | ✅ | 实跑 `doctor` 输出覆盖全部维度（见 §3）；前端 chunk 消费 `binance_rest / futures_ws / announcement_ws / market_intelligence / mcp_servers / scheduler_jobs / overall / reconcile_required / protection_failed / execution_uncertain` 等字段 |
| 3 | `doctor` 可独立运行且完全只读 | ✅ **实证** | `main.go:70` `isCLICommand` 早退（跳过 middleware/静态资源），`main.go:340` `runCommand` 在 `initializeRuntimeDatabase()`（:343，含 Ownership Reconcile/backfill）**之前**返回 → 不启动 Web/WS/Scheduler/交易循环、不执行 Reconcile、不隐式 `sync db`；探针 PC 断言 Report 两次调用后 11 张表行数**零变化**、出网请求**仅 GET**；实跑 doctor 无任何写入与同步日志 |
| 4 | `cleanup logs --before-days N` 只删固定白名单；Fixture 证明真实交易 Audit 不被删 | ✅ **实证** | 白名单硬编码 5 表（`service/logcleanup/service.go:21-27`）且表名/列名与 ORM 模型逐一吻合；`cutoff<=0` 先于任何 DB 访问被拒（`:41-44`）；单一事务 + 回滚（`:46-70`）；现有 Fixture `service_test.go:12-52` 断言 audit 保留；**探针 PA** 在真实 ORM schema 上验证 5 表各删 1 留 1，且 `agent_trade_audits / agent_trade_proposals / futures_managed_positions / futures_managed_orders / test_strategy_results / agent_backtest_trades` 全部存活 |
| 5 | `go test -count=1 ./...`、`go vet ./...`、关键 `-race` 通过 | ✅ | 见 §4 |
| 6 | 前端 `dist` 与后端 `static` 一致 | ⚠️ **产物侧确认** | 后端 `static/` 实测 **92 个文件**，与文档声明一致；前端源码与 `dist` 目录在独立仓库，未审（见 §6） |

**§1 四部分范围核对**：① 系统看板（复用 Observability + `/system/health`）✓；② `doctor` ✓；③ `cleanup logs` ✓；④ 收尾（版本单一常量 v14、无新表、README/Phase 文档更新）✓。**§7「明确不做」未越界**：无数据规模/容量统计、无备份中心、无 Agent Studio、无 Prometheus 集成、无自动修复/自动 Reconcile。

---

## 3. 实跑验证（在用户真实配置 mysql 上执行）

`go build -o /tmp/v38_bin .` 后实跑（未覆盖仓库内 `go_binance_futures` 产物，审计后已删除）：

```
System doctor (2026-09-15T10:59:38+08:00)
[OK]   Database             schema v14
[OK]   Binance REST         reachable in 2639 ms
[OK]   Futures WS           last market update 0s ago
[OK]   Announcement WS      healthy
[OK]   Market Intelligence  3 sources healthy
[WARN] MCP                  1/4 MCP servers are not healthy
[OK]   LLM                  2 routing candidates; 1 LLM failures in 24h
[OFF]  Scheduler            no enabled scheduler jobs
[OK]   Agent Runtime        5 tasks in 24h
[OK]   Trade Safety         no trade safety issue
Overall: warning
EXIT=0
```

- **独立复现了文档 §8 的实跑声明**：Database（v14）、Binance REST（2639ms）、Futures WS、Announcement、Market Intelligence、LLM、Trade Safety 均返回真实状态 ✓
- `Overall` 聚合正确：MCP 警告 → warning ✓
- 启动日志仅有 "use lang" + "use database driver: mysql"，**无** Web/WS/Scheduler/Ownership Reconcile 启动痕迹 ✓ 印证 Gate #3 的"完全只读"边界
- `cleanup logs --before-days 0` / `cleanup logs`（缺参）/ `--before-days -5`：三种情形均 `EXIT=1` 且未触碰数据库（拒绝发生在 `CleanupLogs` 第一行）✓ 复现文档 §8 的"拒绝执行且退出非 0，没有删除数据"

---

## 4. 自动化验证结果（用户工作区原地执行）

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **61 个包 ok，0 FAIL** |
| `go test -count=1 -race ./service/systemhealth ./service/logcleanup ./command ./controllers ./routers` | ✅ systemhealth 1.915s / logcleanup 1.586s / command 2.010s / controllers 2.376s（routers 无测试） |
| `gofmt -l`（V3-8 涉及 10 个文件） | ✅ 无输出 |
| `git diff --check`（排除 static） | ✅ 无异常 |
| 隔离副本 `go build ./...` + 原始测试 | ✅ 通过 |

---

## 5. 实证过的隐式契约（8 个探针，全部 PASS）

探针写在隔离副本（`/tmp/phase_audit`、`/tmp/phase_audit2`），跑完随副本删除；因 beego ORM 的 `RegisterDataBase` 是**进程级全局**，探针与包内既有测试同跑会冲突，故探针以 `-run TestProbe` 单独执行。

| 探针 | 实证内容 | 结果 |
| --- | --- | --- |
| **PA** 清理白名单 vs 真实 schema | 用 ORM 真实模型建表（而非手写裸表）后执行 `Cleanup` | ✅ 5 张白名单表各 `deleted=1 / remaining=1`；`agent_trade_audits` 等 6 张非白名单表旧数据**全部存活**（证明表名/列名与权威 schema 一致，且无模糊匹配） |
| **PB1** CLI 调度分支 | `SchedulerRuntimeAvailable=false` + `agent_tasks` 中一条 `market_scan` 成功任务 | ✅ `lastRunAt` = 插入的 `completed_at`、`LastStatus=succeeded`、Scheduler=healthy → 证明 `skill/status/completed_at/created_at` 列存在且 **Skill 名与生产一致**（`marketregime.Name="market_regime"`、`workflows.MarketScanName="market_scan"`、`DailyMarketBriefName="daily_market_brief"` 与 `schedulerDefinitions` 字面量逐一吻合；生产 Scheduler 恰好注册这 3 个任务） |
| **PB2** LLM 最近失败可见 | 插入较新的 `llm_call` error 观察记录 | ✅ `Status=warning, LastError="boom provider 500"` → 证明被 `_ =` 忽略错误的查询列集正确、失败详情不会丢失 |
| **PB3** Agent Runtime 计数 | 2 条 24h 内任务（1 条 failed 且 `round>=max_rounds` 且 error 含 "maximum rounds"）+ 1 条 48h 前 | ✅ `Tasks24h=2 / Failed24h=1 / MaxRoundsFailed=1`，24h 窗口生效 |
| **PB4** Futures WS 阈值 | 1m / 6m / 20m 三种新鲜度 + 关闭开关 | ✅ healthy / warning / error / disabled 四态与代码阈值（5m/15m）一致 |
| **PC** 完全只读 | 连续两次 `Report`（含 Binance REST 检查）前后比对 11 张表行数 + 记录出网方法 | ✅ 行数**零变化**；出网 `map[GET:2]`，**无 POST/PUT/DELETE** |
| **PE1** 托管**订单** reconcile 计数（安全关键） | 插入 `futures_managed_orders.status='reconcile_required'` | ✅ `Trade.ReconcileRequired=1 / Status=error / Overall=error` → 证明订单表计数列正确（该查询同样忽略错误，若列名错会静默报"无问题"） |
| **PE2** Schema 版本分支 | `config.version` 设为 13（落后）与 15（超前） | ✅ 落后 → `Status=error`、`message="schema v13 is older than required v14"`、`required_version=14`、Overall=error；超前 → healthy（未来升级不误报） |

> 上述行为中，**PB2/PB3/PB4/PE1/PE2 与 CLI 调度分支均无永久回归测试保护**（现有 Fixture 只覆盖 healthy 基线 + execution_uncertain/reconcile_required 的 position 计数），建议将探针断言中的关键项补进正式测试（见 §6）。

---

## 6. 风险与待改进（均为非阻塞）

| # | 级别 | 问题 | 证据 / 建议 |
| --- | --- | --- | --- |
| F1 | P3 | `doctor` **恒以退出码 0 结束**，即使 `Overall=error`（如 DB 不可用、Schema 落后）。实跑时 `Overall=warning` 亦为 `EXIT=0` | `main.go:275-283` 仅在 `Doctor()` 返回 error 时 `os.Exit(1)`，而 `systemhealth.Report` **所有分支都 `return report, nil`**（`service.go:110-151`），error 返回值恒为 nil → `Doctor`/controller 的 error 分支是死代码。建议：`Overall==error` 时返回非 0（便于脚本/告警），或明确文档说明退出码不表达健康状态 |
| F2 | P3 | 多处 `_ =` 忽略查询错误（LLM failures/latest、Agent 三计数、Trade 四计数）→ 若表名/列名将来变更，会**静默退化为"健康"**而非报错 | 本次探针已实证当前全部列名正确；但缺永久回归测试。建议至少为 Trade Safety 四个计数补 Fixture 断言（安全关键） |
| F3 | P3 | 前端未消费 `required_version` 字段（chunk 中无该字面量） | §2 要求展示"当前 Schema Version 与二进制要求版本"；实际通过 `message`（如 "schema v13 is older than required v14"）间接传达。建议 UI 显式展示两值，或在文档中标注以 message 表达 |
| F4 | P3 | Market Intelligence 健康语义偏宽松：只要源记录存在即报 "N sources healthy"，仅 `status=="error"` 才降级 | `service.go:218-244`；`disabled` 源也被计入 healthy。建议区分 enabled/disabled 计数 |
| F5 | 提示 | `/system/health` 每次请求都会出网访问 Binance（5s 超时）并执行 10+ 次全表 SELECT | 页面刷新频率与并发需注意（个人自用单用户场景可接受）；`doctor` 同理会阻塞至多 ~5s（本次实测 2.6s） |
| F6 | 提示 | `command/db_update_test.go:307` 硬编码断言 `config.Version != 14` | 属测试期望值（可视为升级提醒），非生产第二来源；生产侧 `dbVersion` 已统一为 `appversion.DatabaseSchemaVersion`（`main.go:42`）✓ |
| F7 | 提示 | 工作区另有 2 个未被代码引用的 `strategy_templates/*-v15|v16.json` | 疑似人工验证样本；`git add -A` 会一并提交，提交前请确认是否有意纳入（与 V3-7 时同类情况） |

---

## 7. 审计边界与未验证项

- **未验证**：前端源码（独立仓库 `go_binance_futrues_new_ui`）与 `pnpm exec vue-tsc --noEmit` / `pnpm build`；`dist` 与 `static` 的逐文件 diff（仅确认后端 `static/` 为 92 个文件，与文档声明吻合）。前端侧结论均来自**构建产物交叉核对**（菜单 rank、i18n 文案、接口与字段引用）。
- **未验证**：MySQL 上的 `cleanup logs` 真实删除（仅实跑拒绝路径；删除逻辑在 SQLite 真实 ORM schema 上实证）。
- **未验证**：MySQL 实际表结构与 ORM 模型的完全一致性（探针以 `RunSyncdb` 建表为权威）；由于清理只涉及 5 张表各 1 个时间列，且列名与模型一致，风险低。
- 未运行 Scheduler 实跑（当前配置下 3 个调度任务全未启用，doctor 显示 `[OFF]`），故 CLI 调度分支仅在探针层面验证。

---

## 8. 附：交付物清单

| 文件 | 规模 | 作用 |
| --- | --- | --- |
| `appversion/version.go` | 4 行 | `DatabaseSchemaVersion = 14` 单一版本来源 |
| `service/systemhealth/service.go` | 433 行 | 只读健康报告（10 项检查 + Overall 聚合） |
| `service/logcleanup/service.go` | 70 行 | 5 表白名单日志清理（事务） |
| `command/operations.go` | 82 行 | `doctor` 输出与 `cleanup logs` 前置校验 |
| `controllers/system_dashboard.go` | 24 行 | `GET /system/health` 薄封装（注入进程内 Scheduler 状态） |
| `main.go` | +62/-7 | `isCLICommand` 早退、`doctor`/`cleanup` 参数解析、dbVersion 单一来源 |
| `routers/router.go` | +2 | `/system/health` GET 路由 |
| 测试 | 167 行 | systemhealth Fixture（115）+ logcleanup Fixture（52） |

## 9. 复查处理记录（2026-09-15）

根据本审计 §6 的非阻塞项再次复查并收口：

- **F1 已修复**：`doctor` 在 `Overall=error` 时返回退出码 `1`；Healthy/Warning/Disabled 保持 `0`。新增 `DoctorExitCode` 单元测试。
- **F2 已修复**：System Health 不再用 `_ =` 静默忽略 LLM、Agent Runtime、Trade Safety 查询错误；Scheduler freshness 的非 `ErrNoRows` SQL 错误也直接标记为 Error；Market Source 状态读取错误同样显式暴露。
- **F2 回归测试已加强**：永久 Fixture 新增 LLM 最近失败、Agent 24h 失败/最大轮次、CLI Scheduler freshness、Managed Order `reconcile_required`、Schema 落后等断言。
- **F3 已修复**：系统看板 Database 卡片显式展示“当前 Schema / 二进制要求版本”，不再只依赖 message 间接表达。
- **F4 已修复**：Market Intelligence 只把 `status=healthy` 计为 healthy；`error` 或其它非 healthy 状态会降级为 Warning，并分别显示计数。
- **F5 保持现状**：个人单用户场景下按刷新触发一次 Binance REST 和只读 SQL 可接受，不增加缓存或后台健康采集器。
- **F6 不修改**：`command/db_update_test.go` 的 v14 是测试升级哨兵，不是生产版本第二来源。
- **F7 不处理**：未跟踪策略模板 JSON 属用户独立策略工作，不纳入 V3-8。
