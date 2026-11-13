# CODE_REVIEW_06-7-2：两秒自动轮询与执行接线

日期：2026-11-12
后端：`feat/trade_spot`，HEAD **`5c1d4fa feat: 06-7-1`**；**06-7-2 全部改动为未提交工作区**
主方案：[开发方案-06-7-现货量化自动策略执行链路.md](开发方案-06-7-现货量化自动策略执行链路.md)
阶段记录：[开发方案-06-7-2-现货自动轮询与执行接线-实施记录.md](开发方案-06-7-2-现货自动轮询与执行接线-实施记录.md)
前置复评：[CODE_REVIEW_06-7-1-rereview.md](CODE_REVIEW_06-7-1-rereview.md)（F2/F3/F5/F6/F7 已修，F1 保留合约默认阈值语义，F8 移交本阶段）
历史报告：[CODE_REVIEW_06-7-1.md](CODE_REVIEW_06-7-1.md)、[CODE_REVIEW_06-ALL.md](CODE_REVIEW_06-ALL.md) 均保持不变

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 代码与离线 Gate | **✅ PASS（合格研发基线）** |
| 06-7-2 阶段整体验收 | **⚠️ 有条件通过**：认可「只读安全调度骨架 + Fake 可注入业务链路」，**不认可方案 §4 五项完成标准已在运行时达成** |
| 06-7-3 准入 | **⚠️ 有条件放行**，F2 / F3 / F4 必须先闭环或书面接受 |
| 「06-7 整体完成」/「方案 06 整体完成」 | **❌ 不认可** |
| 真实 Spot 自动下单、Testnet、Mainnet、生产发布、回滚、MySQL 8、v25 预检 | **⛔ 全部 BLOCKED** |

一句话：**06-7-2 把「2 秒调度」接进了 `main.go`，但接进去的是一个恒为 `exec=nil` 的只读空转回调；所有自动交易行为只存在于 Fake 注入路径。这是一条合格、诚实、安全边界清晰的研发基线，但不是可运行的自动交易链。** 与 `CODE_REVIEW_06-ALL` F1 同构，属于有意 Gate 而非隐瞒，因此问题不在「做错了」，而在**阶段完成度口径**。

---

## 一、审计范围与基线

### 1.1 工作区状态（实测）

```
 M doc/trade/开发方案-06-7-现货量化自动策略执行链路.md
 M doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md
 M doc/trade/开发方案-总览与实施顺序.md
 M main.go
?? doc/trade/开发方案-06-7-2-现货自动轮询与执行接线-实施记录.md
?? feature/spot_quant_runner.go
?? feature/spot_quant_runner_test.go
```

`git log -6`：`5c1d4fa` → `32d203b` → `ed31061` → `35da095` → `5eb10a2` → `f2f3c9b`。前端仓库 `/Users/zhz/work/binance/go_binance_futrues_new_ui` 当前 clean @ `cf319e1`；后端 `static/`、`conf/`、`routers/` 均无未提交改动。**本阶段为纯后端、首次修改 `main.go` 生产入口**，风险等级高于 06-7-1。

### 1.2 隔离副本与等价校验

按审计纪律建立 `/tmp/phase_audit_0672`：`git archive HEAD` + 按工作区同步全部 7 个 M/?? 文件；副本 `conf/` 仅 `app.conf.example`、无 `strategy_templates/` 研究目录。

```
GO_FILES_IDENTICAL 673 MISSING 0 DIFFER 0
```

`main.go:4` 已导入 `context`，新增 `context.WithTimeout(..., 30*time.Second)` 可编译。

### 1.3 离线 Gate 实证（全部在隔离副本，显式清空许可 env）

