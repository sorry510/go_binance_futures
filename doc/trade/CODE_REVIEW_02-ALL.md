# 方案 02 整体 Code Review（按三条新增评审准则）

审查日期：2026-11-10
审查对象：方案 02（S0～S4 及其 R1～R5 全部整改）的**整体符合性**
报告性质：**新增总报告**，不覆盖 `CODE_REVIEW_02-S1*.md` / `S2*.md` / `S3*.md` / `S4*.md` 任一历史报告

---

## 零、审计基线、范围与边界

### 0.1 基线（本次实测）

| 项 | 实测值 |
| --- | --- |
| HEAD | `95b5a77 feat: spot s1 s2` |
| 当前分支 | `feat/trade-upgrade`，`[origin/feat/trade-upgrade: ahead 2]`；`master` 仍为 `639c932` |
| 工作区 | `git status --porcelain` **217 项**，含 `MM / AM / AD / RD` 混合状态 |
| 审计对象 | **staged 与工作树叠加后的实际源码**，不是纯 staged、也不是纯 unstaged |
| Schema 真源 | `appversion/version.go:4` `DatabaseSchemaVersion int64 = 25` |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui` |
| 隔离副本 | `/tmp/phase_audit_02all`，关键目录 **109 个 Go 文件 SHA256 全等**，`main.go` / `appversion/version.go` / `go.mod` 全等 |

隔离副本等价校验输出：

```text
COPY_EQUIV checked=109 sha_diff=0 missing=0
  main.go: EQUAL
  appversion/version.go: EQUAL
  go.mod: EQUAL
```

### 0.2 本次评审的三条新增准则（用户原话）

> 现在对 02 整体再做一个 review, 检查时要求要尽量复刻合约交易的相关逻辑，不要有和账户绑定相关的逻辑，不要增加复杂的限定逻辑

- **准则 1**：尽量复刻合约交易的相关逻辑；
- **准则 2**：不要有和账户绑定相关的逻辑；
- **准则 3**：不要增加复杂的限定逻辑。

三条准则与既有 S0～S4 方案存在**直接设计冲突**（见 §四、§五）。本报告不作产品裁决，只给出证据、冲突矩阵与可执行的分层化简建议。

### 0.3 审查边界（严格遵守）

未修改任何业务代码、未 commit/push、未改 `conf/app.conf`、未碰 `strategy_templates/`、未对真实库执行 `sync db`、未访问真实 Binance、未重启服务、未操作生产仓位。所有探针/构建在隔离副本或只读统计中完成；测试进程显式清空 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL`。

### 0.4 依据文档与历轮报告

方案：`开发方案-02-现货自定义策略交易.md`、`-S0-现状审计与安全边界.md`、`-S1-配置数据与安全门禁.md`、`-S2-私有REST与订单状态.md`、`-S3-策略执行与库存风控.md`、`-S3-P0-只读账户核验与授权审计.md`、`-S4-前端集成与完整验收.md`、`-S4-现货顶级菜单与代码精简.md`、`-S4-R5-币种列表去账户绑定.md`、`开发方案-总览与实施顺序.md`、`发布检查-02-S4-成套交付与回滚.md`。

历轮报告 8 份：`CODE_REVIEW_02-S1.md` / `-S1-rereview.md` / `-S2.md` / `-S2-rereview.md` / `-S3.md` / `-S3-rereview.md` / `-S4.md` / `-S4-rereview.md`。本轮**采信**其已闭环结论，**不采信**任何「02 已完成 / 实盘可放行」表述。

---

## 一、总体结论

