# Code Review 第三轮复审：方案 01「合约订单审计与部分成交」R2 整改

- 复审时间：2026-11-08 18:22–18:50（北京时间）
- 依据：第二轮报告 `CODE_REVIEW_01-0_01-3-rereview.md`（R2-N1～N9）、整改记录 `CODE_REVIEW_01-0_01-3-rereview_整改记录.md`、方案文档第七～十节
- 审计对象：`feat/trade-upgrade`（HEAD 仍为 `49ba0e5`）**全部工作区未提交改动**；本轮仍未 commit/push
- 方式：`git archive HEAD` + 拷贝全部未提交文件（含未跟踪新目录）建隔离副本 `/tmp/phase_audit_r3`，实跑 Gate，写 2 组探针（共 4 项断言）；**未改业务代码、未对真实数据库执行 `sync db`、未请求真实 Binance**
- 本报告不修改首轮、第二轮报告的历史结论；报告另起文件名，不覆盖历史

---

## 一、结论先行

| 维度 | 结论 |
| --- | --- |
| R2-N1～N9 修复 | **5 项完全修复**（N1 / N5 / N6 / N7 / N8），**4 项部分修复或带边界**（N2 / N3 / N4 / N9）；无「误标已修而实际未动」 |
| 新增永久测试 | **7 个 `TestStageR2*`** + 1 个大幅增强的 `TestSyncDatabaseInitializesAndIsIdempotent` |
| 离线 Gate | ✅ 全绿（build / 9 包 test / 5 包 race / vet / diff-check） |
| v21～v23 迁移 | ✅ **本轮实证** `19→20→21→22→23` 与重复 23 全部执行，N1 已闭环 |
| 新增生产接线 | ⚠️ **`main.go` 首次接进「自动重新挂真实保护单」路径**，无 opt-in、无独立限频 |
| 本轮新发现 | **R3-N1～N9**，其中 **R3-N1 / R3-N2 是实盘授权硬约束** |
| 实盘 Gate / T0 | ❌ **仍未通过、未授权** |
| 是否建议合并 | **离线代码建议合并；但不得据此部署或启用 T0**，`main.go` 新增路径应先加固 |

> ⚠️ 本轮最重要的发现：**R2-N4 的保守快照门禁会饿死 R2-N2 的 REST 补确认**。探针实证生产型 `ExactPositionSideSource` 下，已平仓周期若拿不到「明确零仓位方向 row」，`reconcileClosedCycleFills` **一次都不会被调用**（`rest_calls=0`），周期永久 `incomplete`——即回到第二轮 N2 的原始故障。而 N2 的永久测试用的是不实现该接口的 `fakeAccountPositionSource`，恰好绕过了这条生产分支。

---

## 二、差异对照（第二轮基线 → 本轮）

| 项 | 第二轮 | 本轮 | 变化 |
| --- | --- | --- | --- |
| 已跟踪改动 | 24 个文件，+1226/−72 | 25 个文件，+1626/−72（`git diff -w`：+1612/−58） | +400 行语义改动 |
| 新增文件 | 11 个 | 13 个 | 新增 `service/agenttrade/protection_repair.go`、`doc/trade/CODE_REVIEW_01-0_01-3-rereview_整改记录.md` |
| DB Schema | v23 | **仍为 v23** | `appversion/version.go` 未变，本轮不升版本 |
| **`main.go`** | 未改 | **+9 行** | 每分钟对账成功后调用 `RepairActiveManagedProtections(context.Background())` |
| `command/db_update_test.go` | +90 | +117 | 补 `SyncDatabase(20..23)` 全序列与版本上限断言 |
| `service/futuresownership/reconcile.go` | +140 | +292 | 新增 `recExactSide` 接口门禁、`FreshPositionSide`、`reconcileClosedCycleFills`、告警清理 |
| `service/futuresownership/reconcile_test.go` | +143 | +259 | +3 个 R2 测试 |
| `service/futuresownership/service.go` | 沿用首轮 | +247（累计） | 未发现本轮新增语义漂移 |
| 未变模块 | — | `execution.go`、`settlement.go`、`tradefills/ledger.go`（仅 N5 微调）、`user_data_mirror.go`、`models/futures_trade_fill.go` | 逐段核对无语义漂移 |

