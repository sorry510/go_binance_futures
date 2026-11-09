# Code Review：方案 01 全量复审（01-0 ～ 01-6）

- 审计日期：2026-11-08 23:40 ～ 2026-11-09
- 审计基线：HEAD **`8841d8a feat: 01-4_01-6`**（`feat/trade-upgrade`，2026-11-08 23:39:11 +0800）。**收尾复核时 `origin/feat/trade-upgrade` 追踪引用也已指向 `8841d8a`**，`git branch -vv` 无 ahead/behind；本会话未执行任何 push，也不推断该引用同步的方式。当前 `master` 仍为 `eabf781`，即 01 线**尚未合入主分支**。
- 工作区状态：**代码侧干净**（`git status --porcelain` 仅有本审计报告与 `doc/订单产生与记录完整流程.md` 两个未跟踪文档，无任何代码改动），与前几轮「未提交工作区叠加」有本质区别
- 方案依据：`doc/trade/开发方案-01-合约订单审计与部分成交.md`（536 行，状态行已改为「01-0～01-6 本地开发及阶段回归已执行；01-6 生产上线 Gate 尚未通过、未获授权，T0 尚未启用」）
- 对照文档：`CODE_REVIEW_01-0_01-3*.md`（三轮）、`CODE_REVIEW_01-4.md` + 整改记录、`CODE_REVIEW_01-5.md` + 整改记录、`01-6_本地验收记录.md`
- 副本：`/tmp/phase_audit_01all`（纯 `git archive HEAD`，无需拷贝未提交文件）；探针写在副本内，结束随副本删除

> **对「01 已经全部完成」的口径说明**：本轮实证的是「本地开发、整改与离线回归全部完成并已提交」，**不是**「实盘验收完成」。方案状态行、01-6 验收记录 §五 G1～G7 均明确生产上线 Gate 为 BLOCKED、T0 未启用。本报告不把本地 Mock PASS 写成实盘 PASS。

---

## 一、结论先行

| 项目 | 结论 |
| --- | --- |
| 01-0 ～ 01-6 本地功能与阶段回归 | ✅ 全部落地，跨阶段端到端集成测试通过 |
| 历轮 40+ 项 Findings 的代码侧闭环 | ✅ 绝大多数已闭环；剩余均为**实盘/性能/发布类 Gate**，非代码缺陷 |
| 离线 Gate（build / 目标包 test / race / vet / diff-check / 探针） | ✅ 目标包全绿，`go vet ./...` 全量 EXIT=0 |
| `go test -count=1 ./...` 全量 | ❌ EXIT=1（root/tests 环境 panic + controllers 5 项既有红点），**不得写「全仓测试通过」** |
| 前端类型检查（`tsc` + `vue-tsc --noEmit`） | ✅ 均 `EXIT=0` |
| 生产上线 Gate / T0 / 真实 Binance | ❌ **全部 BLOCKED，未获授权** |
| 是否建议合并 | ✅ **建议合并到本地主分支做代码级交付**；但 **ALL-F1 必须在任何部署动作前处理**（文档与提交口径不一致 + 半发布风险），**不得据此部署生产或启用 T0** |

**本轮新发现 7 项**：ALL-F1（P1，提交已把前端 dist 完整覆盖进后端 `static/`，与三份早期文档声明及发布 SOP 冲突；新流程文档已承认同步，但自身口径仍不一致）、ALL-F2（P2，中英 README 仍写 Schema v19，实际 v23）、ALL-F3（P2，01-4 关键 CLI 守卫测试在干净 CI 不可执行）、ALL-F4（P2，无代码级 T0 门控，方案表述存在落差）、ALL-F5（P2，实盘 Gate 全未关闭，最硬两项仍是 014-F1 / 015-F3）、ALL-F6（P3，全量测试红点归因）、ALL-F7（P3，前端构建本轮未重跑）。

---

## 二、审计范围与基线变化

### 2.1 提交 `8841d8a` 的内容

`git diff --name-status 5356d0c..8841d8a` 显示本次提交一次性纳入了 01-4、01-4 整改、01-5、01-5 整改、01-6 的全部成果：

