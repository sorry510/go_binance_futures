# CODE_REVIEW_06-4：Binance Spot 执行与恢复

日期：2026-11-11
分支 / HEAD：`feat/trade_spot` / `6d8eecc feat: 06-3`（本地 master `7ed6c93`，Schema **v25**）
审计对象：**工作区未提交的 06-4 改动**（HEAD 已提交部分为 06-0～06-3）
依据：`开发方案-06-现货量化交易-复刻合约简化版.md`、`开发方案-06-4-Spot执行与恢复-实施记录.md`、`CODE_REVIEW_06-3.md` / `CODE_REVIEW_06-3-rereview.md`、`CODE_REVIEW_06-2-rereview.md`、`开发方案-06-发布前Schema验证检查单.md`
原报告保留：本报告为 06-4 首轮独立 review，复评须另起 `CODE_REVIEW_06-4-rereview.md`

> **一句话结论**：06-4 的**代码质量、资金安全护栏与离线 Gate 全绿**，是一份合格的研究/开发基线；但存在 **1 个 100% 可复现的业务闭环断裂（SELL 后仓位永不 CLOSED）** 与 **1 个前置 Gate 未闭环（异常订单无恢复入口）**，因此**不认可 06-4 完整验收、不认可方案 06 完成、⛔ 阻塞 06-5 自动信号接线**，生产与实盘 Gate 全部 BLOCKED。

---

## 一、基线与工作区归属

`GIT_OPTIONAL_LOCKS=0 git status --porcelain` 共 8 项，全部属于 06-4：

| 状态 | 路径 |
| --- | --- |
| M | `doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md` |
| M | `doc/trade/开发方案-总览与实施顺序.md` |
| ?? | `doc/trade/开发方案-06-4-Spot执行与恢复-实施记录.md` |
| ?? | `service/spotexecution/`（6 文件，1341 行） |
| ?? | `service/spotledger/lookup.go`（25 行） |
| ?? | `spot/api/binance/spot_quant_adapter.go`（246 行） |
| ?? | `spot/api/binance/spot_quant_adapter_test.go`（44 行） |
| ?? | `spot/api/binance/spot_quant_stream.go`（81 行） |

新增 Go 代码合计 **1819 行**（含测试）。前端仓库 8 项未提交仍为 06-2 及用户既有订单页改动，**06-4 前端零改动**，与实施记录一致。`appversion/version.go:4` 仍为 `25`，`main.go` / `routers/router.go` / `conf/app.conf` 本阶段无改动。

## 二、前置 Gate 独立复核（不采信文档自述）

| 前置项 | 来源 | 独立复核结论 |
| --- | --- | --- |
| 06-3-F1 物化字段不可作为金融权威 | `CODE_REVIEW_06-3-rereview.md` | ✅ 属实。`projectPosition` 从 Fill 重放；06-4 `ExecuteSell:158` / `submitOnce:256` 一律走 `Accounting`，未把 `GetPosition` 当可卖数量 |
| 06-3-F2 终态不可被 Fill 重算覆盖 | 同上 | ✅ 属实。`reconcileRemote:365` 先 `SetExchangeStatus` 再导 myTrades；`reconcileRemote:419` 用 `ReviewPosition` 暴露矛盾 |
| 06-3-F3 外部异常恢复（06-4 硬门槛） | 同上 | ⚠️ **部分闭环**：fail-closed 与错误上报齐备，但「核对该订单后继续」的路径缺失 → 见 **F2** |
| 06-3-F4 全局 + 单币双开关执行门控 | 同上 | ✅ 闭环。`DBGate.Enabled` 双开关每次读 DB，`submitOnce` 在 POST 前再查；`TestStage064DBGateAllSwitchStates` 四态 + 缺币种 fail-closed；`TestStage064GateAndFreeBalanceFailClosed/disabled_just_before_exchange` 证明发送前复查生效 |
| 06-2-F1 发布前 v25 Schema 检查单 | `CODE_REVIEW_06-2-rereview.md` | ⛔ 仍为**发布流程门禁**，本阶段未执行；`main.go.initializeRuntimeDatabase` 依旧只比版本号，生产发布前必须留只读核验证据 |
| 方案 06 06-4 验收行 `:163`「Fake/Testnet 验证单币 BUY/SELL/部分成交/超时去重；关总开关不发新自动单；不误卖手工余额」 | 总方案 | ⚠️ Fake 侧 4 项均 PASS；**Testnet 未执行**；BUY/SELL 的「生命周期闭环」在本实现下不成立 → 见 **F1** |

