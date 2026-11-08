# CODE_REVIEW_01-4 整改记录（2026-11-08）

依据：[CODE_REVIEW_01-4.md](CODE_REVIEW_01-4.md)。本文件记录审查之后的**实际代码修改与离线验证**，不覆盖原报告当时的观测。当前仍处于 **01-4 本地研发阶段**；01-5/01-6 和真实 Binance/T0 Gate 仍未通过。

## 按 review 逐项处理

| Finding | 改动 / 不变边界 | 结论 |
| --- | --- | --- |
| **014-F1 P2** | 删除 `feature/feature.go` 无调用方的 `AutoLossScale`、`autoTradeToTest` 及非持久化计数器；`service/futuresownership/settlement.go::applySettledRisk` 仍在 **verified flat settlement 事务** 中作为唯一生效自动亏损控制。添加唯一入口说明，删除测试交易路径遗留的旧函数注释。现有 `TestStage013DurableLossScaleAndAutoTestAfterTwoLosses`、结算失败回滚与重复结算测试继续通过。**不得用估算 PnL / ACK 创建第二条自动亏损控制回退**；若 Binance 不提供方向级归零证据，持续 incomplete 并进入超过 15 分钟的既有「报警」告警日志。真实归零 API 数据和主动通知/人工兜底仍为上线硬前置。 | 代码整理与回归完成；**实盘 Gate 未关闭** |
| **014-F2 P2** | 新建 `service/tradefills/cooldown.go::RecentClosedSymbols`，使用 `futures_trade_fills.trade_time` 与正的实际成交 `qty`，要求真实 Fill 与 `futures_managed_orders` 精确关联 `client_order_id/account_id/symbol/intent`，只限 USDM 的 CLOSE/TP/SL 和目标账户；允许原始 WS 成交虽费用未 REST 核实，但不将无身份的 manual/orphan 记录混入。两处调用统一替换：`scanner/local_selector_v2.go` main/Lead 和 `feature/strategy/coin/common.go` coin5/coin6。明确查询截止于当前时间，不能用过期订单刷新重置冷却。**新增两套永久反例**：「订单 updated_at 很旧但 Fill.trade_time 最近」必须冷却、「订单 recently updated 但 Fill 很早」不得冷却；账户隔离、零成交、开单、部分保护单成交原测试保留。查询缺表/失败时不把错误当空历史。 | **已修复并测试** |
| **014-F3 P3** | `feature/order_retirement_stage01_4_test.go` 从七个源文件的字符串黑名单提升为遍历生产 Go 源文件的 AST 检查：扫描非测试 .go，跳过注释、测试和独立研究产物；检测旧 ORM `models.Order`（支持 import alias）、`QueryTable("order")`、旧历史 API 函数和以旧表为目标的 SQL 字符串，含带反引号/双引号的情况。额外检查 `main.go` 的唯一归档 Order 使用必须在精确 sync-db IfStmt 内；不是 Go 全程序数据流/反射形式化证明。 | **已加固** |
| **014-F4 P3** | 新增 root `legacy_order_gate_stage01_4_test.go`，验证 `isSyncDatabaseCommand` 对普通启动、doctor、cleanup、缺参、未知子命令、额外 flags 返回 false，对严格 `sync db` 返回 true。AST 对 `orm.RegisterModel(new(models.Order))` 在 `main.go` 的唯一性和必须由精确 sync-db 条件包围进行断言，**普通启动绝不恢复旧 ORM**。如将来扩展 CLI 参数，需同步修改 helper/单测和历史版本兼容验证。 | **已加固** |
| **014-F5 P3** | 完整 controllers 测试的技术指标为空/包含 `family` 字段等模板测试红点属于旧代码/其他研究输入。没有修改他人的策略文件或这组独立业务测试；继续分别报告目标包绿色与 controllers 全量非绿。 | **独立遗留项，未混入 01-4** |
| **014-F6 P3** | 01-4 临时 HTTP 410 API 和升级中页面继续保持本地、**不覆盖运行中二进制所使用的 static**。01-5 要同步替换 API 和 Vue、删除旧 i18n；01-6 部署必须原子发布前后端或在维护窗口同步切换并准备回滚。 | **部署 Gate 保留** |
| **014-F7 P3** | 没有访问生产库 `go_binance`、未猜测当前生产策略类型；`coin5/coin6` 与默认 `smart_local_v2` 已统一到真实 Fill 口径。生产只读核对 FutureStrategyCoin 和启动选币回归需要独立授权。 | **上线 Gate 保留** |
| **014-F8 P3** | AST 允许的历史用途缩窄至 `models/tableStruct.go` 的归档结构、`command/account_scope_migration.go` 的旧 v19 数据迁移、以及单个 `main.go` 的 sync-db 条件注册；另检查 command 非测试 Go 文件不可新增执行旧 order SQL。历史 schema 兼容不等于把旧表纳回交易历史。 | **已加固** |