- **生产代码**：`service/orderhistory/{history.go,history_test.go}`（新）、`service/tradefills/cooldown.go`（新）、`controllers/orders.go`、`routers/router.go`、`main.go`、`feature/feature.go`、`feature/ownership.go`、`feature/strategy/coin/*`、`scanner/local_selector_v2.go`、`service/futuresownership/settlement.go`；删除 `feature/order.go`。
- **测试**：`service/futuresownership/stage016_integration_test.go`（新，153 行）、`feature/order_retirement_stage01_4_test.go`、`legacy_order_gate_stage01_4_test.go`、`feature/strategy/coin/cooldown_stage01_4_test.go`、`controllers/orders_test.go` 大幅增强。
- **文档**：方案 §十七/§十八、`01-6_本地验收记录.md`、历轮 CODE_REVIEW 与整改记录、中英 README。
- **静态资源**：`static/index.html` + 76 个 `static/` 文件变更（见 ALL-F1）。
- `appversion.DatabaseSchemaVersion = 23` 不变，本提交**无新增 migration / DDL**。

前端 UI 仓库 `feat/trade_upgrade` 已提交 `5745528 feat: rec order page`，工作区干净。

### 2.2 与上一轮（01-5 复审）相比的实质变化

1. 015-F1～F9 的代码整改已落地并逐项实证（§四）。
2. 新增 01-6 跨阶段集成测试 `TestStage016EndToEndPartialOpenCancelRestartSettleAndHistory`。
3. 新增正式 `sync db` 在独立本机 MySQL 5.6 测试库从 0 → v23 的幂等验证（仅按验收记录采信，本轮未复跑）。
4. **后端 `static/` 被完整替换为新前端产物**（上一轮报告 015-F2 时 `static/` 尚未改动）。

---

## 三、全量验收矩阵（方案 §四 01-0 ～ 01-6）

| 阶段 | 验收项 | 本轮证据 | 结论 |
| --- | --- | --- | --- |
| 01-0 | 受影响文件清单、未完结仓位策略、不迁移旧历史 | 方案 §七 + 01-4 全仓 AST 守卫 `feature/order_retirement_stage01_4_test.go` | ✅ |
| 01-1 | `position_id` / `actual_order_id` / 唯一新表；不同 owner/方向/同币两轮不串联；DB 出错原子回滚 | `tradefills.SchemaDDL` 单一真源；Stage011 / Stage015 / Stage016 测试；`command/futures_trade_fills_migration.go` 含 `idx_ftf_position_id` | ✅ |
| 01-2 | 逐笔成交、WS/REST 去重、准确均价、Algo 映射、断线补查 | Stage012 / Stage016：两价部分开（2@100 + 2@120 → 均价 110）、100 次重复事件 Fill 数仍为 4、`algoId ≠ actualOrderId` | ✅ |
| 01-3 | 多笔 CLOSE 关联一周期、部分平仓、零仓位但缺 Fill 保持 incomplete、风控一次性 | Stage016：4 开 → 1 平撤单余 3 → 重启后平 3 归零 → `closed + incomplete` → 补 FreshZero + REST Fill 后才 `settled`；连续 3 次对账风控计数不变 | ✅ |
| 01-4 | 无仍执行的旧表读写/删除、安全退出不受影响、亏损控制有效 | 全仓 AST 守卫 + root CLI 守卫；`applySettledRisk` 为唯一入口（Grep 实证无其它调用方）；旧风控函数仅存注释 | ✅ 代码侧；实盘有效性见 ALL-F5 |
| 01-5 | 主表 1 行 / 子表 3 行 / 第二轮新行 / 页面与 Fill 账务一致 | `TestStage015OneCycle...`、`TestStage015SecondSameSymbol...` + 本轮探针 `PROBE_ACCOUNTING` | ✅ |
| 01-6 | 部分开/部分平/撤单/重复乱序 WS/重启/条件单/手工减仓/同币再开/不同 owner/旧活跃仓位退出 | Stage016 覆盖主要路径；`01-6_本地验收记录.md` 列出完整矩阵 | ✅ 离线 Mock |
| 01-6 | 独立测试库 `sync db`、Go 测试/race/vet/build、前端 build、中英 README | 验收记录声明 PASS；本轮独立复跑见 §五；README 见 ALL-F2 | ⚠️ 部分 |
| 01-6 | 上线后仅从启用时间汇总新历史、旧活跃仓位正常管理、生产变更需单独授权 | 生产步骤如实列为 BLOCKED；但**无代码级 T0 门控**（ALL-F4） | ⚠️ Gate 未关闭 |
| 01-6 | Fill 不重复、剩余持仓准确、ACK ≠ 成交、可按 position_id 解释开平、风控未削弱 | Stage016 + 探针 `PROBE_GATE` / `PROBE_ACCOUNTING` | ✅ |

