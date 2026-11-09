# Code Review：方案 01「合约订单审计与部分成交」01-4 旧 `order` 业务退役

- 审计时间：2026-11-08 20:11–20:35（北京时间）
- 依据：方案文档 §四 `01-4`（第 165–177 行）、§四前置约束（第 263–264 行）、§十四「01-4 旧 `order` 业务退役实施记录」（第 410–449 行）
- 审计对象：`feat/trade-upgrade`，**HEAD 已推进到 `5356d0c feat: trade upgrade 01-0_01-3`**（01-0～01-3 与 R3 整改已提交）；工作区仅 01-4 改动，20 个已跟踪文件 +245/−592、2 个新增测试文件，均**未提交**
- 方式：`git archive HEAD` + 拷贝全部未提交文件 + 同步删除 `feature/order.go` 建隔离副本 `/tmp/phase_audit_014`，实跑 Gate，写 2 组临时探针；另对前端 UI 仓库做**只读**交叉核对
- **未改业务代码、未 commit/push、未执行 `sync db`、未访问真实 Binance、未覆盖 `static/`、未重启服务**

---

## 一、结论先行

| 维度 | 结论 |
| --- | --- |
| 01-4 逐项工作 | ✅ **6/6 全部落地**，另补齐了方案清单遗漏的 `scanner` 主策略冷却切换 |
| 「源码无仍会执行的旧表查询/写入/删除」 | ✅ **探针全仓实证**：380 个非测试 Go 文件（35 个归档/迁移白名单），可执行行违规 **0** |
| 「真实交易和既有安全退出不受影响」 | ✅ 离线通过：10 个开平仓路径、通知、风险预检、Lead 闸门、WS 镜像均保留；目标包 test/race 全绿 |
| 「自动亏损控制保持有效」 | ⚠️ **代码链路在，但实盘有效性未验证且已无回退路径**（014-F1） |
| 离线 Gate | ✅ 全绿（build / 10 包 test / 7 包 race / vet / diff-check） |
| 独立既有红点 | ⚠️ `controllers` 全包 3 项失败，与 01-4 无关，但无法用 `go test ./...` 作 Gate |
| 实盘 Gate / T0 | ❌ **未通过、未授权**；01-5 / 01-6 未实施 |
| 是否建议合并 | **建议合并离线代码；不得据此部署或启用 T0**，014-F1 是部署硬前置 |

> 核心判断：01-4 的删除动作本身干净、边界清楚（保留物理表、保留 sync-db CLI 兼容注册、保留 `futures_orders` 实时镜像），是本方案里执行质量较高的一个阶段。**但它把自动亏损控制压成了单入口**：旧触发函数彻底成为死代码，而新入口 `applySettledRisk` 依赖一条尚未取得实盘证据的「零仓位方向 row」证明链。这不是本阶段引入的 bug，却是本阶段之后必须显式承接的 Gate 风险。

---

## 二、审计范围与基线变化