## 三、隔离副本与等价校验

- 副本：`/tmp/phase_audit_064` = `git archive HEAD | tar -x` + 依 `git status --porcelain -z` 同步全部 8 项 M/?? 文件（含未跟踪目录），删除项 0。
- 等价校验：以 `git ls-files '*.go'` + 未跟踪 Go 文件为清单，**661 个 Go 文件 SHA256 全等**，`MISMATCH=0 / MISSING=0 / EXTRA_IN_COPY=0`。
- 探针只写在副本内，结束随副本删除；所有进程显式 `unset SPOT_STRATEGY_LIVE_TRADING_APPROVAL SPOT_LEDGER_MYSQL_DSN`，Fake 与 SDK 构造探针不联网。

## 四、离线 Gate 实测（全部为本次独立执行，`-count=1`）

| 项目 | 命令 / 范围 | 结果 |
| --- | --- | --- |
| 目标包单测 | `go test -count=1 ./command ./controllers ./models ./service/spotquant ./service/spotledger ./service/spotexecution ./spot/api/binance` | ✅ **7/7 ok** |
| Race | 同 7 包 `-race` | ✅ **7/7 ok**（macOS `ld: malformed LC_DYSYMTAB` 为非致命噪音） |
| 06-4 专项 | `TestStage064*`，`./service/spotexecution` + `./spot/api/binance` | ✅ **12 顶层 / 17 用例全 PASS** |
| 06-3 回归 | `service/spotledger -run 'TestStage063\|TestReview063'` | ✅ PASS |
| vet | 同 7 包 | ✅ EXIT=0 |
| build | `go build -o /tmp/spot_quant_064_build_check .` | ✅ EXIT=0（未覆盖现有可执行文件） |
| diff | `git diff --check` | ✅ EXIT=0 |
| 前端 | `tsc --noEmit`、`vue-tsc --noEmit --skipLibCheck` | ✅ 双双 EXIT=0 |
| 产物 | 前端 `dist` ↔ 后端 `static` SHA256 | ✅ `dist=106 static=106 common=106 identical=106 only_dist=0 only_static=0`；`static` 不含 spotQuant 页面（正式菜单/成套发布属 06-5/06-6） |
| 干净副本全量 | `go test -count=1 ./...` | ❌ EXIT=1，**71 包 ok**；红点为既有归因：`controllers` 缺 `../strategy_templates`（`strategy_template_static_test.go:29`）、`tests` / root 缺 `conf/app.conf`（`panic: target config file ... don't exist`）。**不得写「全仓测试通过」** |

实施记录 `:65` 声称「Race Detector / Spot 执行模块重复运行 5 次 PASS」，本次独立复核为 1 次 race（7/7 绿），未复跑 5 次；不否认其真实性，仅标注未复现。

## 五、静态审查（正向不变量）