---

## 四、历轮未决项闭环台账

### 4.1 已闭环（代码 + 永久测试 + 本轮探针实证）

| 来源 | 项 | 闭环证据 |
| --- | --- | --- |
| 首轮 01 | F3（broker 缺 `OrderTradeReader` fail-open）、F5（v22 显式迁移）、F6（6 份 DDL 硬编码）、F7（零手续费中断 WS）、F8（`decimal` 整数位）、F9（Fill 查询未带 account_id）、F12（Lead observer 守卫） | R2/R3 整改 + `command/trade_settlement_migration.go`、`tradefills.SchemaDDL`、AST 白名单 |
| R2 | N1（迁移回归上限）、N4～N9 | R3 复审 5 项全修、R3 整改补齐边界 |
| R3 | N1（零仓位门禁饿死补确认）、N2（自动重挂改默认关闭 opt-in） | `reconcileClosedCycleFills` 前置且 fail-closed；`BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled`（`main.go:478`、`protection_repair.go:22`）；Stage016 验证结算不重放 |
| 014 | F2（冷却按 `updated_at`） | `service/tradefills/cooldown.go::RecentClosedSymbols` 改用 `futures_trade_fills.trade_time` + 精确 `client_order_id/account_id/symbol/intent` 关联；`cooldown_stage01_4_test.go` 通过 |
| 014 | F3/F4/F8（AST 全仓守卫、CLI 条件注册、迁移例外） | `feature/order_retirement_stage01_4_test.go`（252 行）、`legacy_order_gate_stage01_4_test.go`；`main.go:95` 条件注册经静态复核正确；但见 ALL-F3 |
| **015-F1** | Accept 头契约（YAML 非 JSON / XML 500 / Proto panic） | `controllers/orders.go:19-27` 改 `ctrl.JSONResp(...)`；永久测试 `TestStage015HistoryResponseFixedJSONAcrossAcceptTypes`、`TestStage015SuccessResponseAlwaysJSON`；本轮探针 **8/8** 全 Accept（含 proto）返回 `application/json`、可解析、`panicked=false` |
| **015-F4** | 账户级读权限边界 | 方案 §十七明确「单操作员模型」；`OrderController.authorized()` 要求 `jwt.MapClaims` + 非空 `Username`；静态核对链路一致：`middlewares/auth.go:88 SetData("user", claims)` → `utils/jwt.go:18 "Username"` → `orders.go:34-43`；本轮探针 **7/7**（valid→400 通过授权，nil/错类型/空/空白/非字符串/小写键→401） |
| **015-F5** | 前端响应 `any` | `src/api/order.ts` 新增 `OrderHistoryApiResult<T>`，`index.vue:114/150` 直接用 `res.data`，`as` 强转已清除 |
| **015-F6/F7/F8** | 红绿配色 / 模糊匹配提示 / Owner 下拉 | 整改记录明确「保持全站盈利绿亏损红」；`zh-CN.yaml:1626` placeholder 改为「例如 BTC 或 USDT（模糊匹配）」；`index.vue:205-206` `filterable + allow-create` |
| **015-F9** | 直出 ORM 模型 / `source_ref` | `service/orderhistory/history.go:116-147` 新增 `ManagedOrderView` 显式白名单，`Position` 删除 `source_ref`；本轮探针 `PROBE_ALLOWLIST bytes=3355 forbidden=0 required=5 ok`（写入 `SECRET_PROPOSAL_ID_42` 后 JSON 不含 `source_ref`/`last_reconciled_at`，仍含 `client_order_id`/`order_id`/`trade_id`/`rest_verified`） |

