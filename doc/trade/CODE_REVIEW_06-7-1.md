# Code Review：方案 06-7-1（现货策略规则评估）

日期：2026-11-12  
后端：`feat/trade_spot`，HEAD **`32d203b doc: update doc`**，06-7-1 代码为**未提交工作区**（6 个 Go 文件 + 4 个文档）  
前端：本阶段零改动  
审计依据：[06-7 主方案](开发方案-06-7-现货量化自动策略执行链路.md) §三/§四/§五、[06-7-1 实施记录](开发方案-06-7-1-现货策略规则评估-实施记录.md)、[06 主方案 §7](开发方案-06-现货量化交易-复刻合约简化版.md)、[CODE_REVIEW_06-ALL-rereview.md](CODE_REVIEW_06-ALL-rereview.md)  
副本：`/tmp/phase_audit_0671` = `git archive HEAD` + 按 `git status --porcelain -z` 同步全部 10 项 M/??；**670 个工具链可见 Go 文件 SHA256 全等、`mismatch=0`、`only_copy=0`**；探针只在副本内，结束随副本删除。  
本报告不覆盖任何历史报告。

---

## 一、结论先行

> **06-7-1 的代码与离线 Gate 全部通过，是一份合格的研发基线；但阶段整体验收为 ⚠️ 有条件通过，不认可「06-7 整体完成」。**
> 发现 2 个应在 06-7-2 接线前处理的 P2：默认 `profit/loss=100` 使退出规则永不评估（F1）、`EvaluateExit` 无法验证 Snapshot 是否来自 Fill 投影（F2）。
> 生产 / 实盘 / Testnet / 自动交易 Gate **全部 BLOCKED**。

| 维度 | 结果 |
|---|---|
| 代码与离线 Gate | **✅ PASS**（专项 8/8 + AST 回归、10 包单测/race、vet/build/diff-check、全量脚本与 `go test ./...` 干净副本 EXIT=0） |
| 方案 §四 06-7-1 完成标准（「只输出 BUY/SELL，不用 Futures K 线、杠杆或仓位」） | **✅ 达标** |
| 阶段整体验收 | **⚠️ 有条件**（F1/F2 待闭环或书面接受） |
| 06-7-2 准入 | **⚠️ 有条件放行**，须把 F1/F2/F8 列为接线前准入检查项 |
| 自动交易可用性 | **❌ 不可用**：`NewSpotRuleEvaluator` / `LoadVerifiedSpotRuleEnv` 全仓只有定义、零生产调用 |
| 生产 / 实盘 / Testnet | **⛔ BLOCKED**（承接 06-ALL-rereview F1/F5/F6） |

**现在什么已经可用**：`line.NewSpotRuleEvaluator()` 提供了可被 06-7-2 调用的纯信号 API（`EvaluateEntry` / `EvaluateExit`），在注入的 Spot 行情下能正确产出 BUY/SELL 与无杠杆 ROI，且所有异常路径 fail-closed。**没有**任何 HTTP 路由、定时任务、`main.go` 接线或真实下单能力；开关打开仍不会自动交易。

---

## 二、审计范围与基线

### 2.1 与上一轮整体复审的关系

`35da095`（06-ALL 首轮）→ `ed31061 feat: 06-6` → **`32d203b doc: update doc`**。`32d203b` 已包含 [CODE_REVIEW_06-ALL-rereview.md](CODE_REVIEW_06-ALL-rereview.md) 的 F2/F3 整改：README 6 处 v24→v25、`tests/default_test.go` 改为缺失私有配置时加载 `app.conf.example`、`controllers/testdata/strategy_templates/ci-kline-long.json` 最小 fixture 入 Git、新增异常恢复操作边界文档与 06-ALL 复评。

因此**本轮干净副本的 `go test ./...` EXIT=0 与 06-ALL 首轮的 EXIT=1 不是同一代码状态**：首轮红点（`tests` panic、controllers 缺 fixture）已由用户侧整改消除，本轮独立复现确认有效。06-ALL 的 F1（自动信号链无入口）、F4（前端 SHA 组合未固定）、F5（异常恢复无运维入口）、F6（实盘 Gate）在 06-7-1 阶段**仍为未闭 Gate**，06-7-1 本身只解决 F1 的**评估器部分**。

### 2.2 06-7-1 实际改动

