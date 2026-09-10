# V3-5 Code Review：真实交易安全与仓位/订单归属隔离（Trade Safety & Ownership）

- **Phase**：V3-5 / 真实交易安全与仓位/订单归属隔离
- **审查基线**：分支 `feat/trade`，HEAD `18f2c87`（Merge PR #48 `feat/ai-agent-v3`），工作区含 V3-5 未提交改动（新增 `service/futuresownership/*` 全量、`models/futures_managed.go`、`feature/ownership.go`、`service/agenttrade/{lifecycle,notifier}.go` 等）
- **审查类型**：**Code Review（review-only）** —— 仅审查，未修改任何代码
- **审查结论**：**AUTOMATED PASS / 人工验收待定** —— 未发现阻塞级（P0）缺陷；5 个子阶段（5A–5E）Gate 在代码与自动化测试层面均可验证通过；提出 8 项非阻塞改进建议
- **审查日期**：2026-09-10

---

## 1. 审查范围与方法

1. 以 `doc/agent/v3/05-phase-v3-5-trade-safety.md`（含 5 个子阶段 Gate + 11 项 Definition of Done）与 `doc/agent/v3/v3-5-implementation-report.md` 为需求基线。
2. 逐文件通读新增/修改代码：`models/futures_managed.go`、`service/futuresownership/{types,service,execution,reconcile}.go`、`service/agenttrade/{lifecycle,notifier,types,service}.go`、`feature/ownership.go`、`feature/{feature,feature_rush,feature_notice,feature_listen}.go`、`controllers/agent_trade.go`、`routers/router.go`、`main.go`。
3. 用全仓检索交叉验证三条硬约束：**是否存在绕过 ownership 的真实下单旁路**、**是否残留 `reduceOnly`/`closePosition` 语义**、**各模块是否真正接入 owner**。
4. 运行 `go build ./...`、`go vet ./...`、`go test -count=1 -race <受影响包>`、`go test -count=1 ./...` 验证代码健康（`-count=1` 规避缓存）。

---

## 2. 自动化验证结果

| 项目                                                                                                        | 结果                                        |
| --------------------------------------------------------------------------------------------------------- | ----------------------------------------- |
| `go build ./...`                                                                                          | ✅ PASS                                     |
| `go vet ./...`                                                                                            | ⚠️ 仅既有噪声 `main.go:333/338 unreachable code`（历史禁用 goroutine 遗留，非本 Phase 引入） |
| `go test -count=1 -race ./service/futuresownership/... ./service/agenttrade/... ./feature/... ./command/... ./models/... ./controllers/...` | ✅ 全绿（含 `-race`）                             |
| `go test -count=1 ./...`                                                                                  | ✅ 无失败                                      |

---

## 3. 变更清单（关键文件）

| 文件                                        | 变更      | 作用                                                                    |
| ----------------------------------------- | ------- | --------------------------------------------------------------------- |
| `models/futures_managed.go`               | 新增      | 两张 ownership 账本表：`FuturesManagedPosition` / `FuturesManagedOrder`      |
| `service/futuresownership/types.go`       | 新增      | 5 个 owner 常量、4 个 position 状态（含 `reconcile_required`）、7 个 order 状态、4 个 intent |
| `service/futuresownership/service.go`     | 新增      | `ClaimOrder` / `ApplyFill` / `ReconcilePosition` / `CloseQuantity` / `ensureSlotAvailable` |
| `service/futuresownership/execution.go`   | 新增      | Ownership-aware `Executor`（Claim → Submit → ApplyFill；Submit 失败按 clientOrderID 对账） |
| `service/futuresownership/reconcile.go`   | 新增      | `ReconcileOwner` / `PositionOverview` / `ReconcileAll`                |
| `service/agenttrade/lifecycle.go`         | 新增      | `EnsureProtection` / `Close`（V3-5E Agent 仓位生命周期）                       |
| `service/agenttrade/notifier.go`          | 新增      | `WebTradeNotifier.ProtectionFailed`                                   |
| `feature/ownership.go`                    | 新增（AM）  | 主 StartTrade 与其余模块的 mutation 收口层                                       |
| `feature/api/binance/index.go`            | 修改      | 新增 `CreateOwnedOrder`（ownership 通用下单原语）                                |
| `controllers/agent_trade.go`              | 修改      | `Ownership` / `ReconcileOwnership` 视图                                  |
| `routers/router.go`                       | 修改      | 3 条新路由                                                                 |
| `main.go`                                 | 修改      | `dbVersion = 10`、注册表、启动 `ReconcileAll`                                 |