### 4.2 未闭环（均为实盘 / 性能 / 发布 Gate，非代码缺陷）

| 来源 | 项 | 当前状态 |
| --- | --- | --- |
| 首轮 01 F1 / 01-6 G1 | 真实 Binance 部分成交 WS `l/L/n/N/rp/T` 与 REST `userTrades` 逐 `tradeId` 比对 | **BLOCKED**，未授权 |
| 014-F1 / 01-6 G2 | 方向级 FreshZero 真实证据；`applySettledRisk` 是唯一自动亏损控制入口，若真实 `PositionRisk(symbol)` 不返回明确方向零仓位 row，CLOSED 周期将永久 `incomplete`、风控静默 | **BLOCKED**，本轮 Grep 确认仍无第二条路径 |
| 01-6 G3 | 保护单 opt-in 实盘验证、TP/SL Owner 真实恢复、Lead 只读权限、其它 Owner | **BLOCKED** |
| 015-F3 / 01-6 G4 | `trade_time` 无独立索引；本机 MySQL 5.6 新表 0 行，EXPLAIN 无法产生可信 P50/P95；`position_id` 索引已存在但空表计划含 `Using filesort` | **BLOCKED**，v23 未改，明确不走「私改 DDL」 |
| 015-F2 / 01-6 G5 | 双端成套发布、冒烟、回滚演练 | **BLOCKED**，但见 **ALL-F1**（仓库状态已变化） |
| 01-6 G6 | main/Lead 各自 T0、二进制/前端 hash、schema v23、历史启用边界 | **BLOCKED**，T0 模板全为 `null` |
| 014-F5 / 01-6 G7 | `controllers` 全包 5 项策略模板/测试隔离红点 | **OPEN**，独立于交易账本实现 |
| 014-F7 | 生产选币器只读核对 | **BLOCKED** |

> 台账结论：**没有发现「整改记录声明已修但代码实际未修」的项**。所有仍开放的都是记录本身就明确标为 BLOCKED/OPEN 的外部 Gate，文档诚实度较高。唯一例外是 ALL-F1 的 static 口径。

---

## 五、离线 Gate 实证（纯 HEAD 隔离副本）

| 项 | 命令 / 范围 | 结果 |
| --- | --- | --- |
| 全量编译 | `go build ./...` | `EXIT=0` |
| 主程序二进制 | `go build -o /tmp/01all_main_bin .` | `EXIT=0`，37 MB |
| 目标包测试 | `go test -count=1` 11 个业务包 + controllers 01 专项 | **11/11 ok**；`TestStage014\|015\|016` 专项 `ok` |
| Stage016 端到端 | `go test -count=1 -v -run '^TestStage016' ./service/futuresownership` | PASS（0.05s） |
| 01-4 守卫 | `feature` / `coin` / `scanner` 的 Stage014/Cooldown/LegacyOrder | 3/3 ok |
| 竞态 | `go test -count=1 -race` 9 个包 | **9/9 ok**（`ld: malformed LC_DYSYMTAB` 为 macOS 非致命噪音） |
| 静态检查 | `go vet ./...`（**全量**） | **`EXIT=0`** |
| 提交补丁 | `git diff --check 5356d0c 8841d8a` | `EXIT=0` |
| 依赖闭包 | `go list -deps .` | `tradefills=1`、`futuresownership=1`、`orderhistory=1`、`agenttrade=1`、`leadaccount=1`、`testtrades=0`（测试辅助未进生产二进制） |
| 全量测试 | `go test -count=1 ./...` | ❌ `EXIT=1`（见下） |
| 前端类型检查 | 仓库本地 `node_modules/.bin/tsc --noEmit`、`vue-tsc --noEmit --skipLibCheck` | 均 `EXIT=0`（`vue-tsc` 约 7 分钟） |

### 全量 `go test ./...` 失败归因（**禁止写「全仓测试通过」**）

