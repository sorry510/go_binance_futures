# CODE_REVIEW_06-7-ALL：06-7 现货量化自动策略执行链路整体复审

日期：2026-11-13（20:49 起）
后端：`feat/trade_spot`，HEAD **`2c7c2b8 feat: 06-7-all`**，**工作区完全干净（0 项）**
前端：本地 HEAD `cf319e1 feat: spot`，工作区干净，本轮未修改前端
主方案：[开发方案-06-7-现货量化自动策略执行链路.md](开发方案-06-7-现货量化自动策略执行链路.md)（§一/§三/§四/§五/§六）
历轮报告：[CODE_REVIEW_06-7-1.md](CODE_REVIEW_06-7-1.md) + `-rereview.md`、[CODE_REVIEW_06-7-2.md](CODE_REVIEW_06-7-2.md) + `-rereview.md`、[CODE_REVIEW_06-7-3.md](CODE_REVIEW_06-7-3.md) + `-rereview.md`、[CODE_REVIEW_06-7-4.md](CODE_REVIEW_06-7-4.md)（全部保留、不覆盖）
本轮新增依据：[CODE_REVIEW_06-7-统一正式接线.md](CODE_REVIEW_06-7-统一正式接线.md)、[真实 Spot Testnet 订单与 WS 验收记录](开发方案-06-7-4-真实SpotTestnet订单与WS联调-验收记录.md)、[Testnet 自动 Runner 接线记录](开发方案-06-7-4-测试网自动策略Runner接线-实施记录.md)、[外部只读脱敏证据.json](CODE_REVIEW_06-7-4-外部只读脱敏证据.json)、[临时 MySQL8 验证证据.txt](CODE_REVIEW_06-7-4-临时MySQL8验证证据.txt)、[生产只读预检.sql](开发方案-06-7-4-生产只读预检.sql)

---

## 结论摘要

| 维度 | 结论 |
| --- | --- |
| 代码与离线 Gate | **✅ PASS**（682 个 Go 文件全等；全系列专项、count=5、10 包 test/race、vet/build/diff-check、干净副本 73 ok + 13 无测试 + 0 FAIL、脚本 EXIT=0） |
| 真实 Spot Testnet 小额 BUY→SELL→CLOSED + signed WS + myTrades + 自动 Runner | **✅ 已验**（脱敏日志留证，opt-in 测试默认 SKIP，标准回归不联网） |
| 06-7-1 / 06-7-2 / 06-7-3 阶段 | **✅ 闭环**（历史 Findings 均由 rereview 收敛） |
| 06-7-4 阶段 | **⚠️ 部分闭环**：Testnet 与离线证据已补，**生产 / 发布 Gate 仍全部 BLOCKED** |
| **06-7 整体完成** | **❌ 不认可** |
| 主网自动下单代码路径 | **⚠️ 已解锁**（`StartSpotQuantTrade` 现注入真实执行器，`testnet=false` 走 `https://api.binance.com`），但**方案验收标准未同步修订、独立书面部署授权与全部生产 Gate 未满足** → 新增 **F1 P1 治理风险** |
| 主网真实订单、生产 v25 物理预检、生产 MySQL 8 P95、ARM/Linux 成套发布与回滚、异常恢复运维入口 | **⛔ 全部 BLOCKED** |

一句话：**06-7 的代码链路第一次真正接通了（2 秒调度 → 双开关 → 同环境行情/执行 → WS 门禁 → 真实 Processor → Fill 账本），真实 Testnet 小额闭环也有脱敏日志证据，这是历轮最完整的一次；但「接通」不等于「验收完成」——实现把实单授权缩减为两个业务开关并解除了主网硬拒，而主方案 §一原则 7、§3.4 准入 5、§4 完成标准仍明文禁止这种放行，且同一文档内存在「仍不注入执行器」与「以本节为准」的自相矛盾。生产 Gate 一项未过，本轮还发现预检 SQL 与运行时 clientID 口径相反、终态 SELL 订单锁死已 OPEN 仓位两个跨阶段一致性缺陷。**

