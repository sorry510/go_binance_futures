# V3-5 复评报告（大改后的重新审查）

- **对比基线**：`master` = `25f7893`（Merge PR #50，已含 V3-4）；`HEAD` = `0c07530`（`feat: ai agent v3-5`，工作区干净，master 为其祖先）
- **差异规模**：51 个文件 `+5489 / -704`；新增 12 个文件（`feature/ownership.go`、`models/futures_managed.go`、`service/futuresownership/*`、`service/agenttrade/lifecycle.go`、`notifier.go`、`testnet_integration_test.go`）
- **审查类型**：**review-only**（仅审查与验证，未修改任何代码；本报告是新增文件）
- **上一轮报告**：`CODE_REVIEW_V3-5-open-close-consistency.md`（2026-09-12 13:39）。**本报告取代其结论**，其中 6 项问题已在本轮修复
- **审查日期**：2026-09-12（第二轮）

---

## 1. 结论摘要

**AUTOMATED PASS / 人工验收待定。未发现 P0。**

1. 上一轮我提的 **6 项问题全部修复**，且都补了对应单测（§5）。
2. 生产 Futures 下单链路**已彻底收口**：旧直连原语（`BuyMarket/SellMarket/BuyLimit/SellLimit/CreateAgentMarketOrder/OrderTakeProfit/OrderStopLoss`）**已删除**，只剩 `CreateOwnedOrder`（MARKET/LIMIT）与 `CreateOwnedAlgoOrder`（STOP/TP 系列）。
3. **但"平仓逻辑与 master 一致"这句话不再成立**：本轮修复了两处 master 的**恒真/恒假逻辑 bug**，导致两条此前**永不触发**的平仓路径被激活（§7.2）。这是对用户问题最关键的答复。
4. 仍有 2 项需人工决策/确认的风险（1 项行为变更必须知情、1 项做多限制被放宽），以及 4 项 P2/P3 边界问题（§8）。
5. **升级前提**：实施报告已明确"**系统升级不会根据账户现状或旧 `order` 表自动认领历史仓位**"——即升级后既有仓位不再被本系统自动平仓，这是**有意设计**，但必须作为上线前置操作对待（§9.1）。

---

## 2. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./feature/... ./service/futuresownership/... ./service/agenttrade/... ./controllers/...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **57 个包 ok，0 FAIL** |
| `go test -count=1 -race ./service/futuresownership/... ./service/agenttrade/... ./feature/... ./controllers/...` | ✅ 全绿（futuresownership 2.292s / agenttrade 2.750s / feature 3.017s / line 3.277s / controllers 3.736s） |
| `gofmt -l`（仅改动文件） | ✅ 无输出 |
| `git diff --check master HEAD -- '*.go'` | ✅ 无空白错误 |
| 测试网 E2E | ✅ 已实跑通过（见 `TESTNET_TRADE_E2E.md`）：开多→平多、开空→平空、TP/SL 创建-查询-撤销、人工部分减仓→Reconcile→仅平 managed 余量、Algo 挂单清理断言全部 PASS |

> 链接期 `ld: warning ... malformed LC_DYSYMTAB` 为 macOS 工具链噪声。

---

## 3. 变更全景

| 层 | 文件 | 作用 |
| --- | --- | --- |
| Ownership 账本 | `service/futuresownership/{service,execution,reconcile,types}.go` | 订单/仓位记账、执行器、对账器；`client_order_id` 唯一索引；进程内 `mutationMu` |
| 订单原语 | `feature/api/binance/index.go` | 删除全部直连下单函数；新增 `CreateOwnedOrder`（MARKET/LIMIT）与 `CreateOwnedAlgoOrder`（Algo Order API）+ `GetOrderByOrderID`/`GetAlgoOrderByClientOrderID`/`CancelAlgoOrder`；新增 testnet 支持 |
| Feature 接入 | `feature/ownership.go`（434 行）、`feature/{feature,feature_notice,feature_rush,feature_listen,feature_test_strategy}.go` | 6 类真实交易入口全部 Owner 化；`syncStrategyExitPositions`、`RepairNoticeAutoOrderProtections` |
| 策略层 | `feature/strategy/line/{common,line3..line7,line_custom}.go` | **本轮新增改动**：`autoStopNeutralROI`、`baseMarketBreadthPermissions` |
| Agent 生命周期 | `service/agenttrade/{lifecycle,service,notifier,types}.go` | Entry → Stop → 可选 TP → Close → 保护单清理；`protection_failed`/`closed` 状态 |
| 费率套利 | `controllers/eatRate.go` | 合约腿改走 ownership（`OwnerFundingRate`，SourceRef `eat_rate:<id>`） |
| 运行时 | `main.go` | `dbVersion 11→12`；启动 + **每 1 分钟** Ownership 全量对账；testnet 配置门控 |
| API | `routers/router.go` + `controllers/agent_trade.go` | `/agents/trade/ownership(/reconcile)`、`/proposals/:id/close` |