---

## 4. V3-5A：Ownership Foundation

| #  | Gate / 要求                                                       | 实现证据                                                                                                                                            | 状态 |
| -- | --------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 1  | 能明确区分 managed 与 unmanaged                                         | `FuturesManagedPosition` / `FuturesManagedOrder` 独立账本；`PositionOverview` 对无 managed record 的账户仓位标 `unmanaged`                                        | ✅  |
| 2  | ownership 缺失默认 fail closed                                        | `ClaimOrder` 对非 open intent 要求先存在本 owner managed position 且 `requested ≤ managed_qty`，否则拒绝；`ReconcileOwner` 只遍历本地 managed，绝不自动认领                    | ✅  |
| 3  | managed quantity 可独立于 account quantity 表达                        | 字段 `managed_qty`；`ReconcilePosition` 账户多时**不扩大**、账户少时**收缩**、归零时**关闭**                                                                               | ✅  |
| 4  | Schema 通过 SQLite/MySQL sync Gate                                 | `main.go` `dbVersion=10` + `orm.RegisterModel`；ORM-only 迁移，无独立 SQL 文件；`command/db_update_test.go` 已更新                                            | ✅  |
| 5  | 不把 `futures_positions`/`futures_orders` 当 ownership 真相源           | 全仓检索确认：两张镜像表仅用于 Observation（风险/冲突/余额判断），无任何 mutation 路径以其为依据                                                                                         | ✅  |
| 6  | 旧 `order` 表继续用于历史/统计兼容，不升级为 ownership 账本                          | `insertOpenOrder` / `insertCloseOrder` 仍写旧表，但不参与 ownership 判定                                                                                       | ✅  |
| 7  | 同一 `(symbol, position_side)` 首版最多一个 active owner                  | Service 层 `ensureSlotAvailable` 校验 + `mutationMu` 互斥锁；`TestSameOwnerCannotAddOrCreateSecondOpenOrder`                                            | ✅  |
| 8  | 表达 owner / symbol / position_side / managed_qty / source_ref / intent / 双 order id / qty / status / 时间戳 | 两表字段齐全，`FuturesManagedOrder.ClientOrderID` 唯一，`closed_at`、`last_reconciled_at` 齐备                                                                 | ✅  |

**5A 结论：Gate 全部满足。**

---

## 5. V3-5B：修复主 StartTrade 修改边界（风险最高项）

| #  | Gate / 要求                                    | 实现证据                                                                                                                             | 状态 |
| -- | -------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | -- |
| 9  | 手工仓位不会被 `StartTrade` 平掉                       | `feature.go:80` `syncAutoStrategyOwnership(positions)` 把后续遍历的持仓替换为 `owner=auto_strategy` 的 managed 视图（数量取 `managed_qty`），手工仓位不在集合内 | ✅  |
| 10 | 手工 LIMIT 挂单不会被 timeout cancel 撤销             | `feature.go:609` `cancelTimeoutOrder` 已改为薄封装，内部只调 `cancelTimeoutAutoStrategyOrders`；后者只遍历 `OwnerAutoStrategy` 的 `IntentOpen`，跳过 excluded，未知状态先 Reconcile | ✅  |
| 11 | `auto_strategy` 自己创建的仓位仍能正常退出                 | `submitAutoStrategyClose` 走 `CloseQuantity = min(managed_qty, account_qty)` + `IntentClose`                                        | ✅  |
| 12 | 人工加仓后只平 managed quantity                      | `ReconcilePosition`：`account > managed` 时 managed 不增加；`TestReconcileNeverClaimsManualIncreaseAndShrinksAfterManualReduce`       | ✅  |
| 13 | 人工减仓后 managed quantity 自动收缩，不补回                | `0 < account < managed` → managed 下调到 account；同上测试覆盖                                                                             | ✅  |
| 14 | 平仓前重新获取账户数量，`close_qty = min(managed, account)` | `feature/ownership.go:219` `CloseQuantity(...)` 内部先 `currentAccountPositionQty` 再取 min                                          | ✅  |
| 15 | 已存在 unmanaged / 其它 owner 同方向仓位时新开仓拒绝          | `ownership.go:208` 开仓前 `ensureAccountOpenSlotAvailable`，存在则报 "ownership is not safe to merge"                                    | ✅  |
| 16 | `FutureExcludeSymbols` 优先于自动平仓与撤单             | 平仓遍历与 `cancelTimeoutAutoStrategyOrders` 均在 exclude 命中时 `continue`；`TestExecutor...` 与 feature 测试覆盖                              | ✅  |
| 17 | 风险统计仍观察全账户（Observation 可看全）                   | `feature.go:92` `accountTradeRiskCounts(positions)` 注释明确"风险统计继续观察全账户"，与规格 `Observation 可以看全账户` 一致                                | ✅  |