| 文件 | 性质 |
|---|---|
| `feature/strategy/line/spot_strategy.go` | 新增 `SpotRuleEvaluator` / `SpotSignal` / `EvaluateEntry` / `EvaluateExit` |
| `feature/strategy/line/spot_rule_env.go` | 新增 `LoadVerifiedSpotRuleEnv` / `buildVerifiedSpotRuleEnv` / `verifiedSpotKlines` |
| `feature/strategy/line/spot_strategy_0671_test.go` | 新增 5 个顶层用例 |
| `feature/strategy/line/spot_rule_env_0671_test.go` | 新增 3 个顶层用例 |
| `service/spotquant/validation.go` | 修改 AST 禁用名单：allow `ROI`/`Position`/`Positions`（仅 close 侧），加 `IsolatedWallet` |
| `service/spotquant/repository_test.go` | 更新 06-2 AST 回归 + 唯一 ORM alias（`-count=N` 重复运行） |

**零改动确认**：`main.go`、`routers/`、`controllers/`、`service/spotexecution`、`appversion/`、`conf/`、`static/`、前端、`spotledger` 本阶段均未修改（Git diff 与未跟踪清单实证）。

### 2.3 环境前提

- 工作区独有 50 个 Go 文件全在 `.local/`（43）与 `.agents/`（7）私有研究目录，Go 工具链默认忽略；副本剪除后 `only_copy=0`、SHA256 全等。
- `strategy_templates/` 整目录被内层 `.gitignore` 忽略；对其做 glob/`find` 会触发 SIGTERM(137)（本轮复现一次），改用 Grep 工具或 `git ls-files`。
- 06-7-1 实施记录 §四 已诚实声明：**用户工作区**全仓 `go test ./...` 会因 `strategy_templates/research/**` 中重复 `runSignals` 定义变红，不能宣称 PASS。本轮**未在用户工作区重跑全仓测试**以避开该已知限制；副本剪除 research 后全绿。两者不矛盾，报告须分别表述。
- zsh 中 `$PKGS` 不分词，所有 `go test/vet` 包列表显式写全。

---

## 三、离线 Gate 实证（副本内，显式清空许可 env）

| Gate | 命令 / 范围 | 结果 |
|---|---|---|
| 副本等价 | 670 个 Go 文件逐文件 SHA256 | **mismatch=0 / only_copy=0** |
| 06-7-1 专项 | `go test -count=1 -v ... -run 'TestStage0671\|TestStage062SpotRejectFuturesOnlyIdentifiers'` | **8 PASS / 0 FAIL**（line 7 + spotquant 1） |
| 目标包单测 | `go test -count=1` × 10 包 | **10/10 PASS** |
| Race | `go test -race -count=1` × 10 包 | **10/10 PASS** |
| 重复稳定性 | `-count=5 -p=2` | **EXIT=0**（实施记录所述 ORM alias 修复有效） |
| vet / build | `go vet` × 10 包；`go build -o /tmp/pa0671_bin .` | **EXIT=0 / EXIT=0** |
| diff-check | `git diff --check`（用户工作区） | **EXIT=0** |
| 应用包脚本 | `./scripts/test-go.sh -short`（干净副本） | **EXIT=0，73 ok / 0 FAIL** |
| 全量测试 | `go test -count=1 ./...`（干净副本） | **EXIT=0，73 ok / 0 FAIL** |

**前置回归未回退**：06-3/06-4/06-5/06-6 的 `TestReview06*`、`TestStage06*`、`V25VersionReuseMissingSpotLedgerFailsClosed` 随 10 包单测一并执行，均 PASS；Futures/旧 Spot 逻辑与旧 `InitSpotParseEnv` 未改。

---

## 四、方案 §四 06-7-1 完成标准与 §五 最小清单复核