---

## 4. 与 master 的"逻辑一致性"判定（回答核心问题）

| 维度 | 判定 | 依据 |
| --- | --- | --- |
| 开仓**触发条件**（策略信号） | ⚠️ **有变化** | `baseMarketBreadthPermissions` 修复整数除法，4 条市场宽度规则中 3 条的生效条件改变，其中 1 条**放宽**做多（§7.1） |
| 开仓**数量/价格公式与精度** | ✅ 一致 | `(Usdt/price)*Leverage` + `GetTradePrecision` 未改 |
| 开仓**下单方式** | ⚠️ 有意变更 | 全部走 ownership；新增 3 重槽位校验；预警/费率/抢购由"加仓"变为"报错" |
| 平仓**触发条件（止盈/止损阈值）** | ✅ 一致 | `resolveTradeROIThresholds` + 盈亏分支逻辑未改 |
| 平仓**触发条件（风向反转强平）** | ❌ **行为变化** | `autoStopNeutralROI` 修复恒真守卫 → line3~7 的 `AutoStopOrder` 从"永不触发"变为 `|ROI| ≥ 3%` 即参与判定（§7.2） |
| 平仓**触发条件（custom 无平仓策略兜底）** | ❌ **行为变化** | `simpleCloseStrategy` 同一守卫修复 → 5m 反转平仓从"永不触发"变为生效（§7.2） |
| 平仓**执行** | ⚠️ 有意变更 | 全部走 ownership；平仓前必查账户真实数量，`close_qty = min(managed_qty, account_qty)` |
| 平仓**作用范围** | ⚠️ 有意变更 | 由"全账户仓位"改为"managed 仓位"；当前覆盖 `auto_strategy` + `new_coin_rush` + `funding_rate`（非 eat_rate）；`notice_auto_order` 与 `agent_trade` 走各自保护单/接口 |
| 测试策略→实盘自动切换 | ⚠️ 有意变更 | 不再自动开启实盘，改为通知 + 人工确认 |
| 失败后可恢复性 | ✅ **显著改善** | 上轮 6 项"永久卡死"问题已修复（§5） |

---

## 5. 上一轮问题修复核对（逐项证据）

| 上轮编号 | 问题 | 现状 | 证据 |
| --- | --- | --- | --- |
| E2 | 任何提交失败都置 `reconcile_required` 永久占用槽位 | ✅ 修复 | `deterministicSubmitRejection`（`execution.go:95-109`）区分"确定性拒绝→`OrderFailed` 释放槽位"与"结果不确定（-1000/-1001/-1006/-1007 → `reconcile_required`）"；测试 `TestExecutorMarksDeterministicExchangeRejectionFailedAndReleasesSlot`、`TestExecutorKeepsBinanceUnknownResultFailClosed` |
| E3 | 非 auto_strategy owner 无周期对账 | ✅ 修复 | `main.go:371-380` 每 1 分钟 `ReconcileAll` + `RepairNoticeAutoOrderProtections` |
| E4 | 保护单失败后无法重建 | ✅ 修复 | `nextProtectionClientOrderID`（`lifecycle.go:90-117`）：递增 attempt ID；数量变化时先撤旧单再新建；测试 `...ProtectionCanRetryAfterTerminalFailure`、`...ResizesProtectionAfterManagedQuantityShrinks` |
| E5 | 部分成交后无法继续平仓 | ✅ 修复 | `nextCloseClientOrderID`（`lifecycle.go:192-218`）：先 Reconcile，仅终态才换新 attempt ID，仍活动则拒绝；测试 `...CloseCanRetryRemainingQuantityAfterTerminalPartialFill` |
| E6 | 预警保护单静默跳过 | ✅ 缓解 | `RepairNoticeAutoOrderProtections`（`feature/ownership.go:240-314`）每 1 分钟补齐缺失 TP/SL、清理数量不符的旧单 |
| E8 | 重复平仓窗口 | ✅ 修复 | `ClaimOrder` 对 `IntentClose` 增加 live close 单校验（`service.go:119-129`）；测试 `TestClaimOrderRejectsSecondLiveCloseButAllowsRetryAfterTerminal` |
| 旁路下单 | 旧直连原语仍保留 | ✅ 彻底移除 | 6 个直连函数已删除；全仓检索无 futures 直连下单（仅 `spot/` 与测试网 E2E 例外） |
| E1 | 升级不认领历史仓位 | ⚠️ 有意设计 | 实施报告原文"**系统升级不会根据账户现状或旧 `order` 表自动认领历史仓位**"，并补 `TestReconcileDoesNotClaimUnmanagedAccountPosition`、`TestUnknownAccountPositionIsNeverClaimed` |