---

## 三、R2-N1～N9 修复对照表

| 编号 | 复审结论 | 代码 / 测试证据 |
| --- | --- | --- |
| **N1** P2 | ✅ **已完全修复** | `db_update_test.go` 补 `SyncDatabase(20)/(21)/(22)/(23)` 全序列、v21 历史 closed 行空 `settlement_status → legacy` 回填、v22 两个具名索引、v23 旧 Fill 默认 `rest_verified=0`、重复同步幂等，并新增 `version.Version != appversion.DatabaseSchemaVersion` 上限断言（防止未来升版漏测）。**Gate 日志实证** `update database version: 19 -> 20 / 20 -> 21 / 21 -> 22 / 22 -> 23` 及 `already up to date: 23`。<br>📌 整改记录称「原复审『最高只有 19』的判断与当前工作区内容不符」——上轮判断基于当时副本与 `-v` 日志；本轮 diff 显示 20～23 调用均为未提交新增行，属**上轮之后补齐**。措辞应修正，但不影响实质结论。 |
| **N2** P2 | ⚠️ **代码修复正确，但被 N4 门禁在生产路径饿死** | `reconcile.go:347` `reconcileClosedCycleFills` 按 `account_id + position_id` 遍历所有已成交订单（含 FILLED），逐单校验 owner/symbol/side，只读调 `reconcileTradeFills`，不复活订单、不重提交；`TestStageR2ClosedFilledOrderProvisionalRESTAutoRecovers` PASS。但该函数只在 `unsettled` 循环里、`reason == ""` 之后调用，而 N4 的 `unverified` 会在它之前 `continue`（见 R3-N1）。 |
| **N3** P2 | ⚠️ **部分修复，且引入新的真实下单生产路径** | 新增 `service/agenttrade/protection_repair.go`（87 行）+ `main.go` 接线，覆盖 **main 账户的 `OwnerAgentTrade`**（+ 既有 `notice_auto_order`）；只用持久化批准提案的 StopLoss、要求真实 exchange order ID、重复调用不多下单，`TestStageR2AgentRepairRearmsRemainingManagedProtection` PASS。但 Owner/账户覆盖不全、proposal 状态未校验、只修止损不修止盈、错误吞掉、无开关（见 R3-N2～N5、N7）。整改记录第 14、43 行已自认「其他 Owner / Lead 仍属实盘前置未关闭」，与本轮结论一致。 |
| **N4** P3 | ⚠️ **代码修复，但接口层 fail-open** | `FreshPositionSide`（`reconcile.go:475`）要求 symbol-scoped 新鲜 REST 返回**明确的方向 row**，缺失/失败即 `unverified`，阻止 shrink/close/settle；每 Owner 每轮最多 6 次额外请求；`TestStageR2PartiallyMissingBulkSnapshotRequiresExactRESTProof` PASS。但 `ExactPositionSideSource` 是**可选接口断言**，未实现时整段保护静默跳过且无告警（见 R3-N6）。 |
| **N5** P3 | ✅ **已修** | `ledger.go:70-77`：REST 确认且 `CommissionAsset=""` 时，只有精确零手续费被接受；非零费/非法值无资产仍 fail-closed。`TestStageR2RESTZeroFeeMissingAssetIsAuthoritative` + `TestStageR2VerifiedZeroFeeWithoutAssetSettlesCorrectly` 双测试。 |
| **N6** P3 | ✅ **已修** | `alarmIncompletePosition` 上限 4096、超 24h 旧项清理、最旧淘汰；结算成功调 `clearIncompleteAlarm`。`TestStageR2IncompleteAlarmThrottleStorageIsBounded` PASS。 |
| **N7** P3 | ✅ **已修** | `testtrades.Reset()` / `SetAmountsForExchangeOrder()`；`prepareOwnershipDB`、`prepareTradeTestDB`、`leadaccount` fixture 均调 `Reset()`；`go list -deps .` 计数 **0**，未进生产二进制。`TestStageR2FakeRESTSupportsNonzeroMoneyWithoutLeakage` 实证非零 fee/PnL 与复位。 |
| **N8** P3 | ✅ **已修（有运维边界）** | `ensureV22SettlementIndexes` 建 `idx_fmp_settlement_status` / `idx_fmp_settlement_processed`；MySQL 用 `information_schema.statistics`、SQLite 用 `sqlite_master` 做存在性检查；v22 迁移与 `SyncDatabase`（`Version>=22`）均会自检补建。`SyncDatabase` **只在 `sync db` CLI 调用**（`main.go:274`），不阻断服务启动。边界：每次 sync 都跑 DDL 检查，MySQL 权限不足会让 `sync db` 失败；`CREATE INDEX` 的线上锁行为未在本轮验证。 |
| **N9** P3 | ⚠️ **加固但未闭合** | `TestStageReviewLeadReadOnlyStreamCannotPersistTradeFills` 新增 AST `parser.ImportsOnly` 白名单（8 个 import），能防 import alias 绕过；仍保留 4 串黑名单。但白名单**允许 `go_binance_futures/feature/api/binance`**（该包含写 DB 入口），函数体未做 AST 校验，其它写函数/跨包 helper 仍可绕过（见 R3-N8）。整改记录亦自认「是防回归守卫，不代替真实环境只读检查」。 |