| 标准 | 判定 | 证据 |
|---|---|---|
| 相同 Spot K 线/指标下策略结果符合合约 DSL | ✅ | `ParseTechnologyConfigWithKlineLoader` 复用，`ma_5.Data[0]` / `kline_5m.Close[0]` 探针与 `TestStage0671VerifiedSpotEnvPreservesDslAndIndicatorData` PASS |
| 只输出 BUY/SELL，不调用交易所 | ✅ | `SpotSignal` 仅 `Action/Reason/RuleName/RuleHash/ROI`；全仓无 `ExecuteBuy`/`ExecuteSell`/`Place` 引用 |
| 不用 Futures K 线 | ✅ | `verifiedSpotKlines → spotIndicatorKlines → spotbinance.GetKlineData`；`futures.Kline` 仅作 OHLC 数据结构 |
| 不用杠杆 / Futures 仓位 | ✅ | `SpotPositionCode` 无 Leverage/Margin；ROI = `(price-avg)/avg×100` 无杠杆；`spotFuturesOnlyIdentifiers` 拒绝 `FundingRate`/`Leverage`/`MarginType`/`PositionSide`/`LiquidationPrice`/`IsolatedWallet`/`OpenStrategyHash` |
| 只用 Spot 策略仓位数据 | ⚠️ 部分 | `Position`/`Positions` 由 Snapshot 构造，但**来源不可验证**，见 F2 |
| §五 1、2、3、5、6、8（开关即时生效、同轮不重叠、幂等、精确隔离、异常不重单、关开不丢记录） | ⛔ 不属本阶段 | 全部依赖 06-7-2 调度器；当前信号层无开关/仓位/订单参数，见 F8 |
| §五 7（Spot 指标/价格/ROI 全来自 Spot） | ✅ | 静态数据源与探针确认 |
| §五 9（不新增菜单/任务/模式/表，hash 字段先验证） | ✅ | 无新表、无新字段、无 `OpenStrategyHash` 持久化 |
| §五 10（区分「代码接线」与「真实验收」） | ✅ | 主方案状态行与实施记录首段均明确 BLOCKED |

---

## 五、探针实证（副本内临时测试，不联网、不发单）

```
PROBE0671_DEFAULT_THRESHOLD profit=100 loss=100 roi=10 signal="" err=<nil> fallbackKlinesRead=false
PROBE0671_CLOSE_RULE_SKIPPED roi=7 signal="" err=<nil> (nil err => close_long never compiled)
PROBE0671_SNAPSHOT_STATE unpriced BNB fee accepted=true err=<nil>
PROBE0671_SNAPSHOT_STATE impossible inconsistent accepted=false err=Spot managed position accounting state is incomplete
PROBE0671_UNVERIFIED_SNAPSHOT_ACCEPTED signal="" err=<nil> roi="10900"
PROBE0671_KLINE_AGE interval=1M span=31d age=60d -> allowedWindow=1488h5m0s stale=false
PROBE0671_THRESHOLD_EMPTY spot=1000000 err=<nil> (Futures ParseFloat("") would yield 0)
PROBE0671_THRESHOLD_ZERO spot=1000000
PROBE0671_AST "Position.UnrealizedProfit > 0" allowed=true
PROBE0671_AST "Positions[0].UnrealizedProfit > 0" allowed=true
PROBE0671_AST "Position.Leverage > 1" allowed=false
PROBE0671_AST "Positions[0].MarginType == \"CROSSED\"" allowed=false
PROBE0671_AST "OpenStrategyHash == \"x\"" allowed=false
PROBE0671_ENTRY_NO_POSITION_GUARD signal="BUY" err=<nil> (EvaluateEntry takes only coin)
PROBE0671_SHORT_REJECTED err=strategy[0] type "short" is not allowed for Spot (LONG/CLOSE_LONG only)
```

**静态零接线证据**：`NewSpotRuleEvaluator(` 与 `LoadVerifiedSpotRuleEnv(` 全仓**仅有定义行**（`spot_strategy.go:50`、`spot_rule_env.go:26`），无任何生产调用；`main.go` 对 `spotquant`/`spotexecution`/`StartSpotQuantTrade` 0 引用；`routers/router.go` 未新增路由。

---

## 六、本轮 Findings

### F1 — P2／06-7-2 接线前须处理｜默认 `profit/loss=100` 使退出规则永不评估

`EvaluateExit` 先算 `reason`：
```go
case roi <= -lossThreshold: reason = "stop_loss"
case roi >= profitThreshold: reason = "take_profit"
if reason == "" { return hold, nil }   // spot_strategy.go:240-242
```
`checkedSpotThreshold(""/"0")` 返回 `1000000`（禁用），而 `service/spotquant/repository.go:128` 新建币种默认 **`Profit: "100", Loss: "100"`**。因此未显式调低阈值的币种，ROI 需达 ±100% 才会进入 `close_long` 规则评估或 5m Kline 兜底。