1. **零运行时接线**：`spotexecution` 的全仓引用仅在 9 个 06-4 新增文件内；`main.go` / `routers/` / `controllers/` / `loopRun` / `init` **0 处**。无定时器、无信号扫描、无 HTTP 交易路由、无 Schema 变更。
2. **默认只读**：`NewSpotQuantAdapter()` 不授权；`Place` / `Cancel` 首行返回 `ErrDisabled`；`NewSpotQuantAdapterWithClient(c, true)` 仅当 `c.BaseURL == "https://testnet.binance.vision"` 才通过，主网 URL 直接拒绝（永久测试 `TestStage064SpotAdapterCannotTradeByDefault`）。
3. **幂等优先于重试**：`ExecuteBuy/ExecuteSell` 先 `FindOrderByClientID`；`submitOnce` 在任何 POST 前按原 `client_order_id` 查远端；POST 出错/超时只再查一次，绝不二次 POST，返回 `ErrUncertain`；重复 `clientID` 只走 `reconcile`。`TestStage064TimeoutNoDuplicateAndRestartRecovery` 实证 `placeCount=1`。
4. **不误卖手工库存**：SELL 数量来自 06-3 Fill 重放的 `RemainingQty`，并要求 Binance `free >= strategy`；`planSell:136` 在余额不足时报 `ErrNeedsReview`，**不把订单缩小到手工剩余币**；`submitOnce:272-275` 发送前二次校验。`TestStage064WalletShortfallBlocksSELLWithoutManualFallback` 实证。
5. **远端身份与成交一致性**：`checkRemote` 核 symbol / side / type / clientID / origQty；`reconcileRemote` 核 myTrades 身份、累计 Fill 与 `executedQty` 精确相等、`len>=1000` 截断 fail closed、末尾 `ReviewPosition`。`TestStage064ConflictingRemoteAndTradesAreRejected` 5 个子例全 PASS。
6. **WS 归属隔离**：`HandleExecutionReport` 只接受本地已存在 `client_order_id`，手工订单直接忽略；`trade_id` 去重；`TestStage064UserStreamOnlyManagedAndDeduplicated` 实证。
7. **精确十进制**：`parseAmount` 拒绝科学计数/负号/超 18 位，全链路 `decimal.Decimal` 字符串，无 `float64` 参与金额；`prepareQty` 做 step/min/max/MIN_NOTIONAL 预检并注明本地预检非交易所承诺。
8. **Futures 零污染**：测试 fixture 中 `futures_managed_positions.marker='unchanged'` 断言保持。

## 六、探针实证（副本内，Fake/SQLite，不联网）

### 探针 A：BUY → SELL 生命周期是否收敛
```
PROBE064_LIFECYCLE buy_qty=0.99 sell_qty=0.985 remaining=0.005000000000000000 status=OPEN places=2
PROBE064_SECOND_SELL err=Spot quantity violates lot size places=2
PROBE064_NEW_CYCLE err=active Spot strategy position already exists for BTCUSDT places=2
```
方法：复用 `testEngine` 完成一次完整 BUY→SELL，再 `Accounting`，再用**新 clientID** 分别尝试第二次 SELL 与新周期 BUY。

### 探针 B：未确认订单能否恢复
```
PROBE064_UNKNOWN_FIRST err=...ErrUncertain: submit error: timeout after POST; lookup error: <nil> places=1
PROBE064_UNKNOWN_NEW_BUY err=...ErrUncertain: exchange cannot find persisted client id unknown_1 places=1 lookups=3
PROBE064_UNKNOWN_RECOVER err=...ErrNeedsReview: BTCUSDT/unknown_1: ... exchange cannot find persisted client id unknown_1
```
方法：POST 明确失败且交易所无该订单 → 恢复网络后尝试新 BUY → 再跑 `RecoverOutstanding`。

### 探针 C：开关 OFF 时 SELL 的 REST 时序
```
PROBE064_SELL_GATE_OFF err=spot quant execution disabled lookups_delta=1 places=1
```

### 探针 D：WS 端点是否受 Testnet 约束（不联网，仅构造 SDK Client）
```
PROBE064_WS_BEFORE UseTestnet=false UseDemo=false main=wss://ws-api.binance.com:443/ws-api/v3 testnet=wss://ws-api.testnet.binance.vision/ws-api/v3
PROBE064_WS_NEWCLIENT rest_base=https://api.binance.com
PROBE064_WS_AFTER rest_base=https://testnet.binance.vision UseTestnet=false mutation_permitted=true ws_endpoint_would_be=wss://ws-api.binance.com:443/ws-api/v3
```

### 探针 E：FILLED 状态早于 myTrades 明细
```
PROBE064_WS_STATUS_ONLY handled=true err=spot order needs reconciliation: Spot stream requires myTrades reconciliation: order 1 is FILLED but only 0 of 0.99 confirmed in myTrades buy_status=NEW
```

## 七、Findings