| 评审维度 | 判定 | 一句话依据 |
| --- | --- | --- |
| 准则 1 复刻合约交易逻辑 | ⚠️ **部分符合** | 币种列表层次与前端页面结构已复刻，**成交账本→REST 确认→结算的内核未复刻**：`rest_verified` 恒 0，运行链路永不生成 Holding |
| 准则 2 无账户绑定逻辑 | ⚠️ **部分符合，且需产品裁决** | R5 去掉了币种列表/配置层耦合；**数据模型、Holding/Orders/Fills 查询、signal_key、前端 scope 仍绑定 `main/auto_strategy`**，且与总方案「不卖手动库存」安全底线冲突 |
| 准则 3 无复杂限定逻辑 | ❌ **不符合** | `service/spotstrategy` 非测试代码 16 文件 / **3365 行**，按固定关键词清单计 **95 处**复杂限定；大量逻辑内联在运行与旧现货链 |
| 离线 Gate | ✅ **PASS** | 目标包 5/5、race 5/5、vet、build、`git diff --check`、tsc / vue-tsc、dist↔static 119/119 全等 |
| 全量 `go test ./...` | ❌ **EXIT=1** | controllers 5 项既有策略模板红点 + `tests` 包缺 `conf/app.conf` panic；**不得写「全仓测试通过」** |
| 生产 / 实盘 Gate | ⛔ **全部 BLOCKED** | 持久化授权、响应级 Fill 来源、旧仓对账、v25 生产迁移、成套二进制+static 发布、T0 均未通过 |

**核心结论**：

> 02 目前是「**复刻了外形，没有复刻内核；去掉了列表层的账户耦合，没有去掉账本层的归属语义；复杂度不降反升，且其中大部分是历轮 P1 安全整改的副产物**」。
> 本轮**未发现新的越权下单/资金安全 P1**（延续 S4 结论），但三条准则的符合度均不足以宣称方案 02 完成。
> **不认可「方案 02 完成」，不放行依赖生产证据的方案 03 C0，实盘下单/撤单/模式切换继续禁止。**

若要真正满足准则 1，优先事项不是继续精简前端兼容层，而是**补上响应级 `myTrades` 来源认证，让 `rest_verified=1` 可达，从而复刻 01 的 Fill→结算闭环**（§八 D）。

---

## 二、离线 Gate 实证（本次在 `/tmp/phase_audit_02all` 实跑）

| # | 命令 | 结果 | 证据 |
| --- | --- | --- | --- |
| 1 | `go build ./...` | ✅ EXIT=0 | `/tmp/02all_build.log` 空 |
| 2 | `go build -o /tmp/02all_main_bin .` | ✅ EXIT=0 | 产物 38,958,034 B |
| 3 | `go test -count=1 ./service/spotstrategy ./spot ./spot/api/binance ./feature/strategy/line ./command` | ✅ **5/5 PASS** | `spotstrategy 3.960s`、`spot 0.804s`、`spot/api/binance 1.386s`、`line 1.081s`、`command 1.946s` |
| 4 | 同 5 包 `-race` | ✅ **5/5 PASS** | `spotstrategy 5.376s` 等；仅 macOS `ld: malformed LC_DYSYMTAB` 非致命噪音 |
| 5 | `go vet ./...` | ✅ EXIT=0 | `/tmp/02all_vet.log` 空 |
| 6 | controllers S1/S4 专项 | ✅ `ok controllers 0.355s` | `/tmp/02all_ctrl.log` |
| 7 | `git diff --check` | ✅ 无输出 | `/tmp/02all_diff.log` 空 |
| 8 | `tsc --noEmit` / `vue-tsc --noEmit --skipLibCheck` | ✅ 均 EXIT=0 | 拆开执行并加 `NODE_OPTIONS=--max-old-space-size=4096`（合并执行曾 SIGTERM 137） |
| 9 | `dist` ↔ `static` SHA256 | ✅ **119/119 全等** | `dist=119 static=119 common=119 identical=119 only_dist=0 only_static=0` |
| 10 | `go test -count=1 ./...` | ❌ EXIT=1 | 见下 |

**全量红点归因（既有，非本轮引入）**：

```text
--- FAIL: TestQueryStrategyTemplatePageOrdersByIDDescending
--- FAIL: TestStrategyTemplateAITaskUsesRuntimeAndPreservesConversation
--- FAIL: TestStrategyTemplateValidationAcceptsMarketConditionStringComparison
--- FAIL: TestStrategyTemplateValidationAcceptsMultilineLetExpression
--- FAIL: TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals
FAIL  go_binance_futures/controllers  0.666s
panic: the target config file: /tmp/phase_audit_02all/conf/app.conf don't exist
FAIL  go_binance_futures/tests
```

`controllers` 5 项为策略模板 / LLM fixture 既有红点；`tests` 包因 `conf/app.conf` 未纳入 Git，干净副本必然 panic。按项目纪律**未复制用户敏感配置、未绕过、未据此宣称全仓绿**。