探针实证：`profit=100 loss=100 roi=10 close_long="true"` → `signal=""`、`err=nil`、`fallbackKlinesRead=false`；另以引用未定义变量的 close_long 规则验证 ROI=7% 时 `err=nil`，证明**规则从未被编译/评估**。

后果：按方案 §3.2 字面「先判断已启用 `close_long` 规则」，用户会以为配了出场规则就能卖；实际自动 SELL 几乎不会触发，仓位可能长期持有。实施记录 §二.3 已澄清「close_long 为达到阈值后的卖出许可」，且此顺序与 Futures `evaluateTradeExitWithRules`（`feature/account_trade_cycle.go:155-166`，仅阈值触发时调 `canClose`）**实际行为一致**——因此这不是对合约的偏离，而是**方案 §3.2 文字与实现/实施记录不一致**，叠加默认值形成的可用性陷阱。

**建议**（三选一，均需补永久回归）：①方案 §3.2 改为明确的「ROI 阈值为前置门控，close_long 为阈值触发后的许可」；②06-7-2 启用前在配置 API 或启用校验中要求显式设置合理 `profit/loss`；③评估把新建默认从 `100` 下调到与业务一致的数值。

### F2 — P2／06-7-2 接线前须处理｜`EvaluateExit` 无法验证 Snapshot 来自 Fill 投影

签名 `EvaluateExit(coin models.SpotSymbols, snapshot spotledger.Snapshot)` 接受任意手工构造的 `Snapshot`。`validSpotPosition` 只用 `snapshot.Position.RemainingQty` / `AvgCostQuote` 计算数量与 ROI，没有任何字段能证明该值来自 `Ledger.Accounting()` 的 `projectPosition`。

探针把 `RemainingQty` 改为 `999`、`AvgCostQuote` 改为 `1` 后，`EvaluateExit` 正常接受并算出 `ROI="10900"`，未报错。实施记录 §一「持仓必须来自 `Ledger.Accounting(...)`；返回的价格/成交数量绝不能来自可编辑的物化数据库列」是**文档约定，不是类型契约**。

风险边界：`Processor.ExecuteSell` 内部会重新投影并拒绝超卖，**不会造成资金超卖**；但退出信号（是否 SELL、何时 SELL）会基于错误成本价误判。06-3-F1 曾因「信任物化字段」列为 P2，此处在信号层重新打开了同类入口。

**建议**：把 Snapshot 换成由 `Accounting` 构造的不透明类型（私有字段 + 包内构造函数），或在 `spotledger.Snapshot` 增加不可由外部伪造的投影标记并在 `validSpotPosition` 校验；06-7-2 调度器内强制唯一调用路径并加契约测试。

### F3 — P3／测试给了虚假安全感｜`validSpotPosition` 的「财务状态不完整」分支不可达

```go
if !snapshot.FullyPriced && len(snapshot.UnpricedFees) == 0 { return error }
```
而 `Accounting`（`service/spotledger/snapshot.go:62`）恒有 `FullyPriced = (len(UnpricedFees) == 0)`。因此该条件在**文档指定的唯一数据来源**中永假。`TestStage0671SpotFailClosedOnInvalidDataOrRule` 的 `"inconsistent fee accounting"` 用例只把 `FullyPriced` 设为 false、保持 `UnpricedFees=nil`，验证的是人工构造的不可能状态。

真正的「财务投影不可信」由 `Accounting` 在 `projectPosition` 有 `reviewIssues` 时返回 `ErrNeedsReview`（`snapshot.go:32-34`），由调用方负责不构造 Snapshot——即又回到 F2 的契约缺口。建议把该用例改为契约级断言，并在 F2 修复后由 `Accounting` 的 error 路径覆盖。

### F4 — P3／有意安全偏离需在方案中显式标注

以下三处与 Futures 行为不同，方向均更安全，但方案 §一.3 / §3.2 宣称「与合约语义一致」，06-7-2 对照合约调试时会产生困惑：