| ID | 级别 | 结论 | 证据 | 建议 |
| --- | --- | --- | --- | --- |
| **06-4-F1** | **P1 / 阻塞 06-5** | **SELL 固定预留 0.5% 基础币，导致仓位永不 `CLOSED`、同 symbol 无法开启新周期、盈亏永不结算** | `rules.go:14,141` `sellBaseFeeReserve=0.995` → `qty = strategy*0.995`；`fills.go:321-330` 仅 `remaining.IsZero()` 才 `CLOSED`；`ledger.go::CreatePosition` 拒 active cycle；探针 A | 按 `CommissionAsset` 精确处理手续费后卖出全部策略库存，或明确定义 dust 处置与状态收敛路径；永久测试必须断言 `Status==CLOSED && remaining==0` |
| **06-4-F2** | **P1 / 前置 Gate 未闭环** | **一个从未被交易所接受的 `NEW` 订单，令同 symbol 后续 BUY 永久 `ErrUncertain`，`RecoverOutstanding` 恒失败，无人工/只读确认入口** | 探针 B；`processor.go:346-353` 把 `remote==nil` 统一转 `ErrUncertain`；`adapter.go:161-168` 把权威 -2013 映射为 `nil,nil`；`submitOnce:220-226` 却把同一 `nil` 当作「可 POST」→ **同一权威应答两处语义相反**；`types.go:19 ErrNotFound` 已定义但零引用 | 区分「权威 -2013 不存在」与「网络不确定」；为本地 NEW + 远端权威不存在的订单提供显式只读确认/人工标记路径；补齐核对后可继续的永久测试 |
| **06-4-F3** | **P2 / 阻塞 06-5 接线** | **Testnet 限定只覆盖 REST，用户数据 WS 仍指向主网；若改 `sdk.UseTestnet` 会污染全包端点** | 探针 D；`spot_quant_stream.go:34` `sdk.WsUserDataServeSignature`；SDK `websocket_service.go:1184-1192 getWsApiEndpoint()` 由包级 `UseTestnet/UseDemo` 决定，与 `c.BaseURL` 无关；`client.go:365 getAPIEndpoint()` 同样受其控制；项目 Grep 确认**零处**设置根包 `sdk.UseTestnet`（`feature/api/binance/index.go:165` 的 `futures.UseTestnet` 属不同子包） | WS 端点显式化，或在构造器中校验 `sdk.UseTestnet` 与 REST 目标一致；06-5 接线前补 Testnet 签名 WS 实证；实施记录「只有 Testnet 才允许」需注明「仅 REST」 |
| **06-4-F4** | **P2** | **正常竞态「FILLED 先于 myTrades」会令 WS 立即断开，且 06-4 无自动 REST 对账接线** | 探针 E；`user_stream.go:74-80` 有 `ReviewPosition` issues 即 `ErrNeedsReview`；`spot_quant_stream.go:56-59,74-77` 任何 error → `sendError` → `close(stop)` → 返回 | 需核对类事件改为「标记 + 触发只读 `Reconcile`」；仅连接级错误断流；补该竞态永久测试（现有 `TestStage064UserStream...` 的 FILLED 与 Fill 同时到达，未覆盖竞态） |
| **06-4-F5** | P3 | `ExecuteSell` 在 `enabled` 之前先做 `reconcilePositionOrders` 只读 REST；全局 OFF 时仍消耗 1 次查单权重 | 探针 C；`processor.go:141-168`；与 `ExecuteBuy:83` 开头即 gate 不对称 | 把 `enabled` 提到 `reconcilePositionOrders` 前；`Cancel`/`Reconcile` 允许 OFF 保持 |
| **06-4-F6** | P3 | `reconcileSymbolOrders` 每次新 BUY 遍历同 symbol 全部 OPEN/PENDING_BUY 仓位的所有历史订单，逐单 2 次 REST；任一老订单查不到即阻塞新单 | `processor.go:311-334`；探针 B `lookups=3` 已显示放大；实施记录 `:46` 自认「Binance may expire very old order lookups」 | 限定需实时核对的订单范围、加节流/批量，并区分历史订单不可查与网络错误 |
| **06-4-F7** | P3 | `HandleExecutionReport` 用 `err == orm.ErrNoRows` 而非 `errors.Is`；`ExecutionReport` 完整性校验漏 `FeeAsset` | `user_stream.go:35,60-63`；`snapshot.go` 对非零手续费且 `CommissionAsset==""` 会计阶段才报错 | 改 `errors.Is`；把 `FeeAsset` 纳入 TRADE 事件完整性校验；`FeeCost` 可能为 null 需明确处理 |
| **06-4-F8** | P3 | 滚动基线/口径滞后：总方案 `:155` 仍写「06-3 已完成整改，待独立复审」，但 `-rereview.md` 已随 `6d8eecc` 提交；`:4` 写「06-0～06-4 已完成…Gate」而 06-4 独立 review 本报告才完成 | 文档比对 | 同步为「06-3 复评已提交 / 06-4 待独立 review」，与 06-3-F6 同类 |
| **06-4-F9** | P3 / Gate | Testnet 真实签名 REST/WS、真实成交、MySQL 8、生产 v25 Schema 检查单、`dist`+`static`+二进制成套发布与回滚**均未执行** | 实施记录 `:46,:66,:68,:72` 已如实写明；本次复核确认 `RunSpotQuantUserStream` 零网络实证 | 归 06-6；不得以本阶段离线 PASS 宣称实盘可用 |

