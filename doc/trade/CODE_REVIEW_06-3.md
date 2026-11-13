# CODE_REVIEW_06-3：本地仓位、委托、逐笔成交与盈亏

日期：2026-11-11  
分支：`feat/trade_spot`  
范围：方案 06 的 **06-3（本地仓位 / 委托 / Fill / 盈亏）** 阶段代码与离线验收；含对 06-2 整改复评作为**前置 Gate** 的独立复核。  
结论摘要：**06-3 本地代码与 SQLite 离线 Gate 全绿，核心记账、幂等、回滚、重放与 Futures 隔离均有永久回归和探针实证；阶段整体验收 ⚠️ 有条件通过（MySQL 独立证据不足，且 F1/F2 是 06-4 接入真实下单前的阻塞项）。生产 / 实盘 Gate 全部 BLOCKED。**

---

## 一、审计基线与范围

### 1.1 基线（本次实测，非引用文档）

| 项目 | 实测值 |
| --- | --- |
| 后端分支 / HEAD | `feat/trade_spot` / `2e374b4`（2026-11-11 19:06:18 +0800 `fix: add 06-2`） |
| 本地 master | `7ed6c93` |
| Schema 真源 | `appversion/version.go:4` `DatabaseSchemaVersion = 25`（06-3 未改版本） |
| 后端工作区 | **5 项未提交**：`M` × 3、`??` × 2 |
| 前端仓库 | `/Users/zhz/work/binance/go_binance_futrues_new_ui`，HEAD `89f15bd`；8 项未提交与 06-2 复审时**完全一致，06-3 零前端改动** |

后端 5 项：`M doc/trade/开发方案-06-现货量化交易-复刻合约简化版.md`、`M doc/trade/开发方案-总体与实施顺序.md`、`M models/spot_quant.go`、`?? doc/trade/开发方案-06-3-本地仓位订单成交与盈亏-实施记录.md`、`?? service/spotledger/`（7 个 Go 文件，1755 行）。`git diff --stat` 确认 `main.go`、`routers/router.go`、`conf/app.conf`、`static/` 均无改动。

`2e374b4` 提交统计为 15 文件 / 1725 行，含 06-2 全部业务代码、`CODE_REVIEW_06-2.md`、`CODE_REVIEW_06-2-rereview.md`、`开发方案-06-发布前Schema验证检查单.md` 及 06-1 复评勘误。

### 1.2 副本与等价性

`/tmp/phase_audit_063` = `git archive HEAD` + 按 `git status --porcelain -z` 同步 11 个展开条目；无工作区已删文件。基于 git 索引构造集合校验（避免遍历 `strategy_templates/research/**` 巨型未跟踪目录触发 SIGTERM）：

```text
SYNCED 11 REMOVED 0
tracked=644 untracked_go=7 deleted=0 expected=651 copy_go=651
COPY_EQUIV sha_diff=0 missing=0 extra=0
```

**651 个 Go 文件 SHA256 全等**，零内容差异、零缺失、零多余（探针写入前）。

### 1.3 审计边界

未修改任何业务代码；未读取 / 复制 `conf/app.conf`；**所有测试显式 `unset SPOT_LEDGER_MYSQL_DSN` 与 `SPOT_STRATEGY_LIVE_TRADING_APPROVAL`**（MySQL 集成测试实证 SKIP，未连接任何 MySQL）；未访问 Binance；未执行 `sync db`；未操作 `strategy_templates/`、用户其他未提交文件；未 commit / push / 重启服务 / 替换二进制 / 同步 `static/`。探针仅在副本内，结束随副本删除。本会话新增的唯一文件是本报告。

---

## 二、前置 Gate 独立复核（06-2 整改复评）

`CODE_REVIEW_06-2.md` 判 06-2-F1（P1：rereview 声称启动态调用 `VerifyV25SpotQuantSchema`，实际只在 CLI `sync db` 与测试中）为准入阻塞，并给出两个可选修复：① 启动态补只读校验；② 更正声明并把校验责任写进发布 SOP。`CODE_REVIEW_06-2-rereview.md` 选择了 **②**，本次逐条复核：

