# CODE_REVIEW_06-7-3：真实 Processor + Fake Exchange 全链路

日期：2026-11-13
后端：`feat/trade_spot`，HEAD **`1845572 feat: 06-7-2`**；**06-7-3 全部改动为未提交工作区**
主方案：[开发方案-06-7-现货量化自动策略执行链路.md](开发方案-06-7-现货量化自动策略执行链路.md)
阶段记录：[开发方案-06-7-3-真实Processor与FakeExchange全链路-实施记录.md](开发方案-06-7-3-真实Processor与FakeExchange全链路-实施记录.md)
前置报告：[CODE_REVIEW_06-7-2.md](CODE_REVIEW_06-7-2.md)、[CODE_REVIEW_06-7-2-rereview.md](CODE_REVIEW_06-7-2-rereview.md)（均保留原文）

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 代码与离线 Gate | **✅ PASS（合格研发基线）** |
| 06-7-3 阶段整体验收 | **⚠️ 有条件通过**：认可「真实 `spotexecution.Processor` + Fake Exchange 的隔离端到端链路已跑通」，但 F1 / F2 两个 P2 需在 06-7-4 前闭环或书面接受 |
| 06-7-2 整改独立复核 | **✅ 属实**（F2/F3/F4/F6/F7/F8 已在 HEAD 落地，探针 B 实证 F4 闭环）；F1（生产 `exec=nil`）仍为有意 P1 阻塞 |
| 06-7-4 准入 | **⚠️ 有条件放行** |
| 「06-7 整体完成」/「方案 06 整体完成」 | **❌ 不认可** |
| 真实自动下单、Testnet、Mainnet、生产发布、MySQL 8、v25 预检、回滚 | **⛔ 全部 BLOCKED** |

一句话：**这是 06 系列迄今最扎实的一次测试阶段——首次把真实 Processor、真实 SQLite Ledger、真实 ReadModel 串成 BUY→SELL→CLOSED 闭环，并且没有碰生产入口；但生产 `StartSpotQuantTrade` 仍是 `return ctx.Err()` 的空转，且本轮新引入的跨周期去重复用了「可注入事件时钟 + 真实墙钟 created_at」两个时间源，测试靠改写数据库时间才覆盖住分支。离线全绿不等于实盘可用。**

---

## 一、审计范围与基线

### 1.1 工作区状态（实测）

```
 M doc/trade/开发方案-06-7-现货量化自动策略执行链路.md
 M doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md
 M doc/trade/开发方案-总览与实施顺序.md
 M feature/spot_quant_runner.go
 M feature/spot_quant_runner_test.go
 M service/spotledger/query.go
?? doc/trade/开发方案-06-7-3-真实Processor与FakeExchange全链路-实施记录.md
?? feature/spot_quant_runner_0673_integration_test.go
```

`git log -8`：`1845572 feat: 06-7-2` → `5c1d4fa` → `32d203b` → `ed31061` → `35da095` → `5eb10a2` → `f2f3c9b` → `6d8eecc`。`1845572` 的 `git show --stat` 显示 10 files / +1486 行，**06-7-2 首轮整改（rereview 中 F2/F3/F4/F6/F7/F8）已随该提交进入 HEAD**。

**本阶段未修改 `main.go`、前端、`static/`、`conf/`、DDL / Schema / 路由**；`main.go:441-443` Futures `loopRun` 与 `449-455` Spot 独立 `loopRun` 均为 06-7-2 提交内容。改动范围仅 3 个文档 + 3 个 Go 文件 + 1 个新增集成测试（工作区 diff 6 files / +55 −22，另加约 700 行新测试）。

### 1.2 隔离副本与等价校验

`/tmp/phase_audit_0673` = `git archive HEAD` + 按工作区同步全部 **8 个 M/?? 文件**；副本 `conf/` 仅 `app.conf.example`、无 `strategy_templates/` 研究目录。

```
GO_FILES_IDENTICAL 674 MISSING 0 DIFFER 0
```

### 1.3 离线 Gate 实证（隔离副本，显式清空许可 env）

