# 实盘开仓/平仓逻辑一致性审查（master → 当前分支）

- **对比基线**：`master` = `25f7893`（Merge PR #50，已含 V3-4）；`HEAD` = `abb0525`（Merge branch 'feat/trade' into feat/ai-agent-v3），工作区与 HEAD 一致
- **差异规模**：36 个文件 `+3488 / -414`；新增 12 个文件（`feature/ownership.go`、`models/futures_managed.go`、`service/futuresownership/*`、`service/agenttrade/lifecycle.go`、`service/agenttrade/notifier.go`）
- **重要前提**：`service/backtest/**`（V3-4 自适应回测）**不在差异内** —— master 已包含 V3-4。本次差异**只有 V3-5「合约订单归属（ownership）与交易安全」**，即新增 `service/futuresownership` 包，并把所有**合约**下单/撤单改道托管层
- **审查类型**：**review-only** —— 仅静态对照 + 运行构建/测试，未修改任何代码
- **审查日期**：2026-09-12

---

## 1. 结论（先说重点）

1. **交易决策逻辑 100% 一致。** 判断「何时开、开多还是开空、何时平、平多少比例」的代码**完全没有改动**：`feature/strategy/**`（`line1~line7`、`line_custom`）、`AutoStopOrder`、`GetCanLongOrShort`、ROI 阈值解析、KC/均线等指标计算、数量与价格计算公式、精度处理、`SetMarginType/SetLeverage` 顺序、平仓通知模板、`insertOpenOrder/insertCloseOrder` 记账 —— 逐项核对无差异。
2. **执行逻辑全部替换，且不是等价替换。** 所有合约下单从「直接调用 Binance」改为「先写 ownership 表 → 带 `clientOrderId` 下单 → 按成交回写托管数量」。同时新增**三重槽位校验**，因此存在明确的**行为变更**，而不是重构。
3. **存在 4 类会导致「永久性卡死 / 静默失效」的新报错路径，以及 1 个升级兼容性缺口。** 其中最重要的是：**升级后既有仓位不会被自动平仓**（无 ownership 记录，且无任何回填逻辑）。
4. **自动化验证全部通过**：`go build ./...` ✅、`go vet`（feature / futuresownership / agenttrade）✅、`go test -count=1 ./...` ✅ 56 包 ok / 0 FAIL、`git diff --check` ✅。

> 一句话回答用户问题：**「什么时候开仓/平仓」和以前完全一样；「开仓/平仓怎么发出去、发出后怎么记账」完全换了实现，其中一部分场景以前能成功、现在会直接报错或永久失效。**

---

## 2. 执行链路变更总览

| 业务入口 | master 的下单方式 | 当前分支的下单方式 | 归属 Owner |
| --- | --- | --- | --- |
| 自动策略开仓（市价/限价） | `binance.BuyMarket / SellMarket / BuyLimit / SellLimit` | `submitAutoStrategyOpen` → `Executor.Execute` | `auto_strategy` |
| 自动策略平仓（策略触发/止盈止损） | `binance.SellMarket / BuyMarket` | `submitAutoStrategyClose` → `Executor.Execute` | `auto_strategy` |
| 预警币自动下单（开仓） | `binance.BuyMarket / SellMarket` | `submitNoticeAutoOpen` → `Executor.Execute` | `notice_auto_order` |
| 预警币止盈/止损挂单 | `OrderTakeProfit / OrderStopLoss`（**`ClosePosition(true)` 全平触发单**） | `submitNoticeProtection`（**定量 `STOP_MARKET` / `TAKE_PROFIT_MARKET`**） | `notice_auto_order` |
| 资金费率自动下单 | `binance.BuyMarket / SellMarket` | `submitFundingRateOpen` → `Executor.Execute` | `funding_rate` |
| 新币抢购 | `binance.BuyMarket / SellMarket / BuyLimit / SellLimit` | `submitNewCoinRushOpen` → `Executor.Execute` | `new_coin_rush` |
| AI 受控交易（V2-12） | `binance.CreateAgentMarketOrder` | `Executor.Execute`（+ 新增保护单/平仓生命周期） | `agent_trade` |
| 超时撤单 | 遍历**交易所全部 NEW 挂单**并撤单 + 删本地 `order` 行 | 只遍历 **ownership 表中的托管开仓单** | — |
| 测试策略达标 | 自动置 `FutureEnable=1 / AllowLong=1 / AllowShort=1 / FutureTest=0` | **不再自动开启实盘**，仅发通知要求人工确认 | — |