1. **`go_binance_futures`（root）**：`init()` → `registerModels()` → `setDriver()` 连 MySQL 失败，`panic: register mysql database: ... dial tcp :0`。根因：`conf/app.conf` 未纳入 Git（`git ls-files conf/` 仅 `app.conf.example`），纯 HEAD 副本必然缺配置。本会话沙箱**拒绝读取该文件**，按安全策略未重试、未绕过。
2. **`go_binance_futures/tests`**：`panic: the target config file: .../conf/app.conf don't exist`，同一根因。
3. **`controllers`**：5 项失败 —— `TestQueryStrategyTemplatePageOrdersByIDDescending`、`TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation`、`TestStrategyTemplateValidationAcceptsMarketConditionStringComparison`、`TestStrategyTemplateValidationAcceptsMultilineLetExpression`、`TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`。全部位于未改动的策略模板 fixture / 研究目录，01-6 验收记录 §四 已如实列为既有红点。
4. **`service/leadaccount`**：第一次全量并行时 `TestStage56SupervisorRecoveryStartsBeforeWSAndStopsAllWorkers` 报 `recovery fault did not reach alert queue`；单独跑目标包、race、第二次全量均 PASS → **并发时序抖动**，非 01 引入。

`01-6_本地验收记录.md` 声称「root 包 PASS」只在本机具备 `conf/app.conf` 与可达 MySQL 时成立，**本轮未在隔离副本独立复现**（见 ALL-F3）。前端本轮实跑 `tsc --noEmit` 与 `vue-tsc --noEmit --skipLibCheck`，两个命令均 `EXIT=0`（耗时约 7 分钟）；未执行会重写 `dist` 的 `vite build`，01-6 记录声明的 ESLint / Vite build PASS 本轮未重复执行。

---

## 六、临时探针实证（副本内、隔离执行）

```
PROBE_JSON     8/8 全 Accept（含 application/x-protobuf、x-yaml、xml、text/xml）
               code=200/400/410、Content-Type="application/json; charset=utf-8"、
               json=true、panicked=false        → 015-F1 已闭环
PROBE_AUTH     7/7 valid→400；nil / map[string]any / 空 claims / 空白 Username /
               非字符串 Username / 小写键 → 401，全部 JSON 无 panic
                                            → 015-F4 防御性授权成立
PROBE_ALLOWLIST bytes=3355 forbidden=0 required=5 ok
                                            → 015-F9 白名单成立
PROBE_ACCOUNTING vwap=110/137.5 net=109.2 roi=24.818181818181818182
                 leverage=null lead_detail=ErrNotFound
PROBE_GATE     5/5 fail-closed，reason 逐字匹配：
               fee_and_pnl_not_rest_verified / order_fill_quantity_mismatch /
               mixed_commission_assets / settled_pnl_sign_mismatch /
               legacy_cycle_without_complete_fills
STAGE016       部分开→100 次重复→部分平撤单→重启→归零 incomplete→补证 settled→
               3 次重放风控计数不变，4 Fill / 3 子订单 / 净盈亏 109.55 USDT
```

静态产物实证（Python 逐文件 SHA256）：

```
static files: 106   dist files: 106
only_in_static: 0   only_in_dist: 0   same_path_different_content: 0
identical_files: 106   index_html_equal: True
```

---

## 七、本轮新发现

### ALL-F1 — P1｜提交已把前端 dist 完整覆盖进后端 `static/`，与三份文档及发布 SOP 冲突

**事实**：提交 `8841d8a`（2026-11-08 23:39:11）包含 `static/index.html` 修改与 76 个 `static/` 文件变更；逐文件 SHA256 比对证明后端 `static/` 与前端 `dist/` **106/106 完全一致**，`index.html` 哈希 `7c95ac4b5b43b7cd70a166c44d7d8cd2189bac50bec9c40f8f233464fc1c36f7`；主入口 JS 含新订单页 `orderPage` 键。

**冲突**：
- `CODE_REVIEW_01-5_整改记录.md:10`（23:00）：「后端 `static` 和根目录主程序保持不动……**不得只替换新后端 API 或先覆盖旧进程的 `static`**」。
- 方案 §十七:520（23:27 前）：「**不得先把 dist 复制到活动 static 或只替换二进制**」，要求维护窗口内成套切换。
- `01-6_本地验收记录.md:8` 与 §四：「运行中应用二进制和后端 `static` **均未修改**」「未更改后端活动 `static`」。