**永久回归证据（引用现有测试名，未重复造轮子）**：`TestS4SymbolListingDoesNotRequireAnyTradingTables`、`TestS4SymbolManagementIndependentOfAccountAndMode`、`TestS3F1ManualModeAndEnvCannotAuthorizeLiveWrites`、`TestS3F3EndpointAndFakeCannotProvideResponseEvidence`、`TestS4F1RuntimeUnverifiedTradeNeverCreatesHolding`、`TestS2F2FakeReadOnlyEngineCannotForgeRESTVerified`、`TestS3ForgedHoldingWithoutSignedFillsCannotSell`。

---

## 三、准则 1：尽量复刻合约交易的相关逻辑

### 3.1 参照链定义与实测

以 `controllers/feature.go`（合约 `GET /features`）、`controllers/spot.go`（旧现货列表）、`feature/feature_notice.go` / `feature_listen.go` / `feature_rush.go`（合约自动链）、`feature/account_trade_cycle.go`（合约账户 runner）为基准：

```text
controllers/feature.go:         lines=348 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 owner=0 AccountID=0
controllers/spot.go:            lines=208 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 owner=0 AccountID=0
feature/feature_notice.go:      lines=272 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 AccountID=0
feature/feature_listen.go:      lines=224 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 owner=0 AccountID=0
feature/feature_rush.go:        lines=226 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 owner=0 AccountID=0
feature/account_trade_cycle.go: lines=185 spotstrategy=0 AuthorizeLegacyOrder=0 account_id=0 owner=0 AccountID=15
```

口径说明：`feature_notice.go` 的 `owner` 计数全部来自 import 包名 `futuresownership`，**不是账户绑定字段**；`account_trade_cycle.go` 的 `AccountID=15` 是合约 **main/lead 带单分支**，与 02 的 `account_id="main"` 单一标签语义不同。即：**合约链确实没有 02 式的 `account_id/owner` 标签与注入式门禁**。

### 3.2 复刻度对照表

| 合约参照 | 02 现状 | 判定 |
| --- | --- | --- |
| `GET /features`：JWT → 查表 → 搜索/筛选 → COUNT + 分页 | `GET /spots/symbols` 同层次；`ListSettings(ctx, db, req)` 无 account/owner；`routers/router.go` 中 `/spots/symbols=1`、`/spots/strategy/configs=0` | ✅ **已复刻**（S4-R5） |
| `futures/symbol.vue` / `strategyTemplate.vue` 页面结构 | `spot/trading.vue` / `templates.vue` / `rulesEditor.vue` / `technologyDialog.vue` 按 S4-R3 复刻；`controlled.vue` 仅 11 行薄封装复用 `account.vue` | ✅ **已复刻**（视觉/交互） |
| 合约自动链直接复用 `line` / `expr` 求值环境 | 02 新建 `feature/strategy/line/spot_closed_env.go`（104 行）过滤未收盘 K 线、禁 FundingRate / ROI | ⚠️ 额外实现，非复刻 |
| **方案 01 内核**：逐笔 Fill → REST `userTrades` 按 `tradeId` 确认 `rest_verified=1` → 完整平仓 `settled` → 一次性风控 | 02 明确「**不直接复用** `futures_managed_*` / `futures_trade_fills`」，自建三表；`canAttestLiveFills()` 恒 `false`，S3 新回放 Fill 恒 `rest_verified=0` | ❌ **内核未复刻** |
| 合约链无 `spotstrategy` / `AuthorizeLegacyOrder` 注入 | `spot/spot.go`：`spotstrategy.=12`、`AuthorizeLegacyOrder=2`、`LegacyIntentForSide=2`、`SourceNotice=3`、`SourceRush=5`；`spot/api/binance/index.go`：`spotstrategy.=21`、`AuthorizeLegacyOrder=6` | ❌ **反向偏离**：02 向旧现货链注入门禁，合约链没有 |
| 合约 01-4 后旧 `order` 运行时读写退役、三端点 410 | 02 保留旧现货自动/通知/抢新链并加 mode 门禁 | ⚠️ 语义不同（旧链仍可能承担 SELL 退出职责，S1-F1 历史原因） |

### 3.3 关键证据：Holding 业务闭环断裂（准则 1 的根本阻塞）