| 复评声明 | 独立实证 | 判定 |
| --- | --- | --- |
| F1：不重加运行态门禁，纠正旧文档与注释，发布前强制 CLI Schema 验证 | `CODE_REVIEW_06-0_06-1-rereview.md:7-15` 新增勘误，明确「运行态也拒绝启动」是删除前的历史描述；`command/spot_quant_migration.go:204-208` 注释改为「normal runtime startup does NOT call it」；新增 `开发方案-06-发布前Schema验证检查单.md`（27 行），§1 明确「启动不会调用」，§2 要求发布前只读 `information_schema` 核对 2 开关 + 4 表 + 10 个 `DECIMAL(36,18)`，§2 末尾 Gate 保持 BLOCKED；`main.go` 实测仍 0 处特殊检查 | ✅ **口径已统一**，采用上一轮报告允许的第二方案 |
| F2：基线文档已修正 | 总方案 `:5` 标为「06-0 历史基线快照」、`:6` 新增 2026-11-11 当前基线 | ✅ 属实（见 06-3-F6） |
| F4：Futures-only 变量禁用 | `service/spotquant/validation.go:27` 禁用名单 + `:113` `ast.Find` 标识符检查；`controllers/spot_quant.go` 试算 env 已移除 `ROI` | ✅ 属实 |
| F5：同名模板导入确认 | `src/views/spotQuant/strategyTemplate.vue:538` 按名称查全部匹配、`:550` `ElMessageBox.confirm` | ✅ 属实 |
| F3 / F6 / F7 | 检查单明确 06-2 未成套部署；F6 语义冻结（币级 enabled 仅为配置偏好，06-4 双开关门控）、F7 后移 06-5 | ✅ 属设计裁决 / 阶段后移，符合方案简化原则 |

**复核结论**：06-2 的准入阻塞以「更正声明 + 发布 SOP + 生产 Gate 继续 BLOCKED」闭环，不再是「声明与代码不符」。⚠️ 复评所述「按用户此前明确指示删除运行态检查」属开发侧记录的用户决策，本会话无法独立验证该指示本身；但代码、注释、历史勘误、检查单四处口径一致，风险已由**运行态代码门禁**显式转移为**发布流程门禁**。**06-3 本地研发准入成立**；生产发布仍须完成检查单并留存证据，否则 BLOCKED。

---

## 三、06-3 验收矩阵

依据 `开发方案-06-现货量化交易-复刻合约简化版.md` §6 的 06-3 行（「SQLite/MySQL 测试可完整演练部分成交、手续费、重复回报、重启恢复、买卖关系，不能读取/修改 Futures 表」）与 `开发方案-06-3-本地仓位订单成交与盈亏-实施记录.md` §二 / §三 / §四。

| # | 验收项 | 判定 | 证据 |
| --- | --- | --- | --- |
| 1 | BUY→SELL 完整生命周期、分批成交、加权均价、已实现盈亏 | ✅ PASS | `TestStage063SpotPartialFillsFeesAndFullLifecycle`、`...PricedPNLAndFilters` |
| 2 | base / quote / 第三方 BNB 手续费三态 | ✅ PASS | 永久测试断言 `AvgCostQuote=101.0101`、`UnpricedFees["BNB"]`、`FullyPriced=false` |
| 3 | 重复 Fill 幂等 + 同 ID 内容冲突拒绝 + 12 线程并发 | ✅ PASS | `TestStage063SpotRejectedFillsRollbackAndDedup`：`applied=1`、fills=1 |
| 4 | 重启重放、被破坏的物化字段可修复 | ✅ PASS | `TestStage063SpotStatusValidationAndRebuild`；探针 H |
| 5 | 非法方向 / 超量 / 错币种 / 手续费超额整笔回滚 | ✅ PASS | 8 类非法 Fill 全部拒绝，order binding 与 fills 均未落库 |
| 6 | 同币种并发建仓串行化 | ✅ PASS（SQLite） | `TestStage063SpotSimultaneousOpenCycles` 12 并发仅 1 个周期 |
| 7 | 状态机：终态不回退、PENDING_CANCEL 迟到 Fill 不抹除、FILLED 缺 myTrades 阻断新单 | ✅ PASS（有 F2 语义冲突） | `...StatusValidationAndRebuild`、`...OutOfOrderRecoveryAndPendingCancel` |
| 8 | 仓位 / 委托 / Fill 查询、筛选、分页、关联 | ✅ PASS | `ListPositions` / `ListOrders` / `ListFills` 永久测试 |
| 9 | **MySQL 完整交易与重启** | ⚠️ **开发侧声明，本会话未复跑** | `mysql_integration_test.go` 默认 SKIP（实证 `--- SKIP`）；实施记录称 MySQL 5.6 Docker 容器 PASS 且容器已删除；**MySQL 8 明确未验证** |
| 10 | Futures 三张表零污染 | ✅ PASS | 3 张哨兵表永久测试 + 探针 `marker=futures-untouched` |
| 11 | 无真实下单 / 无自动触发器 / 无新 HTTP 接口 | ✅ PASS | 全仓 Grep：`spotledger` 引用只在 `service/spotledger/` 与 `models/`；`main.go`、`routers/`、`bootstrap/` 零引用；无任何网络 / Binance SDK import |
| 12 | 06-4 的双开关门控 | N/A（属 06-4） | 库层零开关引用，见 06-3-F4 |