**5B 结论：Gate 全部满足。** 特别确认了 `cancelTimeoutOrder` 已被彻底改造（不再是旧的全账户撤单），这是本 Phase 风险最高的单点。

---

## 6. V3-5C：其它真实交易模块 Owner 化

| #  | Gate / 要求                                    | 实现证据                                                                                                                                                              | 状态 |
| -- | -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 18 | `new_coin_rush` 接入                            | `ownership.go:176` `submitNewCoinRushOpen` → `feature_rush.go:125/128/133/136` 全部替换为 owned 入口，`sourceRef = new_coin_rush:<coin.ID>`                                 | ✅  |
| 19 | `notice_auto_order` 接入                        | `ownership.go:180/195` `submitNoticeAutoOpen` / `submitNoticeProtection` → `feature_notice.go:99/130/140/146/177/187` 接入，保护单用 `noticeManagedQuantity` 取 managed 数量而非账户数量 | ✅  |
| 20 | `funding_rate` 接入                             | `ownership.go:203` `submitFundingRateOpen` → `feature_listen.go:353/377` 接入                                                                                           | ✅  |
| 21 | `agent_trade` 接入                              | `service/agenttrade/default.go:115` + `lifecycle.go` 全链路 `OwnerAgentTrade`                                                                                             | ✅  |
| 22 | 不同 owner 之间不能互相平仓/撤单/改单                       | `Executor.Cancel` 校验 `order.Owner == expectedOwner`；`ClaimOrder` 拒绝跨 owner；`TestClaimOrderIsIdempotentAndRejectsOtherOwner`                                        | ✅  |
| 23 | Agent 创建的仓位不能被 `auto_strategy` 处理             | `syncAutoStrategyOwnership` 只取 `OwnerAutoStrategy`；`TestReconcileDoesNotClaimUnmanagedAccountPosition`                                                                 | ✅  |
| 24 | 同方向存在其它 owner / unmanaged 时新 owner 默认拒绝开仓      | `ensureAccountOpenSlotAvailable` 在 `submitOwnedFeatureOpen` 中对**所有 owner 统一生效**（含 auto_strategy / rush / notice / funding）                                           | ✅  |
| 25 | 部分成交按实际成交数量归属                                 | `ApplyFill` 仅按 delta 增减，累积量不可降；`TestPartialFillOwnsOnlyActuallyFilledQuantity`、`TestExecutorRequiresConfirmedMarketFill`                                           | ✅  |
| 26 | 生成 clientOrderId → 登记 → 提交 → 按真实成交更新 → 仅原 owner 管理 | `Executor.Execute` 严格按此顺序；clientOrderID 按 owner 前缀 `aut_/rush_/notice_/fund_/agt_`                                                                              | ✅  |

**5C 结论：Gate 全部满足，四个模块均已真实接入而非仅提供接口。**

---

## 7. V3-5D：Restart / Reconcile