---

## 四、实证过的隐式契约（本轮探针，副本内执行，随副本删除）

| 探针 | 结果 | 判定 |
| --- | --- | --- |
| **N2 × N4 交互**：构造完整 closed 周期 + 一笔 `rest_verified=0` Fill，分别用「实现 `ExactPositionSideSource` 但零仓位 row 不可得」「不实现该接口」「实现且返回明确零 row」三种 PositionSource 跑 `ReconcileOwner` | `PROBE_EXACT_MISSING: unresolved=1 status=incomplete rest_verified=0 rest_calls=0`<br>`PROBE_NONEXACT_CONTROL: unresolved=0 status=settled rest_verified=1 rest_calls_delta=1`<br>`PROBE_EXACT_ZERO_OK: unresolved=0 status=settled rest_verified=1 rest_calls=1` | ⚠️ **确认缺口**：生产型 source 下 REST 补确认根本没被调用；永久测试恰好绕过该分支（R3-N1） |
| **单周期失败是否阻塞整轮修复**：两个活动 agent 周期，第一个无可信提案，第二个正常 | `PROBE_BAD_CYCLE_FIRST: count=0 err=报警: missing approved proposal ... submit_stop=0 submit_tp=0` | ⚠️ **确认缺口**：后续周期完全未处理（R3-N3） |
| **proposal 状态校验**：`Status=rejected`、`RiskStatus=risk_fail`、`RejectedAt>0`、`ExpiresAt` 已过期 | `PROBE_REJECTED_PROPOSAL: count=1 err=<nil> submit_stop=1 submit_tp=1` | ⚠️ **确认缺口**：被拒绝/风控失败/过期提案仍触发真实 SL+TP 下单（R3-N2） |
| **止盈恢复**：止损数量精确、TP 已被撤销后跑修复 | `PROBE_MISSING_TP: count=0 err=<nil> tp_status=canceled tp_submit_delta=0` | ⚠️ **确认缺口**：只修止损不修止盈（R3-N5） |
| 迁移版本序列 | `go test -count=1 -v -run TestSyncDatabaseInitializesAndIsIdempotent ./command` 日志含 `19->20 / 20->21 / 21->22 / 22->23 / already up to date: 23` | ✅ N1 闭环 |
| 生产依赖闭包 | `go list -deps .`：`tradefills=1`、`futuresownership=1`、`agenttrade=1`、`testtrades=0` | ✅ 测试辅助未进生产二进制 |