---

## 一、审计范围与基线

### 1.1 相对上一轮（12:59 的 06-7-4 复审）的变化

`git log -10`：`2c7c2b8 feat: 06-7-all` → `a19efe4 feat: 06-7-4` → `3e1a79f feat: 06-7-3`。**工作区从 8 项未提交变为 0 项**，本次为纯 HEAD 审查。

| 变化 | 内容 |
| --- | --- |
| `a19efe4` | 15 files / +1395 −19：06-7-4 报告与实施记录、脱敏证据 JSON、MySQL8 合成数据脚本与验证证据、预检 SQL（136 行 PASS/FAIL 断言）、`spot_quant_runner.go`（+90）、0674 集成测试（+373）、`spot_quant_adapter.go`（新增）、`conf/app.conf.example`（+6，独立 Spot Testnet 密钥字段） |
| `2c7c2b8` | 18 files / +1427 −62：新增 `spot_quant_runtime.go`（177）、`spot_quant_market_strategy.go`（69）、`spot_quant_market.go`（103）、统一接线复评、真实 Testnet 验收记录、Testnet 自动 Runner 记录、`spot_quant_testnet_live_e2e_test.go`（368）；`main.go`（7 行注释）、`spot_quant_runner.go`（+78）、`spot_quant_stream.go`（+66）、`spot_quant_adapter.go`（+69） |

**核心变化**：`StartSpotQuantTrade` 从 `return ctx.Err()` 改为 `return spotQuantRunLive(ctx)`；新增生产 Runtime 构造真实 `QuantAdapter` + `Processor` + `SpotQuantRunner`；`NewSpotQuantTradingAdapterFromConfig(false)` 返回 `allowOrderMutation=true` 的主网适配器。此前所有报告反复守住的「生产零接线 / 主网硬拒」边界**在代码层已被解除**。

### 1.2 隔离副本与等价校验

工作区干净，副本 `/tmp/phase_audit_067all` = 纯 `git archive HEAD`（无 M/?? 同步）；`conf/` 仅 `app.conf.example`，无 `strategy_templates/` 研究目录。

```
TRACKED_GO_FILES 682
GO_FILES_IDENTICAL 682 MISSING 0 DIFFER 0 COPY_ONLY 0
```

### 1.3 离线 Gate 实证（隔离副本，清空全部 opt-in 许可 env）

| Gate | 结果 |
| --- | --- |
| 06-7 全系列专项 | `./feature` `^(TestStage067\|TestReview067\|TestSpotQuant)`、`./feature/strategy/line` 0671 系列、`./spot/api/binance` 064/Testnet config 系列 → **全部 PASS** |
| 重复稳定 | `./feature` 全系列 `-count=5 -p=2` → **PASS** |
| 10 包 test / race | 显式 10 包 `-count=1` / `-race` → **10/10 ok，无 DATA RACE** |
| vet / build / diff-check | 10 包 `go vet`、`go build -o /tmp/spot067all_review_build .`、工作区 `git diff --check` → **EXIT=0** |
| 干净副本全量 | 探针删除后 `go test ./... -count=1` → **73 ok + 13 无测试 + 0 FAIL，EXIT=0** |
| 项目脚本 | `./scripts/test-go.sh -short` → **EXIT=0 / 73 ok** |
| opt-in 隔离 | 未设置许可 env 时 `TestSpotQuantLiveTestnetBUYSELLAndWS`、`TestStage0675RealTestnetAutomaticRunnerCycle`、`TestSpotQuantTestnetMarketPreflight` **全部 SKIP** → 标准回归不访问网络、不发单 |

> 审计过程说明：首轮把 `env -u ...` 整串放进变量后由 zsh 当作单个命令名，8 项 Gate 全部报 `EXIT=127 / file name too long`。这是**审计命令自身错误，不是代码缺陷**；改为 shell 层 `unset` 后重跑全部 EXIT=0。

---

## 二、逐阶段最终判定