| Gate | 命令 / 结果 |
| --- | --- |
| 06-7-3 专项 | `go test ./feature -run '^TestSpotQuant0673' -count=1` → **PASS**，`-list` 列出 **10 个顶层测试** |
| 重复稳定 | `^TestSpotQuant0673\|^TestSpotQuant0672\|^TestReview0672` `-count=5 -p=2` → **PASS** |
| 06-7-1 整改回归 | `TestReview0671\|TestStage0671` in `line` + `spotquant` → **PASS** |
| 10 包 test | 显式 10 包全列 `-count=1` → **10/10 ok** |
| 10 包 race | `-race` 同 10 包 → **10/10 ok，无 DATA RACE** |
| vet / build / diff-check | `go vet` 10 包、`go build -o /tmp/spot0673_review_build .`、`git diff --check` → **全部 EXIT=0** |
| 干净副本全量 | `go test ./... -count=1` → **73 ok + 13 无测试包 + 0 FAIL，EXIT=0** |
| 项目脚本 | `./scripts/test-go.sh -short` → **EXIT=0** |

> 审计过程说明：首轮批量命令因 zsh 中 `$PKGS` 不自动分词，把整串包列表当成一个目录参数，导致 test/vet/race 三项报 `directory not found`（`EXIT=1`）。这是**审计命令自身的错误，不是代码缺陷**；已按项目约定显式写全包列表重跑，三项均 EXIT=0。全量 EXIT=0 沿用 06-ALL-rereview 的 `tests/default_test.go` + `controllers/testdata` 修复；**用户工作区含 `strategy_templates/research/**` 的全仓测试仍不得写成 PASS**。

### 1.4 06-7-2 整改独立复核（前置准入核对）

逐项核对 `CODE_REVIEW_06-7-2-rereview.md` 的声明与 HEAD 代码，**结论：属实**。

| 项 | 独立证据 |
| --- | --- |
| F2 Kline 事件 clientID | `spotSignalInterval`（`runner.go:85-118`）+ `spotCandleOpen`（`:122-137`）；探针 A `rule_only=1h tech5m_rule1h=1h tech_only=5m`，证明规则文本 `kline_1h` 与指标配置都能推高主周期 |
| F3 Fill 投影权威 / 全部未结订单 | `activeSpotPosition`（`:254-283`）对每页历史仓位逐个 `Accounting`；`reconcileUnsettled`（`:285-305`）遍历 `ListUnsettledOrders` **全部**结果并在任一对账轮直接 return |
| F4 空转无 ORM / panic 隔离 | `StartSpotQuantTrade`（`:411-413`）**函数体只有 `return ctx.Err()`**；`main.go:449-455` 用 `RunSpotQuantProtected` 单独包住 Spot 回调，Futures `loopRun`（`:441-443`）未改。探针 B：`PROBE0673_START_ENTRY err=<nil> no_panic=true`（干净副本无 default ORM alias 也不 panic） |
| F5 / F6 / F7 / F8 | `running` 注释澄清为实例级；`logSymbolError` + `ReportSpotQuantScheduler` 按错误文本去重；`reconcileUnsettled` 全部订单；`warnings` map 对默认 `100/100` 阈值首次提示 |
| F1 | **仍 BLOCKED（有意）**：`StartSpotQuantTrade` 不构造 Runner、不注入执行器；全仓生产注入点为零 |

---

## 二、方案契约核对

### 2.1 §4「06-7-3 完成标准」

| 完成标准 | 核对 |
| --- | --- |
| `go test ./...`、Race、Build | **✅** 干净副本 73 ok / race 10/10 / vet / build 全 EXIT=0 |
| Futures / 旧 Spot 回归通过 | **✅** 10 包 test+race 覆盖 `feature`/`spot`/`controllers`/`command`；全量 86 包 0 FAIL；Futures sentinel `untouched` |
| 不误卖钱包资产 | **✅** `TestSpotQuant0673RealProcessorBUYSELLFillReadModelAndNextCycle` 断言 Fake 钱包 50 手工 BTC 时 `SELL.OrigQty == BUY.RequestedQty`；dust / 余额不足 / 全局 OFF 三个 fail-closed 用例 |
| 没有重复 BUY/SELL | **⚠️ 基本满足但有边界**：20 轮重复、UNKNOWN 重启、-2013、部分成交、WS 重复 TradeID、5m/1h 同根 K 线、动态周期切换均有回归；但 F1 所示的「时钟分离」场景会漏，且测试靠改写 `created_at` 覆盖 |
| 真实 Processor + Fake Exchange 全链路 | **✅ 执行/账本/读模型层**；**⚠️ 策略行情层为最小 Fake**（F3） |