---

## 四、逐文件静态审查要点

**`service/spotledger/decimal.go`**：`exactDecimal` 正则 `^[0-9]{1,18}(\.[0-9]{1,18})?$`，拒绝科学计数法、负号、超 18 位整数 / 小数；`numeric()` 固定 18 位并做溢出检查。全链路使用 `shopspring/decimal`，资产计算无 `float64`。

**`ledger.go`**：`New` 拒绝 nil DB 与未知 dialect；`CreatePosition` 校验 `base+quote==symbol` 与 USDT/USDC/FDUSD 报价资产，MySQL/Postgres 用 `SELECT ... FROM spot_symbols ... FOR UPDATE` 串行化同币新周期，SQLite 用进程内 mutex；`CreateOrder` 限 BUY/SELL、MARKET/LIMIT、`client_order_id` 幂等（同参数返回原单、异参数拒绝）、每仓仅一笔未结算订单、SELL 必须 `OPEN` 且 `qty <= remaining - reserved`。

**`fills.go`**：`BindExchangeOrderID` 只绑定不可重绑；`SetExchangeStatus` 用 `transitionAllowed` 阻止终态回退；`validateFillEvent` 校验身份、`qty*price≈quote_qty`（0.000001 绝对容差）、手续费必须显式；`RecordFill` 按 `symbol+exchange_order_id+trade_id` 去重、冲突内容报错、事务内先绑定再插入，随后 `reprojectPosition` 从不可变 Fill 重算；`RebuildPosition` 供重启恢复。

**成本模型**：BUY 基础币手续费减少入库数量、报价币手续费增加成本；SELL 基础币手续费计入扣减量、报价币手续费冲减收入；按 `cost×debit/remaining` 分摊成本；全卖完时分摊剩余全部成本；第三方资产手续费原样保留且不折价。`snapshot.go::Accounting` 在无可信 Spot 现价时不伪造浮盈亏 / 总盈亏，第三方费用返回 `unpriced_fees` + `fully_priced=false`。

**`models/spot_quant.go`**：仅为 3 个账本 DTO 补 `orm:"column(...)"` 映射以支持 `QueryRows`，**仍未注册 `orm.RegisterModel`**，保持 `DECIMAL(36,18)` 不被 ORM 改写。生产 v25 DDL（`command/spot_quant_migration.go:27-94`）与 SQLite/MySQL 测试 fixture 的列、类型、`uq_spot_orders_client`、`uq_spot_orders_exchange`、`uq_spot_fill_exchange` 一致。

**`mysql_integration_test.go`**：默认 SKIP；`ParseDSN` 后强制 `DBName == "spotledger063_isolated"`，否则 Fatal；使用独立 ORM alias 不覆盖 `default`；`t.Cleanup` 只 DROP 指定 5 张表。安全边界合理，是正向不变量。

**运行时接线（关键边界）**：06-3 是**纯本地 Go 库**，当前没有任何生产调用路径——没有 loopRun、没有路由、没有控制器、`main.go` 零引用。因此本地账本不会自动运行，也不构成当前资金风险；所有资金安全结论都只对 06-4 接线后成立。