| 阶段 | 交付 | 最终状态 |
| --- | --- | --- |
| **06-7-1** 现货策略规则评估 | `line.SpotRuleEvaluator`、`LoadVerifiedSpotRuleEnv`、严格 Spot ticker/24h OHLC/Kline 新鲜度/指标完整性 | **✅ 闭环**。默认 `Profit/Loss="100"` 语义按首轮 F1 裁决保留并文档化；F2/F3/F5/F6/F7 由 rereview 收敛。当前生产经 `SpotMarketStrategy` 与执行链使用同一 `QuantAdapter` 行情源，06-7-3 F3（Fake 环境替换）也已闭环 |
| **06-7-2** 两秒自动轮询接线 | `main.go` 2 秒 `loopRun` + `RunSpotQuantProtected`、双开关、Kline 事件 clientID v4、Fill 投影权威、全部未结订单对账 | **✅ 闭环**。原 F1「生产 `exec=nil`」在 `2c7c2b8` 被接线替代，但**替代方式与阶段验收标准冲突**（见 F1） |
| **06-7-3** 真实 Processor + Fake Exchange 全链路 | 10 个 0673 顶层测试 + rereview 的 5 个整改用例，覆盖 BUY→SELL→CLOSED、部分成交、UNKNOWN、-2013、迟到 myTrades、BNB fee、dust、双币隔离 | **✅ 闭环**。首轮 F1（双时钟）/F2（超成交静默空转）/F6（双币真实 Processor）已由 rereview 收敛并进入 `3e1a79f` |
| **06-7-4** Testnet 与交付 Gate | 严格 Fake 行情 E2E、3d 拒单、临时 MySQL8 v25/EXPLAIN、只读预检 SQL、真实 Testnet BUY/SELL/WS、统一正式接线 | **⚠️ 部分闭环**：Testnet 侧与离线证据侧已补齐（详见 §四）；**生产 MySQL 8 P95、生产 v25 物理预检、ARM/Linux 成套发布与回滚、主网真实订单、异常恢复运维入口仍 BLOCKED** |

**06-7-4 上一轮 Findings 的最终闭环状态**：

| 上轮 Finding | 状态 |
| --- | --- |
| **F1 P1** 3d 门禁阻断 SELL | **✅ 已闭环**。`runSymbol` SELL 分支（:513-544）不再调用 `resolveSpotEventOpen`，改用 `spotQuantExitClientID(symbol, positionID)` 固定身份；新增 `TestReview0674ThreeDayMissingSourceAllowsVerifiedPositionSELL`、`TestReview0674StrictThreeDayExitSignalUnblocked`、`TestReview0674ExitClientIDStableAcrossClockAndTimeframe` 等永久回归 |
| **F2 P2** 阶段目标未达成 | **⚠️ 部分**：真实 Testnet 已验，生产 Gate 仍未达成 |
| **F3 P2** 外部证据不可复核 | **✅ 基本闭环**：新增脱敏 JSON、MySQL8 合成数据脚本 + 验证证据（含 `sync db` EXIT=0 日志、10 DECIMAL、4 条 `EXPLAIN ANALYZE`）、两份真实 Testnet 脱敏日志；详见 F4 剩余缺口 |
| **F4 P3** 3d 生产永久禁用 | **✅ 已闭环**：`newSpotQuantRuntime` 注入 `runner.spot3dOpen = SpotMarketThreeDayOpen(exchange)`，从当前交易所环境 3d Kline 取真实 OpenTime |
| **F5 P3** 预检 SQL 口径/人工判读 | **⚠️ 部分**：PASS/FAIL 断言与哈希绑定已加，但**前缀口径反转**，见 F2 |
| **F6 P3** 评估价/执行价两源 | **✅ 已闭环**：`SpotMarketStrategy` 用同一 `exchange.MarketSnapshot` / `MarketKlines` |
| **F7 P3** 06-ALL 未决项 | 见 §五 F5 |

---

## 三、06-7 整体目标核对（主方案 §一 / §五 / §六）