时间上三份记录都写在提交之前，可解释为「验收执行时未改、提交时才纳入仓库」。

**收尾补充证据（本轮复核新增）**：未跟踪文档 `doc/订单产生与记录完整流程.md:299-302` 已明确写出：

> - **前端产物**：已执行 `pnpm typecheck`、`pnpm build`，并把前端 `dist` 同步至本地后端 `static`（106 个文件，内容校验一致）。
> - **后端二进制**：本次静态资源同步**没有替换或启动新的后端二进制**；新页面需要与包含 01-5/01-6 API 的新后端配套使用。
> - **代码提交**：保留当前工作区，未自动 commit/push。

这与本轮 106/106 逐文件 SHA256 比对结论**相互印证**，说明 static 入库是有意为之、不是误操作；该文档对「新页面必须配新后端」的半发布风险也已有清醒描述。因此 ALL-F1 的准确定性是**多份文档之间口径不一致 + 部署 SOP 未同步**，而非「无人知晓 static 被改动」。

但该文档自身仍有两处冲突：
- `:302` 的「未自动 commit/push」与当前事实不符 —— 代码已在 `8841d8a` 本地提交、工作区代码侧干净，`origin/feat/trade-upgrade` 追踪引用也已指向该提交；
- 三份早期记录（01-5 整改记录 / 方案 §十七 / 01-6 验收记录）仍保留「后端 `static` 未修改」的绝对表述，未随新文档一起修订。

即：**已有一份文档说清了事实，但没有任何一份文档把「仓库已含新 static + 已提交」写成部署硬约束**；执行者按不同文档仍会得出相反结论。

**风险**：仓库里已存在「新前端页面 + 旧生产二进制/旧生产 DB/未启用 T0」的组合。beego 静态目录为磁盘直读，一旦基于该 commit 部署、`git pull` 到运行目录或打包发布，新页面会**立即**对外提供，而后端 API / schema / T0 可能尚未成套切换 —— 正是整改记录与方案要避免的半发布状态。执行者若以三份早期文档为准，会误判 static 未被触碰；新流程文档虽然已提示，但未进入部署 SOP。

**建议**：① 以 `doc/订单产生与记录完整流程.md:299-302` 的事实描述为准，修订三份早期文档的「static 未修改」表述，并更正 `:302` 的「未 commit/push」；② 在部署 SOP 中把「该 commit 已含前端」写成硬约束，任何部署必须同时切换二进制 + static + schema，且不得用 `git pull` 到运行目录的方式单独更新静态文件；③ 若希望保持「代码仓只放源码」的约定，考虑把 static 产物拆到独立 release 提交/产物仓库；④ 在所有生产动作前保持 G5 BLOCKED。

### ALL-F2 — P2｜中英 README 仍写 Schema v19，与代码 v23 不符

`README.md:38`「截至当前 Stage 7-1～7-4 代码，数据库要求 **Schema v19**」；`README.EN.md:38` 同样写 v19，第 107 行「MySQL 8 schema v19」，第 137 行「The current binary requires **schema v19**」（当前时态，明确错误）。而 `appversion/version.go` 为 **v23**，`01-6_本地验收记录.md` 的 sync db 也到 v23。

01-6 验收记录 §四 声称「中英 README 已同步：删除旧订单历史『基于订单估算利润』说法」，实际只更新了订单历史段落（`README.md:89`），**数据库版本描述未同步**。方案 §四 01-6 明确要求「核对中英文/README」。v19 → v23 是发布前置的核心信息，运维按 v19 判断会导致升级判断错误。

另 `README.md:89` 仍写「新订单历史（方案 01-5，**待 01-6 正式发布**）」，与 ALL-F1 的 static 已入库状态口径不一致。建议统一修订并加「以 `appversion.DatabaseSchemaVersion` 为唯一真源」。

### ALL-F3 — P2｜01-4 关键 CLI 守卫测试在干净 CI 不可执行

`legacy_order_gate_stage01_4_test.go` 位于 `package main`，而 root 包 `main.go:70 init()` 会 `registerModels()` → `setDriver()` 连数据库。`conf/app.conf` 未纳入 Git，因此任何干净 checkout / CI 上 `go test .` 都会在 init 阶段 panic（本轮纯 HEAD 副本实证）；`tests` 包同理。