---

## 6. 新增能力复核（本轮新引入）

1. **Algo Order API 分流**（`execution.go:253-316`）：`STOP/TAKE_PROFIT/STOP_MARKET/TAKE_PROFIT_MARKET/TRAILING_STOP_MARKET` 走 `CreateOwnedAlgoOrder`，其余走普通 Order API；查询/撤销按 `OrderType` 分流；`exchangeFromAlgoLookup` 会通过 `ActualOrderId` 反查真实订单以取得真实成交状态。Go SDK 的 `AlgoOrderType` 常量与所用字符串完全一致（已核对模块源码）。
2. **方向与数量语义校验**（`execution.go:60-93`）：`IntentOpen + LONG ⇒ BUY`、`+ SHORT ⇒ SELL`，平仓/保护相反；数量必须有限且 > 1e-12。已核对全部 7 个调用点均符合。
3. **错误类型判定**：已确认 go-binance v2.8.12 的 `callAPI` 返回 `*common.APIError`（`client.go:397 return nil, &res.Header, apiErr`），`errors.As` 可正确命中，`deterministicSubmitRejection` 的分类是有效的（非"看起来像但打不中"）。
4. **浮点序列化修复**（`index.go formatOwnedOrderDecimal`）：修复 `0.0024000000000000002` 与 trigger price 二进制尾数导致的 `-1111 Precision is over the maximum defined for this asset`（测试网实测问题，已由 `index_order_format_test.go` 固化）。
5. **孤儿保护单清理**（`reconcile.go:100-123`）：仓位已消失时自动撤销其 TP/SL/Close 挂单，覆盖"保护单成交后 sibling 残留"。
6. **testnet 支持**：`futures.UseTestnet` + 自定义 base URL；启动对账与 ws 用户数据统一由 `futuresTradingConfigured()`（`main.go:244-249`）门控，避免无 key 时误连。

---

## 7. 必须知情的两处"平仓/开仓语义变化"（P1）

### 7.1 P1-A：市场宽度开关被修正，且**放宽**了做多限制

`feature/strategy/line/common.go:155-175`（新）对照 master：

| 规则 | master 行为（整数除法 `riseCount/len`） | 现行为 | 变化 |
| --- | --- | --- | --- |
| 上涨占比 > 75% 禁做空 | 恒为 0/1 → **几乎永不触发** | `risePercent >= 75` 禁做空 | **新增限制** |
| 下跌占比 > 75% 禁做多 | 恒不触发 | `fallPercent >= 75` 禁做多 | **新增限制** |
| 上涨 ≥60% 且 BTC > 5% 禁做空 | 恒不触发（`>60` 不可达） | 生效 | **新增限制** |
| 下跌 < 60% 且 BTC < -5% 禁做多 | `0/1 < 60` 恒真 → **只要 BTC 跌超 5% 就禁做多** | 需 `fallPercent >= 60` | ⚠️ **放宽**：BTC 大跌但下跌币种不足 60% 时，如今**允许做多** |
| 币列表为空 | `riseCount/len(coins)` → **整数除零 panic 风险** | `total <= 0 → 双向禁止` | 修复 panic |