| 契约 | 核对 |
| --- | --- |
| §一目标：像合约一样，开启总开关 + 单币开关后自动按周期读币种、算指标、LONG BUY / CLOSE_LONG SELL | **✅ 代码层已实现**：2 秒调度 → `spotQuantRunLive` → 双开关 → Runtime（WS 门禁）→ Runner（`SpotMarketStrategy` + `Processor`）→ `SpotManaged` 账本。真实 Testnet 自动 Runner 5 tick / ~37 秒完成 BUY→SELL→CLOSED 有日志留证 |
| §五 #1/#2 双开关即时生效、20 轮不重单 | **✅** 06-7-2 / 06-7-3 永久回归 + `DBGate` 每次重读 |
| §五 #3/#4 部分成交只认 Fill、只卖策略净持有量 | **✅** 06-3 `projectPosition` 权威；真实 Testnet 0.00028 BTC 闭环、未动测试账户 1 BTC 手工余额 |
| §五 #5 BTCUSDT/WBTCUSDT 精确隔离 | **✅** `TestReview0673TwoSymbolsThroughRealProcessorStayExactAndIndependent` |
| §五 #6 UNKNOWN / REST 失败 / dust fail-closed | **✅** 06-7-3 用例 + 新 WS 门禁；真实交易所的部分成交/超时/429/418 长稳**未验**（验收记录 §5 自认） |
| §五 #7 Spot 指标/价格/ROI 均来自 Spot、Futures 不受影响 | **✅** `SpotMarketStrategy` 只用 Spot 适配器；Futures `loopRun` 未改 |
| §五 #8 总开关关闭不自动卖出、异常订单人工可核对 | **⚠️ 部分**：不自动卖出已实现；「人工可核对」仍缺 HTTP/CLI/UI 恢复入口（06-ALL F5），且 F3 使锁死状态更难处理 |
| §五 #9 不新增独立菜单/任务中心/模式/Owner/审核表 | **✅** 无新表、无第三开关、无 Owner/审批状态机 |
| §五 #10 明确区分「代码已接线」与「Testnet/主网已真实验收」 | **⚠️ 不满足**：主方案 line 156 仍写「主程序 StartSpotQuantTrade 仍不注入真实执行器」，line 166 才写「以本节为准…已被最新实现替代」，同一文档内自相矛盾；见 F1 |
| §六 明确不做 | **✅** 未引入独立 cron / worker 池 / 队列 / AI 决策 / 自动选币 |

**「开发完成」结论**：按 §一原则 7、§3.4 准入 5、§4 完成标准，**不认可 06-7 整体完成**。理由不是代码缺失，而是（a）实现与未修订的验收标准直接冲突；（b）生产/发布 Gate 全部 BLOCKED。

---

## 四、正向结论（已实证）