| Gate | 命令 | 实测结果 |
| --- | --- | --- |
| 06-7-2 专项 | `go test ./feature -run '^TestSpotQuant0672' -count=1` | **PASS**，8 个顶层测试 |
| 重复稳定 | `-count=5 -p=2` | **PASS** |
| 06-7-1 整改回归 | `line` + `spotquant` `-run 'TestReview0671\|TestStage0671\|TestStage062SpotRejectFuturesOnlyIdentifiers'` | **PASS** |
| 10 包 test | `-count=1` 显式 10 包全列 | **10/10 ok** |
| 10 包 race | `-race` 同 10 包 | **10/10 ok**（macOS `LC_DYSYMTAB` 非致命警告） |
| vet | 同 10 包 | **EXIT=0** |
| build | `go build -o /tmp/spot0672_review_build .` | **EXIT=0**（未覆盖运行中程序） |
| diff-check | `git diff --check`（用户工作区只读） | **EXIT=0** |
| 干净副本全量 | `go test ./... -count=1` | **73 ok + 13 无测试包 + 0 FAIL，EXIT=0** |
| 项目脚本 | `./scripts/test-go.sh -short` | **EXIT=0** |

> 全量 EXIT=0 与 06-ALL 首轮 EXIT=1 不是同一代码状态：`CODE_REVIEW_06-ALL-rereview` 已修 `tests/default_test.go`（缺私有配置时回退 `app.conf.example`）并把 `controllers/testdata/strategy_templates/ci-kline-long.json` 入 Git。**用户工作区全仓仍因 `strategy_templates/research/**` 变红，不得写「用户工作区全仓测试通过」。**

### 1.4 `main.go` 改动（完整 diff）

```go
// 06-7-2: match the Futures two-second cadence and loopRun no-overlap.
// The Spot Runner currently has NO live order executor in production: ...
loopRun(func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    if err := feature.StartSpotQuantTrade(ctx); err != nil {
        logs.Warning("spot quant scheduler: ", err)
    }
}, 2*time.Second)
```

只在既有 Futures `loopRun(..., time.Second*2)` **之后**新增一个独立 `loopRun` 调用，未改动 Futures 循环、路由、DDL、Schema、前端或 `static`。`main.go:572-585` 的 `loopRun` 内每个调用持有独立 `running int32` 并以 `atomic.CompareAndSwapInt32` 防重入，因此 Spot 与 Futures 两个循环**调度层互不影响**（正向）。

---

## 二、方案契约逐项核对

### 2.1 §4「06-7-2 完成标准」五项

| 完成标准 | 核对结果 |
| --- | --- |
| 开关即时生效 | **⚠️ 仅 Fake 路径 ✅**：`Run` 每轮 `globalEnabled` 读真实 DB；`runSymbol` 开头 + BUY/SELL 前各用 `DBGate.Enabled` 复查（`gate_db.go:14-33` 同查总/单币开关）；`TestSpotQuant0672SwitchOffDuringEntryBlocksSubmission` 验证评估期间关闭即时阻断。**运行时 `exec=nil` 在 `globalEnabled` 之前返回，连开关都不读**（F1） |
| 同轮不重叠 | **✅ 代码层**：`loopRun` CAS 有效。**⚠️ 实例级 `running` 声明不成立**（F5） |
| 同币不重复订单 | **⚠️ 仅「同一分钟」**：跨分钟会重新开周期（F2） |
| 已有未完成委托优先恢复 | **⚠️ 正常路径 ✅**（NEW / PARTIALLY_FILLED / PENDING_CANCEL 先 `Reconcile` 并 return，等下一轮）；物化状态异常或无活动仓位时跳过（F3）；多订单只恢复第一个（F7） |
| Futures 循环不受影响 | **✅ 调度层**；**⚠️ 进程级不成立**（F4 panic 会终止整个进程） |

### 2.2 §3.4 五条接线准入条件