| #  | Gate / 要求                              | 实现证据                                                                                                                    | 状态 |
| -- | -------------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | -- |
| 27 | 重启后 ownership 不丢失                       | ownership 持久化在两张独立表；`main.go:233` 启动执行 `ReconcileAll`；`TestReconcileRecoversOnlyRegisteredOrderByClientID`              | ✅  |
| 28 | 历史手工仓位不会因为启动 Reconcile 被认领              | `ReconcileOwner` 只遍历本地 managed positions，绝不根据账户仓位创建 ownership；`TestReconcileDoesNotClaimUnmanagedAccountPosition`、`TestUnknownAccountPositionIsNeverClaimed` | ✅  |
| 29 | 网络超时优先查询 clientOrderId，不盲目重发            | `Executor.Execute` Submit 失败 → `Broker.Lookup(ctx, symbol, clientID)`；查到则按真实状态 `ApplyFill`，查不到则置 `OrderReconcile` 要求先对账      | ✅  |
| 30 | 非 pending 状态订单直接走 Reconcile，防重复提交       | `Execute` 开头判断已有 managed order 状态，非 pending 直接 Reconcile 返回；`TestExecutionIsIdempotentAndNeverSubmitsTwice`              | ✅  |
| 31 | 无法确定时进入 `reconcile_required`             | 位置状态 `reconcile_required`、订单状态 `OrderReconcile`；`TestExecutorLeavesUnknownResultInReconcileRequired`、`TestReconcileKeepsUnknownExchangeResultFailClosed` | ✅  |
| 32 | 不根据旧 `order` 表自动认领历史仓位                   | Reconcile 数据源为 Binance 账户状态 + 本地 managed 表，不读旧 `order` 表                                                                  | ✅  |
| 33 | 本地落后时以 Binance 外部事实修正数量和状态              | `ReconcilePosition` 账户少则收缩、归零则关闭；`TestReconcileShrinksButNeverExpandsManagedQuantity`                                     | ✅  |

**5D 结论：Gate 全部满足。** 第 4 条 Gate（"WS 模式和直接 API 模式 ownership 结果一致"）代码逻辑统一走同一 Executor，但**未做实测**，列入未验证项。

---

## 8. V3-5E：Agent 受控交易仓位生命周期

| #  | Gate / 要求                                          | 实现证据                                                                                                                                        | 状态 |
| -- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------- | -- |
| 34 | Entry → owner=agent_trade Managed Position → Stop → 可选整仓 TP → 整仓 Close → 清理保护单 | `lifecycle.go` `EnsureProtection` + `Close` 完整链路                                                                                              | ✅  |
| 35 | Entry 成功后 Stop 成功，或明确进入 `protection_failed`          | `service.go:403-475` `reconcileExecutedLifecycle`：Stop 失败 → `StatusProtectionFailed` + 审计 + `Notifier.ProtectionFailed`；`TestStopFailureBecomesProtectionFailedWithoutResubmittingEntry` | ✅  |
| 36 | Stop 失败不重发 Entry                                    | 同上测试明确断言不重发；`submitProtection` 失败按 clientOrderID 重试 2 次后 Reconcile                                                                           | ✅  |
| 37 | TP 失败仅 warning，不置 `protection_failed`               | `TestOptionalTakeProfitFailureDoesNotMarkProtectionFailed`                                                                                       | ✅  |
| 38 | 无 managed position 时不为外部仓位建保护单（fail closed）         | `EnsureProtection` 返回 `ErrManagedPositionClosed`；`TestOwnershipLifecycleStopFailureNeverResubmits`                                              | ✅  |
| 39 | 重复 Execute / Close / Reconcile 不产生重复订单              | `TestExecutionIsIdempotentAndNeverSubmitsTwice`、`TestManagedCloseIsIdempotent`                                                                  | ✅  |
| 40 | Close 用 `min(managed, account)`，先撤 TP 保留 Stop，部分成交不提前删 Stop | `lifecycle.go` `Close`；`TestOwnershipLifecycleCloseUsesManagedQtyAndCleansProtection`                                                          | ✅  |
| 41 | 外部/手工仓位不提供 Agent 平仓能力                                | `Close` 入口校验 managed position 存在；`TestOwnershipLifecycleDetectsProtectiveClose`                                                               | ✅  |
| 42 | Fake Broker 覆盖 Entry / 超时 / Stop 失败 / Close / Restart Recovery | `service_test.go` + `lifecycle_test.go` 共 9 个相关用例（fake broker + in-memory/SQLite），覆盖上述 5 类场景                                                   | ✅  |
| 43 | 提供"重新对账"和"关闭此系统仓位"入口                                 | `POST /agents/trade/ownership/reconcile`、`POST /agents/trade/proposals/:id/close`                                                             | ✅  |
| 44 | LLM 不参与 Stop 创建失败、紧急平仓等确定性安全动作                       | `EnsureProtection` / `Close` 为纯确定性代码路径，无 LLM 调用                                                                                                | ✅  |