| 项 | Futures | 06-7-1 Spot |
|---|---|---|
| 5m Kline 获取失败 | `simpleCloseStrategy` 置 `Complete=true`（**fail-open 平仓**，`line_custom.go:225-229`） | 返回 error，**fail-closed**（实施记录 §二.3 已声明） |
| `profit/loss` 为空串 | `resolveTradeROIThresholds("")` → `ParseFloat("")=0`（忽略 error）→ 阈值 0，ROI≥0 即平仓 | `checkedSpotThreshold("")` → `1000000`（禁用） |
| 规则编译/运行错误 | `CanOrderComplete` 记日志后 `continue` 跳过 | 直接 `return hold, err` |

另：`simpleCloseStrategy` 在 `lines` 长度不足时会越界 panic，Spot 侧显式校验 `len==2`，属改进。

### F5 — P3／长周期 Kline 新鲜度窗口过宽

`verifiedSpotKlines` 用 `age > 2*span + 5*time.Minute` 判 stale（`spot_rule_env.go:236`）。探针：`interval="1M"` → `span=31d`，`age=60d` 时 `allowedWindow=62d+5min`，判定 `stale=false`，即**两个月前的月 K 被视为有效**。方案 §3.1 要求「缺指标历史、最近 Spot Kline 过期…直接跳过本轮」。建议对长周期改用与 span 解耦的窗口，或按 `CloseTime` 判定，并补 `1d/1w/1M` 的边界回归。

### F6 — P3／指标样本有效性检查过弱

`hasSpotIndicatorSamples`（`spot_rule_env.go:61-81`）只要指标结构体中**任一切片** `len>0` 即返回 true，不校验该指标所需的最小样本数（如 MA(5) 至少 5 根）。`requiredSpotIndicators` 因此可能在 warmup 不足时放行。方案 §3.1 要求「必要指标样本」有效；建议按 `IndicatorConfig.Period` 校验对应 `Data` 长度。

### F7 — P3／AST 放行例外基于属性名而非宿主

`validation.go:129`：
```go
return ok && property.Value != "UnrealizedProfit" && spotFuturesOnlyIdentifiers[property.Value]
```
`UnrealizedProfit` 是无宿主限定的全局例外，任何 `X.UnrealizedProfit` 都被放行。当前 env 中只有 `Position`/`Positions` 具备该字段，**探针确认无实际绕过**（`Position.Leverage`、`Positions[0].MarginType`、`OpenStrategyHash` 均正确拒绝）；但白名单应按「宿主标识符 + 属性」组合判定，避免未来新增 env 变量时放大。

### F8 — P2（阶段边界）／06-7-2 准入条件必须显式承接

- `EvaluateEntry(coin)` 只有币种参数，**不读双开关、不检查已有 OPEN 仓位/未完成订单/UNKNOWN 状态**。探针 `PROBE0671_ENTRY_NO_POSITION_GUARD signal="BUY"`。方案 §3.1、§3.2、§五第 1/2/3/5/6/8 条的全部幂等与开关保证**目前零代码支撑**，完全由 06-7-2 调度器承担。
- `SpotSignal.RuleHash`：入场取当前 long 规则 hash、出场取当前 close_long 规则 hash，**5m Kline 兜底分支（`spot_strategy.go:293`）的 `RuleName`/`RuleHash` 为空**。方案 §3.3 计划用「币种+方向+规则 hash+K 线时间」生成可重算的 `clientOrderID`；若照此实现，用户中途编辑策略会改变 hash，同根 K 线可能生成不同 clientID；兜底路径空 hash 更会造成 ID 退化。06-7-2 必须定义稳定 ID 生成规则并加重复轮询回归。
- 零运行时接线本身是**正向**的：`main.go`/routers/loopRun 对评估器 0 引用，两个开关打开仍不会自动交易，符合 06-ALL-rereview F1 与 06-7 主方案状态行。

### F9 — P3／分层耦合加深

`feature/strategy/line`（技术指标层）新增依赖 `service/spotledger`（账本）与 `service/spotquant`（配置校验）。该包原已依赖 `service/binanceapiusage`、`service/strategy`，非首次倒置，编译无循环；但指标层现在直接引用账本 `Snapshot` 类型，与 F2 的契约问题同源。建议 06-7-2 前评估把 `SpotSignal` 评估器迁至 `service/spotquant` 或独立包。

### F10 — P3／红点归因与文档口径