1. **信号与订单职责分明 / 先查开关与未结订单**：**⚠️ 部分满足**。开关与活动仓位检查已落地；但「检查同币**全部**未结订单」只在 `exists=true` 分支按 position 查，无活动仓位时不查（F3）。
2. **平仓来源不可伪造**：**✅ 闭环**。runner 调用 06-7-1 整改后的 `EvaluateExit(coin, r.ledger, active.ID)`，评估器内部强制 `Ledger.Accounting` 从不可变 Fill 投影取数；`ErrNeedsReview` 会终止该币。
3. **持久化订单幂等（不能只靠 RuleHash）**：**⚠️ 部分满足**。`spotQuantClientID` 用 `symbol｜side｜position_id｜UTC 分钟桶` SHA256 截断，确实**不含可编辑 `RuleHash`**（06-7-1 F8 的这一半已闭环），34 字符符合 Binance ≤36 与 `orderIDPattern ^[A-Za-z0-9_-]{1,64}$`；「改策略/重启/UNKNOWN」有回归。**但缺方案要求的「对应 K 线时间」**（F2），「任意混合周期去重」被推给 06-7-3。
4. **环境差异显式保留**：**✅**。06-7-1 的三处有意安全偏离（Kline 失败 fail-closed、空阈值禁用、规则错误终止本轮）未被 06-7-2 改动。
5. **零生产放行**：**✅ 明确成立且文档如实**。实施记录 §9、方案 §5/§3.3、三个状态行均写明「生产执行器 `exec=nil`、只读、不发单」，未虚报实盘。

### 2.3 §5 最小测试清单 10 项

8 个 Fake 顶层测试覆盖 #1（双开关 5 轮 OFF / ON 下一轮）、#2（20 轮不重复，**固定 clock**）、#3（部分成交 / UNKNOWN）、#5（BTCUSDT ↔ WBTCUSDT 精确隔离，`symbol=?` 来自 06-5 整改）、#6（重启、clientID 稳定、跨币/方向碰撞、长度）、#7（Futures sentinel `untouched`）、#8（总开关 OFF 不自动 SELL）、#9（无新菜单/任务表/模式）、#10（文档区分代码接线与真实验收）。

**缺口**：#2 用固定 `clock=1794517200` 跑 20 轮（等价 40 秒），不覆盖跨分钟；#4「确认 CLOSED 后下一轮才可重新 BUY」实际是「下一分钟」；#6 未覆盖「同 K 线策略修改」这一项（只测了 `RuleHash` 不参与 ID，未测真实策略文本变更）；方案 §3.3 要求的「任意混合 K 线周期去重」无任何回归。

---

## 三、正向结论（已实证）

1. **真实 06-4 Processor 可直接注入**：隔离探针编译期断言 `var _ spotQuantExecutor = (*spotexecution.Processor)(nil)` **PASS**。三个方法签名完全一致：`ExecuteBuy(ctx, symbol, budget, clientID)`（`processor.go:77`）、`ExecuteSell(ctx, positionID, clientID)`（`:135`）、`Reconcile(ctx, localOrderID)`（`:363`）。06-7-3 无需写适配器。
2. **双层幂等**：runner 侧 `FindOrderByClientID` 去重；`Processor.ExecuteBuy/ExecuteSell` 侧 `p.mu.Lock()` + `validClientID` + `p.enabled` 双开关复查 + `FindOrderByClientID` 再查 + `reconcile` 复用旧意图，网络 UNKNOWN 不会二次 POST。
3. **06-3 账本兜底确实有效**：`CreatePosition`（`ledger.go:161-180`）会遍历该币全部历史仓位做 `projectPosition`，物化 `status` 被篡改也不能据此开新周期（探针 C 以 `ErrNeedsReview` 拦住）。**因此 F3 不构成重复开仓 / 超卖漏洞**。
4. **双开关与精确隔离**：`DBGate.Enabled` 每次读真实 DB、总开关与单币开关同查；`ListPositions` 默认精确 `symbol=?`，UI 模糊搜索走显式 `SymbolLike`（06-5 F1 整改保持有效）。
5. **Futures 隔离**：`loopRun` 每次调用独立 `running`；Futures sentinel 在测试中保持 `untouched`；本阶段未改 `routers/`、`static/`、`conf/`、前端、DDL。
6. **文档诚实**：方案 §4 阶段标题补了「本地代码及 Fake Gate 已完成；运行时无真实执行器」，三个文档状态行均写明 `exec=nil`、Testnet/生产 BLOCKED。**未发现虚报实盘的声明**。