```text
s3_live_gate.go:132  func canAttestLiveFills(_ *Engine) bool { return false }
s3_live_gate.go:25   const spotRealOrderReleaseApproved = false
s3_ledger.go:263     ... AND holding_id IS NULL AND rest_verified=1
s3_cost.go:19        ... AND rest_verified=1
s3_recovery.go:29    ... AND f.rest_verified=1 AND f.holding_id IS NULL
```

`ApplyVerifiedFills` / `replayHoldingCost` / `RecoverAndSettle` 三条消费路径**全部以 `rest_verified=1` 为前置**，而 S3-R1 后运行链路无法产生该值。因此：

- `TestS4F1RuntimeUnverifiedTradeNeverCreatesHolding` 已永久回归该事实；
- `CODE_REVIEW_02-S4-rereview.md` 明确将 E2E-02/03/04 改口径为「SQL Fixture PASS／运行链路 BLOCKED」；
- 持仓页、成本、收益、PnL 在真实运行下**零数据**，S4 仅为「只读 UI 骨架 + 夹具演示」。

> **判定**：准则 1 **⚠️ 部分符合**。已复刻的是「列表与页面」，未复刻的是 01 的「成交→确认→结算」内核；而后者才是合约交易相关逻辑的主体。

---

## 四、准则 2：不要有和账户绑定相关的逻辑

### 4.1 量化（`service/spotstrategy` 16 个非测试文件 / 3365 行）

```text
AccountMain=37   OwnerAuto=28   account_id=42   owner=69
ValidateScope=13 ValidateS4Scope=1  active_slot=8
```

按文件分布（账户相关关键词合计）：`s4_read.go=38`、`s3_legacy_preflight.go=32`、`s2_engine.go=24`、`s3_ledger.go=23`、`s3_exit.go=21`、`config.go=12`、`schema.go=12`、`s2_reconcile.go=10`、`s3_pnl.go=6`、`s3_recovery.go=6`、`s3_live_gate.go=4`、`mock_reservation.go=4`、`s2_read.go=2`、`s3_signal.go=2`、`s3_cost.go=1`。**15/16 个非测试文件均含账户绑定语义**。

### 4.2 R5 已完成的部分（实证）

| 位置 | 实测 |
| --- | --- |
| `service/spotstrategy/config.go` | `Setting` / `Patch` DTO `account_id=0`；`LoadSetting` 不再调用 `CurrentMode`；`UpdateSetting` 不校验 Account/Owner |
| `service/spotstrategy/s4_read.go` | `func ListSettings(ctx, db, req)` 无 account/owner 参数 |
| `controllers/spot_strategy_s4.go` | `ListSymbols=1`，仅 `authorized()` JWT 校验 |
| `routers/router.go` | `/spots/symbols=1`、`/spots/strategy/configs=0` |
| `src/views/spot/trading.vue` | 注释 `// Symbol management is independent of private account, owner and trade mode.`；`account_id=0` |
| 永久测试 | `TestS4SymbolListingDoesNotRequireAnyTradingTables`、`TestS4SymbolManagementIndependentOfAccountAndMode` |

### 4.3 仍然存在的部分（实证）

| 类别 | 证据 |
| --- | --- |
| 数据模型 | `schema.go` DDL：`account_id` 列 5 处、`owner` 列 3 处；`UNIQUE (account_id,owner,symbol,signal_key)`；`PRIMARY KEY (account_id,symbol,order_id,trade_id)`；`UNIQUE (active_slot)` |
| 只读查询 | `s4_read.go` 中 `ListHoldings` / `ListOrders` / `ListOrderFills` 三个签名均带 `account, owner string`，`ValidateScope=4`；`controllers/spot_strategy_s4.go` `s4ReadScope=4` |
| 信号幂等 | `s3_signal.go:51` `strings.Join([]string{AccountMain, OwnerAuto, symbol, ...})` → `signal_key` 把账户身份写入哈希 |
| 前端 API | `src/api/spotStrategy.ts`（142 行）仍导出 `spotStrategyScope = { account_id:"main", owner:"auto_strategy" }`、`getSpotStrategyMode()`；`scope()` 出现在 4 处请求 |
| **前端页面（与先前口径不同，本次实测更正）** | `src/views/spot/account.vue:44` 仍 `await getSpotStrategyMode()`；`:46` 仍检查 `status.data?.contract_version===4`；`:111` 仍显示 `<span>{{tr("scope")}}: main / auto_strategy</span>`。即 **R5 只去了交易页的账户绑定，账户页仍依赖 mode API 与 scope 文案** |