**前端与产物**：06-3 零前端改动；`dist` ↔ `static` 实测 `106/106 SHA256 全等`，`static` 不含 06-3 内容。隐藏路由 / 本地账户页 / 成套交付仍属 06-5 / 06-6，本阶段不提前发布，符合 06-2-F3 边界。

---

## 五、离线 Gate 实测

`go test` 均带 `-count=1`；`go` 路径显式 `/usr/local/go/bin`；所有进程清空 `SPOT_LEDGER_MYSQL_DSN`。

| 验证 | 结果 |
| --- | --- |
| 用户工作区 `go test -count=1 ./command ./controllers ./models ./service/spotquant ./service/spotledger` | ✅ **5/5 PASS** |
| 用户工作区 `go test -count=1 -race` 同 5 包 | ✅ **5/5 PASS**（macOS `ld: malformed LC_DYSYMTAB` 为非致命噪音） |
| 副本同 5 包 / race | 4 包 ok；`controllers` 因副本缺未跟踪 `strategy_templates/` 报 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals` → 既有归因 |
| `-run '^(TestStage063\|ExampleLedger)' -v ./service/spotledger` | ✅ EXIT=0，**9 个 `TestStage063*` + 1 个 Example = 10 PASS**，MySQL 集成测试 `--- SKIP` |
| `go vet ./command ./controllers ./models ./service/spotquant ./service/spotledger` | ✅ EXIT=0 |
| `go build -o /tmp/063_main_bin .` / 包 build | ✅ EXIT=0 / EXIT=0 |
| `git diff --check`（后端工作区） | ✅ EXIT=0 |
| 前端 `tsc --noEmit` / `vue-tsc --noEmit --skipLibCheck` | ✅ EXIT=0 / EXIT=0 |
| 副本 `go test -count=1 ./...` | ❌ EXIT=1：**70 个包 ok**；`controllers` fixture 缺 `strategy_templates/`、`tests` 包 `panic: ... /tmp/phase_audit_063/conf/app.conf don't exist`。均为既有环境归因，**不写「全仓测试通过」** |
| 删除探针后副本 4 包复跑 | ✅ 4/4 PASS，无残留影响 |

---

## 六、探针实证（副本内临时探针，结束随副本删除）

探针：`/tmp/phase_audit_063/service/spotledger/zz_063_probe_test.go`，同包复用 `fixture` / `event` / `createTestOrder`。

```text
PROBE063_BASELINE remaining=1 status=OPEN
PROBE063_MATERIALIZED_OVERSELL err=<nil> existing=false requested=999 status=NEW
PROBE063_OVERSELL_PERSISTED id=2 requested=999
PROBE063_ACCOUNTING_MATERIALIZED realized=999 total=10989.000000000000000000 err=<nil>
PROBE063_FUTURES_SENTINEL marker=futures-untouched
PROBE063_CANCELED_FULL_FILL before=CANCELED after=FILLED err=<nil>
PROBE063_CANCELED_DOWNGRADE err=illegal Spot order status FILLED -> NEW
PROBE063_CLOSED status=CLOSED closedAt=2000 remaining=0 err=<nil>
PROBE063_LATE_BUY_REOPEN err=order 1 fill qty exceeds requested qty
PROBE063_AFTER_CLOSE_REJECT err=cannot append a new trade to already-closed Spot position
PROBE063_QUOTE_TOL within=<nil> beyond=fill quote quantity does not match qty * price
PROBE063_CONSTRUCTOR nil=nil Spot ledger DB dialect=unsupported Spot ledger dialect "cockroach"
PROBE063_REBUILD before_err=<nil> rebuild_err=<nil> remaining=1 pnl=0 after_err=SELL exceeds remaining strategy-owned base asset
```

**正向不变量**：晚于 `closed_at` 的新 Fill 被拒；`quote_qty` 容差边界符合预期；nil DB / 非法 dialect 被拒；Futures 哨兵表零污染；`RebuildPosition` 能把被破坏的物化字段修回 `remaining=1, pnl=0`，修复后 `SELL 999` 被正确拒绝。

**边界观察**：物化字段被篡改后 SELL 上限与 Accounting 均失效（F1）；终态订单被 reproject 改写（F2）；异常恢复是硬失败（F3）。