> 探针文件 `zz_r3_probe_test.go` 写在副本内并用 `-run '^TestProbeR3'` 单独执行（避免 beego ORM 全局重复注册），跑完随副本删除。**建议把前两条关键断言转为永久测试**。

---

## 五、本轮新发现（R3）

| 编号 | 级别 | 发现 | 建议 |
| --- | --- | --- | --- |
| **R3-N1** | **P1** | **N4 门禁饿死 N2 补确认**。`reconcile.go:296-321` 中，`unverified[key]` / `uncertainEmpty` 产生的 `reason` 在第 309-313 行 `continue`，而 `reconcileClosedCycleFills` 在第 317 行。已平仓周期的 key 必然不在全量快照里（`positions()` 跳过零仓位行），因此**每个 closed 周期都依赖 `FreshPositionSide` 返回明确零 row**。整改记录第 44 行自认「生产 API 是否始终返回明确方向 row 须在正式 API 环境确认」——若不返回，N2 等于未修，全部周期永久 `incomplete`。探针已实证全套链路 | 实盘授权前**必须**先抓真实 Binance symbol-scoped `PositionRisk` 的零仓位响应；把「REST 补确认」与「仓位快照证明」解耦（补确认是只读事实核对，不应被仓位快照门禁阻断，或至少允许先补确认再拒绝结算）；把探针 A 转为永久测试，并让 N2 永久测试改用实现 `ExactPositionSideSource` 的 source |
| **R3-N2** | **P1/P2** | **`main.go` 首次接进自动真实下单，且提案身份校验缺失**。每分钟 `loopRun` 在 `ReconcileAll` 无 error 时无条件调 `RepairActiveManagedProtections`，后者经 `EnsureProtection → Executor.Execute → Broker.Submit` 真实挂 SL/TP；只校验 `Symbol/Side/StopLoss>0`，**不校验 `Status`、`RiskStatus`、`ExpiresAt`、`RejectedAt`、`ApprovedAt`**（对比 `service.go:251` 原执行路径有 `StatusApproved + RiskPass` 门禁）。探针实证 rejected+risk_fail+过期提案仍下单。这是方案 01 线从「账本/对账/结算」扩展到「自动真实下单」的 Gate 级变化，整改记录未把这一点作为独立 Gate 列出 | ① 至少加 `Status ∈ {executed, protection_failed}`、`RiskStatus == RiskPass`、`ExpiresAt` 未过期的校验；② 为这条路径加显式 opt-in 开关与独立限频/审计（默认关闭）；③ 部署前逐 Owner 确认「是否允许自动重挂」并留存实盘订单身份证据 |
| **R3-N3** | P2 | **单周期失败阻塞整轮 sweep**。`RepairActiveManagedProtections` 遇到无提案、pending/reconcile 保护、身份不符、提交失败等任一情形直接 `return`，后续周期不再处理。一个长期异常周期会让其它周期**永久失去止损** | 改为按周期收集错误（continue + 汇总），至少保证其它周期继续修复；错误单独打「报警」并进入 `ReconcileSummary`；补一条「多周期中一个失败仍修复其余」的回归测试 |
| **R3-N4** | P2 | **「对账成功」判定过松 + 无超时**。`ReconcileAll` 只在 `ReconcileOwner` 返回 error 时才失败，`summary.OrdersUnresolved` 不阻断；因此 `main.go` 的 `else` 分支可能在对账未完成时用**陈旧 `ManagedQty`** 挂超量保护单（下一轮再被 `cancelExcessProtections` 撤销 → 挂单/撤单抖动）。同时新增调用使用 `context.Background()` 无 deadline，而 `loopRun` 用 `atomic.CompareAndSwapInt32(&running,0,1)` 保证串行——一旦 REST 阻塞，`running` 永不释放，整个每分钟任务（含 `ReconcileAll` 本身）**永久停摆** | ① 用带超时的 `context.WithTimeout`（如 30s）；② 只有全部 owner 的 `OrdersUnresolved == 0` 才允许重挂；③ 或把重挂拆成独立任务，避免拖垮对账 |
| **R3-N5** | P3 | **只修止损不修止盈 + 错误吞掉**。`hasExactStop` 只查 `IntentStopLoss`，TP 被撤销后不会恢复（探针实证）；`EnsureProtection` 把 TP 失败写进 `ProtectionResult.TakeProfitError` 却返回 `nil error`，`RepairActiveManagedProtections` 完全不检查该字段（对比 `service.go:439` 原路径会处理），TP 挂单失败会静默 | 同时检查 TP 覆盖；修复路径显式检查 `TakeProfitError` 并打「报警」；补回归测试 |
| **R3-N6** | P3 | **N4 的 `ExactPositionSideSource` 是可选接口、fail-open**。`if exact, ok := r.Account.(ExactPositionSideSource); ok { ... }` 未实现时整段保护**静默跳过**、无告警，退回「缺失 key = 0」——与首轮 F3（`OrderTradeReader` 缺失静默跳过）同型。生产 `DefaultReconciler`/`NewAccountReconciler` 用的是 `BinanceAccountPositionSource`，当前安全，但任何包装/自定义 source 会破坏保证。另 `budget=6` + `range needed`（map 随机序），超过 6 个缺失 key 时部分周期会长期 unverified | 把 `FreshPositionSide` 并入 `AccountPositionSource` 必需接口，或在 `ok==false` 时显式 fail-closed / 打「报警」；`needed` 改为确定性排序并给出可观测计数 |
| **R3-N7** | P3 | **新增真实下单路径无独立治理**：无 opt-in 开关、无独立限频/预算标记、无审计落库（对比既有下单路径有 `binanceapiusage` 预算与审计）；与对账共用同一每分钟任务。`futuresTradingConfigured()` 只检查 tradeKey 非空，不是专门开关 | 至少加开关 + 每次重挂写审计（proposal_id / position_id / 剩余量 / exchange order ID / 触发原因），并把「重挂次数」纳入看板 |
| **R3-N8** | P3 | **N9 AST 白名单仍有逃逸面**：白名单允许 `go_binance_futures/feature/api/binance`（自身含 `persistFuturesAccountEvent` 等写入口），且只解析 `ImportsOnly`，函数体仍靠 4 个字符串黑名单；其它写函数、跨包 helper 可绕过 | 增加「`service/leadaccount` 只读宿主不依赖 `tradefills` / 不调用 DB 写」的依赖闭包或调用图白名单断言，并给守卫加反例自测 |
| **R3-N9** | P3 | **验收矩阵盲区与记录措辞**：N2 永久测试用 `fakeAccountPositionSource`（仅 `Positions`），完全没走生产 `ExactPositionSideSource` 分支；N1 整改记录用「当前工作区内容」反驳上轮基于当时快照的判断，措辞不准确。用户自述 18:15 对 `go_bn_test` 执行两次 `sync db`，本轮**未复跑**，仅按自述记录 | 把探针 A/B 的关键断言转为永久测试；验收矩阵中每行标注所用的 source 类型与是否覆盖生产路径；历史评审结论以「当时快照」表述，不与整改后状态对撞 |