> 精确口径：`config.go` 中 `owner=9` 全部位于 `ValidateScope`、`AuthorizeLegacyOrder` 错误消息、`RequestMode` 与注释，**不在 `Setting`/`Patch` DTO 内**；不能据此说「配置层仍有账户绑定」，也不能说「账户绑定已彻底清除」。

### 4.4 与既有设计的冲突矩阵（必须由用户裁决）

| 用户准则 | 既有正式文档 | 冲突性质 |
| --- | --- | --- |
| 不要账户绑定 | `开发方案-02-现货自定义策略交易.md:5`「**仅主账户 `main` 的 `auto_strategy`**」 | 直接冲突 |
| 不要账户绑定 | 同文档 `:38`「**未知 Owner**、未核实 free/locked…全部停止新增风险」；`:三-1`「不将其余用户余额一起卖掉」 | 账户/归属标签是**防误卖手动库存**的安全约束，不是多账户功能 |
| 不要账户绑定 | `S0:12`「当前只有一个 Spot API Key，计划先支持 `market=SPOT, account_id=main, owner=auto_strategy`；未来增加账户前需重新设计凭据映射，**绝不静默回退 main**」 | 02 的 `account_id/owner` 本质是**单账户下的归属标签**，不是多账户路由 |
| 不要账户绑定 | `S0:115/120`「旧 SELL 退出职责未清，不得简单全停」「后端门禁必须放在实际订单发送处」 | 若删掉归属校验，将同时削弱旧单/人工资产隔离 |
| 不要账户绑定 | `S4-R5` 文档「`main/auto_strategy` **只保留在订单、Holding、Fill 和资金/交易模式风控领域**」 | 现行设计只实现了「API/UI 层解耦」，与用户字面要求仍有距离 |

**结论**：准则 2 有两种解读，不能由审查者擅自选：

- **(a) 去掉 API/UI 层不必要的账户参数耦合** → R5 已做，属合理精简，与本准则一致；
- **(b) 去掉数据模型与风控中的归属语义** → 会破坏总方案「不卖手动/旧策略库存」与 S0「绝不静默回退 main」的底线，需要用户明确接受风险后重设计表结构（`schema.go` 改动会牵动 v25 → v26 迁移）。

本报告**不执行 (b)**，也不把 (b) 的未完成判为代码缺陷，而列为**待裁决设计项**（02ALL-F2）。

---

## 五、准则 3：不要增加复杂的限定逻辑

### 5.1 量化

`service/spotstrategy` 非测试代码 16 文件 / 3365 行；含测试共 27 文件 / 5661 行。按固定关键词清单统计：

```text
StateReconcile=17  markReconcile=11  reconcile_required=7  pending_slot=10
AuthorizeLiveMode=8  AuthorizeLiveWrite=2  signal_key=6  Provenance=2  provenance=6
LiveOrdersApproved=3  canAttestLiveFills=3  spotRealOrderReleaseApproved=2
LegacyPreflight=7  blockers=2  LegacyIntentForSide=2  AuthorizeLegacyOrder=2
frozen=1  freeze=4                                  COMPLEX_TOTAL = 95
```

（口径：本次实测的 20 个关键词计数之和；与历轮报告中不同关键词集合的计数不可直接比较。）

02 专属新增/改动文件（不含 `main.go` / `routers/router.go` / `spot/spot.go` 等整体文件）：`spot_strategy_runner.go` 203、`s2_readonly.go` 181、`s3_live.go` 99、`spot_closed_env.go` 104、`controllers/spot_strategy.go` 172、`controllers/spot_strategy_s4.go` 172 ≈ **931 行**。

### 5.2 复杂度来源分类

**A. 无业务价值、可安全删除/下沉的限定（建议优先精简）**