---

## 四、Findings

### F1 · P1 · 「运行时接线」在生产路径恒为空转，§4 完成标准不可达

`feature/spot_quant_runner.go:239-254` 的 `StartSpotQuantTrade` 固定 `NewSpotQuantRunner(db, ledger, line.NewSpotRuleEvaluator(), **nil**)`；`Run`（`:91-96`）在 `exec==nil` 时**先于 `globalEnabled` 直接 `return nil`**。全仓 Go 引用点静态扫描（`git ls-files` + 本阶段 3 个新/改文件）：

```
feature/spot_quant_runner.go:239: func StartSpotQuantTrade(ctx context.Context) error
feature/spot_quant_runner.go:249: runner, err := NewSpotQuantRunner(db, ledger, line.NewSpotRuleEvaluator(), nil)
main.go:452:                     if err := feature.StartSpotQuantTrade(ctx); err != nil
feature/spot_quant_runner_test.go:98 / :303: NewSpotQuantRunner(..., ex)   // Fake only
```

**全仓不存在任何生产注入点、配置项或路由能把非 nil 执行器交给 runner。** 因此方案 §4 的五项完成标准（开关即时生效、同币不重复订单、未完成委托恢复）以及 §3.1 的行为，在运行时**全部不可达**，只在测试 Fake 注入路径成立。这与 `CODE_REVIEW_06-ALL` F1（`NewProcessor` / `RunSpotQuantUserStream` 零运行时引用）是同一个问题的延续：本阶段补上了「调度骨架」，但没补上「可运行交易链」。

**定性**：这是有意的安全 Gate，实施记录与方案均已披露，不是隐瞒，也不是缺陷。**问题是完成度口径**——不能把「`main.go` 已注册 2 秒回调」写成「06-7-2 完成标准已达成」。建议把 §4 完成标准列同步标注「仅 Fake 注入路径验证；运行时需 06-7-4 独立授权后注入」，避免与 06-ALL F1 的整改承诺混淆。

### F2 · P2 · clientOrderID 用固定 1 分钟桶，缺方案要求的 K 线事件时间；跨分钟可无限重建周期

`spotQuantClientID`（`:63-68`）的 `identity` 为 `symbol｜side｜position_id｜now.UTC().Unix()/60`，**不含 K 线周期或 K 线 OpenTime**。方案 §3.3 明确要求「结合币种、BUY/SELL 方向、持久化持仓周期、**对应 K 线时间**、现存订单」，并写「确认 CLOSED 后**同根 K 线**仍不能无限 BUY」；§3.2 写「不做自动加仓」。

隔离探针（真实 SQLite + Fake Executor，逐分钟推进 clock，策略恒 BUY + 恒 SELL）：

```
PROBE0672_MINUTE_BUCKET minutes=6 polls=12 buys=6 sells=6 reconciles=0
PROBE0672_CLIENTID len=34 same_minute_equal=true distinct_ids_within_one_5m_bar=5 sample=sqefd45d56faf8d393640a4060eeaae8e8
```

即：**每过一个 UTC 分钟，BUY 的 clientID 就变化一次；只要上一周期已 CLOSED，就会新建一个完整 BUY→SELL 周期**。对 1m 策略这与「一根 K 线一个事件」等价；对 5m 策略，同一根 K 线内可产生 5 个不同 clientID / 5 个周期；1h 策略可产生 60 个。