**统一下单原语**：`feature/api/binance/index.go:512 CreateOwnedOrder` —— 与 master 的 `BuyMarket/SellMarket/BuyLimit/SellLimit` 相比，新增 `ClientOrderID`（幂等键）、`StopPrice`；`Limit` 单仍为 `TimeInForce=GTC`，`Market` 单不带价格。旧的直连函数**仍保留在文件里**（`index.go:411/431/451/470/489`），只是不再被合约链路调用（存在被误用回去的可能）。

**下单前必须先落库**（`execution.go:67 ClaimOrder`）：先写 `futures_managed_orders`（`client_order_id` 唯一索引，`models/futures_managed.go:33`），再提交交易所；成交后 `ApplyFill` 回写 `futures_managed_positions.ManagedQty`。

---

## 3. 开仓逻辑对照

### 3.1 与 master 完全一致的部分（已逐行核对）

- 开仓条件：`coin_line_strategy.GetCanLongOrShort(strategy.OpenParams{Symbols: coin})`（策略包零改动）。
- 数量公式：`quantity = (Usdt / buyPrice) * Leverage`，再 `GetTradePrecision(quantity, StepSize)`；价格精度 `GetTradePrecision(price, TickSize)`。
- 开仓前跳过已有持仓/挂单的判断（`feature.go:411-441`，基于账户快照）。
- 杠杆与保证金设置：`UpdateSymbolTradeInfo` → `SetMarginType` → `SetLeverage`（调用顺序与位置未变）。
- 成功后写 `order` 表、发开仓通知（`Status: success/fail`）的字段与文案未变。

### 3.2 新增的三重校验（行为变更点）

| 校验 | 位置 | 触发后的行为 |
| --- | --- | --- |
| ① 账户槽位校验（仅 feature 系入口） | `feature/ownership.go:147-170`，在 `submitOwnedFeatureOpen:208` 调用 | 账户上同 `symbol+positionSide` **已有持仓**或**已有开仓方向挂单** → **直接拒绝开仓并返回错误**（master 会继续下单、与既有仓位合并） |
| ② 托管槽位校验 | `service/futuresownership/service.go:140-163`，`ClaimOrder:115` 调用 | DB 中同 `symbol+positionSide` 已有 **active 托管仓位** 或 **live 托管开仓单**（**不限 owner，跨 owner 互相阻塞**）→ 拒绝 |
| ③ 市价成交确认 | `service/futuresownership/execution.go:91-101` | 市价单提交成功但 `executedQty` 未确认 → 再查一次 → 仍为 0 → 报错并置 `reconcile_required` |

**影响面**：
- 自动策略主流程**基本无感知**：`feature.go:443` 本来就要求 `hasPositionLong == false && hasBuyOrderLong == false`，与校验①等价；差别仅在于校验①是「提交前再查一次真实账户」，能挡住快照过期。
- **预警币 / 资金费率 / 新币抢购有实际差异**：master 会加仓（合并持仓），现在会**报错**。预警币会推送「失败」通知；资金费率与抢购只写 `logs.Error`。
- **新币抢购路径变慢**：每次下单前多 2 次账户读取（`GetTransformPositions` + `getTransformOpenOrders`，`feature/ownership.go:147-170`）。开启 `wsFuturesUserData` 时读本地表，否则走 REST —— 抢购是时间敏感路径，属性能回退。

---

## 4. 平仓逻辑对照

### 4.1 与 master 完全一致的部分