| 项 | 状态 |
| --- | --- |
| HEAD | `49ba0e5` → **`5356d0c`**（01-0～01-3 与 R3 整改已提交） |
| 工作区 | 20 个已跟踪文件 +245/−592；新增 `feature/order_retirement_stage01_4_test.go`、`feature/strategy/coin/cooldown_stage01_4_test.go` |
| 删除文件 | `feature/order.go`（64 行，30 分钟旧订单猜配/误删任务） |
| DB Schema | **仍为 v23**，本阶段不升版本、不迁移历史、不 DROP 物理 `order` 表 ✅ |
| 前端 UI 仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`：4 文件 +13/−290（临时页面 / i18n / 删除 `src/api/order.ts`），**未提交**；后端 `static/` 未被覆盖 ✅ |
| 副本一致性 | `feature/feature.go`、`main.go`、`controllers/orders.go` 与用户工作区 sha256 完全一致；`feature/order.go` 已在副本中同步删除 |

> 副本构建注意：`git archive HEAD` 会带上 HEAD 中仍存在的 `feature/order.go`，必须按 `git status --porcelain` 的 ` D ` 行显式删除，否则副本与工作区不等价（本轮首次构建即遇到并已修正）。

---

## 三、验收清单逐项核对

| 方案 01-4 工作项 / 验收 | 结论 | 代码与测试证据 |
| --- | --- | --- |
| 删除 `insertOpenOrder` / `insertCloseOrder` 及 Runner 的 `RecordOpen/RecordClose` | ✅ | `feature/feature.go` 删除两函数（−67 行）与 10 个开平仓路径的调用；`account_trade_cycle.go` 删除字段、赋值与 `validate()` 校验；Stage3/4-2/4-5/4-6/7-4 回归测试移除 fake history writer |
| 删除 `UpdateOrderStatus` 与 `main.go` 30 分钟定时调用 | ✅ | `feature/order.go` 整文件删除；`main.go` 移除 `loopRun(...UpdateOrderStatus, 30min)` |
| 删除 `feature/ownership.go` 撤单后 `DELETE FROM order` | ✅ | −3 行；真实 Binance `Cancel`、Managed 状态与部分已成交 Fill 保留 |
| 清理 `coin/common.go`、`coin5.go`、`coin6.go` 旧表查询 | ✅ | 新增 `getLimitMinManagedClose(5)`：基于 `futures_managed_orders`，`account_id='main'` + `intent__in(close,take_profit,stop_loss)` + `filled_qty>0` + `updated_at>=cutoff`；DB 错误时 coin5/coin6 返回空池（fail-closed，不把历史缺失当作可开仓） |
| 停止旧 `controllers/orders.go` GET/DELETE；注册前完成引用扫描 | ✅（有边界） | 三端点 `Get/Delete/DeleteAll` 一律 HTTP **410 Gone** 且不查库；路由保留 tombstone 注释；`models.Order` ORM 注册收窄到 `if isSyncDatabaseCommand(os.Args[1:])`（`main.go:94-95`），**探针实证 scoped=true** |
| 不迁移旧数据、不 DROP 物理 `order` 表 | ✅ | 全仓无旧表 DROP/数据迁移；`models.Order` 定义与 `TableName()="order"` 保留；`command/**` 中 v19 迁移与测试仍引用（CLI 兼容例外，非交易运行时） |
| **验收：源码无仍会执行的旧表查询/写入/删除** | ✅ **实证** | 探针 A 遍历副本全部 `.go`：扫描 380 个非测试文件、白名单 35 个（`models/**`、`command/**`、`main.go`）、跳过注释行后，可执行行命中 `models.Order` / `` `order` `` / `QueryTable("order")` / `DELETE FROM order` / `insertOpenOrder` / `RecordOpen` 等 11 类危险串 **0 次**。`TestStage014LegacyOrdersRoutesReturnGoneWithoutDB` 三端点在无 DB 注册下 PASS |
| **验收：真实交易和既有安全退出不受影响** | ✅ 离线 | 逐段 `git diff -w` 核对：10 处 `SubmitOpen/SubmitClose` 调用、`NotifyOpen/NotifyClose`（含 MARKET 上浮 0.1% / 下调 0.12% 估算价）、`riskAllowsOpen`、`EnsureConfig`、Lead mock 白名单与风险预检全部保留；Stage3 golden 断言由 `RecordOpen` 改挂 `NotifyOpen`，价格期望仍为 `LONG:100.1` / `SHORT:99.9` 且通过。`futuresOrders.go` 只删注释掉的死代码，`futures_orders` WS/REST 实时镜像查询保留 |
| **验收：自动亏损控制保持有效** | ⚠️ **代码在、实盘未验证** | `applySettledRisk`（`settlement.go:283`）仍由 `settleVerifiedFlat` 调用，复刻旧 `AutoLossScale` / `autoTradeToTest` 的 `loss_max_count` / `future_enable` 逻辑；但两旧函数**运行时调用方为 0**（探针 `live_callers=0/0`，全仓 Grep 只剩定义 `feature.go:1101/1131` 与一处注释）。详见 014-F1 |
| 方案 §四.7：`coin5/coin6` 仍可选，退役前确认运行中无旧选择器、做选币回归 | ⚠️ 部分 | 三个旧选择器已改到 Managed 并补 `cooldown_stage01_4_test.go`（账户隔离、ACK 零成交、部分止损成交、过期、开仓不算平仓、DROP TABLE 后 fail-closed）；但 **`GetCoinStrategy` 仍注册 coin1～coin6 与 `smart_local_v2`（`feature.go:934-950`），生产配置实际选用的选择器未获证据**。详见 014-F7 |

---

## 四、实证过的隐式契约（临时探针，副本内执行，随副本删除）

| 探针 | 输出 | 判定 |
| --- | --- | --- |
| **A. 全仓旧表退役扫描**（比永久守卫更强：全部非测试 Go 文件、跳过注释、白名单归档/迁移面） | `PROBE_REPO_WIDE: scanned=380 allowlisted=35 violations=0`<br>`PROBE_MAIN_GUARD: condition_lines=[94 273 280] register_line=95 scoped=true`<br>`PROBE_RISK_CALLERS: AutoLossScale( live_callers=0`<br>`PROBE_RISK_CALLERS: autoTradeToTest( live_callers=0` | ✅ 旧表运行时引用归零；`models.Order` 注册严格限定在 sync-db CLI；⚠️ 两个旧风控函数已无调用方 |
| **B. 冷却口径**（构造「真实成交但 `updated_at` 陈旧」「一小时前成交但状态被刷新」「零成交」四组 Managed 订单） | `PROBE_COOLDOWN_UPDATED_AT: fresh=true stale_updated=false refreshed_old=true zero_fill=false` | ⚠️ 冷却实际按 `updated_at` 而非真实成交时间：既可能**漏冷却**，也可能**过度冷却**。详见 014-F2 |
| 控制器 410（永久测试，独立复跑） | `TestStage014LegacyOrdersRoutesReturnGoneWithoutDB` 3 个子用例全 PASS，`-race` 亦 PASS，且无 ORM 注册 | ✅ fail-closed、不查库、不删历史 |
| 迁移/回归测试 | `scanner`、`feature`、`coin`、`futuresownership`、`tradefills`、`agenttrade`、`leadaccount`、`command`、`models`、`feature/api/binance` 共 10 包 `-count=1` 全 ok | ✅ |

> 审计过程如实记录：探针 A 初版把注释里合法提及的 `` `order` `` 计为违规（3 处），探针 B 与同包永久测试同跑时触发 beego ORM 全局重复注册 panic；两者都是**探针自身缺陷**，已修正（跳过注释行、探针用 `-run '^TestProbe014'` 单独执行、Gate 前先移出探针文件重跑干净结果），**不是用户代码缺陷**。修正后干净 Gate 与两个隔离探针均通过。

---

## 五、风险与待改进（014-F）

| 编号 | 级别 | 发现 | 建议 |
| --- | --- | --- | --- |
| **014-F1** | **P2 / 实盘硬前置** | **自动亏损控制变成单入口且依赖未验证的实盘 API**。01-3 已把风控从 `insertCloseOrder` 迁到 `applySettledRisk`，01-4 删除旧函数后 `AutoLossScale` / `autoTradeToTest` 成为死代码（探针 callers=0）。`applySettledRisk` 只在 `settleVerifiedFlat` 中触发，而后者要求 `futuresownership` 拿到「symbol-scoped 新鲜 REST 明确零仓位方向 row」。R3 整改记录第 31 行明确：若真实 Binance 不返回该行，CLOSED 周期保持 `incomplete`、**收益风控不触发**。方案 §四.6 要求「不得直接删除这两项风险机制」——逻辑确由 `applySettledRisk` 承接，但**实盘无第二条路径、无回退、无人工兜底** | ① 上线前必须实盘抓一笔已归零的 symbol-scoped `PositionRisk` 响应，确认方向 row 存在；② 删除两个死函数或加「已迁移至 settlement」注释，避免后续维护者误以为仍被调用；③ 对超期 `incomplete` 提供主动告警/人工兜底（承接 R2-F2 遗留）；④ 在 01-6 验收里把「结算触发一次亏损控制」列为**实盘**用例，而非仅离线用例 |
| **014-F2** | P2 | **冷却口径与「真实成交后 5 分钟」不等价**。新查询用 `futures_managed_orders.updated_at`，而非 `futures_trade_fills.trade_time`。探针实证：`stale_updated=false`（有真实成交但 `updated_at` 陈旧 → **不冷却**，可能过早允许再开仓）与 `refreshed_old=true`（一小时前成交因状态刷新 → 仍冷却，过度保守）。实施记录第 427 行只自认「可能保守延长冷却」，未覆盖缩短方向 | 冷却改以真实 Fill 的 `trade_time` 为准，或至少 `MAX(updated_at, 该订单最后 Fill 时间)`；补一条「成交在窗口内但 `updated_at` 陈旧」的永久测试；在 01-5 历史汇总中继续严格用 Fill 时间与成本，不要复用 `updated_at` |
| **014-F3** | P3 | **静态防回归守卫偏弱**。`TestStage014NoLegacyHistoryRuntimeReferences` 只扫 7 个文件的字符串黑名单，注释中的合法提及都会误报（本轮探针初版即复现），且无法覆盖新增文件/包、import alias、跨行拼接。真正给出「全仓零违规」结论的是本轮临时探针，而探针随副本删除 | 把探针 A 的全仓扫描（跳过注释 + 明确归档白名单 + `main.go` 条件注册断言）固化为永久测试；或改为 `go list -deps` / AST 级符号断言 |
| **014-F4** | P3 | **`models.Order` 条件注册无回归保护**。`isSyncDatabaseCommand(os.Args[1:])` 要求恰好 `sync db` 两参数；当前 `scoped=true` 已实证，但若未来 `sync db` 加 flag、拆子命令或在其它 CLI 路径复用注册，条件会静默失效，导致低版本 v19 升级漏注册或普通启动重新引入归档模型 | 补永久测试：断言「普通启动参数不注册 `models.Order`」「`sync db` 参数注册」；把条件抽成单一函数并加注释说明这是唯一例外 |
| **014-F5** | P3 | **`controllers` 全包测试仍红**。副本中 3 项失败：2 项策略模板「至少需要一个已启用技术指标」校验、1 项 `lstat ../strategy_templates: no such file or directory`（副本不含用户未跟踪的 research 目录）。与 01-4 `/orders` 无关，HEAD 干净副本同样会红，但意味着 **`go test ./...` 不能作为 Gate**，只能用目标包 + `-run` 子集 | 单独立项处理策略模板测试隔离；本阶段报告须明确「目标包绿 / 全量红」的拆分，不要修别人的研究产物来凑全绿 |
| **014-F6** | P3 | **过渡态前后端发布顺序风险**。后端三端点 410 + 前端临时「订单历史升级中」页；实施记录明确**未覆盖后端 `static/`**（避免新旧前后端版本不一致，纪律正确）。但运行中的旧二进制仍提供磁盘上的旧静态文件，若后端先部署 410，用户点旧页面会得到 410 报错；i18n 中 `orderPage.button.search/delete` 等旧键亦残留 | 在 01-5/01-6 部署方案写明前后端发布顺序、灰度与回滚；01-5 上线新页面时一并清理旧 i18n 键；上线前不要覆盖 `static/` |
| **014-F7** | P3 | **生产选币器未确认**。`GetCoinStrategy` 仍可返回 coin1～coin6（`feature.go:934-950`），方案 §四.7 要求退役前确认运行中无旧选择器并做启动/选币回归。本轮把 coin5/coin6 一并改到 Managed 并补测试是正确补偿，但生产 `config` 实际使用的选择器没有证据（既有事实：测试库用 `smart_local_v2`，生产未验证） | 上线前只读核对生产配置的选择器值，并对该值跑一次启动 + 选币回归；若确为 `smart_local_v2`，确认其冷却已走 `recentClosedSymbolsForAccountWithOrm` 的账户隔离路径 |
| **014-F8** | P3 | **归档/迁移面不受守卫覆盖**。`models/**`、`command/**` 仍保留旧表模型定义、`INSERT INTO order` 测试夹具与 v19 迁移路径，属方案允许的 CLI 兼容例外；但永久守卫完全不扫这两个目录，未来若在此混入交易运行时代码不会被发现 | 在扫描白名单旁加注释与「仅允许迁移/归档用途」的符号级约束，并定期人工复核 |

---

## 六、独立复跑的 Gate 记录

| 项目 | 命令 | 结果 |
| --- | --- | --- |
| 全仓编译 | `go build ./...`（副本，不含未跟踪 research 目录） | `EXIT=0` |
| 主程序构建 | `go build -o /tmp/014_main_bin .` | `EXIT=0` |
| 目标包测试 | `go test -count=1` 10 包（scanner / feature / coin / futuresownership / tradefills / agenttrade / leadaccount / command / models / feature/api/binance） | **全 ok** |
| Race | `go test -count=1 -race` 7 包 | **全 ok**；仅 macOS `ld: malformed LC_DYSYMTAB` 非致命噪音 |
| Vet | `go vet` 11 包（含 `controllers`） | `EXIT=0` |
| 控制器 01-4 专项 | `go test -count=1 -v -run '^TestStage014' ./controllers`（含 `-race`） | **PASS**（3 子用例） |
| 控制器全量 | `go test -count=1 ./controllers` | **FAIL（既有 3 项，见 014-F5）** |
| 空白错误 | `git diff --check`（用户仓库） | `EXIT=0` |
| 探针 | 2 组，`-run '^TestProbe014'` 隔离执行 | 均 PASS；A 给出全仓零违规与风控死代码，B 量化冷却口径偏差 |
| 前端 | UI 仓库 4 文件只读交叉核对；`src/api/order.ts` 已删且无残留引用；后端 `static/` 未改动 | ✅ 边界正确；未在本轮执行 `pnpm build`（用户自述已通过，未独立复跑） |

---

## 七、实盘 Gate：仍未关闭（不得上线 / 启用 T0）

1. **014-F1**：真实已归零 symbol-scoped `PositionRisk` 的方向 row 证据；否则 `applySettledRisk` 不触发，自动亏损控制在实盘静默。
2. **F1（历轮遗留）**：真实多次部分成交的 WS `l/L/n/N/rp/T` 与 REST `userTrades` 逐 `tradeId` 对照。
3. **014-F7**：生产配置实际选币器确认与启动/选币回归。
4. **保护单重挂**：R3 整改后 `BINANCE_AGENT_PROTECTION_AUTO_REPAIR=enabled` 默认关闭，实盘订单身份/触发价/全 Owner 覆盖均未验证。
5. **01-5 / 01-6 未实施**：新主子表 API、前端 `incomplete/settled` 展示、隔离测试库全场景回归、T0 启用记录均缺；当前 410 + 临时页面为过渡态，不能作为最终生产订单页。
6. 生产 schema 仍 v23（本阶段不升），但**二进制替换、重启、静态资源发布、生产 `go_binance` 只读复核（含 5 条未归属非零镜像仓位）均需单独授权**。
7. 实施记录自述的 `pnpm build` 与「未覆盖 static / 未重启 PID 65353」按用户自述记录，本轮未独立复跑。

**离线通过 ≠ 实盘可用。** 本轮通过结论仅覆盖本地代码、SQLite 隔离测试、静态检索与探针证据。

---

## 八、审计纪律声明

- 未修改用户任何业务代码、未 commit/push、未改 `conf/app.conf`、未触碰 `strategy_templates/` 与后端 `static/`。
- 前端 UI 仓库仅执行 `GIT_OPTIONAL_LOCKS=0 git diff/status/log` 等只读命令，未做任何写操作；避免了 `.git/index.lock` 残留。
- 副本 `/tmp/phase_audit_014` 中的探针在报告完成后随副本删除；用户仓库仅新增本报告。
- 副本未执行 `sync db`、未连接真实 Binance、未执行任何生产写操作；未重启服务、未操作生产仓位。

---

## 九、复审后的整改状态（2026-11-08）

本报告保留原独立复审快照与 F1～F8 结论。代码层面的 F2/F3/F4/F8 已完成整改并增加永久测试；F1 已删除旧内存风控死代码，**仍由唯一可核验的 settlement 事务触发，实盘零仓位证据未取得、Gate 未关闭**。F5～F7 分别维持既有控制器测试独立红点、前后端原子发布要求与生产选币配置只读验证条件。

详细差异、测试结果和剩余风险见 [CODE_REVIEW_01-4_整改记录.md](CODE_REVIEW_01-4_整改记录.md)。其中真实 Fill 成交时间冷却查询在数据库大规模积累后的性能需独立 EXPLAIN 验收；当前不改变 v23 Schema、不更改生产数据库。