### F1 补充分析（为什么是必然失败而非边界情况）

06-3 `fills.go:270-272` **已经**在 SELL Fill 重放时按 `CommissionAsset == BaseAsset` 把手续费计入 `debit`，即账本层已正确处理 base 手续费；06-4 又在下单前预扣 0.5% 数量，构成**双重扣减**。若手续费以 quote/BNB 收取（Binance 现货常见），这 0.5% 更无任何依据。结果恒为 `remaining > 0` → `OPEN`：

- 第二次 SELL 数量约为首次的 0.5%，很快低于 `MinQty` / `LOT_SIZE` → `Spot quantity violates lot size`（探针 A）；
- `CreatePosition` 拒绝同 symbol 新周期（探针 A）；
- `realized_pnl` 可累计，但仓位状态与结算口径永不收敛。

现有 `TestStage064FakeBuySellAndWalletIsolation:242-244` 只断言 `sellQty.LessThan(qty)`，**把 dust 写成了预期行为**，因此 12 个专项测试全绿也不会发现该问题。这是本报告最需要优先整改的一项。

## 八、验收矩阵与结论

| 项目 | 结论 |
| --- | --- |
| 工作区归属 / 副本等价 | ✅ 8 项全部属 06-4；661 Go 文件 SHA256 全等 |
| 06-3 前置 Gate F1 / F2 / F4 | ✅ 已闭环且被 06-4 正确消费 |
| 06-3 前置 Gate F3（外部异常恢复） | ⚠️ 部分：fail-closed 到位，恢复路径缺失（**F2**） |
| 代码与离线 Gate（test/race/vet/build/diff/前端/产物） | ✅ **PASS 研发基线** |
| 资金安全护栏（默认只读、幂等、双开关、手工库存隔离、远端一致性） | ✅ 均有永久测试或探针实证 |
| 方案 06-4 阶段整体验收 | ⚠️ **有条件，不认可完整验收**（F1 / F2 未闭环） |
| **06-4 → 06-5 自动信号接线** | ⛔ **阻塞**（F1、F2、F3、F4） |
| 方案 06 整体 / Testnet 真实下单 / 生产发布 | ⛔ **全部 BLOCKED** |

**建议整改顺序**：① F1（SELL 数量与 CLOSED 收敛 + 永久测试断言）→ ② F2（权威 -2013 与网络不确定分流 + 恢复入口）→ ③ F3/F4（WS 端点显式化、竞态改只读对账）→ ④ F5～F8 → ⑤ 06-6 统一做 Testnet 端到端、MySQL 8、发布前 Schema 检查单与成套发布。整改复评另起 `-rereview.md`，不覆盖本报告。

## 九、审计边界声明

本次仅做只读代码审查、本地 `go test/vet/build`、`Fake`+临时 SQLite 探针与静态产物比对；**未**连接真实 Binance（含 Testnet）、**未**建立 WS 长连、**未**发送任何真实订单、**未**执行生产或实验库 `sync db`、**未**修改 `conf/app.conf`、**未**重启服务、**未**改用户业务代码、**未**自动 commit/push。探针仅在 `/tmp/phase_audit_064` 内，收尾随副本删除。实施记录中「MySQL 5.6 Docker 复跑 PASS」属 06-3-rereview 的开发侧声明，本会话未复跑，MySQL 8 仍未验证。