- 触发条件：`AutoStopOrder(CloseParams{Symbols, Position, NowProfit})`（策略包零改动），以及止损/止盈/趋势反转的三段判断结构（`feature.go:127/205/282/…`）。
- 平仓方向与市价单类型：`LONG → SELL`、`SHORT → BUY`，均为市价。
- `insertCloseOrder`、平仓通知字段与文案未变。

### 4.2 行为变更点

| # | 变更 | 位置 | 影响 |
| --- | --- | --- | --- |
| B1 | **自动平仓对象从「全账户持仓」缩小为「仅 auto_strategy 托管持仓」** | `feature.go:80-84` 先 `syncAutoStrategyOwnership`，`feature.go:94` 循环 `managedPositions` | 手工开的仓、预警币仓、资金费率仓、AI 仓**不再被自动策略平掉**（master 会平）。这是 V3-5 的设计目标，但对既有仓位是重大行为变化，见 §5.1 |
| B2 | 平仓数量从「账户持仓量」改为「`min(托管量, 账户量)`」 | `feature/ownership.go:214-232`、`service.go:364-373` | 手工加仓的部分不会被机器人平掉（符合设计） |
| B3 | 预警币止盈/止损从 `ClosePosition(true)` 改为**定量触发单** | `feature/feature_notice.go:128-140/175-187` | ① 数量固定为挂单时的托管量，后续仓位变化不会自动调整；② 若托管数量取不到，**只写日志、不挂任何保护单**（见 E6） |
| B4 | 超时撤单只处理托管开仓单 | `feature/ownership.go:108-139`（`feature.go:89` 调用） | 手工挂单不再被机器人撤销（更安全）；但 master 时代遗留的挂单也不会再被撤（见 §5.1） |
| B5 | 风控计数语义变化 | `feature.go:93` `accountTradeRiskCounts(positions)`（全账户），`feature.go:368-377` 使用 | master 只统计「非白名单 + 未平仓」的仓位；现在白名单持仓也计入 `positionCount`，亏损计数覆盖全账户 → **更容易触达 `FutureMaxCount`/`LossMaxCount` 而停止开新仓** |
| B6 | 测试策略达标后不再自动切实盘 | `feature/feature_test_strategy.go:486-493` | master 会自动 `FutureEnable=1/AllowLong=1/AllowShort=1/FutureTest=0`；现在只发站内通知并重置计数，且不再清理 `test_strategy_results` → 需人工确认，测试数据持续累积 |
| B7 | AI 交易新增保护单/平仓生命周期 | `service/agenttrade/lifecycle.go`、`service.go:400-470` | 新增 `EnsureProtection`（止损必挂、止盈最多 1 个）与 `Close`；新增状态 `protection_failed` / `closed` |

---

## 5. 会产生报错 / 永久失效的场景（核心）

### E1（P1）升级后**既有仓位失去自动平仓能力**，且系统没有回填机制

- 事实：托管仓位**唯一创建点**是 `service/futuresownership/service.go:240`（某笔 `intent=open` 的托管单成交时）。
- 事实：全仓检索无任何 backfill / 迁移 / 接管逻辑；`ReconcilePosition`（`service.go:343`）只更新**已存在**的托管行。
- 后果：升级瞬间账户里已存在的仓位（包括 master 自动策略开的仓）**没有托管行** →
  1. `feature.go:94` 的平仓循环看不到它们 → **再也不会被自动止损/止盈/策略平仓**；
  2. master 时代遗留的超时挂单也不再被 `cancelTimeoutOrder` 撤销；
  3. 它们仍会计入 `positionCount`（全账户统计），继续占用开仓名额。
- 恢复方式：人工在网页/交易所平掉，或人工补写 `futures_managed_positions` 行。
- 建议：上线前先人工核对并清理既有仓位/挂单，或提供一个一次性「接管现有仓位」操作。

### E2（P1）任何一次下单失败都会**永久占用槽位**，阻塞该 `symbol+positionSide` 的所有 owner