**5E 结论：Gate 全部满足。**

---

## 9. 统一安全原则逐条核对（规格 §2 / §11）

| 原则                                | 核对结果                                                                                                       | 状态 |
| --------------------------------- | ---------------------------------------------------------------------------------------------------------- | -- |
| Observation 可以看全账户；Mutation 必须先确认 ownership | 全仓检索确认：生产代码（`feature/`、`service/`，排除测试与 `feature/api/` 薄封装层）**无任何直接下单调用**；所有 mutation 必经 `futuresownership.Executor` | ✅  |
| 谁创建谁管理（5 owner + unmanaged 只读）     | owner 常量齐备，`Executor.Cancel` 校验 owner，Reconcile 不跨 owner                                                     | ✅  |
| Fail Closed（不自动认领/平仓/撤单/改单）       | 见 5A #2、5D #28                                                                                               | ✅  |
| managed_qty 而非 managed symbol      | 见 5A #3、5B #12/#13                                                                                           | ✅  |
| 升级前已有仓位/挂单无 managed record 保持 unmanaged | `ReconcileAll` 只同步已有 managed，不新建                                                                              | ✅  |
| `FutureExcludeSymbols` 与 ownership 是两层独立保护 | exclude 在平仓/撤单前优先短路，不受 ownership 影响                                                                            | ✅  |
| 删除真实下单旁路 `GoTestOrder()`          | 全仓检索 `GoTestOrder` **零命中**，已彻底移除                                                                              | ✅  |

---

## 10. Hedge Mode 与下单语义审查

| 检查项                                              | 结果                                                                                                                  |
| ------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------- |
| V3-5 mutation 路径不使用 `reduceOnly`                  | ✅ `CreateOwnedOrder`（`feature/api/binance/index.go:453`）仅设 `Symbol/Side/PositionSide/Type/Quantity/ClientOrderID`，无 `ReduceOnly` |
| V3-5 mutation 路径不使用 `closePosition=true`          | ✅ 同上，平仓用明确 `positionSide` + opposite side + 固定 quantity                                                               |
| 全仓残留 `ClosePosition(true)`                        | ⚠️ 仅 `feature/api/binance/index.go:595/616`（`OrderTakeProfit` / `OrderStopLoss`），**合约侧已无任何调用点**（残留死代码），见建议 #1         |
| 订单类型白名单                                          | ✅ `BinanceOrderBroker` 限定 MARKET/LIMIT/STOP_MARKET/TAKE_PROFIT_MARKET                                                  |
| clientOrderID 按 owner 前缀隔离                        | ✅ `aut_/rush_/notice_/fund_/agt_`，便于交易所侧按前缀对账与人工核查                                                                  |

---

## 11. DB 迁移 9 → 10 兼容性

- `main.go` `dbVersion = 10`，新增两张表通过 ORM `RegisterModel` + `Syncdb` 注册，ORM-only、无独立 SQL 迁移文件，与项目既有 V3-x 迁移方式一致。
- 两张新表为**纯新增**，不修改既有表结构；旧 `order`、`futures_positions`、`futures_orders` 语义不变，升级不破坏历史数据。
- `command/db_update_test.go` 已同步更新并通过。
- 降级风险：从 10 回退到 9 时新表为孤儿表（不阻塞旧版本运行），无数据损坏风险。

---

## 12. 测试覆盖评估

新增/更新的核心测试共 **20+ 个用例**，与风险点一一对应：