---

## 六、独立复跑的 Gate 记录

| 项目 | 命令 | 结果 |
| --- | --- | --- |
| 全仓编译 | `go build ./...`（副本） | `EXIT=0` |
| 主程序构建 | `go build -o /tmp/r3_main_bin .` | `EXIT=0` |
| 目标包测试 | `go test -count=1` 9 包（appversion / command / models / feature/api/binance / tradefills / futuresownership / agenttrade / leadaccount / feature） | **全 ok**，`TEST_EXIT=0` |
| Race | `go test -count=1 -race` 5 包 | **全 ok**，`RACE_EXIT=0`；仅 macOS `ld: malformed LC_DYSYMTAB` 非致命噪音 |
| Vet | `go vet` 9 包 | `EXIT=0` |
| 空白错误 | `git diff --check`（用户仓库） | `EXIT=0` |
| 迁移覆盖 | `go test -count=1 -v -run TestSyncDatabaseInitializesAndIsIdempotent ./command` | PASS，日志含 **19→20→21→22→23** 与重复 23 |
| 依赖闭包 | `go list -deps . \| grep -c` | `tradefills=1`、`futuresownership=1`、`agenttrade=1`、`testtrades=0` |
| 探针 | 2 文件、4 组断言，`-run '^TestProbeR3'` 单独执行 | 全部 PASS；其中 4 项**确认缺口** |