调用链：`Executor.Execute`（`execution.go:78-86`）

```go
result, submitErr := e.Broker.Submit(ctx, request, clientID)
if submitErr != nil {
    lookup, lookupErr := e.Broker.Lookup(ctx, managed.Symbol, clientID)
    if lookupErr == nil && ... { return e.applyExchange(...) }
    _ = e.Ownership.SetOrderStatus(ctx, clientID, OrderReconcile)   // ← 行 84
    return ExchangeOrder{}, fmt.Errorf("managed order submission is uncertain; ...")
}
```

- `OrderReconcile` 属于 `liveOrderStatus`（`service.go:73-75`）→ `ensureSlotAvailable`（`service.go:159`）会一直报 `"... already has a live managed open order owned by ..."`，且**不区分 owner**。
- 自愈路径只有 `Reconcile`（`execution.go:105-115`）用 `origClientOrderId` 反查：若交易所**根本没有这笔单**（例如被 `-2013 Order does not exist`、或下单被明确拒绝 `-1013/-2010/-2027` 等），反查永远失败 → 状态被再次置回 `reconcile_required` → **永不释放**。
- 关键缺陷：代码把**「明确的拒绝」和「不确定的网络异常」当成同一类**处理。master 的行为是「这一次失败，下一次 tick 继续按策略重试」，现在变成「该币种该方向永久无法开仓，直到人工改库或重启」。
- 恢复方式：重启进程（`main.go:235` 启动时 `ReconcileAll`）或调用 `POST /agents/trade/ownership/reconcile` —— 但对「交易所从未受理」的单，这两条路径同样失败，只能人工删/改 `futures_managed_orders` 行。
- 建议：区分错误类型 —— 确定性 API 拒绝 → 直接置 `failed`（终态，释放槽位）；只有超时/EOF/5xx → 才置 `reconcile_required`。并为 `reconcile_required` 增加 TTL / 最大重试。

### E3（P1）`notice_auto_order` / `funding_rate` / `new_coin_rush` **没有周期性对账**，保护性平仓后永久拒绝再次开仓

- 事实：2 秒循环里只有 `syncAutoStrategyOwnership`（`feature.go:80-84`）对账，且写死 `OwnerAutoStrategy`（`feature/ownership.go:27/37`）。其余 owner 只依赖**进程启动时**的 `ReconcileAll`（`main.go:235`）与**手动 API**。
- 后果链：预警币仓位由交易所侧止损/止盈触发成交 → 没有任何代码调用 `ApplyFill` → `futures_managed_positions` 仍是 `active` 且 `ManagedQty > 0` → 下次同币种触发预警时，`ClaimOrder` 报 `"... already has an active managed position owned by notice_auto_order"`（`service.go:147`），**该币种自动下单永久失效**，直到重启或手动对账。
- 同理适用于手工平仓、强平、资金费率仓、抢购仓。

### E4（P2，设计取舍）AI 交易保护单失败后**无法重建**，仓位可能长期无止损

- `lifecycleClientOrderID`（`lifecycle.go:195-201`）对同一 proposal 生成**确定性** clientOrderId（`agt_sl_/agt_tp_/agt_close_<proposalID>`）。
- `ClaimOrder`（`service.go:106-108`）遇到同 `(owner, symbol, side, intent)` 的既有行时直接返回该行；随后 `Execute` 走 `Reconcile` 分支 —— 由于 clientOrderId 被那个失败行永久占用，**重试永远不会真正下新单**。
- `TestOwnershipLifecycleStopFailureNeverResubmits`（`lifecycle_test.go:109-122`）**明确固化了「止损单只提交一次」**，说明这是有意设计（fail-closed）。但运维后果是：一次瞬时网络故障即可让 AI 仓位**永久缺失止损**，且状态停在 `protection_failed`，只有 `webnotification` 一条通知（未接钉钉）。
- 建议：让确定性 ID 带序号（`agt_sl_<pid>_1/_2`），或在既有行为终态（`failed`/`canceled`）时允许换新 ID；并把该通知接入主告警通道。