实施记录 §36 已诚实披露「现有设计不宣称已覆盖任意混合 K 线周期的严格事件去重」并推给 06-7-3，**但方案 §4 完成标准「同币不重复订单」与 §5 #2「连续 20 轮最多一个 BUY」未同步收窄**，且 `TestSpotQuant0672SwitchesRepeatedSignalsAndSecondCycle` 用固定 clock 跑 20 轮（40 秒），随后又**显式断言** `r.clock` +60s 后 `ex.buys == 2`（测试名 `new bar must allow new cycle` 把 1 分钟称作 bar）。测试与方案口径互相加固，掩盖了真实 2 秒 × N 分钟行为。

**要求（06-7-3 注入真实 Processor 前）**：要么把 clientID 绑定到策略主时间框架的 K 线 OpenTime，要么把 §3.3 / §4 / §5 的文字收窄为「1 分钟事件桶」并补 5m/1h 策略下不重复的永久回归；不得只靠固定 clock 测试宣称幂等达成。

### F3 · P2 · `activeSpotPosition` 读物化 status，绕开 06-3「Fill 投影为唯一金融权威」

`activeSpotPosition`（`:130-147`）用 `ledger.ListPositions(PositionFilter{Symbol, Status})` 读物化 `status IN ('OPEN','PENDING_BUY')`（`query.go:54-60` 白名单只有这三种）。06-3 整改后 `projectPosition` 才是金融权威，`Accounting` 会强制从不可变 Fill 重算；06-7-1 F2 也是为此把 `EvaluateExit` 改成 `(coin, *Ledger, positionID)`。**但调度层判断「有没有活动仓位」用的仍是物化列**，把 06-7-1 F2 的闭环在入口处绕开了半圈。

隔离探针（真实 `PENDING_BUY` 仓位 + 未结 `NEW` BUY 订单，把物化 `status` 篡改为 `CLOSED`，clock 推进一分钟）：

```
PROBE0672_MATERIALIZED pid=1 tampered=CLOSED accounting_status=PENDING_BUY accounting_err=<nil> reconciles_delta=0 buys_delta=1 run_err=BTCUSDT: spot order needs reconciliation: previous Spot cycle 1 still holds strategy inventory
```

含义：
- `Accounting` 明确知道该周期仍是 `PENDING_BUY`、仍有未结订单；
- runner 却 `reconciles_delta=0`，**完全没有对账那笔未结订单**，直接走到 `EvaluateEntry` 发出新 BUY 意图（`buys_delta=1`），违反 §3.4 准入 1「检查同币全部活动仓位**和未结订单**，先对账再判断是否允许买入」；
- 最终由 06-3 `CreatePosition` 的 `projectPosition` 历史兜底以 `ErrNeedsReview` 拦住，**没有造成重复开仓或超卖**（正向，不夸大级别）。

风险在于：`exists=false` 分支不查「该 symbol 是否还有任何 NEW/UNKNOWN 订单」，异常仓位会被调度层静默跳过、永不恢复，是 `CODE_REVIEW_06-ALL` F5「异常恢复无运维入口」在自动链路里的延续；且若 06-7-3 换用一个不做 `CreatePosition` 检查的执行器，此处会直接变成重复开仓。

**建议**：`activeSpotPosition` 改以 `Accounting` / `projectPosition` 投影判断活跃仓位；并在「无活动仓位」分支增加按 symbol 的全局未结订单兜底对账；补篡改物化状态 + 无活动仓位孤儿订单的永久回归。

### F4 · P2 · 只读空转回调仍有前置副作用，且 `loopRun` 无 recover，异常会 panic 终止整个进程

`StartSpotQuantTrade` 在 `exec==nil` 早退**之前**依次执行：`config.String("database::driver")` → `orm.NewOrm()` → `spotledger.New` → `line.NewSpotRuleEvaluator()` → `NewSpotQuantRunner`。即生产每 2 秒构造一整套对象却立刻丢弃。更关键的是 `main.go:572-585` 的 `loopRun` 在独立 goroutine 里直接 `callback()`，**没有任何 `defer recover`**。

隔离探针（干净副本，default ORM alias 未注册）实证：

```
panic: <Ormer.Using> unknown db alias name `default`   → go test 进程终止 / EXIT=1
```