这意味着 01-4 用于防止「归档 `models.Order` 被重新无条件注册」的关键守卫，实际上**只在开发者本机有配置时可自动验证**。01-6 验收记录把 root 包列为 PASS 却未披露这一环境依赖。本轮只能以静态复核替代（`main.go:95` 的 `if isSyncDatabaseCommand(os.Args[1:])` 条件注册正确、函数定义与 `isCLICommand`/`runCommand` 一致）。

**建议**：把 `isSyncDatabaseCommand` 提取为不触发 `main init` 的独立纯函数包并把守卫测试迁过去；或在 README/CI 配置中明确 root/tests 两包的前置条件。

### ALL-F4 — P2｜无代码级 T0 门控，与方案 §四 01-6 存在表述落差

全仓 Go 源码检索无 `T0` / `history_enabled_at` / `activation_time` / `cutover` 等机制；`Store.List` 直接查询全部 `futures_managed_positions`（含 `settlement_status=legacy` 的旧周期，按设计显示为 `incomplete`）。方案 §四 01-6 要求「上线后**仅从功能启用时间开始汇总新历史**」，01-6 验收记录把 T0 列为 G6 BLOCKED 且模板所有字段为 `null`。

当前「明确未实现 + 已列为 Gate」是自洽的，但方案未界定 T0 究竟是**运维记录**还是**代码级过滤**。若仅为记录，则「仅从启用时间开始汇总」这句验收项实际无法成立；若为代码门控，则需在启用前实现。建议在方案中明确二选一，避免上线时无法界定哪些周期属于新历史。

### ALL-F5 — P2｜实盘 Gate 全部未关闭，最硬两项仍是 014-F1 与 015-F3

G1～G7 全为 BLOCKED/OPEN。本轮复核确认：
- `applySettledRisk`（`service/futuresownership/settlement.go:287`）仍是**唯一**自动亏损控制入口，`feature/feature.go:1100` 旧计数器仅存注释，无回退路径 → 一旦真实 `PositionRisk(symbol)` 不返回明确方向零仓位 row，CLOSED 周期永久 `incomplete`、风控静默（G2）。
- `futures_trade_fills.trade_time` 仍无索引；本机测试库 0 行，01-6 只读 EXPLAIN 明确「不能推导生产 P95」→ 性能 Gate 未关闭（G4）。
- 保护单自动重挂仍是默认关闭 opt-in `BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled`（G3）。

这三项是「代码正确但实盘证据缺失」，**不能**用 Stage016 的 Mock FreshZero 或 SQLite/MySQL5.6 结果替代。

### ALL-F6 — P3｜全量测试红点归因与偶发抖动

见 §五。`go vet ./...` 全量 EXIT=0 值得肯定；但 `go test ./...` 仍红，`controllers` 5 项与 root/tests 2 项环境 panic 必须在交付说明中如实标注。`leadaccount` 的 supervisor 告警队列测试在并行全量下偶发失败，建议单独立项做时序加固。

### ALL-F7 — P3｜前端构建本轮未重跑

前端仓库已提交且工作区干净；01-6 验收记录声明 ESLint / `pnpm typecheck` / `pnpm build` PASS（Vite ~17.96s、4.07MB）。本轮独立实跑两个 `--noEmit` 类型检查均 `EXIT=0`，但**未**执行会重写 `dist` 的构建；`dist` 与 `static` 已逐文件 106/106 一致，构建产物可追溯。

---

## 八、01-6 上线门禁状态（按 `01-6_本地验收记录.md` §五，本轮逐项复核）

| Gate | 状态 | 本轮复核 |
| --- | --- | --- |
| G1 真实 WS vs REST 单笔成交 | BLOCKED | 无实盘证据 |
| G2 方向级真实归零证据 | BLOCKED | 与 014-F1 互为硬阻塞 |
| G3 TP/SL 重挂与 Owner 安全 | BLOCKED | opt-in 默认关闭，代码侧确认 |
| G4 生产 MySQL 8 性能与 v23 结构 | BLOCKED | v23 未改、`trade_time` 无索引，文档诚实 |
| G5 双端成套发布 / 真实运行验证 | BLOCKED | **但仓库状态已变化，见 ALL-F1** |
| G6 main/Lead T0 与版本持久记录 | BLOCKED | 模板全 `null`，且无代码级门控（ALL-F4） |
| G7 全量 controllers 测试隔离 | OPEN | 5 项既有红点，本轮复现 |