### E5（P2）AI 交易**部分成交/过期后无法继续平仓**

- `Close`（`lifecycle.go:95-143`）使用固定的 `agt_close_<pid>`。
- 若首次平仓单被交易所判为 `EXPIRED`（市价单部分成交后剩余作废）或 `PARTIALLY_FILLED` 后终结，`applyExchange` 把行置为 `canceled`/`filled`；重试时 `ClaimOrder` 命中既有终态行 → `Execute` 走 `Reconcile` → 返回**旧成交结果**，不会下新单 → `Close` 读到 `remaining.ManagedQty > 0` → 报 `"managed close is only partially filled; X remains"`。
- 结果：**剩余仓位通过该接口永远平不掉**（只能人工处理或依靠止损单）。建议同上：终态允许新 clientOrderId。

### E6（P2）预警币保护单**静默跳过**

`feature/feature_notice.go:128-140`（LONG）/`175-187`（SHORT）：`noticeManagedQuantity` 取不到托管数量时只 `logs.Error` 并**跳过挂单**，既不重试也不发通知。触发条件为 ownership 数据与交易所不一致（例如 E3 场景下托管数量已归零/缺失）→ **仓位裸奔无止损**。

### E7（P3）数据库版本

`main.go:39 dbVersion = 12`；`main.go:220-221` 若 `systemConfig.Version < 12` 直接 **panic**：`database version 11 is older than required version 12; run 'go_binance_futures sync db' first`。必须执行 `sync db`（`command/db_update_test.go` 已覆盖 v12 建表与幂等）。

### E8（P3）重复平仓窗口（非回归）

`service.go:114-126` 的 `ClaimOrder` 对 `intent=close` 只校验数量，不校验是否已有 live 平仓单；若首笔平仓单在 2 秒 tick 内尚未确认成交（`ManagedQty` 未递减），下一 tick 会用**新的随机 clientOrderId** 再发一笔平仓单。master 同样存在该风险（每 tick 都可能再发市价平仓），故非本次回归 —— 但 ownership 层本可顺手加上 live close 单校验。

### E9（P3）`Execute` 在 `applyExchange` 出错时丢弃已受理订单信息

`execution.go:87-90` 返回 `(applied, err)`，而 `feature/ownership.go:287-289` 只取 `err` → 上层认为失败、不写 `order` 表、不发成功通知；但交易所可能已成交（`MarkOrderSubmitted/ApplyFill` 已在出错前执行）。auto_strategy 能靠周期对账自愈，其余 owner 不能。

### E10（P3）REST 权重上升

新增的周期性调用（2 秒 tick）：对**每一个** live 托管单执行一次 `GetOrder`（`syncAutoStrategyOwnership` → `Executor.Reconcile`），以及在 `wsFuturesUserData == "1"` 或有 live 单时每 tick 一次**全账户** `GetPosition`（`feature/ownership.go:251-263`）。master 只有 30 分钟一次的 `UpdateOrderStatus` 兜底。`main.go:359` 注释的 2400 权重/分钟预算需要重新评估。

---

## 6. 「是否与以前一致」逐项判定

| 维度 | 判定 | 说明 |
| --- | --- | --- |
| 开仓/平仓的**触发条件与方向** | ✅ 完全一致 | `feature/strategy/**`、`AutoStopOrder`、`GetCanLongOrShort` 均未改动 |
| 数量/价格**计算公式与精度** | ✅ 完全一致 | 公式与 `GetTradePrecision` 调用未变 |
| 杠杆/保证金模式设置 | ✅ 完全一致 | 调用位置与顺序未变 |
| 本地 `order` 表记账与通知文案 | ✅ 基本一致 | 字段与文案未变，但失败分支的触发条件变多（见 §3.2） |
| 开仓**是否会被执行** | ⚠️ 有意变更 | 新增三重槽位校验；预警币/资金费率/抢购由「加仓」变为「报错」 |
| 平仓**作用于哪些仓位** | ⚠️ 有意变更 | 由「全账户」缩小为「仅 auto_strategy 托管仓位」 |
| 预警币止盈止损挂单形式 | ⚠️ 有意变更 | `ClosePosition(true)` → 定量 `STOP_MARKET/TAKE_PROFIT_MARKET` |
| 超时撤单范围 | ⚠️ 有意变更 | 由「交易所全部挂单」改为「仅托管挂单」 |
| 测试策略→实盘自动切换 | ⚠️ 有意变更 | 取消自动开启，改为人工确认 |
| 风控计数语义 | ⚠️ 有意变更 | 白名单持仓也计入，更早停止开仓 |
| **失败后的可恢复性** | ❌ 明显劣化 | 见 E1/E2/E3/E4/E5/E6 —— 多处由「下次重试」变为「永久失效，需人工介入」 |