- `s3_legacy_preflight.go` **286 行**只读预检，`ReadyForLive` 每次恒 `false`；`LegacyPreflightBlocked` 等入口无生产消费者。属于「永远不放行却完整实现一套检查」的最典型冗余，建议下沉为独立 `doctor` 子命令，不进入交易包。
- `contract_version=4` / `capabilities` 兼容协商：`controllers/spot_strategy.go` 的 `spotModeStatus` 仍返回，`account.vue:46` 仍在检查；在「不要复杂限定」下属于前后端版本握手包袱。
- `LegacyIntentForSide` / `AuthorizeLegacyOrder` 注入旧现货链（`spot/spot.go` 12 处、`index.go` 21 处 `spotstrategy.` 引用）：合约链完全没有，是 02 独有的耦合。
- `signal_key` 中拼入 `AccountMain` / `OwnerAuto`（准则 2 + 3 双重冗余）。
- `quote_filter` 严格白名单 / 旧页兼容分支：R5 已声明「不再引入能力协商或旧页兼容分支」，但历史设计与注释仍留有多层说明。

**B. 资金安全底线，绝不可按准则 3 删除**

- `s3_live_gate.go:25` `spotRealOrderReleaseApproved = false` 编译期硬封闭；`LiveOrdersApproved()`、`AuthorizeLiveWrite()`；
- `canAttestLiveFills()` 恒 false（阻止伪造 `rest_verified=1`）；
- `s3_ledger.go` `holding_id IS NULL` CAS + `RowsAffected==1` 幂等、禁止加仓；
- `s3_exit.go` 从 linked verified Fill 重建 `provenBase` 精确比对（伪造 Holding 不能卖）；
- `trade_id` 主键去重、精确 `DECIMAL(36,18)`、`ACK≠Fill`、`CANCELED≠零成交`、UNKNOWN 按原 `clientOrderId` 查询不盲重发。

这些是 S2-F1/F2、S3-F1/F3、S4-F1 等 **P1 整改的直接产物**。删除它们 = 恢复已修复的越权/伪造漏洞。

**C. 应「下沉」而非「删除」的限定**

状态机（`reserved/submitting/unknown/.../reconcile_required`）、`pending_slot` 并发防重、`freeze` 冻结态、模式矩阵，属于订单正确性必要机制，但当前**内联在 3365 行运行包中**，与配置/查询/API 混在一起。建议抽成单一 `orderstate` 子包 + 一份状态迁移表，主链只保留调用。

### 5.3 判定

> 准则 3 **❌ 不符合**：02 从 S1 起持续新增限定，R1～R5 又以整改形式追加了更多门禁。当前复杂度不是偶发，而是**贯穿数据模型、状态机、门禁、只读预检、前端兼容层的系统性特征**。
> 但同时必须指出：**复杂度与安全整改高度耦合**，不能整体回滚。正确路径是「保留 B、下沉 C、删除 A」，而不是「删门禁」。

---

## 六、本轮 Findings

> 编号 `02ALL-F*` 表示**准则符合性 / 设计裁决类问题**，与历轮 `S*-F*` 资金安全缺陷分开计；本轮**未发现新的越权下单或资金损失型 P1**。