---

## 七、Findings

| ID | 级别 | 问题 | 实证与影响 | 建议 |
| --- | --- | --- | --- | --- |
| **06-3-F1** | **P2（阻塞 06-4 接真实下单）** | **SELL 上限与盈亏视图直接信任物化列，而非不可变 Fill 重放** | `ledger.go:309-319` 用 `position.RemainingQty` 减 `reserveOpenOrders` 判断；`snapshot.go:23/:61` 直接返回物化 `RealizedPnL`。探针：把 `remaining_qty` 改为 999 后 `CreateOrder(SELL 999)` **成功并持久化**（`requested=999 status=NEW`）；把 `realized_pnl` 改为 999 后 `Accounting` 返回 `realized=999 total=10989`。正常路径下 `RecordFill` 每次 `reprojectPosition` 会保持一致，风险窗口是**外部改库 / 历史脏数据 / 重启后未 Rebuild**；而 06-3 没有任何运行时调用 `RebuildPosition` 的接线。06-4 若把该 `RequestedQty` 直接发往 Binance，可能误卖手工钱包资产——正是方案 §1.5 与实施记录 §五 的核心防线 | 在 `CreateOrder` 事务内先按 Fill 重放可信剩余量（或复用 `reprojectPosition` 校验）再比较；`Accounting` 独立重放或显式要求先 `RebuildPosition`；06-4 发单前增加「从 Fill 重算的净持有量」二次校验并写永久测试 |
| **06-3-F2** | **P2（需 06-4 前裁决）** | **状态机两套规则冲突：`reprojectPosition` 可把终态订单改写为 FILLED** | `fills.go:284-285` 对 `filled == requested` 的订单无条件 `status="FILLED"`，绕过 `SetExchangeStatus::transitionAllowed` 的 `terminalStatus` 保护。探针 `before=CANCELED after=FILLED`；而永久测试对「CANCELED + 部分成交」断言保持 `CANCELED`。「成交事实优先」与「终态不可变」两种语义并存，会让 06-4 恢复逻辑与 06-5 页面对同一订单给出不同解释 | 明确裁决：若「成交事实优先」，同步修正 `transitionAllowed` / 文档 / 测试注释；若「终态不可变」，`reprojectPosition` 应保留 CANCELED/REJECTED 并单独标记异常。裁决前不得把状态机当作 06-4 恢复依据 |
| 06-3-F3 | P3 | 异常 / 迟到 Fill 恢复是硬失败，无人工修复入口 | `RecordFill` 允许 `TradeTime <= ClosedAt` 的历史 Fill，但探针 `PROBE063_LATE_BUY_REOPEN` 被 `order 1 fill qty exceeds requested qty` 整笔回滚。06-4 的 REST `myTrades` 恢复若遇到本地 `requested_qty` 与实际成交不符（部分撤单、手工干预、历史数据），将无法恢复；当前无 API / CLI / 状态标记可介入 | 06-4 定义最小化的「需人工复核」标记或恢复流程并写异常测试；按方案简化原则不新增复杂状态机 |
| 06-3-F4 | P3 | 库层零开关门控，06-4 必须显式接线 | Grep 实证 `service/spotledger` 无 `spot_quant_enable` / `spot_quant_enabled` 引用。符合 06-3 阶段边界，但 06-2-F6 只是「语义冻结」，**不是已实现的门控**；检查单 §2 的要求也尚无代码 | 06-4 在调用 `CreatePosition` / `CreateOrder` 前强制「全局 `config.spot_quant_enable=1` **且** 单币 `spot_quant_enabled=1`」，总开关 OFF 时保留币级偏好但不发新自动单；必须以永久测试覆盖「全局 OFF / 单币 OFF / 双 ON」三态 |
| 06-3-F5 | P3 | MySQL 证据边界与索引未测 | 集成测试默认 SKIP（本会话实证 `--- SKIP`，未连任何 MySQL）；实施记录所称 MySQL 5.6 Docker 容器测试由开发侧执行、容器已删除，本会话无法复跑；MySQL 8 生产环境明确未验证。生产 v25 DDL 有 4 个索引（`idx_spot_positions_symbol_status` 等），测试 fixture 未建任何索引；且 `spot_trade_fills.trade_time` 无索引，而 `reprojectPosition` 每次 `ORDER BY trade_time,id` → 大表 P95 / EXPLAIN 未测 | 06-6 前在目标 MySQL 版本补只读 `EXPLAIN` / P95 与索引评估；生产 DDL 与权限预检按检查单执行并留存证据。**不得写「MySQL 已验证」** |
| 06-3-F6 | P3 | 文档滚动基线轻微滞后 | 总方案 `:6` 当前基线仍写 HEAD `af07191`，实测 `2e374b4`；`:4` 状态行已更新为「06-0～06-3 本地开发与离线验收完成，06-3 待 Review」 | 每次阶段提交后同步基线行；不影响本阶段代码结论 |