- **ownership 归属语义**：`TestUnknownAccountPositionIsNeverClaimed`、`TestClaimOrderIsIdempotentAndRejectsOtherOwner`、`TestPartialFillOwnsOnlyActuallyFilledQuantity`、`TestMutationRequiresOwnedQuantity`、`TestSameOwnerCannotAddOrCreateSecondOpenOrder`
- **reconcile 收缩/不扩张**：`TestReconcileNeverClaimsManualIncreaseAndShrinksAfterManualReduce`、`TestCloseFillReducesOnlyManagedQuantity`、`TestReconcileDoesNotClaimUnmanagedAccountPosition`、`TestReconcileShrinksButNeverExpandsManagedQuantity`
- **执行与对账**：`TestExecutorPersistsOwnershipBeforeAndAfterFill`、`TestExecutorRecoversAmbiguousSubmitByClientOrderID`、`TestExecutorLeavesUnknownResultInReconcileRequired`、`TestExecutorRequiresConfirmedMarketFill`
- **Agent 生命周期**：`TestExecutedEntryRequiresConfirmedProtection`、`TestStopFailureBecomesProtectionFailedWithoutResubmittingEntry`、`TestOptionalTakeProfitFailureDoesNotMarkProtectionFailed`、`TestManagedCloseIsIdempotent`、`TestOwnershipLifecycle*`（4 个）

覆盖度对本 Phase 的关键安全属性是充分的。

---

## 13. 非阻塞改进建议（P1/P2）

| #   | 级别 | 问题                                                                                                                                             | 建议                                                                        |
| --- | -- | ---------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 1   | P1 | `feature/api/binance/index.go:588/609` `OrderTakeProfit` / `OrderStopLoss` 仍使用 `ClosePosition(true)`，与 V3-5 "不用 closePosition" 原则冲突；当前合约侧无调用点（死代码），但未来被误用会造成 Close-All 越权平仓 | 删除这两个函数，或加 `Deprecated:` 注释明确禁止使用                                           |
| 2   | P1 | 同一 `(symbol, position_side)` 单 active owner **仅由 Service 层 `ensureSlotAvailable` + 包级 `mutationMu` 保证**，DB 无唯一约束。单进程正确，但多实例/多进程部署（主程序 + 旁路进程）存在竞态窗口 | 在 `futures_managed_positions` 上加 `(symbol, position_side, status)` 部分唯一索引，或在文档中明确"单实例部署"前提 |
| 3   | P1 | `reconcile_required` / `OrderReconcile` 状态目前只有 UI 查询与手动触发入口，**无主动告警**；若停在该状态，仓位可能长期无人处置                                                        | 复用 `WebTradeNotifier` 增加 reconcile_required 通知，或在 ownership 概览页做醒目提示         |
| 4   | P2 | `feature/ownership.go` 中全局变量 `autoStrategyOwnership` 实际被用于查询 `OwnerNoticeAutoOrder` 等任意 owner（如 `:185`），命名与实际语义不符，易误导后续维护者                          | 重命名为 `sharedOwnership` 或按用途拆分                                               |
| 5   | P2 | `syncAutoStrategyOwnership` 失败时 `feature.go:81-84` 直接 `return`，主循环本轮整体停摆。fail closed 方向正确，但连续失败会静默停止策略                                            | 增加连续失败计数与告警（可复用现有钉钉/Web 通知通道）                                                |
| 6   | P2 | `feature/feature_test_strategy.go` 有未提交改动，且历史上是真实下单旁路的藏身之处（V3-5 已删 `GoTestOrder`）；建议再次确认该文件当前无真实下单路径                                             | 人工复核该文件，或加 lint 规则禁止其直接调用下单 API                                             |
| 7   | P2 | `qtyEpsilon = 1e-12` 为固定绝对精度；对价格/数量量级差异极大的币种（如低价 meme 合约），可能仍需按 symbol 的 stepSize 归一化                                                              | 后续接入 `GetExchangeInfo` 的 quantityPrecision 做按 symbol 的量化比较                    |
| 8   | P2 | 旧 `order` 表仍记录开平仓但不含 owner 字段，历史/统计侧无法区分归属                                                                                                        | 若在 UI 上展示历史订单，建议补 `source_ref`/owner 冗余列或明确标注其为非权威账本                          |

---

## 14. 未验证项 / 审查边界（本次 review 未覆盖）