## 代码/回归测试

- 新生产实现：`service/tradefills/cooldown.go`；调整 `scanner/local_selector_v2.go`、`feature/strategy/coin/common.go`；清理 `feature/feature.go` 老风险死代码、`feature/feature_test_strategy.go` 已失效注释；`service/futuresownership/settlement.go` 标明唯一有效风险结算入口。
- 增强测试：`scanner/local_selector_v2_db_test.go`、`feature/strategy/coin/cooldown_stage01_4_test.go`（SQL 逐笔 Fill 事实），`feature/order_retirement_stage01_4_test.go`（AST 全仓与迁移例外），`legacy_order_gate_stage01_4_test.go`（纯 CLI 入参门禁）。
- `go test -count=1`：scanner、feature、feature/strategy/coin、tradefills、futuresownership、agenttrade、leadaccount、feature/api/binance、command、models 共 **10 包全部通过**；root `TestStage014LegacyOrderSchemaCLISelection` 及 controllers `TestStage014LegacyOrdersRoutesReturnGoneWithoutDB` 分别通过。
- `go test -race -count=1`：scanner / feature / coin / tradefills / futuresownership 相关 01-4/01-3/R3 回归 **全部通过**；macOS `LC_DYSYMTAB` 链接器有已知非致命 warning。
- 以上相关包 + controllers + root 的 `go vet`、真实入口 `go build -o /tmp/... .`、`git diff --check` **通过**。完整 controllers 包的既有模板测试红点不应被误写为全部 Go 测试通过；research 目录包含独立运行脚本，同目录同名 `main`，不作为正式主程序 Gate。
- 前端本轮未修改，继承 01-4 通过的 `pnpm build` 与不发布 static 约束；没有执行新的 `pnpm build`。

## 需要真实交易所/生产授权的剩余条件

1. **014-F1 不得提前关单/结算**：必须证明 Binance `PositionRisk(symbol)` 对已归零方向返回明确可核验 row；在生产/正式 Testnet 下抓真实样本（只读验证）并核对 `applySettledRisk` 每个 cycle 只触发一次。不具证明的 incomplete 继续报警、阻止风控副作用；明确值守与人工核查流程。
2. **014-F2 性能 Gate**：新冷却查询使用 `(market,account_id)` 主键前缀和真实 trade_time，但 v23 未单独建立 `trade_time` 辅助索引。真实成交表增长后，必须用目标数据库 **EXPLAIN / 运行时间和 QPS** 实测；若不能满足扫描频率，另开正式版本迁移建立按账户/时间/intent 的索引，绝不能在 v23 下偷偷修改生产 DDL，也不能为提速退回错误的 `updated_at` 时间来源。
3. 前后端 01-5 主子表、结算 incomplete/settled 可视化，01-6 生产灰度/回滚、Lead/其他 Owner 保护单实盘核验、R3 真实部分成交 WS/REST 对照仍待完成。

**安全界线：**DB Schema 仍 v23；未执行 `sync db`，未更改 `conf/app.conf`、生产库、真实 Binance 订单、运行中 PID/二进制、后端 static，也未 commit/push。