1. **完整链路接通且环境一致**：`spotQuantRunLive`（`spot_quant_runtime.go:126-177`）先检查 `orm.GetDB("default")`、读 `config.spot_quant_enable` 与启用币种，再 `sync.Once` 构造 Runtime；`newSpotQuantRuntime` 用同一 `testnet` 选择端点与凭据、同一 `QuantAdapter` 供 `SpotMarketStrategy`（信号）与 `Processor`（执行），注入 `spot3dOpen` 与 `orderPrefix`（`sqt`/`sqm`）。
2. **WS 门禁有效**：`spotQuantLiveExecutor.ExecuteBuy/ExecuteSell` 在 `wsReady == false` 时返回 `spotexecution.ErrDisabled`；`Reconcile` 在断线时仍允许 REST 只读恢复；`streamLoop` 5 秒重试并在失败时 `ready.Store(false)`。`spot_quant_stream.go` 对「订阅 ACK 前收到事件」返回错误。
3. **跨环境防误卖 + 身份防伪造**：`runSymbol` 检查已有 BUY 订单前缀必须匹配当前 `orderPrefix`；`hasBUYInEvent` 用 `spotQuantClientID` 重新计算并比较 `check[3:]`，语法合法的伪造 ID 也会 fail-closed（`TestReview0674SyntacticallyValidForgedV4OrderFailsClosed`）；`spotQuantSafeMessage` 脱敏签名 URL（日志实证 `signature=[REDACTED]`）。
4. **真实 Testnet 证据可复核**：`/tmp/spot_testnet_live_buy_sell_ws_run.log`（17:50，26.78s）记录 signed WS ACK、MARKET BUY/SELL `FILLED`、1 条 Fill、CLOSED、剩余 0、WS executionReport buy/sell 均见；`/tmp/spot_quant_testnet_autorun_live_20261113.log`（19:54，37.10s）记录自动 Runner `ticks=5 BUYFill=true SELLFill=true CLOSED=true`；两份日志均无 `signature=`/`api_key=`/secret 字面量。验收记录给出 BUY `13340589` / SELL `13340593`、0.00028 BTC、手续费 0。
5. **离线与临时 MySQL8 证据补齐**：`CODE_REVIEW_06-7-4-临时MySQL8验证证据.txt` 含 MySQL 8.0.46 原始 SELECT、`sync db` EXIT=0 与日志行、v25 / 500 周期 / 1000 订单 / 1000 Fill、索引清单、10 个 `decimal(36,18)`、4 条 `EXPLAIN` 与 `EXPLAIN ANALYZE`（明确标注非 P95）、合成脚本 SHA；预检 SQL 已改为输出 `PASS/FAIL` 且对缺表、config 多行、10 DECIMAL、超成交、非法 BUY ID 均有显式门控。
6. **06-ALL F4 接近闭环**：后端 HEAD `2c7c2b8` 且工作区干净、前端 `cf319e1` 且工作区干净、`dist`↔`static` **115/115 SHA256 全等**（`index.html` = `612b3b18…`），首次存在可引用的前后端固定 SHA 组合。
7. **标准回归不联网**：清空 opt-in env 后全量 73 ok / 0 FAIL，3 个真实 Testnet 用例全部 SKIP。

---

## 五、Findings

### F1 · P1 · 方案契约与实现冲突：主网自动下单路径已解锁，验收标准未修订

**代码事实**（HEAD `2c7c2b8`）：

```
main.go:449-455  2 秒 loopRun → RunSpotQuantProtected → StartSpotQuantTrade
runner.go:589-591  StartSpotQuantTrade → spotQuantRunLive(ctx)
runtime.go:126-176 读 config.spot_quant_enable + 任一启用币 → newSpotQuantRuntime
adapter.go:55-73  testnet=false → 主网 api_key/api_secret + https://api.binance.com，
                  NewSpotQuantAdapterWithClient(client, true)  ← allowOrderMutation=true
```

即**只要部署后的 `app.conf` 为 `testnet=false`、全局开关 ON、任一币开关 ON、WS 签名订阅 ACK 成功，系统就会自动向 Binance 主网发 MARKET 单**。此前 06-4 的「主网硬拒」、06-7-4 历史版本的 `SPOT_QUANT_TESTNET_AUTOTRADE` / 独立测试库匹配 / BTCUSDT 与 20 USDT 上限，均在「统一正式接线」中被明确删除。「实单授权」被缩减为两个业务开关 + `DBGate` POST 前复查，**没有持久化授权凭据（签名/有效期/操作者/撤销）、没有 append-only 模式切换与解冻审计**。

**契约冲突**：主方案（同一文件）仍保留以下未修订条款：

| 位置 | 原文要点 |
| --- | --- |
| §一原则 7（:19） | 「真实 Testnet、生产发布、主网下单需单独验收与授权，**不得靠翻业务开关绕过**」 |
| §3.4 准入 5（:116） | 「Mainnet 下单硬限制、生产库和部署 Gate **不因本阶段接线而解除**」 |
| §4 阶段表（:125） | 「未有真实 Testnet 证据与**单独部署授权**前，**不解除 Mainnet 订单硬限制、不启动主网自动交易**」；标题仍写「真实 Testnet/生产继续 BLOCKED」 |
| §5 补充（:156） | 「主程序 `StartSpotQuantTrade` 仍不注入真实执行器；自动轮询下单…仍未完成」 |
| §5 补充（:166） | 「以本节为准：旧文档…**主程序永远不下单**的描述，均是历史开发阶段事实，已被最新实现替代」 |