实施记录 §四已诚实区分「工作区全仓测试因私有研究归档重复 `runSignals` 变红」与「`scripts/test-go.sh -short` 73 包 PASS」。本轮独立复现：**干净副本** `go test ./...` 与脚本均 EXIT=0 / 73 ok / 0 FAIL（HEAD `32d203b` 已含 06-ALL-rereview F3 修复，且副本不含私有研究归档）。报告须写「干净副本全绿」，**不得**简写为「用户工作区全仓测试通过」。

---

## 七、Gate 状态总表

| Gate | 状态 |
|---|---|
| 06-7-1 代码 / 离线 / 探针 | ✅ PASS（研发基线） |
| 06-ALL F1 自动信号链接线 | ⛔ 有意 Gate，仅评估器完成，06-7-2 才接线 |
| 06-ALL F4 前端固定发布 SHA 组合 | ⛔ 未闭（本阶段前端零改动） |
| 06-ALL F5 异常恢复运维入口 | ⛔ 未闭（仅 Go 方法 + [操作边界文档](方案06-现货异常订单恢复操作边界.md)） |
| 06-ALL F6 Testnet `-2015` / 真实 BUY·SELL·Cancel·WS / MySQL 8 P95 / 生产 v25 预检 / 成套发布回滚 / 真仓授权 | ⛔ 全部 BLOCKED |
| 发布前 Schema 检查单 §2 七项 | ⛔ 全部未勾选（承接上一轮） |
| 06-7-2 准入 | ⚠️ 有条件：F1/F2 先闭环或书面接受，F8 三条列为接线验收项 |

---

## 八、纪律与边界声明

所有 ✅ 均指**离线 / 注入 Fake / 纯本地环境 / 静态比对**通过，不自动升级为实盘结论。本轮未改 `conf/app.conf`、未执行 `sync db`、未连真实 Binance（含只读）、未连生产库、未启动服务、未操作仓位、未改 `static/` 与用户前端文件、未自动 commit/push；探针与临时二进制仅存在于副本，结束随副本删除。

**建议收尾顺序**（均需用户单独授权）：
1. 闭环 F1（方案文字对齐 + 阈值默认值/启用校验 + 永久回归）与 F2（Snapshot 来源类型化）；
2. 修 F3 测试口径，记录 F4 三处有意偏离，视优先级处理 F5/F6/F7/F9；
3. 把 F8 的开关即时生效、同币幂等、先恢复后开仓、稳定 `clientOrderID`、信号优先级写成 06-7-2 的验收清单与永久测试；
4. 06-7-3 Fake 全链路、06-7-4 真实 Testnet 凭据与小额成交/WS/手续费/dust、MySQL 8 与生产 Schema 预检、成套发布与回滚，全部通过后才讨论主网自动交易准入。

---

## 附录：关键证据位置索引

| 主题 | 位置 |
|---|---|
| 评估器 / 信号结构 | `feature/strategy/line/spot_strategy.go:19-55` |
| 入场规则顺序与 hash | `spot_strategy.go:109-145` |
| 退出阈值前置门控（F1） | `spot_strategy.go:225-242` |
| Snapshot 校验（F2/F3） | `spot_strategy.go:147-168` |
| 5m Kline 兜底（F4/F8 空 hash） | `spot_strategy.go:265-295` |
| Spot 严格环境 | `spot_rule_env.go:26-29, 98-177` |
| Kline 新鲜度（F5） | `spot_rule_env.go:211-240` |
| 指标样本检查（F6） | `spot_rule_env.go:61-96` |
| AST 禁用名单（F7） | `service/spotquant/validation.go:22-28, 113-136` |
| 新建币默认 `profit/loss=100`（F1） | `service/spotquant/repository.go:125-130` |
| Futures 退出优先级对照 | `feature/account_trade_cycle.go:145-166`、`feature/strategy/line/line_custom.go:83-181, 213-242`、`feature/trade_semantics.go:9-23` |
| Spot 数据源隔离 | `feature/strategy/line/spot_parse_env.go:37-45` |
| `Accounting` / `FullyPriced` 定义（F3） | `service/spotledger/snapshot.go:13-93` |
| 零运行时接线 | `main.go`、`routers/router.go` 全仓 Grep 0 引用 |
| 前置整改 | `CODE_REVIEW_06-ALL-rereview.md`、`tests/default_test.go`、`controllers/testdata/strategy_templates/ci-kline-long.json` |