生产正常启动时 `initializeRuntimeDatabase` 已注册 default alias，故**不是必然触发**；但一旦配置加载异常、DB 别名失效或未来把 `orm.NewOrm()` 换成会 panic 的初始化，这条每 2 秒的 Spot 回调会 **panic 掉整个进程，连带 Futures 循环一起终止**，与方案 §3.1「若 Futures 循环、Spot 监听、旧 Spot Rush 发生异常，不得导致另一条自动循环永久退出；保留各自的日志和已有限流防护」直接冲突。

**建议**：`exec==nil` 时在 `StartSpotQuantTrade` 最开头直接 `return nil`（连配置与 ORM 都不要碰）；或给 `main.go` 的 Spot 回调加 `defer recover()` + 一次性告警。补「Spot 回调异常不影响 Futures 循环」的回归。

### F5 · P3 · 实例级 `running atomic.Bool` 在生产接线模式是死代码，防重入声明与测试不等价

`SpotQuantRunner.running` 是**实例字段**；而 `StartSpotQuantTrade` 每 2 秒 `New` 一个新 runner（静态引用点见 F1），因此该门禁在生产路径永远是「新实例的 false→true」，**不提供任何跨轮防重入**。真实防重入完全依赖 `loopRun` 的 per-call `atomic.CompareAndSwapInt32`。当前 `loopRun` 确实有效，故**无实际并发下单漏洞**；但 `TestSpotQuant0672OverlappingCyclesSkipWithoutQueue` 只测「同一实例并发 `Run`」，与生产接线不等价，实施记录 §29「`SpotQuantRunner` 同时有 `atomic.Bool` 防止手工/测试中并发 `Run`」作为安全声明具有误导性。

> 审计过程说明：一例用于动态演示「跨新实例绕过门禁」的并发探针因探针自身同步原语阻塞而超时（`-timeout 45s`），**已弃用且不作为证据**；本条结论仅由静态引用点与 `loopRun` 源码证明。所有临时探针已删除，副本复验 `go test ./feature -count=1` PASS。

**建议**：runner 提为包级单例，或删除误导性声明并在注释中明确「防重入由 `loopRun` 持有」。

### F6 · P3 · 异常日志每 2 秒重复，违反方案 §3.3 的告警约束

`runSymbol`（`:116-119`）对每个异常币 `logs.Warning`，`Run` 用 `errors.Join` 汇总返回后 `main.go:453` 再 `logs.Warning("spot quant scheduler: ", err)`。任一币持续 `ErrNeedsReview` 或 REST 异常时，**每 2 秒至少 2 条重复日志**。方案 §3.3 明确「不要每 2 秒发送重复报警，订单状态变更与异常才通知」。建议做状态变化去重 / 指数退避。

### F7 · P3 · 未结订单只恢复第一个

`runSymbol:174-184` 遍历 `ListOrders(active.ID)`，遇到**第一个** `NEW / PARTIALLY_FILLED / PENDING_CANCEL` 就 `Reconcile` 并 `return nil`，本轮不再处理其余订单；只要第一个状态不变，该币会一直卡在对账同一笔订单。方案 §3.3「启动与每轮优先恢复**已有订单**」应覆盖全部未结委托。建议一轮内对账全部未结订单后再等下一轮决策。

### F8 · P3 · 06-7-1 F1 的流程门禁在本阶段无代码承载

`CODE_REVIEW_06-7-1-rereview` 把「新币默认 `profit=100` / `loss=100` 导致 `close_long` 极少触发」选择为**保留合约语义 + 文档约束 + 启用前人工核实阈值**，把风险转为流程门禁。但 06-7-2 的 runner、`StartSpotQuantTrade` 与配置 API 没有任何启动期校验或告警，`方案 §3.2` 的「06-7-2 开启前应由用户核实阈值」在自动链路里完全依赖人读文档。建议在 06-7-3 / 06-7-4 注入执行器时，至少对启用币的默认阈值做一次显式日志提示或拒绝启用。