line 156 与 line 166 直接矛盾；§一/§3.4/§4 的硬性禁令未被修订，只在 §5 末尾以「以本节为准」覆盖。`CODE_REVIEW_06-7-统一正式接线.md` 也自述「没有使用正式 Binance API Key 执行主网 BUY/SELL」「只有重新构建并部署到目标服务器后才会改变服务器行为」——即**代码已具备能力，靠「未部署」这一事实性状态而非代码门禁保证安全**。

**要求（部署前必须裁决）**：由用户明确书面选择其一——
1. **修订方案**：同步改写 §一原则 7 / §3.4 准入 5 / §4 完成标准 / line 156，并补「持久化部署授权 + 生产 v25 检查单 + MySQL 8 P95 + 成套发布/回滚 + 异常恢复入口」作为新的发布 Gate；或
2. **恢复约束**：保留主网硬拒或部署级授权，使「两个业务开关」不足以单独解锁主网自动下单。

在此之前不得判定「06-7 完成」，不得把代码部署到 `testnet=false` 的生产环境，不得打开双开关触发主网自动交易。

### F2 · P2 · 生产只读预检 SQL 与运行时 clientID 身份口径完全相反

预检 SQL（`:107-110`）用 `^sq4(01|03|...)[0-9a-z]{7}[0-9a-f]{22}$` 判定 `buy_client_id_gate`；而统一接线后 `buyClientID`（`runner.go:207-213`）把 `sq4` 前缀替换为 `orderPrefix`（`sqt` / `sqm`），`spotStoredBUYEvent`（:223-243）接受 `sq4|sqt|sqm` 三种前缀。隔离探针：

```
PROBE067ALL_PREFIX orderPrefix=""    id=sq4010tofw005f733f974f353b30d80dbd len=34 runner_parseable=true preflight_sq4_regex=true
PROBE067ALL_PREFIX orderPrefix="sqt" id=sqt010tofw005f733f974f353b30d80dbd len=34 runner_parseable=true preflight_sq4_regex=false
PROBE067ALL_PREFIX orderPrefix="sqm" id=sqm010tofw005f733f974f353b30d80dbd len=34 runner_parseable=true preflight_sq4_regex=false
```

后果是双向误判：**生产发布后所有新 BUY 订单（`sqt/sqm`）都会被预检判为 `invalid_buy_id` → `buy_client_id_gate=FAIL`**；反过来，预检认为格式合法的 `sq4` 旧 ID，Runner 在 `orderPrefix != ""` 时会返回 `ErrNeedsReview`。运维若据预检 FAIL 去「人工核对/清理历史订单」，存在误改不可变账本的风险。

**要求**：SQL 正则改为 `^(sq4|sqt|sqm)` 并对 `sq4` 单独输出「需人工复核」而非计入合法；哈希绑定校验（`SHA2(CONCAT('spot_quant_0673_v4|',...))`）与前缀替换后的身份语义对齐；补一条用真实 `sqt/sqm` 订单跑预检的回归。

### F3 · P2 · 终态（CANCELED）历史 SELL 订单使已 OPEN 仓位永久锁死

`runSymbol` SELL 分支（`:517-526`）遍历该仓位全部订单，只要存在 `Side == "SELL"` 就返回 `ErrNeedsReview`，**不区分订单状态、不检查是否有 Fill**。隔离探针（真实 Processor + SQLite Ledger，插入一笔 `CANCELED`、零 Fill 的历史 SELL）：

```
PROBE067ALL_TERMINAL_SELL err=BTCUSDT: ... Spot position 1 remains OPEN after prior SELL order 2; review the remaining Fill and dust needs_review=true places=1
PROBE067ALL_TERMINAL_SELL_CONTROL err=<nil> places=2 status=CLOSED
```