### 2.2 §3.3 幂等与恢复

- `clientOrderID` v3 = `symbol｜side｜position_id｜interval｜K线起点UTC`，SHA256 截断 34 字符，前缀 `spot_quant_0673_v3`；**不含可编辑 `RuleHash`** ✅；符合 Binance ≤36 与 `orderIDPattern` ✅。
- 新增 `hasBUYInEvent`（`:154-171`）：`created_at` 落在当前 K 线内 **OR** `client_order_id` 命中任一周期当前对齐 ID → 阻断新 BUY，用于覆盖动态切换主周期。
- `ListUnsettledOrders` 新增 `FILLED AND filled_qty <> requested_qty`，修复「交易所 ACK=FILLED 但 myTrades 迟到」时订单永久不可恢复的缺口（`TestSpotQuant0673RemoteFILLEDWithoutMyTradesRecoversLateFill` 验证补齐后不再返回）✅。

### 2.3 §5 最小测试清单

10 个集成测试覆盖 #1（双开关）、#2（20 轮不重单）、#3（部分成交）、#4（CLOSED 后新周期）、#6（UNKNOWN/重启/-2013/余额不足/dust）、#7（Futures 不受影响）、#8（全局 OFF 不丢 Fill、仍可显式 `Reconcile`）、#9（无新表/菜单）、#10（文档区分隔离联调与真实验收）。

**缺口**：#5 的 **BTCUSDT ↔ WBTCUSDT 精确隔离未在真实 Processor 下重测**（Fake Exchange 的 `Rules`/`Price`/`Free` 硬编码 `BTCUSDT`，10 个测试全部单币）；本轮探针 F 仅验证 `ListUnsettledOrders` 的精确匹配（`btc=1 wbtc=1`）。

---

## 三、正向结论（已实证）

1. **真实业务组件串联属实**：`spot0673Fixture`（integration test `:181-203`）用 `spotexecution.NewProcessor(ex, DBGate{DB: db}, r.ledger)` 注入真实 `Processor`，通过 `r.ledger`（真实 SQLite 06-3 表）、`spotledger.NewSpotReadService`（06-5 只读聚合）验证 BUY→SELL→CLOSED→新周期；10 个测试覆盖部分成交、`accepted_unknown` 重启、权威 -2013、FILLED 迟到 myTrades、BNB 手续费、基础币 dust、WS 重复 TradeID 三次 + REST 对账、5m/1h 去重、1h↔1m 动态周期切换。
2. **生产零接线保持**：`StartSpotQuantTrade` 只有 `return ctx.Err()`；`main.go` 的 Spot 回调独立且带 panic 隔离；打开两个业务开关**不会**发真实订单。文档三处状态行均如实写明，未虚报实盘。
3. **共享读模型未被破坏**：`service/spotledger/query.go` 本轮只改 `ListUnsettledOrders` 的 WHERE 条件，`ListPositions` 的默认精确 `symbol=?` / `SymbolLike` 显式开关（06-5 F1 整改）保持；`ListUnsettledOrders` 全仓仅被 runner 与测试调用，**不影响 06-5 只读 API / controllers**。
4. **`now()` 与 `hasBUYInEvent` 时间单位一致**：`ledger.go:104` `now()` 返回 `UnixMilli()`，`hasBUYInEvent` 用 `start.UnixMilli()` / `now.UnixMilli()`；`writeProjection`（`fills.go:380`）确实回写 `filled_qty` 与 `status`，FILLED 补齐后能退出未结集合。
5. **同源时钟下跨周期门禁有效**（对照探针，见 F1）：`PROBE0673_SAME_CLOCK seeded=2 places=2 created_at_le_now=true`。
6. **F4 闭环**：探针 B 实证干净副本无 default ORM alias 时 `StartSpotQuantTrade` 返回 `nil` 且不 panic。
7. **规则周期识别有效**：探针 A `PROBE0673_RULE_KLINE rule_only=1h tech5m_rule1h=1h tech_only=5m`。
8. **手工资产隔离**：Fake 钱包 50 BTC 时 SELL 数量等于策略 BUY 的 `RequestedQty`；dust / 余额不足 / OFF 均 fail-closed，不虚构 CLOSED、不挪用人工币。