> 环境说明：副本由 `git archive HEAD` 生成并补齐未跟踪目录，不含 `strategy_templates/research/**`，因此全仓编译为绿；`controllers` 既有 ORM/模板测试问题不在本轮范围。副本 `conf/app.conf` 指向本地测试库，**全程未执行 `sync db`**。

---

## 七、实盘 Gate：仍未关闭（不得上线 / 启用 T0）

1. **R3-N1 是新增硬前置**：必须先确认真实 Binance symbol-scoped `PositionRisk` 对零仓位是否返回明确方向 row；否则 N2 的 REST 补确认在生产上不可达，已平仓周期永久 `incomplete`、风控不触发。
2. **R3-N2 是新增硬前置**：`main.go` 自动重挂路径需先补提案身份校验、开关与审计，并逐 Owner 完成实盘订单身份验证；整改记录第 43 行所列「其他 Owner / Lead 无通用可信触发价恢复链」仍未关闭。
3. **F1 真实部分成交证据仍缺**：需人工授权抓一笔真实多次部分成交的 WS `l/L/n/N/rp/T` 与 REST `userTrades` 逐 `tradeId` 比对。
4. **生产 schema 仍需从当前版本升到 v23**（新增 `rest_verified`）；整改记录称仅在 `go_bn_test` 完成两次 `sync db` 且均为 `v23 already up to date`，本轮**未复跑用户数据库**，生产 `go_binance` 未动。
5. **F2 仍只是应用内 Warning + `ReconcileSummary` 字段**，无主动推送、无人工兜底开关；不得手工改 `settlement_status=settled`。
6. **01-4 / 01-5 未实施，T0 未启用**：旧 `order` 读写仍在，前端 `incomplete/settled` 不可见，新主子表 API 未实现；生产 5 条未归属非零镜像仓位须上线前独立只读复核。
7. 生产 schema 变更、二进制替换、重启与实盘验证按方案 §四 01-6 均需**单独授权**。

**离线通过 ≠ 实盘可用。** 本轮通过结论仅覆盖本地代码、SQLite 隔离测试、静态检索与探针证据。

---

## 八、审计纪律声明

- 未修改用户任何业务代码、未 commit/push、未改 `conf/app.conf`、未触碰 `strategy_templates/` 与 `static/`。
- 副本 `/tmp/phase_audit_r3` 中的探针在报告完成后随副本删除；用户仓库仅新增本报告。
- 副本未执行 `sync db`、未连接真实 Binance、未执行任何生产写操作。
- 对用户自述的 `go_bn_test` 两次 `sync db` 与本地 MySQL 索引自检，按「未独立复跑」标注，未当作本轮实证。

---

## 九、第三轮复审后代码整改（2026-11-08）

本报告是修复前的独立审计快照，不覆盖原结论。R3-N1～N9 的后续修复、逐项离线回归、严格默认禁用的 AgentTrade 真实重挂开关及仍未关闭的实盘 Gate，详见 [CODE_REVIEW_01-0_01-3-rereview2_整改记录.md](CODE_REVIEW_01-0_01-3-rereview2_整改记录.md)。特别注意：**REST Fill 补核实与零仓位确认现已解耦，缺少零仓位证据仍禁止最终结算**；自动 AgentTrade 保护单重挂默认关闭，不会因升级二进制自动启用。