撤单后重新平仓是正常业务场景；当前实现要求人工改库才能恢复，而 **06-ALL F5「异常订单恢复运维入口」仍未交付**，故该仓位处于「锁死且无合规恢复路径」状态。`ReportSpotQuantScheduler` 按错误文本去重日志，运维侧可能只看到一条告警。

**要求**：仅对有实际 Fill 或活动状态（`NEW/PARTIALLY_FILLED/PENDING_CANCEL/FILLED`）的 SELL 阻断；对 `CANCELED/EXPIRED/REJECTED` 且零 Fill 的 SELL 允许生成补卖 clientID；或提供受控的 HTTP/CLI 恢复入口（同时闭环 06-ALL F5）。补永久回归。

### F4 · P2 · 外部证据仍有时序与完整性缺口

| 证据 | 状态 |
| --- | --- |
| Testnet 3d klines 原始响应、`-2015` 401 原始状态码/时间/错误文本 | ✅ 可复核 |
| MySQL8 `sync db` EXIT=0、10 DECIMAL、EXPLAIN ANALYZE、合成脚本 SHA | ✅ 可复核（明确标注非 P95） |
| 两份真实 Testnet 脱敏日志 | ✅ 可复核（无密钥字面量） |
| **Mainnet 3d Kline 原始响应** | **❌ 缺失**：脱敏 JSON 中 `spot_mainnet` 为 `network_error: "URLError"`，但 06-7-4 实施记录 §2 仍写「Mainnet 11-07/11-10/11-13」并据此得出「两环境锚点不同」结论 |
| MySQL8 证据末尾状态 | ⚠️ 写「Production DB preflight, production performance P95 and **real Testnet remain NOT VERIFIED**」（13:17 生成，早于 17:50 / 19:54 两次真实 Testnet 验收），与后续记录时序冲突 |
| 生产库只读预检结果 / 检查单 §2 七项 | ❌ SQL 就绪但未在目标库执行，七项 0 勾选 |

**要求**：补主网 3d 公开行情原始响应，或把结论降级为「仅 Testnet 实测、主网锚点未验」；统一证据时间戳与状态表述，避免把历史快照当成当前事实。

### F5 · P3 · 06-ALL 未决项的最终状态

| 项 | 状态 |
| --- | --- |
| F1 自动信号链运行时无入口 | **✅ 已闭环**（`StartSpotQuantTrade` 现调用真实 Runtime），但闭环方式触发本轮 F1 |
| F4 固定发布 SHA 组合 | **⚠️ 接近闭环**：前后端 HEAD 均已提交且工作区干净、dist/static 115/115；仍**无 ARM Linux 生产构建物、备份与回滚演练** |
| F5 异常恢复运维入口 | **❌ 仍 BLOCKED**：无 HTTP/CLI/UI；F3 使其更紧迫 |
| F6 实盘 / 发布 Gate | **❌ 全部 BLOCKED** |

### F6 · P3 · 其他边界

- `spotQuantRuntimeState` 用 `sync.Once` 缓存实例，`streamLoop(context.Background())` 使用全局 context，WS goroutine 不受请求 ctx 或服务生命周期控制，无法优雅停止；`testnet` 改变需重启（文档已注明）。
- `spotQuantExitClientID` 固定 `sqe` 前缀、不带环境身份；跨环境隔离目前依赖 BUY 前缀检查（有效），若未来同一库混用两个环境需复核。
- 自动 Runner 真实 Testnet 日志收尾出现一次 `user stream requires REST reconciliation ... context canceled`；长期 WS 重连、429/418、真实部分成交与极端网络超时未做实网演练（验收记录 §5 已承认）。
- 预检 SQL 检查单 §2 七项仍 0 勾选；本次审计未对任何真实库执行 SQL。

---

## 六、实盘 / 发布 Gate（全部 BLOCKED）