| ID | 级别 | 问题 | 证据 | 影响 |
| --- | --- | --- | --- | --- |
| **02ALL-F1** | **P1（准则符合性阻塞）** | 准则 1 的内核未复刻：真实运行链路 `rest_verified` 恒 0，`ApplyVerifiedFills` / 成本重建 / `RecoverAndSettle` 永不触发，Holding 与收益闭环为零 | §3.3；`TestS4F1RuntimeUnverifiedTradeNeverCreatesHolding`；`CODE_REVIEW_02-S4-rereview.md` | 阻断「02 完成」与「03 C0 放行」；**不是新漏洞**，是既有 S4-F1 未闭环在新准则下的升级定性 |
| **02ALL-F2** | **P1（待裁决）** | 三条准则与既有正式方案冲突未裁决：总方案 `:5/:38`、S0 `:12/:115` 要求 `main/auto_strategy` 归属，S4-R5 只解耦 UI 层 | §4.4 冲突矩阵 | 若按字面执行准则 2 的 (b)，需重设计三表并升 v26，且会削弱「不卖手动库存」；**须用户明确取舍** |
| **02ALL-F3** | P2 | 账户绑定残留：DDL 5+3 列、3 个只读函数 `ValidateScope=4`、`signal_key` 含账户身份、`spotStrategyScope` 前端常量、`account.vue` 仍调 `getSpotStrategyMode()` / 检查 `contract_version===4` / 显示 `scope: main/auto_strategy` | §4.3 实测行号 | 与准则 2/3 不一致；`account.vue` 依赖 mode API 意味着「币种列表去账户化」并未覆盖整个现货模块 |
| **02ALL-F4** | P2 | 复杂度集中在运行链且存在恒 false 的重实现：`s3_legacy_preflight.go` 286 行只读预检从不放行；95 处复杂限定关键词；旧现货链被注入 33 处 `spotstrategy.` 引用 | §5.1 / §5.2 | 与准则 3 不符；维护成本与误改风险升高 |
| **02ALL-F5** | P2 | 半发布风险仍未消除：S4-R2 文档 `:78` 记录曾发现 **PID 63542** 旧二进制在 3333 端口运行、新 `static` 可能被旧进程直接提供；R4 `:158` 称同步前已确认无进程。本次只读查询未发现 3333 监听，但**无法区分「确无监听」与「查询被环境限制」，亦未核实实际运行二进制是否为包含 `/spots/symbols` 的新构建** | `发布检查-02-S4-成套交付与回滚.md:9-11,22`；`README.md:781`、`README.EN.md:680` | `static` 与后端二进制版本不一致时，新页面 4 个 GET 在旧二进制上 404；生产成套发布 Gate 仍 BLOCKED |
| **02ALL-F6** | P2 | 文档口径自相矛盾：总方案 `:5`「仅主账户 main 的 auto_strategy」与 `:70` S4-R5「去账户绑定」并存；`:38`「未知 Owner…停止新增风险」未被 R5 说明覆盖 | §4.4 / §0.4 | 后续开发者无法确定归属语义是否仍为强制约束 |
| **02ALL-F7** | P3 | `go test ./...` 全量 EXIT=1（controllers 5 项 + tests 包 panic） | §二 | 既有归因，非本轮引入；**禁止在任何验收记录中写「全仓测试通过」** |
| **02ALL-F8** | P3 | `feature/strategy/line/spot_closed_env.go` 104 行为 02 自建 K 线求值环境，未复用合约链 `line`/`expr` | §3.2 | 与准则 1「复刻」有偏差，但有「拒绝 FundingRate/ROI、过滤未收盘 K 线」的合理业务理由 |

---

## 七、历轮未决项台账（本轮未闭环，全部 BLOCKED）

| 未决项 | 来源 | 状态 |
| --- | --- | --- |
| 持久化授权凭据（签名 + 有效期 + 操作者 + 撤销）、append-only 模式切换/解冻审计 | S1 / S3 / S3-P0 | ⛔ BLOCKED |
| 真实 `myTrades` 响应级来源证明、逐 `tradeId` 与手续费资产核对、`rest_verified=1` 可达 | S3-R1 / S4-R1 | ⛔ BLOCKED；直接导致 02ALL-F1 |
| 旧自动 SELL 退出职责 / 库存 / 余额 / 外部挂单对账；`eatRate` 手动路由停用确认 | S0 / S1-F1 | ⛔ BLOCKED |
| **014-F1**：`applySettledRisk` 唯一自动亏损控制入口依赖实盘 symbol-scoped 零仓位方向 row 证明 | 01 线 | ⛔ BLOCKED |
| **015-F3**：`futures_trade_fills.trade_time` 无索引 + MySQL 8 P95 / EXPLAIN | 01 线 | ⛔ BLOCKED |
| 生产 v25 升级、二进制 + `static` + Schema 成套发布与回滚、T0 代码级定义 | S4 / 发布 SOP | ⛔ BLOCKED（02ALL-F5） |
| 浏览器人工 E2E、E2E-02/03/04 完整交易场景 | S4-R1～R5 | ⛔ BLOCKED（夹具演示 ≠ 业务闭环） |

---

## 八、按三条准则的分层化简建议

**A. 零资金风险、可立即执行（不改表、不碰门禁）**