---

## 四、Findings

### F1 · P2 · `hasBUYInEvent` 混用两个时间源，测试靠改写 `created_at` 才覆盖住分支

`hasBUYInEvent`（`runner.go:154-171`）的 SQL 上界是 `now.UnixMilli()`，其中 `now` 来自**可注入的 `r.clock()`**（事件时钟）；而 `spot_managed_orders.created_at` 由 `spotledger.now()` 写入，是**真实墙钟**。生产 `clock=time.Now` 时二者同源，门禁有效（探针 G 已证）；但二者一旦分离，`created_at<=now` 恒为 false，`client_order_id IN (...)` 又只覆盖各周期**当前对齐**的一个 ID，短周期→长周期切换的去重会**静默失效**。

隔离探针（1m 策略 BUY+SELL → 切 1h → 推进 2 分钟，**不改写任何 `created_at`**）：

```
PROBE0673_DUAL_CLOCK seeded_places=2 places_after_switch=3 simulated_now_ms=1794484920000 db_created_at_ms=1794537710236 created_at_after_sim_now=true last_err=<nil>
对照：PROBE0673_SAME_CLOCK seeded=2 places=2 created_at_le_now=true
```

即时钟分离时**同一根 1h K 线内重复 BUY 了一次，且无任何错误**。

更关键的是测试有效性：`TestSpotQuant0673SwitchingIndicatorTimeframeCannotReBUYPreviousSignal` 第二个子测试（integration test `:682-686`）注释写着「The Fill fixture uses the real wall clock, so bring this test order's creation time into the simulated 12:01 event」，并直接 `UPDATE spot_managed_orders SET created_at=<模拟时间>`。**这是改写被测数据以让断言成立**，掩盖了生产时钟同源这一隐含契约；一旦未来把 `r.clock` 换成基于 K 线 OpenTime 的事件时钟、发生 NTP 回拨、或订单 `created_at` 被回填/迁移，该回归不会报警。

**要求（06-7-4 前）**：把 `created_at` 上界改为真实 `time.Now()`（或去掉上界只保留下界），或将事件时间持久化到订单表；补一条**不改写 DB 时间**的 1m↔1h 回归；在方案 §3.3 写明该门禁依赖「事件时钟与账本时钟同源」。

### F2 · P2 · 超成交 FILLED 订单使该币永久、静默空转

`query.go:85` 的新条件是 `status='FILLED' AND filled_qty <> requested_qty`，会命中 **`filled_qty > requested_qty`**（超成交异常）；而 `ledger.go:343` 的 `CreateOrder` 未结检查用的是 `filled_qty < requested_qty`，**两处口径不一致**。`projectPosition`（`fills.go:298-300`）会把超成交标为 `ErrNeedsReview`，但 `Processor.Reconcile` 在本场景既不修复也不返回错误：

```
PROBE0673_OVERFILL pending_before=1 pending_after=1 places=1 last_err=<nil>
```

3 轮后 `ListUnsettledOrders` 仍返回同一订单，`reconcileUnsettled` 返回 `recovered=true, err=nil`，`runSymbol` 直接 return，`logSymbolError(symbol, nil)` 按「正常/恢复」处理 → **该币每 2 秒空转、永不评估信号、永不下单，且不产生任何告警**。这是 `CODE_REVIEW_06-ALL` F5「异常恢复无运维入口」在自动链路里的延续，且比 F5 更隐蔽（静默）。

**要求**：对 `>` 分支显式返回 `ErrNeedsReview` 并至少一次性告警，或把条件统一为 `<` 并让 `Accounting` 的 `reviewIssues` 经现有日志暴露；补永久回归。这样 06-7-4 才有可诊断的异常清单。

### F3 · P2 · 端到端测试替换了 06-7-1 的严格 Spot 环境，评估链未真正贯通

