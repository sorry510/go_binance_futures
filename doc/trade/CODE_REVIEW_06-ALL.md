# Code Review：方案 06 整体复审（06-0 ～ 06-6）

日期：2026-11-12  
后端：`feat/trade_spot`，HEAD **`35da095 feat: 06-6`**，工作区干净（Schema `appversion.DatabaseSchemaVersion=25`）  
前端：`go_binance_futrues_new_ui`，HEAD `89f15bd`，分支 `feat/trade_spot`，**7 个已修改 + 3 个未跟踪（共 10 项）业务改动未提交**  
审计对象：用户提出「现在 06 整体完成了，整体再 review 下」；依据 [方案 06](开发方案-06-现货量化交易-复刻合约简化版.md) §6 阶段验收、§8 验收总标准、[06-6 实施记录](开发方案-06-6-完整联调与交付-实施记录.md)、[发布前 Schema 检查单](开发方案-06-发布前Schema验证检查单.md) 及 06-0/06-1 ～ 06-5 全部首轮与整改复评报告。  
副本：`/tmp/phase_audit_06all`（纯 `git archive HEAD`，后端工作区干净），**666 个 Go 文件 SHA256 全等、mismatch=0**；探针/临时二进制仅存在于副本，结束随副本删除。  
本报告**不覆盖任何历史报告**，历轮结论原文保留。

---

## 一、结论先行

> **不认可「方案 06 整体完成」。**
> 代码与离线 Gate 是一份**合格、可继续研发的基线**；但方案 §8「开发完成」总标准**未满足**，方案 §5 描述的完整自动交易链**在运行时没有任何入口**，阶段文档自身与 06-6 实施记录也明确 Testnet / 生产发布 Gate 全部 BLOCKED。
> 可以合并本地代码；**不得宣称上线、不得启用自动现货量化、不得发主网订单**。

| 维度 | 结果 | 说明 |
|---|---|---|
| 代码与离线 Gate（test / race / vet / build / diff-check / 探针） | **✅ PASS** | 详 §五；7/7 + race 7/7、54 条 06 系列永久回归、06-6 Fake E2E 2/2、Futures/旧 Spot race 6/6 |
| 阶段 Findings 代码级闭环 | **✅ 基本闭环** | 06-F1、06-3-F1/F2、06-4-F1/F2、06-5-F1/F2/F3/F4/F5/F7 均有独立复核与永久测试，详 §四 |
| 方案 §8「开发完成」八项总标准 | **❌ 不满足** | 「有真实测试结果」「Spot BUY/SELL 正确」仅有 SQLite/Fake；自动信号链未接线，详 §三、F1 |
| 前后端成套发布可复现性 | **⚠️ 有条件** | `dist↔static` 115/115 全等且清单一致，但前端源码未提交、产物无法从任何已提交 SHA 复现，详 F4 |
| 生产 / 实盘 / Testnet Gate | **⛔ 全部 BLOCKED** | Testnet 签名账户 -2015、MySQL 8 P95、生产 v25 预检、二进制+静态+Schema 成套发布与回滚、真仓授权，详 §八 |
| 是否放行 06-6 之后的自动信号接线 / 主网准入 | **⛔ 不放行** | 需先关闭 F1/F5 与 §八 全部 P1 Gate，并经用户单独明确授权 |

**一句话**：06 **做完了「可以安全研发的代码」，没有做完「可以运行的现货量化交易系统」**；把「离线全绿」表述为「整体完成」会直接误导发布决策。

---

## 二、审计范围与基线

### 2.1 本次提交链

```
3f85add feat: spot 06-01  →  7ed6c93 (merge master)
  → 2e374b4 fix: add 06-2 → af07191 Merge → 6d8eecc feat: 06-3
  → f2f3c9b feat: 06-4 → 5eb10a2 feat: 06-5 → 35da095 feat: 06-6  (当前 HEAD)
```

后端 `git status --porcelain` 为空，故副本用纯 `git archive HEAD` 即可等价；等价校验 `find` 666 个 Go 文件、`git ls-files '*.go'` 666 个、逐文件 SHA256 mismatch=0、工作区无未跟踪 Go 文件。

### 2.2 已知环境前提（必须复述，否则会误判红点）