文档对 BLOCKED 的标注**诚实且充分**，本轮未发现把本地 Mock PASS 写成实盘 PASS 的表述；新流程文档甚至明确写了「静态资源同步没有替换或启动新的后端二进制」，风险意识到位。遗留问题是 ALL-F1 的多份文档之间 static / 提交状态口径不一致，以及 ALL-F2 的 README 版本。

---

## 九、纪律与边界声明

- 本报告仅代表 **离线 PASS**（Go build / 目标包 test / race / 全量 vet / diff-check + SQLite 夹具 + 临时探针 + 源码/AST 静态核对 + 逐文件产物比对）。**实盘 Gate / T0 / 生产部署一律未通过、未授权**。
- 未修改任何业务代码；未 commit / push；未改 `conf/app.conf`；未对真实库执行 `sync db`；未访问真实 Binance；未重启服务；未操作生产仓位；未触碰 `strategy_templates/`。
- `conf/app.conf` 含连接信息，本会话沙箱拒绝读取；按安全规则**未重试、未推断、未绕过**，root/tests 两包的测试结论据此降级为「静态复核 + 引用验收记录」。
- 副本 `/tmp/phase_audit_01all` 内的 2 个临时探针随副本删除；前端仅做只读核对与 `--noEmit` 检查。
- 本报告为**新文件**，不覆盖 `CODE_REVIEW_01-0_01-3*.md` / `CODE_REVIEW_01-4.md` / `CODE_REVIEW_01-5.md` 等历史报告。

---

## 附录：关键证据位置索引

| 位置 | 内容 |
| --- | --- |
| `controllers/orders.go:19-45` | `JSONResp` 固定 JSON + `authorized()` 单操作员防御 |
| `middlewares/auth.go:88` / `utils/jwt.go:18` | `SetData("user", jwt.MapClaims)` / `Username` claim |
| `service/orderhistory/history.go:116-158` | `ManagedOrderView` 白名单、`Position` 无 `source_ref` |
| `service/futuresownership/stage016_integration_test.go` | 01-1～01-5 跨阶段端到端 |
| `service/futuresownership/settlement.go:287` | `applySettledRisk` 唯一入口 |
| `service/tradefills/cooldown.go:29-43` | 冷却按 `trade_time` + 精确身份关联 |
| `command/futures_trade_fills_migration.go:18-38` | `idx_ftf_position_id` |
| `main.go:95` / `main.go:478` | 归档模型条件注册 / 保护单 opt-in 注释 |
| `static/index.html` vs UI `dist/index.html` | SHA256 `7c95ac4b…`，106/106 逐文件一致 |
| `README.md:38,89` / `README.EN.md:38,107,137` | 仍写 v19 / 待发布（ALL-F2） |
| `doc/订单产生与记录完整流程.md:299-302` | 承认 dist→static 106 文件同步、未换二进制；`:302` 仍写「未 commit/push」（ALL-F1） |

---

## 审计后整改追记（2026-11-09）

本报告前文记录的是 `8841d8a` 的审计快照。后续已新增修复与测试，详见 [CODE_REVIEW_01-ALL_整改记录.md](CODE_REVIEW_01-ALL_整改记录.md)：ALL-F1/F2 的文档与 Schema v23 口径已更正；ALL-F3 的纯 CLI 判断已迁至不触发主包初始化的 `internal/cliargs`；ALL-F4 已实现按 main/Lead 独立 T0 的只读历史收益认证门禁（**未批准 T0 时不认证收益**），并为无 T0、前 T0、真实 T0 分别补回归；ALL-F7 已重新构建前端且 106 个本地静态文件一致。原审计结论的生产 G1～G6 仍 BLOCKED，G7 仍 OPEN，本轮未提交/推送整改，不代表已生产发布。