- 该行为已由 `TestBaseMarketBreadthPermissionsUsesPercentages` 固化（含 `missing universe` 用例）。
- 注释仍写"判断当前所有币种涨跌数量是否 **80%**"，与实际 75% 不一致，建议更正注释。
- **需人工确认**：第 4 条由"BTC 跌超 5% 即禁多"变为"需下跌币种 ≥60%"，属于**放宽风控**，是否符合预期需明确。

### 7.2 P1-B：两条此前**永不触发**的平仓路径被激活

`feature/strategy/line/common.go:151-153`：

```go
func autoStopNeutralROI(nowProfit float64) bool { return nowProfit > -3 && nowProfit < 3 }
```

master 对应处为 `if closeParams.NowProfit < 3 || closeParams.NowProfit > -3 { return }` —— 该表达式对任意实数**恒为真**（`x<3 || x>-3` 覆盖全体实数），因此：

| 路径 | master | 现在 |
| --- | --- | --- |
| `AutoStopOrder`（line3/4/5/6/7，"达到止盈止损前风向反转强平"） | 守卫恒真 → **立即 return false，永不强平** | `|ROI| ≥ 3%` 时执行 `MarketReversal`（日线 KDJ/MA 组合）判定并强平 |
| `TradeLineCustom.simpleCloseStrategy`（custom 未配置平仓策略时的兜底） | 同一恒真守卫 → **永不触发** | `|ROI| ≥ 3%` 时按 5m 收盘价反转判定平仓 |

- `NowProfit` 单位为**百分比**（`utils.FuturesLeveragedROI` 末尾 `*100`），故阈值语义为 **±3%**，与其余分支（`resolveTradeROIThresholds`）单位一致。
- 已由 `TestAutoStopNeutralROI` 固化（`-3/-10/3/10` 放行、`±2.999/0` 中立）。
- **这是"平仓逻辑与以前不一致"的主要来源，且属于对 master bug 的修复**。上线前必须确认：这是否是期望的交易行为（会新增强制平仓信号，尤其在日线反转时）。若不希望启用，应通过配置显式关闭，而不是依赖旧 bug。
- 复核完整性：全部 6 个调用点（line3/4/5/6/7 + line_custom）均已替换，无残留旧式守卫。

---

## 8. 新发现的边界问题

### P2-1 保护单**部分成交**后不再重新保护

`service/agenttrade/lifecycle.go:101-103`：`nextProtectionClientOrderID` 遇到该 intent 已有 `OrderFilled` 订单时**直接复用其 clientOrderID**。若止损被**部分**成交（managed qty 已按成交量减少但仓位仍 active），下一次 `EnsureProtection` 会复用已终结的订单：`Execute` → `ClaimOrder` 命中既有行 → 非 pending → `Reconcile` → 返回 FILLED → **`EnsureProtection` 返回成功**，但市场里**没有**新的止损单，残余仓位实际裸奔，DB/UI 却显示已保护。
建议：`OrderFilled` 且 `position.ManagedQty > 0` 时按"新 attempt"处理（或明确返回需人工介入的错误）。

### P2-2 `-1008`（服务端过载）被当作"确定性拒绝"

`execution.go:103-108` 仅把 `-1000/-1001/-1006/-1007` 视为不确定，其余带 code 的错误一律置 `failed` 并允许换新 clientOrderId 重试。`-1008 Server is currently overloaded` 属边界情况：若该请求实际已被受理，重试会产生**重复订单**。建议与 Binance 官方说明核对后，把 `-1008` 归入不确定集合（`-1003/-1015` 归入 failed 是合理的）。

### P3-1 `formatOwnedOrderDecimal` 的 8 位小数下限未与校验对齐

`formatOwnedOrderDecimal` 以 `'f', 8` 四舍五入，`|x| < 5e-9` 会输出 `"0"`；而 `validateOrderRequest` 只要求 `> 1e-12`，因此 1e-12~5e-9 的数量可通过校验后被格式化为 `"0"` 发给交易所（表现为 `-1111/-4003` 报错，被归为确定性拒绝，不会静默成交）。当前 Binance 最小 tick 为 1e-8，属理论风险。建议把数量下限校验对齐到 1e-8，或在格式化为 0 时提前返回明确错误。

### P3-2 未使用/未赋值的状态