---

## 五、实盘 / 发布 Gate（全部 BLOCKED）

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 生产运行时自动 BUY/SELL | ⛔ | `exec=nil` 恒空转（F1），且无注入点 |
| Spot Testnet 签名账户 | ⛔ | 06-6 记录 `GET /api/v3/account` **-2015**，本阶段未修复 |
| 真实 Testnet 小额 BUY / SELL / Cancel / WS 恢复 | ⛔ | 06-7-4 范围，本阶段未执行 |
| Mainnet 订单硬限制 | ⛔ | 06-4 适配器默认只读、主网硬拒，本阶段未解除（**正确**） |
| MySQL 8 物理 Schema / EXPLAIN / P95 | ⛔ | 未执行；仅 SQLite + Fake |
| 生产 v25 发布前 Schema 检查单 | ⛔ | `开发方案-06-发布前Schema验证检查单.md` §2 七项仍未勾选；06-2 起风险由代码门禁转为流程门禁 |
| 二进制 + `static` + schema 成套发布 / 回滚 | ⛔ | 本阶段未改前端/`static`，不构成发布；06-ALL F4（固定可发布 Git SHA 组合）仍未闭 |
| 异常订单运维入口（HTTP / CLI / UI） | ⛔ | 06-ALL F5 仍未闭；本阶段 F3 使自动链路里异常仓位更难被发现 |
| 任何真实 Binance 请求 / 下单 / 仓位操作 | ⛔ | 审计全程未访问；探针使用 SQLite + Fake，transport 探针不联网 |

**06-ALL 的 F1 / F4 / F5 / F6 与 06-7-3 / 06-7-4 的 Gate 均不因本阶段接线而解除。**

---

## 六、06-7-3 准入条件

1. **F2 必须闭环或书面接受**：clientOrderID 绑定策略主 K 线事件时间，或方案 §3.3 / §4 / §5 口径显式收窄为 1 分钟事件桶并补 5m/1h 永久回归；禁止继续用固定 clock 的 20 轮测试宣称「同币不重复订单」。
2. **F3 必须闭环或书面接受**：活跃仓位判断改走 `Accounting` 投影，补「无活动仓位但存在未结订单」的全局兜底对账与篡改物化状态回归。
3. **F4 必须闭环**：`exec==nil` 入口直接返回 / 给回调加 recover，并证明 Spot 回调异常不影响 Futures 循环。
4. 建议同步处理 F5 / F6 / F7 / F8。
5. 06-7-3 使用真实 `spotexecution.Processor` + Fake Exchange 时，**生产默认仍必须保持 `exec=nil`**；任何真实执行器注入都需 06-7-4 的独立书面授权 + 有效 Testnet 凭据。
6. 补方案 §5 #6 的「同 K 线策略修改」去重回归，以及完整 BUY→SELL→新周期在真实 Processor 下的 REST/WS 仿真。

---

## 七、最终结论与审计边界

**06-7-2 的代码与离线 Gate 全部通过，是一份合格、边界诚实的研发基线；阶段整体验收为 ⚠️ 有条件通过：认可「只读安全调度骨架 + Fake 可注入业务链路已实现」，不认可「方案 §4 五项完成标准已在运行时达成」，不认可「06-7 整体完成」或「方案 06 整体完成」。06-7-3 有条件放行，F2 / F3 / F4 需先闭环或书面接受。所有实盘、Testnet、生产与发布 Gate 保持 BLOCKED；可以合并本地代码，但不得宣称自动现货量化可用、不得启用真实执行器、不得发主网订单。**

**审计边界**：本次未修改任何业务代码、未自动 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/` 与 `static/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。临时探针只写在 `/tmp/phase_audit_0672` 副本内并已全部删除；副本仅含 HEAD + 7 个 M/?? 同步文件，`conf/` 只有 `app.conf.example`。本报告为新增文件，未覆盖任何历史 `CODE_REVIEW_*` 报告。