**未发现**：资金安全 P1（零运行时接线、零网络调用）、SQL 注入可绕过（WHERE 全参数绑定、LIMIT/OFFSET 为已 clamp 的 int）、`float64` 参与资产计算、SHORT 方向放行、Futures 表被写、自动下单接线、旧方案 02 模式 / Owner / 审批状态机复活。

---

## 八、Gate 结论

| Gate | 判定 |
| --- | --- |
| 06-3 本地代码与 SQLite 离线验收 | ✅ **PASS**（研发基线）：验收矩阵 1~8、10、11 全绿，10 个永久回归 + 探针正向不变量全绿，651 文件副本等价 |
| 06-3 阶段整体验收 | ⚠️ **有条件通过**：验收项 9（MySQL 完整交易与重启）只有开发侧声明、本会话未复跑，MySQL 8 未验证；06-3-F1/F2 是面向 06-4 的阻塞项 |
| 06-2 → 06-3 前置 Gate | ✅ 已闭环（采用上一轮报告允许的「更正声明 + 发布 SOP」方案，四处口径统一） |
| 06-3 → 06-4 | ⚠️ **有条件**：允许 06-4 本地 Fake/Testnet 开发；**接入真实 Binance 下单前必须先闭环 06-3-F1、F2、F4 并补永久测试**（尤其「从 Fill 重算净持有量」的二次校验与双开关门控） |
| 方案 06 完成 / 06-5 菜单对外 / 06-6 成套交付 | ⛔ 不认可、未放行 |
| 生产发布、真实 Binance Spot、真实账户下单、T0、MySQL 8 生产 DDL / 权限、`dist→static` + 二进制成套发布、发布前 Schema 检查单执行记录 | ⛔ **全部 BLOCKED** |

一句话：**06-3 把「账本该怎么写」这件事做对了——精确十进制、Fill 不可变、重放可修复、手续费不折价、Futures 零污染、零运行时接线；但「谁可以依据这个账本去下单」还没做对——SELL 上限与盈亏视图信任的是可被改坏的物化列，状态机还有两套语义，开关门控尚未接线。这三件必须在 06-4 摸到真实下单 API 之前闭环。**

---

## 九、遗留与操作边界

**遗留项（按优先级）**：① 06-3-F1（Fill 重放校验 + Accounting 可信度）；② 06-3-F2 状态机语义裁决；③ 06-4 双开关门控永久测试（F4）；④ 异常恢复标记（F3）；⑤ 06-6 前 MySQL 8 / 索引 / EXPLAIN（F5）；⑥ 发布前 Schema 检查单执行记录；⑦ 文档基线同步（F6）。

**本次未执行**：不读 `conf/app.conf`、不连任何 MySQL（集成测试 SKIP）、不访问真实 Binance、不执行 `sync db`、不操作真实仓位、不重启服务、不替换后端二进制、不同步 `static/`、不 commit / push、不碰 `strategy_templates/` 与用户其他未提交文件。探针仅在 `/tmp/phase_audit_063` 内，收尾随副本删除。

**报告不覆盖历史**：`CODE_REVIEW_06-0_06-1.md`、`CODE_REVIEW_06-0_06-1-rereview.md`、`CODE_REVIEW_06-2.md`、`CODE_REVIEW_06-2-rereview.md` 原文保留；若后续整改 06-3-F1/F2，复评须另起 `CODE_REVIEW_06-3-rereview.md`。