`PositionReconcileRequired` 被 `activePositionStatus`（`service.go:70`）引用，但全仓**没有任何代码把它赋给仓位**；`SuspendPosition`（`service.go:399`）亦无调用方。属死代码/死状态，建议清理或补齐用途。

### P3-3 agent proposal 状态不随周期对账自动同步

止损/止盈自行成交时，1 分钟对账会关闭 managed 仓位并撤销 sibling 保护单，但 `AgentTradeProposal.Status` 仍停留在 `executed`，直到人工 `/execute`、`/reconcile` 或点击"关闭此系统仓位"。建议在周期对账中同步一次 proposal 状态（或在 UI 明确提示）。

### P3-4 周期对账循环未按交易配置门控

`main.go:371-380` 的 1 分钟循环未加 `futuresTradingConfigured()` 判断（启动对账已加）。DB 无 managed 行时不会发起网络请求，但若存在历史 managed 行而 key 未配置/失效，会每 60 秒记录一次 Warning。建议与启动路径保持一致。

### P3-5 权重与单实例前提（需运维知情）

- REST 权重上升：2 秒 tick 内对**每个 live 托管单**执行一次 `GetOrder/GetAlgoOrder`（`feature/ownership.go:87-91`）；存在 live 单或 `wsFuturesUserData=1` 时每 tick 一次**全账户** `GetPosition`（`:97`、`ownershipAccountQuantities`）；另有 60 秒一次 `ReconcileAll`。`main.go:369` 的注释仍是旧预算（"1min 中不能超过 2400 权重"），建议重新核算并更新注释。
- **单实例前提**：`mutationMu`（`service.go:18`）是**进程内**互斥；跨进程/多实例无保护。实施报告 §9 已声明该前提，我复核代码确认无 DB 级锁（`client_order_id` 唯一索引只能防重复插入，不能防"校验-插入"竞态）。

---

## 9. 上线前运维要求（非代码缺陷）

1. **既有仓位不会被自动接管**（有意设计）：升级前必须先人工核对 `futures_positions` / 交易所持仓与挂单；既有持仓与旧挂单在升级后**不会被本系统平仓/撤单**，也不会计入 managed。需要人工平仓，或按业务决定是否补一次性接管脚本（当前无此脚本）。
2. **必须执行 `./go_binance_futures sync db`**：`main.go:39 dbVersion = 12`，旧库启动会 `panic("database version N is older than required version 12; run 'sync db' first")`。
3. **确认 testnet 开关**：`binance::testnet = true` 时全部 futures 客户端切到测试网（含 `futures.UseTestnet` 全局）；上生产务必确认为 `false`。
4. **单实例部署**：如前述，多实例会破坏 ownership 槽位互斥。

---

## 10. 建议（review-only，未改代码）

按优先级：

1. 明确 §7.1 第 4 条"放宽做多限制"与 §7.2"激活两条平仓路径"是否符合预期（**这是本次唯一真正改变交易结果的点**），并在文档/变更说明中显式记录。
2. 修复 P2-1（保护单部分成交后重新保护）与 P2-2（`-1008` 归类）。
3. 上线前执行 §9 的 4 项运维前置；在实施报告里补一节"升级步骤与既有仓位处置"。
4. 清理 P3-2 死状态；补 P3-3 proposal 状态同步；统一 P3-4 门控；更新 `main.go` 权重预算注释。
5. 建议补一条**策略层等价性回归测试**：对 line3~7 的 `AutoStopOrder`，用固定 `NowProfit` 与 mock K 线断言"master 恒 false / 新逻辑按预期 true"，避免将来再出现"守卫写反导致路径静默失效"。

---

## 11. 未验证项

- 未连接真实 Binance 账户复跑（本轮为静态审查 + 单测；测试网 E2E 结果引用 `TESTNET_TRADE_E2E.md` 的既有实测记录）。
- 前端独立仓库 `go_binance_futrues_new_ui` 对本轮新增接口（ownership / close / 展示文案）的适配未审。
- Binance 侧细节未实测：Algo Order 部分成交后的 `ActualOrderId` 行为、hedge 模式下超量平仓的返回、`-1008` 的官方归类。
- `main.go` 既有 `unreachable code` vet 噪声未复查（非本次引入，且不在本次更改范围内）。