1. **未做真实资金 / Binance 测试网端到端验证** —— 全部结论基于代码与 Fake Broker 测试。
2. **未在 MySQL 环境验证 sync** —— 测试均在 SQLite 下运行；MySQL 的 DDL/索引行为未实测。
3. **未核对独立前端仓库 `go_binance_futrues_new_ui`** —— 规格 §9 要求"开启合约交易的警告文案改成只管理本系统创建并登记的仓位/订单"、"AI → 受控交易展示 Managed Position / Stop/TP / order id / clientOrderId / 最近 Reconcile 时间"、"对 unmanaged 与其它 owner 仓位显示归属但不提供越权操作按钮"。本仓 `static/` 有构建产物变更，但**前端是否真正消费了 3 条新接口未验证**。
4. **未实测 WS 模式与直接 API 模式的 ownership 结果一致性**（5D Gate 第 4 条）—— 代码路径统一，但无 WS 环境下的对账实测。
5. **未做并发压测** —— `mutationMu` 在高频并发下的行为仅由 `-race` 单测间接覆盖。
6. **未验证 Binance 双向持仓模式下保护单的真实下单行为** —— STOP_MARKET/TAKE_PROFIT_MARKET 在 hedge mode 下的实际撮合语义需测试网验证。

---

## 15. 人工验收待办

建议按 `doc/agent/v3/v3-5-agent-trade-ownership-testing-guide.md` 执行，至少覆盖：

1. **手工仓位保护**：交易所手工开一个仓位 → 启动系统 → 确认系统不会平掉、不会认领，UI 显示 `unmanaged`。
2. **手工挂单保护**：手工挂一个 LIMIT 单 → 等待超过 `FutureBuyTimeout` → 确认未被撤销。
3. **人工加仓/减仓**：在系统 managed 仓位上人工加仓 → 确认只平 `managed_qty`；人工减仓 → 确认 managed 收缩且不补仓。
4. **重启对账**：系统运行中创建仓位 → 重启 → 确认 ownership 不丢失，历史手工仓位仍为 unmanaged。
5. **超时对账**：模拟下单请求超时但交易所已接单 → 确认按 clientOrderID 查回，不重复下单。
6. **Agent 生命周期**：Approved → Entry → Stop 成功；人为使 Stop 失败 → 确认进入 `protection_failed` 并收到通知，且未重发 Entry；执行整仓 Close → 确认只平 managed 数量并清理保护单。
7. **UI 归属展示**：确认 unmanaged / 其它 owner 仓位显示归属但无越权操作按钮。
8. **`FutureExcludeSymbols` 优先级**：加入排除列表后确认自动平仓与撤单均跳过。

---

## 16. 结论

**AUTOMATED PASS / 人工验收待定。**

V3-5 的 5 个子阶段（5A Ownership Foundation / 5B 主 StartTrade 修改边界 / 5C 其它模块 Owner 化 / 5D Restart-Reconcile / 5E Agent 仓位生命周期）在代码层面**均已落地且可验证**，44 项 Gate 检查点全部通过：

- **核心风险项已消除**：主 `StartTrade` 的平仓与超时撤单已收口到 `auto_strategy` managed 集合（手工仓位/挂单不再被误伤）；`GoTestOrder` 真实下单旁路已删除；生产代码无绕过 ownership 的直接下单调用。
- **安全语义正确**：fail closed、managed_qty 独立表达、Reconcile 只收缩不扩张且不自动认领、超时优先按 clientOrderID 对账而非重发、Hedge Mode 下不使用 `reduceOnly`/`closePosition`。
- **构建与测试健康**：`go build` 通过，`go vet` 仅有历史噪声，受影响包 `-race` 全绿，全仓 `go test -count=1` 无失败。

**无阻塞级（P0）缺陷。** 建议在人工验收前优先处理建议 #1（删除/标注含 `ClosePosition(true)` 的死代码函数）与 #2（明确单实例部署前提或补 DB 唯一约束），其余 6 项可排入后续迭代。

**放行建议**：可进入人工验收；待第 15 节 8 项人工用例通过且在 Binance 测试网完成一轮真实下单验证后，方可在真实资金环境启用。