`spot0673Fixture`（integration test `:190-201`）把 `r.strategy` 设为手工构造的 `line.SpotRuleEvaluator`：`LoadEnvironment` 只返回 `NowPrice` / `NowSymbolClose` / `NowTime`，`LoadKlines` **永远返回 error**；测试规则只有 `NowPrice > 90` 与 `ROI > 8`。因此 **`LoadVerifiedSpotRuleEnv`、Spot ticker / 24h OHLC / Kline 新鲜度、指标样本完整性、`kline_*` DSL 变量、ticker 5 分钟窗口**在 06-7-3 的端到端链路里**零覆盖**；`spotSignalInterval` 读的是 `coin.Technology` 配置文本，与评估器实际加载的行情完全脱节。

实施记录 §25 只声明「不宣称真实 Binance WS/REST、主网技术指标网络链路已验」，未说明**本地严格环境也被替换**；「策略采用真实 `line.SpotRuleEvaluator` / expr / Spot ROI」容易被误读为整条评估链已验。阶段目标是 Fake Exchange 全链路，用 Fake 行情本身合理，但应把「已验」与「未验」列清楚。

**要求**：06-7-4 补一条使用 `LoadVerifiedSpotRuleEnv`（本地 Fake 行情源）的端到端测试，或在方案/实施记录中显式列出「严格 Spot 环境、指标完整性、Kline 新鲜度未进入 06-7-3 联调」。

### F4 · P3 · 长周期事件起点按 UTC 推导，非交易所实际 Kline OpenTime

探针 E：

```
PROBE0673_CANDLE interval=1m open=2026-11-12T12:34:00Z
PROBE0673_CANDLE interval=5m open=2026-11-12T12:30:00Z
PROBE0673_CANDLE interval=1h open=2026-11-12T12:00:00Z
PROBE0673_CANDLE interval=1d open=2026-11-12T00:00:00Z
PROBE0673_CANDLE interval=1w open=2026-11-09T00:00:00Z
PROBE0673_CANDLE interval=1M open=2026-11-01T00:00:00Z
PROBE0673_CANDLE interval=3d open=2026-11-10T00:00:00Z
```

`1w`（周一）/`1M`（月初）单独处理正确；`3d` 用 `Truncate(72h)` 基于 Unix epoch，与 Binance 3d K 线的真实对齐未必一致。rereview §15 与实施记录已披露「不宣称直接读取交易所每根实际 Kline 的 OpenTime」，属已知边界；06-7-4 接真实 Testnet 时须核对。

### F5 · P3 · 每轮完整 Fill 投影 + 无索引范围查询，MySQL 8 成本未实测

`activeSpotPosition` 每轮对启用币**全部历史仓位**逐个 `Accounting`（完整 Fill 投影）；`reconcileUnsettled` 每轮全表查 `ListUnsettledOrders`；`hasBUYInEvent` 每次 BUY 信号执行一次 `created_at` 范围 + 15 个 `client_order_id IN (?)` 的 `COUNT(*)`，而 `created_at` 无索引。实施记录 §82 / rereview §45 已承认「未声明 P95 达标、EXPLAIN 未实测」。个人单账户规模下安全优先可接受，但不得据此写「性能已验收」。

### F6 · P3 · 真实 Processor 下未重测跨币精确隔离

10 个集成测试的 Fake Exchange `Rules`/`Price`/`Free` 硬编码 `BTCUSDT`，故方案 §5 #5（BTCUSDT / WBTCUSDT 精确隔离）只在 06-7-2 的 Fake Executor 测试覆盖，本阶段真实 Processor 链路未重测。本轮探针 F 仅验证 `ListUnsettledOrders` 的 `WHERE symbol=?` 精确隔离（`btc=1 wbtc=1`）。建议 06-7-4 补一条双币真实 Processor 回归。

### F7 · P3 · clientID v2→v3 不兼容，历史订单只能靠 `created_at` 分支兜住

`spotQuantClientID` 前缀由 `spot_quant_0672_v2` 改为 `spot_quant_0673_v3`，且身份新增 `interval`。任何 06-7-2 时期生成的 v2 `client_order_id` 都不会被 `hasBUYInEvent` 的 `IN (...)` 命中，只剩 F1 所示的 `created_at` 分支。实施记录 §64 说明「此前自动运行时一直 `exec=nil`，不会修改已在 Binance 提交过的真实 clientID」——本阶段核对属实（生产零订单）；但 06-7-4 首次接真实账户时，必须一并验证 v3 ID 与任何既有本地订单、以及 Binance 侧 clientID 规则的兼容性。