- `conf/app.conf` **未入 Git**（干净副本只有 `app.conf.example`）→ 干净副本中 `tests` 包必 panic；
- `strategy_templates/` 整目录被内层 `.gitignore`（内容 `*`）忽略，但 `controllers/strategy_template_static_test.go` 会 `filepath.Walk("../strategy_templates")`；本次在副本内同步**剪除 `research/`** 的 53 个文件后 `controllers` PASS；
- `strategy_templates/research/**` 全仓 `os.walk` / `find` 会触发 SIGTERM（Exit 137），本次已显式剪枝；
- macOS 链接器 `ld: malformed LC_DYSYMTAB` 为非致命警告，退出码 0 即通过；
- zsh 中 `$PKGS` 不自动分词，所有 `go test/vet` 包列表均显式写全（首次两次调用因该问题 setup failed，已重跑）。

### 2.3 审计纪律

未改 `conf/app.conf`、未执行 `sync db`、未连生产库、未访问真实 Binance（含只读）、未启动服务、未操作仓位、未自动 commit/push、未运行 `pnpm build`（会覆盖用户 `dist`）、未碰 `static/` 与用户前端未提交文件。所有测试进程显式清空 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL` / `SPOT_LEDGER_MYSQL_DSN`。

---

## 三、方案 §8「开发完成」总标准逐项复核

方案原文：「全局 OFF 安全生效、每币独立开关工作、Spot BUY/SELL 正确、持仓不混入手动库存、订单 Fill/PnL 可追溯、四个菜单操作与合约 UI 一致、未破坏任何合约表和旧 Spot 业务，**且有真实测试结果**。仅构建成功或页面能显示，不能认定完整完成。」

| # | 标准 | 本轮判定 | 证据 |
|---|---|---|---|
| 1 | 全局 OFF 安全生效 | ✅ 代码级 PASS / 实盘未验 | `gate_db.go` 每次读 `config.spot_quant_enable`；`TestStage064DBGateAllSwitchStates`、`TestReview064GateOffBeforeAnySellREST`；06-6 只读核验实验库总开关=0、809 币全 OFF |
| 2 | 每币独立开关工作 | ✅ 代码级 PASS | `spot_symbols.spot_quant_enabled` 独立列，双开关 AND 门控；01/02 旧 `enable` 未复用 |
| 3 | **Spot BUY/SELL 正确** | **⚠️ 仅 Fake 正确** | `TestStage066EndToEndBUYSELLLedgerReadModelAndSecondCycle`、`TestReview064FullSellClosesCycleAndNewBuy` 全 PASS；**真实 Testnet BUY/SELL 因 -2015 未执行** |
| 4 | 持仓不混入手动库存 | ✅ 代码级 PASS | `TestStage064FakeBuySellAndWalletIsolation`、`TestStage064WalletShortfallBlocksSELLWithoutManualFallback` |
| 5 | 订单 Fill / PnL 可追溯 | ✅ 代码级 PASS | 06-3 `projectPosition` 唯一金融权威、终态不可改写、`DECIMAL(36,18)`、逐 `trade_id` 去重 |
| 6 | 四个菜单操作与合约 UI 一致 | ✅ 静态 PASS | 13 条 `/spot-quant/*` 路由 ↔ `src/api/spotQuant.ts` 13 个字面量一一对应；四菜单 + 中英 i18n key 齐备；`spotQuant-BnAMNvmr.js` 已入 `static/static/js/` |
| 7 | 未破坏合约表和旧 Spot 业务 | ✅ 离线 PASS | Futures/旧 Spot race **6/6 PASS**；`TestStage066*` 含 Futures 表哨兵未变断言；v25 回归验证 Futures 历史行数不变 |
| 8 | **且有真实测试结果** | **❌ FAIL** | 全部为 SQLite / Fake / httptest / 拦截 transport；Testnet 签名账户 `-2015`，真实小额 BUY/SELL/Cancel/WS、MySQL 8 P95、生产 v25 物理 Schema 均无证据 |

**结论：8 项中 6 项代码级达标、1 项仅 Fake 达标、1 项明确 FAIL。**因此按方案自己的定义，**不能认定「开发完成」**。此外 §5「一条最简单但完整的交易链」的上半段（信号求值 → 预算 → 自动创建订单）在任何运行时均无入口，见 F1。

---

## 四、历轮 Findings 闭环台账

| 阶段 | 首轮关键 Findings | 本轮独立复核 | 状态 |
|---|---|---|---|
| **06-0/06-1** | 06-F1 **P1** v25 与旧方案 02 同号复用、`version==target` 静默跳过迁移 | `command/db_update.go:452-462` 每次 `SyncDatabase` 都执行 `VerifyV25SpotQuantSchema`（2 开关列 + 4 表全字段 + **10 个 `DECIMAL(36,18)`**）；子测试 `TestSyncDatabaseInitializesAndIsIdempotent/V25VersionReuseMissingSpotLedgerFailsClosed` **PASS** | ✅ 代码闭环 |
| 06-0/06-1 | F2～F4、F7 配置/JSONResp/永久回归 | rereview 声明与代码一致；F5（MySQL 8 DDL/权限）仍未测 | ✅ / F5 ⛔ |
| **06-2** | F1 **P1** 「`main.go` 启动态调用 Schema 校验」声明不实 | 全仓 `VerifyV25SpotQuantSchema` 仅出现在 `command/db_update.go`、`command/spot_quant_migration.go`、`command/db_update_test.go`；**`main.go` 0 引用**。按用户指示以「更正声明 + 发布 SOP」替代运行态门禁，未重加门禁 | ⚠️ 流程闭环、代码仍无运行态门禁（见 F9） |
| 06-2 | F4 Futures-only AST 禁用名单、F5 同名模板确认、F7 `authorized()` | `TestStage062SpotPreviewRejectsFuturesOnlyVariables` PASS；F7 在 06-5-rereview 落地 `SpotQuantController.authorized()` + `TestReview065*RejectsMissingOperatorClaim` | ✅ |
| **06-3** | F1 **P2** 物化字段可致超卖 / F2 **P2** 终态被 Fill 改写 | `projectPosition` 为唯一权威、`snapshot.go` 独立投影、终态不覆盖；`TestReview063TamperedMaterializedInventoryAndAccounting`、`TestReview063TerminalOrderNeverRewrittenToFilled` PASS | ✅ |
| 06-3 | F3 异常恢复人工入口 / F4 双开关 | F4 由 06-4 `DBGate` 闭环；**F3 仅 Go 方法级闭环，运维入口仍未接入**（见 F5） | ⚠️ |
| **06-4** | F1 **P1** SELL 0.5% 预留致永不 CLOSED | `rules.go` 已无 `0.995`；`planSell` 精确卖出 + step 整除校验，`TestReview064FullSellClosesCycleAndNewBuy`、`TestReview064BaseFeeDustIsExplicitReviewNotFakeClose` PASS | ✅ |
| 06-4 | F2 **P1** 权威 -2013 与网络错误不分流、异常 NEW 永久卡死 | `ErrNotFound` / `ConfirmNeverAccepted` / `ReconcileByClientID` 已落地；`TestReview064ExplicitAbsentOrderRecovery` 等 10 个 `TestReview064*` PASS | ✅ 代码级（入口见 F5） |
| 06-4 | F3 WS 走主网 / F4 FILLED 先于 myTrades 断流 / F5～F8 | WS 固定 `wss://ws-api.testnet.binance.vision/ws-api/v3`；`Place`/`Cancel` 复查 BaseURL；`TestReview064WS*` PASS | ✅ |
| 06-4 | F9 Testnet / MySQL 8 / 部署 | 06-6 记录仍全部 BLOCKED | ⛔ |
| **06-5** | F1 **P2** `ListPositions` 改 `LIKE '%...%'` 致执行层跨币扩散 | `query.go:46-52` 恢复默认 `symbol=?`，`SymbolLike` 仅 UI 显式开启；`TestReview065ExactSymbolRecoveryNoCrossCoin` PASS | ✅ |
| 06-5 | F2 **P2** `fully_priced=false` 仍下发 `total_pnl` | `readmodel.go:128-137` 双处 `if view.FullyPriced` 门控；`TestReview065ClosedUnpricedFeeNeverPublishesTotalPnL` PASS | ✅ |
| 06-5 | F3 **P2** `dist↔static` 106/115 断裂、须成套替换 | 06-6 已整体 `rsync --delete`；实测 `dist=115 static=115 common=115 identical=115 differing=0`，清单 `declared=115 actual=115 hash_mismatch=0` | ✅ 本地产物闭环（发布见 F4） |
| 06-5 | F4 Prettier / F5 JWT 二次鉴权 / F7 异常仓位物化字段清空 | ESLint 声明已修正；`authorized()` 二次校验 + `respondRead` 错误分类；`readmodel.go:110-113` 置空 | ✅ |
| 06-5 | F6 跨页 summary N+1 | 已有 `maxSummaryPositions=1000` + `ErrSummaryTooLarge` + HTTP 422（`TestReview065SummaryTooLargeReturnsActionable422` PASS），**但批量预取与 MySQL 8 P95 未做** | ⚠️ 部分（见 F7） |
| **06-6** | 新增 Fake E2E 2 测试 / 静态同步 / 只读核验 | `TestStage066EndToEndBUYSELLLedgerReadModelAndSecondCycle`、`TestStage066RestartRecoveryGateAndNoDoublePOST` **PASS**；Testnet `-2015`、MySQL 8、生产 Gate 未闭 | ⚠️ 本地完成、实盘 BLOCKED |

**共同正向结论**：全仓未检出已废弃方案 02 的 `legacy/strategy/off`、`spotRealOrderReleaseApproved`、`canAttestLiveFills`、`spot_auto_trade_mode`、`SPOT_STRATEGY_LIVE_TRADING_APPROVAL` 任何残留（0 match），符合方案 §7「绝不复活旧状态机」。

---

## 五、离线 Gate 实证（纯 HEAD 隔离副本，`go test -count=1`）

| Gate | 命令 / 范围 | 结果 |
|---|---|---|
| 副本等价 | 666 Go 文件逐文件 SHA256 | **mismatch=0** |
| 单测 | `go test -count=1` × 7 目标包 | **EXIT=0，7/7 PASS**（补齐 53 个剪除 `research` 的 gitignored 模板 fixture 后重跑，输出无 FAIL） |
| Race | `go test -race -count=1` × 7 目标包 | **7/7 PASS** |
| 06 系列永久回归 | `-run 'TestReview06\|TestStage06\|TestStatic'` × 5 包 | **54 PASS / 0 FAIL** |
| v25 同号复用 | `TestSyncDatabaseInitializesAndIsIdempotent/V25VersionReuseMissingSpotLedgerFailsClosed` | **PASS**（单独补跑，子测试需父名匹配） |
| 06-6 Fake E2E | `-run '^TestStage066'` | **2/2 PASS** |
| Futures / 旧 Spot 回归 | `go test -race -count=1 ./feature/... ./spot/...` | **6/6 PASS** |
| vet | `go vet` × 7 包 | **EXIT=0** |
| build | `go build -o /tmp/phase_audit_06all_bin .` | **EXIT=0** |
| diff-check | `git diff --check`（后端工作区） | **EXIT=0** |
| 路由契约 | 13 条 `/spot-quant/*` ↔ 前端 13 个 API 字面量 | **13/13，无下单/撤单/对账路由** |
| 静态资源 | `dist↔static` + SHA256 清单 | **115/115、115/115，0 mismatch** |
| 前端 typecheck | `pnpm typecheck`（`tsc --noEmit && vue-tsc --noEmit --skipLibCheck`） | **EXIT=0**；另独立 `tsc --noEmit` **EXIT=0** |
| 前端 ESLint | Spot 文件 `eslint --max-warnings 0` | **EXIT=0**（两次独立执行均 EXIT=0） |
| 全量 `go test ./...` | 干净副本 | **EXIT=1**，72 ok / `tests` panic |
| `./scripts/test-go.sh -short` | 干净副本 / 用户工作区 | **EXIT=1（72 ok）/ EXIT=0（73 ok、0 FAIL）** |

### 5.1 全量红点归因（**禁止写「全仓测试通过」**）

- 副本 `go test ./...` 与 `./scripts/test-go.sh -short` 均因 `panic: the target config file: /tmp/phase_audit_06all/conf/app.conf don't exist` 使 `go_binance_futures/tests` 失败；
- `conf/app.conf` 未入 Git，用户工作区存在该文件 → 在工作区重跑脚本 **EXIT=0、73 包 ok、0 FAIL**，与 06-6 实施记录 §五「退出码 0；73 个包 PASS」**完全一致**；
- 因此 06-6 记录**不是虚报**，但该结论**依赖未入 Git 的私有配置**，干净 CI / 独立复核不可复现；报告必须加此限定，不得简写为「全仓测试通过」；
- `controllers` 在纯 HEAD 副本还需 gitignored `strategy_templates/` fixture，属同一类既有环境依赖。

---

## 六、探针实证（副本内 / 只读静态探针）

```
PROBE06ALL_DIST_STATIC dist=115 static=115 common=115 identical=115
                       only_dist=0 only_static=0 differing=0
  index.html sha=612b3b1884c6f6d13a1ed5995eeef09765d66b780523a4d843f38df664b6b600
PROBE06ALL_MANIFEST declared=115 actual=115 only_declared=0 only_actual=0 hash_mismatch=0
PROBE06ALL_ROUTER spot_routes=13
PROBE06ALL_API_LITERALS 13（与 13 条路由一一对应）
PROBE06ALL_MUTATING_ORDER_ROUTES []          # 无 execute/sell/buy/cancel/reconcile/place
PROBE06ALL_I18N zh-CN: spotQuantTrading=True spotQuantPage=True
PROBE06ALL_I18N en:    spotQuantTrading=True spotQuantPage=True
PROBE06ALL_FIXTURE_SYNC copied=53 pruned_dirs=1   # 剪除 research，避免 SIGTERM
```

**静态零接线证据**：`main.go` 对 `spotexecution` / `NewProcessor` / `RunSpotQuantUserStream` / `ConfirmNeverAccepted` / `RecoverOutstanding` **0 引用**；全仓 `NewProcessor` 仅出现在 `processor_test.go`、`integration_066_test.go`；`RunSpotQuantUserStream` 仅在 `spot_quant_stream.go` 定义、无调用方；`ConfirmNeverAccepted` / `RecoverOutstanding` 仅在测试中被调用。`VerifyV25SpotQuantSchema` 全仓 4 处引用全在 `command/` 与测试，`main.go` 0 引用。

**适配器安全证据**：`NewSpotQuantAdapter()` 默认只读；`NewSpotQuantAdapterWithClient` 在 `permitOrderMutation=true` 且 BaseURL ≠ `https://testnet.binance.vision` 时直接构造失败；`Place`/`Cancel` 请求前再次复查 BaseURL；`spotQuantStreamEndpoint` 固定 `wss://ws-api.testnet.binance.vision/ws-api/v3`。

---

## 七、本轮新发现

### F1 — P1／阻塞「整体完成」宣称｜方案 §5 完整交易链在运行时无入口

`spotexecution.Processor` 的 `ExecuteBuy/ExecuteSell/Cancel/Reconcile/RecoverOutstanding/ConfirmNeverAccepted` 全部只有测试调用方；`main.go`、`routers/router.go`、定时循环均未构造 `Processor`，用户数据流 WS `RunSpotQuantUserStream` 也无调用方。因此：

- `TestStage066EndToEndBUYSELLLedgerReadModelAndSecondCycle` 是**测试代码手工编排** `DBGate → ExecuteBuy → Fill → ReadModel → ExecuteSell → 新周期` 的 Fake 链路，**不是系统运行时链路**；
- 方案 §5 上半段「读取已启用币种 → 复用策略表达式求值 → LONG/CLOSE_LONG 信号 → 计算预算 → 创建本地待提交订单」**没有任何实现入口**；`config.spot_quant_enable` 与 `spot_quant_enabled` 的运行期消费者只有 `gate_db.go`（且只在被测试构造的 Processor 内）和配置 API；
- 配置文件里的 LONG/CLOSE_LONG 规则、`SpotRuleTest` 均只做保存/试算，不驱动下单。

06-6 实施记录第 14 行把此称为「安全保留」，06-6 §七第 7 条也明确「在用户单独批准前不注册主网自动交易循环」——**这不是代码缺陷，而是有意 Gate**。但由此「自动现货量化交易系统」尚未成为可执行产品；用户所称「06 整体完成」若指可运行/可交付，则不成立。**在实盘 Gate 关闭并经单独授权前不应接线**，本条只用于纠正完成度表述。

### F2 — P2／发布误导｜中英 README 仍宣称「当前 Schema 要求 v24」

`appversion/version.go:4` 为 `DatabaseSchemaVersion=25`，但：

- `README.md:38`「**当前二进制数据库 Schema 要求 v24**」、`:135`「MySQL 8 Schema v24」、`:165`「当前代码要求 **Schema v24**」；
- `README.EN.md:38`「**Current binary requires database schema v24**」、`:138`「MySQL 8 schema v24」、`:168`「The current binary requires **schema v24**」。

（另有 `README.md:110/118`、`README.EN.md:113/121` 的 v24 章节属 01 阶段历史迁移描述，可保留。）06-6 只新增了现货菜单说明，未同步版本声明。**风险直接命中发布 SOP**：[检查单](开发方案-06-发布前Schema验证检查单.md) §2 要求「记录当前 `config.version`」，运维按 README 会记录错误版本；同号 v25 复用旧 02 结构恰恰是 06-F1 的 P1 风险。与 01-ALL 的 ALL-F2 同类。修法：仅改 6 处现在时表述指向 v25 并注明「以 `appversion.DatabaseSchemaVersion` 为准」。

### F3 — P2／CI 可复现性｜06-6 的 `test-go.sh` 结论依赖未入 Git 的私有配置

详见 §5.1。06-6 记录 §五写「退出码 0；73 个实际含测试用例的 Go 包 PASS，0 个 FAIL」在用户工作区属实；但纯 HEAD 副本实跑 **EXIT=1**（`tests` 因缺 `conf/app.conf` panic），`go test ./...` 同样 EXIT=1。建议在 06-6 记录与任何发布证据中补一句「该结果依赖工作区 `conf/app.conf`，干净 CI 需注入等价配置」，避免被当作「全仓测试通过」的发布证据。

### F4 — P2／发布组合不可复现｜已提交 `static/` 产物无法由任何已提交前端 SHA 构建

后端 HEAD `35da095` 提交了 115 个前端构建产物（含 `static/static/js/spotQuant-BnAMNvmr.js`），但前端 HEAD `89f15bd` 的 `git ls-tree -r --name-only` 中 **`spotQuant` 文件数 = 0**；10 项 06-5/06-6 源码改动仍是未提交状态（7 M + 3 ??），其中 `src/api/spotQuant.ts`、`src/router/modules/spotQuant.ts`、`src/views/spotQuant/` 为新增。即：

- 没有任何已提交的前端 commit 能重建当前 `static/`；
- 后端与前端**没有可发布、可审计的固定 SHA 组合**；后端运行中的二进制也未替换（06-6 记录第 101、120 行已承认）。

06-5-F3 要求的「整体成套替换 + 回滚」在**开发机产物层**已闭环，但**版本控制层**未闭环。成套发布前必须：固定并提交前端 SHA → 记录该 SHA 的 `dist` 清单 → 与新后端二进制 SHA、v25 Schema 预检、回滚包一并留证。

### F5 — P2／操作级未闭环｜异常恢复只有 Go 方法，没有 HTTP / CLI / UI 入口

06-4-rereview F2 结尾已写明「**入口仅为 Go 方法，06-5 页面尚未接入**」；06-5/06-6 均未补。当前 `ConfirmNeverAccepted`、`ReconcileByClientID`、`RecoverOutstanding` 全仓仅在测试中被调用，`routers/router.go` 13 条路由中唯一 POST 是 `positions/:id/rebuild`（本地重算），`controlledTrade.vue` 的真实卖出/撤单/对账按钮 `disabled`。后果：06-4-F2 的「异常 `NEW` 订单永久卡死」在**代码能力层**已修，但真实运行一旦出现权威 -2013 异常，**运维没有任何可操作路径**去确认或恢复，只能改库（违反审计纪律）。06-3-F3 移交的「人工异常入口」因此只完成一半。

### F6 — P1／实盘与发布 Gate 全部 BLOCKED

- **Testnet**：专用 Key 签名 `GET /api/v3/account` 返回 **-2015**；真实小额 BUY/部分成交/SELL/Cancel、用户数据 WS 长连、断线重启恢复**全部未执行**。公开 `exchangeInfo` 成功不构成交易准入证据。
- **主网**：仅一次签名只读 GET（连接与签名成功），**未发任何订单**；默认适配器仍硬拒主网 `Place`/`Cancel`；自动循环未注册。
- **数据库**：只读核验只在开发机 `go_bn_test`（MySQL **5.6.51**），`config.version=25`、2 开关列、4 表、10 个 `DECIMAL(36,18)`、3 个唯一索引 + 查询索引均存在，但 **Position/Order/Fill 全 0 行**；EXPLAIN 在空表上做，**不能推断 MySQL 8 的 P95/EXPLAIN**。
- **发布**：[检查单](开发方案-06-发布前Schema验证检查单.md) §2 七项复选框**全部未勾选**；生产库 v25 物理 Schema 预检与备份、维护窗口、`sync db`、二进制 + 静态资源 + Schema 一次性切换、回滚演练、JWT/路由/Futures/旧 Spot 健康检查均未执行。
- **授权**：持久化授权凭据、T0 代码级定义、真仓人工授权均不存在。

### F7 — P3／性能仅有限额保护，无实证

`maxSummaryPositions=1000` + `ErrSummaryTooLarge` + HTTP 422 已防止无界重放（06-5-F6 安全限额闭环），但 1～1000 条仍逐仓位重放 Fill/Order，无批量预取；06-6 在 0 行业务数据上做 EXPLAIN。需 MySQL 8 有代表性数据压测后才能确认，不得把 SQLite PASS 当性能证明。

### F8 — P3／Dust 真实处置流程未验证

06-4-F1 移除 0.995 预留后，`planSell` 对不满足 `MARKET_LOT_SIZE` 的策略残留返回 `ErrNeedsReview` fail-closed（正确）；但基础币手续费导致的真实 dust 如何人工处置、页面如何提示、是否会造成长期 `OPEN` 仓位，**无真实 Testnet 成交证据与操作规范**。06-4-rereview F1 与 06-6 记录 §8 P2 均将其后移，本轮仍未闭环。

### F9 — P3／风险由代码门禁转为流程门禁，且流程尚未执行

按用户明确指示，`main.go` 启动只比 `config.version` 与 `dbVersion`（`main.go:240-241`），**不调用** `VerifyV25SpotQuantSchema`；`VerifyV25SpotQuantSchema` 只在显式 `sync db` 触发。06-2-F1 用「发布前 Schema 检查单 SOP」替代运行态门禁。当前该 SOP 七项全未勾选 ⇒ **同号 v25 的旧 02 物理结构在运行时没有兜底拦截**，一旦跳过 SOP 就会静默放行。这是可接受的架构取舍，但必须作为发布硬约束写入交接清单，不能表述为「已自动拦截」。

---

## 八、实盘 / 发布 Gate 状态总表

| Gate | 06-6 记录 | 本轮独立复核 | 状态 |
|---|---|---|---|
| v25 物理 Schema（生产库） | 未检查 | 仅开发机 MySQL 5.6、空表 | ⛔ BLOCKED |
| 2 开关 / 4 表 / 10 DECIMAL / 索引 | 实验库只读 PASS | 代码 `VerifyV25SpotQuantSchema` 覆盖完整；生产未核 | ⚠️ 代码就绪 / 生产 BLOCKED |
| Testnet 签名账户 | -2015 | 未复跑（审计不联网），采信记录 | ⛔ BLOCKED |
| Testnet 小额 BUY/SELL/Cancel/WS | 未执行 | — | ⛔ BLOCKED |
| MySQL 8 EXPLAIN / P95 | 未执行 | — | ⛔ BLOCKED |
| `dist↔static` 全量哈希 | 115/115 | **115/115 + 清单 115/115，0 mismatch** | ✅ 本地产物 PASS |
| 前后端固定 SHA 组合 | 未形成 | 前端 10 项未提交、HEAD 无 spotQuant | ⛔ BLOCKED |
| 后端二进制替换 / 成套发布 / 回滚 | 未执行 | — | ⛔ BLOCKED |
| 自动信号接线 | 未注册 | `main.go`/routers/循环 0 引用 | ⛔ 有意 Gate（F1） |
| 主网订单授权 | 未给 | 适配器默认只读 + 主网硬拒 | ⛔ BLOCKED |

---

## 九、纪律与边界声明

本报告只对**代码、离线测试、静态证据、文档一致性**负责；不对「生产库已升级」「服务器已部署」「真实交易可用」作任何背书。所有 ✅ 均指**离线 / Fake / SQLite / Mock / 静态比对**通过，不自动升级为实盘结论。未发现需要紧急回滚的资金安全缺陷——**当前最危险的不是代码，而是把「离线全绿」当成「整体完成」去触发发布**。

**建议的收尾顺序**（均需用户单独授权，本轮未执行）：
1. 修 F2（README v25）、F3（脚本结论加限定），把 F4 的前端 SHA 固定并提交；
2. 在隔离环境补 F5 的运维恢复入口（或明确书面接受「异常只能人工改库」的风险）；
3. 完成 [检查单](开发方案-06-发布前Schema验证检查单.md) §2 七项，尤其是生产库 v25 只读预检与备份；
4. 用独立、IP 白名单正确的 Spot Testnet Key 关闭 -2015，执行真实小额 BUY→部分成交→SELL、Cancel、断线重启、dust 处置，留存脱敏 orderId/clientId/myTrades 证据；
5. MySQL 8 有代表性数据的 EXPLAIN / P95；
6. 二进制 + `static` + Schema 一次性成套发布与回滚演练；
7. 最后才讨论 F1 的自动信号接线与主网准入。

---

## 附录：关键证据位置索引

| 主题 | 位置 |
|---|---|
| Schema 版本真源 | `appversion/version.go:4`（=25） |
| 启动态仅比版本号 | `main.go:235-241` |
| v25 同号复用 fail-closed | `command/db_update.go:452-462`；`command/db_update_test.go:851` |
| v25 物理校验（2 列/4 表/10 DECIMAL） | `command/spot_quant_migration.go:209-266` |
| 双开关门控 | `service/spotexecution/gate_db.go`；`processor.go:35` |
| SELL 精确数量 / dust fail-closed | `service/spotexecution/rules.go:135-148` |
| 权威未找到 / 人工确认恢复 | `service/spotexecution/processor.go:381-432` |
| Fill 重放权威 / 终态保护 | `service/spotledger/fills.go`、`snapshot.go` |
| 精确 symbol / UI 模糊开关 | `service/spotledger/query.go:13,46-52` |
| PnL 门控 / 1000 上限 / 异常抹除 | `service/spotledger/readmodel.go:110-113,128-137,162-164` |
| 13 条只读路由 / 零交易路由 | `routers/router.go:129-143` |
| JWT 全局白名单 | `middlewares/auth.go:19-27`（`/spot-quant/*` 不在白名单） |
| 主网硬拒 / Testnet WS | `spot/api/binance/spot_quant_adapter.go:25-38,176-210`；`spot_quant_stream.go:24-32,113-116` |
| 06-6 端到端 Fake 测试 | `service/spotexecution/integration_066_test.go` |
| README 过期版本声明 | `README.md:38,135,165`；`README.EN.md:38,138,168` |
| 静态产物证据 | `开发方案-06-6-静态资源SHA256清单.txt`、`static/static/js/spotQuant-BnAMNvmr.js` |
| 发布 SOP | `开发方案-06-发布前Schema验证检查单.md` §2（7 项未勾选） |

---

> **前端工具链说明**：本轮在前端工作区实跑 `pnpm typecheck`（EXIT=0）、Spot Quant 相关文件 `eslint --max-warnings 0`（EXIT=0），并额外独立复跑 `tsc --noEmit`（EXIT=0）与同一组 ESLint（EXIT=0），均在本轮独立复现，不依赖历史记录。因本机 VSCode / Volar 服务争用，`vue-tsc` 单次耗时约 11 分钟，但退出码为 0、无诊断输出。**未运行 `pnpm build`**（会覆盖用户 `dist`，与历轮审计纪律一致）；构建结论采信 06-6 实施记录 §六（`pnpm build` PASS、2550 modules），并以本轮 13/13 路由契约、中英 i18n key、`spotQuant` chunk 存在性与 115/115 SHA256 静态比对交叉验证。