1. 删除 `src/api/spotStrategy.ts` 中未被 views 使用的 `getSpotStrategyMode` 与 `contract_version` 校验路径，或把 `account.vue` 的兼容检查改为纯静态版本常量；清理 `account.vue:111` 的 `scope: main / auto_strategy` 展示。
2. 把 `s3_legacy_preflight.go`（286 行、恒不放行）从交易包移入 `doctor` / 独立只读工具，保留其 9 项回归但不再计入运行链复杂度。
3. 统一文档口径：在总方案 `:5/:38` 增加「归属标签仅用于资金风控，不是多账户绑定；UI/列表层不传账户参数」的显式说明，消除 02ALL-F6。
4. 删除或合并 R1～R5 文档中已被后续章节覆盖的历史设计段落（S4-R2 `:164` 已自行声明「此节覆盖上文历史说明」）。

**B. 需用户裁决后才能动（02ALL-F2）**

- 是否接受「去掉数据模型中的 `account_id` / `owner` / `active_slot`」→ 若是，需 v26 迁移、重写三表主键与唯一约束、重新定义「不卖手动库存」的替代证据，并重新评估 S0/S2/S3 全部回归用例；**在响应级来源认证落地前不建议做**。

**C. 明确禁止按准则 3 删除**

- `spotRealOrderReleaseApproved=false`、`canAttestLiveFills=false`、`AuthorizeLiveWrite`、Fill CAS 幂等、`trade_id` 去重、`provenBase` 比对。任何「简化」不得以恢复这些已修复漏洞为代价。

**D. 若目标是真正满足准则 1（推荐路线）**

按 01 的成熟链路复刻，而不是继续加前端兼容层：

1. 实现签名 REST 响应的**来源认证**（请求参数、时间窗口、账户/订单/tradeId 身份关联、可审计证据）；
2. 让 `canAttestLiveFills` 在证据齐备时返回 true，真实 `myTrades` 逐 `tradeId` 确认后置 `rest_verified=1`；
3. 打通 `ApplyVerifiedFills` → Holding → `PrepareExit` → 结算/PnL，把 E2E-02/03/04 从「SQL Fixture PASS」升级为「运行链路 PASS」；
4. 之后再评估是否把 02 的旧现货链门禁收敛为与合约链一致的最小形态。

> 顺序理由：在 D 未完成前删除 B 类归属校验，会同时失去「不误卖手动库存」保护与「可核实 Fill」依据，属于风险放大。

---

## 九、Gate 结论矩阵

| Gate | 结论 |
| --- | --- |
| 本地代码编译 / 目标包单测 / race / vet / build | ✅ **PASS** |
| `git diff --check` / dist↔static 119/119 / tsc / vue-tsc | ✅ **PASS** |
| 三条准则符合度 | ⚠️ 准则 1 部分、⚠️ 准则 2 部分且待裁决、❌ 准则 3 不符合 |
| 历轮 S1/S2/S3/S4 的 P1 整改 | ✅ 已闭环（采信各 `-rereview`），**本轮无新增资金安全 P1** |
| 方案 02 整体完成 / 业务闭环 | ⛔ **不认可**（02ALL-F1） |
| 方案 03 C0 准入 | ⛔ **不放行** |
| 真实 Binance 下单/撤单、模式切换、生产 v25、成套发布、T0 | ⛔ **全部 BLOCKED** |

**允许**：把当前代码作为**只读研发与人工 UI 验收基线**继续开发。
**禁止**：宣称 02 完成、部署生产、单独同步 `static`、启用实盘或 T0、以夹具演示冒充业务闭环。

---

## 十、审查边界与未验证项

- 本次未连接真实 Binance、未访问生产 MySQL、未执行 `sync db`、未重启服务、未操作生产仓位；
- 未对本地 3333 端口做进一步探测，也未确认当前运行二进制哈希；02ALL-F5 的半发布风险按「未证实消除」处理；
- 全量 `go test ./...` 的 root/tests 包因 `conf/app.conf` 未入 Git 无法在干净副本运行，按项目纪律未复制敏感配置；
- `strategy_templates/research/**` 未跟踪研究目录对全量 build/vet/test 的干扰按既有归因记录，未删除用户任何研究材料；
- 所有量化数字均为**本次对当前工作区实跑统计**，不同关键词集合的历史计数不可直接比较；
- 探针与构建仅在 `/tmp/phase_audit_02all` 隔离副本或纯只读统计中完成，未修改用户业务代码。