---

## 五、实盘 / 发布 Gate（全部 BLOCKED）

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 生产运行时自动 BUY/SELL | ⛔ | `StartSpotQuantTrade` = `return ctx.Err()`，全仓无生产执行器注入点（06-7-2 F1 延续，有意 Gate） |
| Spot Testnet 签名账户 | ⛔ | 06-6 记录 `GET /api/v3/account` **-2015**，本阶段未修复 |
| 真实 Testnet 小额 BUY/SELL/Cancel/WS 恢复 | ⛔ | 本阶段仅 Fake Exchange，无 Binance 网络 |
| Mainnet 订单硬限制 | ⛔ | 06-4 适配器默认只读、主网硬拒（**正确**） |
| MySQL 8 物理 Schema / EXPLAIN / P95 | ⛔ | 仅 SQLite；见 F5 |
| 生产 v25 发布前 Schema 检查单 | ⛔ | `开发方案-06-发布前Schema验证检查单.md` §2 七项仍未勾选（06-2 起为流程门禁） |
| 二进制 + `static` + schema 成套发布 / 回滚 | ⛔ | 本阶段未改前端 / `static`；06-ALL F4（固定可发布 Git SHA 组合）仍未闭 |
| 异常订单运维入口（HTTP / CLI / UI） | ⛔ | 06-ALL F5 未闭；F2 使自动链路里异常更静默 |
| 任何真实 Binance 请求 / 下单 / 仓位操作 | ⛔ | 审计全程未访问；探针使用 SQLite + Fake，transport 不联网 |

**06-ALL 的 F1 / F4 / F5 / F6 与 06-7-4 的 Gate 均不因本阶段隔离联调通过而解除。**

---

## 六、06-7-4 准入条件

1. **F1 必须闭环或书面接受**：明确事件时钟与账本时钟同源契约，去掉对 `created_at <= r.clock()` 的依赖，补不改写 DB 时间的动态周期回归。
2. **F2 必须闭环或书面接受**：超成交 / 无法修复的 FILLED 订单必须产生可诊断的 `ErrNeedsReview` 而非静默空转；统一 `query.go:85` 与 `ledger.go:343` 的 `<>` / `<` 口径。
3. **F3 建议闭环**：补严格 Spot 环境的端到端测试，或文档显式列出未覆盖项。
4. F4 / F5 / F6 / F7 建议同步处理，尤其是 Testnet 真实 Kline 对齐、MySQL 8 EXPLAIN/P95、双币真实 Processor 回归、v3 clientID 首次接线验证。
5. 生产默认必须保持 `exec=nil`；任何真实执行器注入需 **06-7-4 独立书面授权 + 有效 Testnet 凭据**，并按「测试账户 → MySQL 8/发布核查 → 成套发布与回滚」顺序验收。

---

## 七、最终结论与审计边界

**06-7-3 的代码与离线 Gate 全部通过，10 个真实 Processor + Fake Exchange 集成测试构成 06 系列迄今最完整的隔离端到端证据，是一份合格研发基线；阶段整体验收为 ⚠️ 有条件通过：认可「隔离 Fake 全链路跑通」，不认可「06-7 整体完成」或任何实盘可用表述。06-7-4 有条件放行，F1 / F2 需先闭环或书面接受。所有实盘、Testnet、生产与发布 Gate 保持 BLOCKED；可以合并本地代码，但不得宣称自动现货量化可用、不得注入真实执行器、不得发主网订单。**

**审计边界**：本次未修改任何业务代码、未自动 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/` 与 `static/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。临时探针只写在 `/tmp/phase_audit_0673` 副本内并已全部删除，清理后复验 `go test ./feature -count=1` PASS；副本仅含 HEAD + 8 个 M/?? 同步文件，`conf/` 只有 `app.conf.example`。本报告为新增文件，未覆盖任何历史 `CODE_REVIEW_*` 报告。