| Gate | 状态 | 依据 |
| --- | --- | --- |
| 主网真实 BUY/SELL / 主网 signed WS | ⛔ | 「统一正式接线」§五自述未使用正式 API Key 发单；代码路径已具备但无任何主网成交证据 |
| 生产 MySQL 8 物理 Schema / P95 | ⛔ | 仅临时合成 500 周期兼容性证据，明确非 P95；生产库未只读核验 |
| 生产 v25 发布前 Schema 检查单 | ⛔ | SQL 已就绪且带 PASS/FAIL，但 `开发方案-06-发布前Schema验证检查单.md` §2 七项 0 勾选 |
| 独立书面部署授权 + 持久化凭据 / append-only 审计 | ⛔ | 无；实单授权被缩减为两个业务开关（F1） |
| 二进制 + static + schema 成套发布 / 备份 / 回滚 | ⛔ | 前后端 SHA 组合已可引用，但无 ARM Linux 产物、无部署与回滚演练 |
| 异常订单恢复运维入口 | ⛔ | 06-ALL F5 未闭；F3 显示锁死场景无合规恢复路径 |
| 真实交易所长稳 / 多币 / BNB fee / dust / 429/418 | ⛔ | 验收记录 §5 自认仅单币 BTCUSDT、20 USDT、零手续费一次闭环 |
| 真实 Spot Testnet 小额 BUY→SELL→CLOSED + signed WS + myTrades + 自动 Runner | **✅ 已验** | 脱敏日志 + 验收记录；这是本轮相对 06-7-4 复审的唯一实网 Gate 闭环 |

---

## 七、后续准入条件（按优先级）

1. **F1 必须先裁决**（P1）：书面选择「修订方案验收标准并补发布 Gate」或「恢复主网硬拒/部署级授权」；同时消除主方案 line 156 与 line 166 的自相矛盾。裁决前不得部署、不得打开主网双开关。
2. **F2 / F3 必须闭环**（P2）：预检 SQL 与 `sqt/sqm` 身份对齐；终态无 Fill SELL 不得锁死已 OPEN 仓位，并补受控恢复入口（同步闭环 06-ALL F5）。
3. **F4 补证据**（P2）：主网 3d 原始响应或结论降级；统一留证时序。
4. **生产 Gate 顺序**：生产库只读预检（7 项勾选留证）→ MySQL 8 真实体量 EXPLAIN/P95 → ARM Linux 构建 + static + schema 成套发布与回滚演练 → 持久化部署授权 → 最后才考虑主网真实小额验证。禁止因 Testnet 成功而跳过。
5. **06-7 无后续阶段编号**：整体完成只能由「方案契约裁决 + 生产 Gate 全部通过 + 独立书面授权」共同确认，不能由更多离线测试或一次 Testnet 闭环替代。

---

## 八、最终结论与审计边界

**06-7 的代码与离线 Gate 全部通过，真实 Spot Testnet 小额自动交易闭环有脱敏日志证据，是 06 系列迄今最完整的研发基线；但整体复审结论为 ❌「不认可 06-7 整体完成」：主网自动下单代码路径已解锁而方案验收标准未同步修订、独立书面部署授权与全部生产/发布 Gate 未满足，且新增 F1（P1 治理风险）、F2/F3/F4（P2 跨阶段一致性缺陷）尚未闭环。可以合并本地代码；不得宣称现货量化自动实盘可用、不得部署到 `testnet=false` 生产环境、不得打开双开关触发主网自动交易。**

**审计边界**：本次未修改任何业务代码、未自动 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/` 与 `static/`、未对真实库执行 `sync db` 或预检 SQL、未访问真实 Binance（Testnet 声明仅按脱敏日志与验收记录核对，opt-in 真实测试未运行）、未重启服务、未操作生产仓位。隔离探针只写在 `/tmp/phase_audit_067all` 副本内，删除后复验干净副本 `go test ./... -count=1` 为 73 ok / 0 FAIL；副本与本次临时产物在审计结束删除，开发侧 `/tmp/spot_*` 留证只读核验未动。本报告为新增文件，未覆盖任何历史 `CODE_REVIEW_*` 报告。