---

## 7. 自动化验证结果

| 项目 | 结果 |
| --- | --- |
| `go build ./...` | ✅ PASS |
| `go vet ./feature/... ./service/futuresownership/... ./service/agenttrade/...` | ✅ 无输出 |
| `go test -count=1 ./...` | ✅ **56 包 ok，0 FAIL** |
| `go test -count=1 ./service/{futuresownership,agenttrade}/... ./feature/...` | ✅ 全绿 |
| `git diff --check master -- '*.go'` | ✅ 无空白错误 |

测试覆盖情况：`service/futuresownership` 有 4 个测试文件（含 `TestExecutorLeavesUnknownResultInReconcileRequired`、`TestExecutorRequiresConfirmedMarketFill`、`TestSparseTradeRangeCached...`），`service/agenttrade` 新增 `lifecycle_test.go` 5 个用例，`command/db_update_test.go` 覆盖 v12 建表与幂等。**但没有任何测试覆盖 E1/E2/E3/E5 的「卡死」语义**（例如「失败的 open 单不再阻塞后续开仓」）。

---

## 8. 建议（仅建议，未改代码）

按优先级：

1. **上线前**：人工核对账户既有持仓与挂单；提供一次性「接管既有仓位」的迁移/脚本（否则 E1 会直接让既有仓位失去自动平仓）。
2. **错误分类**（解 E2）：在 `Executor.Execute` 中区分确定性 API 拒绝与传输层异常；前者置 `failed` 释放槽位，后者才置 `reconcile_required`。
3. **补周期对账**（解 E3）：把 `ReconcileAll`（或至少 4 个 feature owner）纳入定时执行；或为 `reconcile_required` / 长期 active 托管仓位增加 TTL 与自动降级。
4. **解除确定性 ID 的死锁**（解 E4/E5）：终态行允许新 clientOrderId，或 ID 追加递增序号。
5. **保护单失败必须告警**（解 E4/E6）：接入钉钉/主通知通道，并明确「无止损持仓」的运维处置流程。
6. **补测试**：为「下单失败后不阻塞后续开仓」「保护单失败可重建」「部分成交后可继续平仓」补回归测试。
7. **权重评估**（解 E10）：重新核算 2 秒 tick 下 `GetOrder × live 单数 + GetPosition` 的总权重，必要时降频或按 owner 分级。

---

## 9. 未验证项

- **未连接真实 Binance 账户做实测**（无网络/密钥）：本报告全部结论来自静态对照 + 单元测试，属「代码层面可判定」的范围。建议在测试网或小额账户做一次升级演练，**必测场景**：升级时账户已有仓位 + 已有挂单。
- 前端独立仓库 `go_binance_futrues_new_ui` 对新增 `/agents/trade/ownership(/reconcile)`、`/proposals/:id/close` 的适配未审（本仓仅 `static/` 构建产物）。
- 未核对 Binance 侧行为细节：hedge 模式下 `STOP_MARKET + quantity` 与超量平仓的实际返回、`GetOrder(origClientOrderId)` 的查询保留窗口。
- `main.go` 既有 `unreachable code` vet 噪声未在本次范围内复查（非本次引入）。
