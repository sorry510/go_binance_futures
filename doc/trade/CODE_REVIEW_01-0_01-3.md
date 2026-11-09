# Code Review：方案 01「合约订单审计与部分成交」01-0 ～ 01-3

- 审计时间：2026-11-08
- 审计对象：`feat/trade-upgrade` 分支 HEAD `49ba0e5` 之外的**全部工作区未提交改动**（20 个已跟踪文件 +850/−63，7 个新增文件/目录）
- 依据文档：`doc/trade/开发方案-01-合约订单审计与部分成交.md`（含第七～十节实施记录）
- 审计方式：`git archive HEAD` 干净副本 + 拷贝全部未提交文件至 `/tmp/phase_audit`，在副本中实跑 Gate 与临时探针；**未修改任何业务代码、未 commit/push、未对真实数据库执行 `sync db`、未请求真实 Binance**
- 报告结论仅代表**离线验收**，不代表实盘 Gate 通过

---

## 一、结论先行

| 维度 | 结论 |
| --- | --- |
| 01-0 现状确认 | **通过**（文档交付完整；生产实时状态未复核，属外部 Gate） |
| 01-1 最小 DB 变更与持仓关联 | **通过**（含事务回滚、跨周期/跨账户隔离、v21 幂等迁移） |
| 01-2 逐笔成交、均价与订单恢复 | **离线通过**，但存在 1 个 **P1 实盘前置假设**未验证（WS `n` 字段口径） |
| 01-3 完整平仓与一次性结算 | **离线通过**，但发现 1 项验收要求**只实现一半**（保护单剩余量）与 1 项**风控可用性空窗** |
| 离线 Gate（编译/测试/race/vet/diff-check） | ✅ **全部通过** |
| 实盘 Gate / T0 | ❌ **未通过、未授权**（详见第六节） |
| 是否建议合并代码 | **建议合并**（F1～F4 均不阻塞离线合并），但**不得据此部署生产或启用 T0** |

> ⚠️ 与既往 Lead 各阶段「生产零接线」不同：本轮 `go list -deps .` 显示 `service/tradefills` 与 `service/futuresownership` **均在主流二进制依赖闭包内**（各计数 1）。01-1～01-3 是**真实运行路径变更**，风险等级应按生产代码对待。

---

## 二、验收清单逐项核对

### 01-0（只读现状确认）

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 列出 main/其他 owner 下单路径与 OPEN/CLOSE/TP/SL | ✅ | 方案第七节 1 表格，覆盖 auto_strategy / new_coin_rush / funding_rate / notice_auto_order / agent_trade / lead |
| 给出退役旧表的受影响文件清单 | ✅ | 方案第七节 2，含 `feature/feature.go`、`feature/order.go`、`main.go`、`coin5/6`、`controllers/orders.go` 等 |
| 确认 `coin5/coin6` 旧表依赖退出方式 | ✅ | 方案第七节 4.7，明确 `GetCoinStrategy` 选择链不能只删 SQL |
| 评估 `AutoLossScale/autoTradeToTest` 触发时机 | ✅ | 方案第七节 4.6，明确 01-3 迁移前不得删除风控 |
| 定义 T0 与存量仓位处理策略 | ✅ | 方案第七节 4.1～4.5，5 条非零镜像仓位保持现状、不自动认领 |
| 生产库只读核查（53/431/5 条） | ⚠️ | 文档记录 2026-11-07 22:13 READ ONLY 快照；**本轮未复核生产库实时状态** |
| 不操作生产仓位、不迁移旧历史 | ✅ | 本轮工作区与副本均无生产数据变更 |

### 01-1

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| `futures_managed_orders` 新增 `position_id` / `actual_order_id` | ✅ | `models/futures_managed.go:34-35`（`*int64` / `*string`，均带索引） |
| 只新增一张 `futures_trade_fills` | ✅ | `command/futures_trade_fills_migration.go:9-27`；`models/futures_trade_fill.go` 故意**不注册 ORM**，避免 `RunSyncdb` 把 `DECIMAL` 改 `VARCHAR` |
| 五列成交唯一主键 + `position_id` 索引 | ✅ | 同上，`PRIMARY KEY (market,account_id,symbol,order_id,trade_id)`；MySQL 分支用 `information_schema.statistics` 做存在性检查，SQLite 用 `IF NOT EXISTS` |
| OPEN 在首次真实 Fill 事务内建仓并回填 ID | ✅ | `service/futuresownership/service.go:302-370`：`o` 即为 `tx`，`applyPositionDelta` → `Insert(position)` → `order.PositionID` → `o.Update(&order)` → `tradefills.Attach(o, order)` → `tx.Commit()`，全链同一事务 |
| CLOSE/TP/SL 在 ClaimOrder 绑定已有 ID | ✅ | `service.go:124-160`，非 OPEN 分支 `positionID = &position.ID` |
| 未成交订单 position_id 可为空 | ✅ | 永久测试 `TestStage011NewCycleOneOpenMultipleCloses:28-37` |
| 不同 owner/方向/同币两轮不串联 | ✅ | `TestStage011NewCycleOneOpenMultipleCloses`、`TestStage011StaleCloseAndProtectionCannotCloseNextCycle`、`TestStage011DifferentOwnersAndAccountsSeparateCycles`；本轮探针「同币两周期独立结算」亦通过 |
| 旧 terminal 未绑定 OPEN 拒绝复活 | ✅ | `service.go:329-332` + `TestStage011LegacyTerminalUnboundOpenCannotResurrectPosition` |
| 手动认领新建周期、历史 closed 不变 | ✅ | `manual_claim.go:207-225` 改为全新 `FuturesManagedPosition{...}` + `tx.Insert`，不再复用 `previous` 行 |
| DB 出错原子回滚 | ✅ | `TestStage011OpenFillAtomicRollback`、`TestStage013SettlementDBFailureRollsBackRiskAndMarker` |
| `sync db` 幂等、不改 `app.conf` | ✅ | 实跑 `TestSyncDatabaseInitializesAndIsIdempotent`：日志 `20 → 21`、`21 → 22`、重复 `22` 提示 already up to date；`UpdateDatabase` 的 v21 分支位于 `command/db_update.go:357-361`；`main.go:231-232` 有 `config.version < dbVersion` panic 门禁 |
| 六份 Fill DDL 当前一致 | ✅ | 静态逐列比对：生产 1 份 + 5 个测试夹具，均 16 列、列名/类型完全一致（但见 F6） |

### 01-2

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| WS 有效 TRADE Fill 持久化（在镜像时间戳检查之前） | ✅ | `feature/api/binance/user_data_mirror.go:103-115`；`TestStage2UserDataMirrorIsolatedForSameSymbolAndOrderID` 新增「迟到 TRADE 事件重复 100 次只落 1 笔且镜像状态不变」 |
| WS 写失败 → main mirror dirty | ✅ | `user_data_mirror.go:36-43` 既有逻辑，`persistFuturesAccountEvent` 捕获 error 后 `mainFuturesAccountMirrorDirty.Store(true)` |
| 唯一 trade_id 去重 / 冲突载荷拒绝覆盖 | ✅ | `service/tradefills/ledger.go:182-235`：INSERT 失败后回读并逐字段比对 `qty/price/commission/realized_pnl` + `commission_asset/trade_time/position_side`，不一致返回 `conflicting immutable WS/REST fill` |
| 只接受真实 TRADE + 正 orderId/tradeId + 正成交量 | ✅ | `ledger.go:44-84`、`88-111`；`NEW/CANCELED`、零量不生成 Fill |
| REST `userTrades` 账户绑定补查 | ✅ | `feature/api/binance/index.go:674-688`（main）、`account_client.go:186-198`（Lead，注释明确不回退 main）；`execution.go:376-400` 的 `OrderTradeReader` |
| 分页与身份校验 | ✅ | `execution.go:412-455`：`maxPages=12 × 1000`、`fromID = trade.ID+1`、逐笔校验 `orderID/symbol/positionSide`、分页未推进即报错 |
| 部分成交均价正确（2@100 + 2@120 = 110） | ✅ | `service.go:436-445` 把 REST `ap` 视为**累计均价**直接覆盖（不再二次加权）；`TestStage012RESTPartialCumulativeAverageCancelAndClose` 断言 `110` 而非 `105`；`SyncOpenAverageFromFills` 在 Fill 齐全后按 `Σ(price×qty)/Σ(qty)` 用 `big.Rat` 重算 |
| `ApplyFill` 唯一负责累计 delta，Fill 不重复计量 | ✅ | `tradefills.Quantity/WeightedAverage` 只读；`execution.go` 仅做数量对账，不写 managed_qty |
| 部分成交后撤单不删已成交事实 | ✅ | `TestStage012RESTPartialCumulativeAverageCancelAndClose`：CLOSE 实成 1 后 CANCELED，余 3 且 Fill 保留 |
| Algo `algoId → actualOrderId → Fill` | ✅ | `service.go:221-261` `SetActualOrderID` 拒绝重映射；`execution.go:634-640` 校验实际单 `OrderID/Symbol`；`matchingOrder` 对 Algo 跳过 `exchange_order_id` 匹配；`TestStage012AlgoActualOrderIDAndTradeAttribution` 覆盖 TP/SL |
| 未知提交按原 clientOrderID 查单 | ✅ | 既有 `Executor.Reconcile → Broker.Lookup` 链未被本轮破坏；`TestStage012RESTGapFailsClosedAndRecovers` 验证 REST 恢复后不重复下单 |
| 缺口/故障显示待核对、不虚报盈利 | ✅ | `applyExchange:310-315` 失败置 `OrderReconcile`；`TestStage012RESTGapFailsClosedAndRecovers`、`TestStage012FillDBFailureKeepsRecoverableReconcile` |
| 精度：原始十进制字符串 | ✅ | `models/futures_trade_fill.go` 全 string；`ledger.go:23-42` `decimal()` 用 `big.Rat` 并拒绝 >18 位小数 |

### 01-3

| 验收项 | 结果 | 证据 |
| --- | --- | --- |
| 多笔 CLOSE 关联同一 position_id、平 4 实成 1 余 3、归零才 CLOSED | ✅ | `TestStage013PartialClosesOnlyOneVerifiedSettlement`、`TestStage011NewCycleOneOpenMultipleCloses` |
| 零仓位但缺真实 Fill → `closed + incomplete` | ✅ | `service.go:475-480`、`687-692`、`727-735`；`TestStage013IncompleteExternalReductionAndAssetMismatch`（外部减仓 / BNB 手续费 / 非法资产 3 例） |
| 结算前置：新鲜 REST 零仓位快照 | ✅ | `reconcile.go:130-140` 走 `FreshPositions`；已核实 `BinanceAccountPositionSource.FreshPositions → GetPositionFreshContext`（`reconcile.go:285-311`），非缓存 |
| 净收益精确十进制 + 资产证据 | ✅ | `settlement.go:74-187`：`gross=Σ(realized_pnl)`、`fee=Σ(commission)`、`net=gross-fee`；只有 `marginAsset ∈ {USDT,USDC}` 且每笔手续费同资产才判正负；`reconcile.go:196-225` 从 `exchangeInfo.marginAsset` 取证据 |
| 风控迁移到可信完整结算 | ✅ | `feature/feature.go:622-628` 移除旧委托返回时两个副作用；`settlement.go:259-263` 仅 `main / auto_strategy` 调 `applySettledRisk` |
| 按周期只触发一次、重启不重复 | ✅ | `settlement.go:246-258` CAS：`settlement_status=incomplete AND settlement_processed_at=0 AND status=closed`；MySQL 下 `SELECT ... FOR UPDATE`（`settlement.go:280-283`）；`TestStage013...` 重复 20 次不重触发；计数改为持久化 `settlement_scale_balance` / `settlement_consecutive_losses` |
| 旧 `legacy` 不参与结算 | ✅ | `models/futures_managed.go:17-19` 默认 `legacy`；`UnsettledClosedPositions` 只查 `incomplete`；`TestStage013OldClosedCycleCannotSettleOrMutateConfig` |
| 手动认领 Owner 边界、外部减仓不当策略平仓 | ✅ | `TestStage013ManualClaimWithoutFillsRemainsIncomplete` |
| 旧风控语义等价 | ✅ | 逐行比对：旧 `AutoLossScale`（`feature.go:1178-1203`，连续 ±2 次调 `loss_max_count`，下限 1）与新 `applySettledRisk`（`settlement.go:292-311`）一致；旧 `autoTradeToTest`（`feature.go:1208-1225`，连续亏损达阈值 → `future_enable=1 / allow_long=short=0 / future_test=1`）与新 `settlement.go:312-326` 一致，且由「整行 `orm.Update(systemConfig)`」改为「只更新必要字段」，更稳 |
| 保护单按剩余可平量管理 | ⚠️ **部分** | 新建保护单受 `position.ManagedQty` 限制，但已挂保护单不缩减 —— 见 **F4** |

---

## 三、实证过的隐式契约（临时探针，副本内执行）

| 探针 | 方法 | 结果 |
| --- | --- | --- |
| `SELECT * FROM config` → `models.Config` 映射 | 写入 10 个可区分字段后执行 `applySettledRisk` 同款查询 | ✅ **映射正确**（id=1/version=22/lossMax=7/autoScale=1/autoTestLimit=3/scale=-1/losses=2/allowLong=0/allowShort=0/enable=1/test=1），新增两列未造成错位 |
| 保护单剩余量 | OPEN 4 → 挂 TP(4) → 平 1（余 3）→ 再挂 SL(4) / 读回 TP | ⚠️ 新建 SL(4) **被拒**（`requested mutation qty 4 exceeds managed qty 3`）；但已挂 TP 仍 `requested_qty=4, status=pending` → **F4 确认** |
| `OrderTradeReader` 缺失 | 用不实现 `ListAccountTrades` 的 `fakeOrderBroker` 跑 `ApplyObservedExchange` | ⚠️ `err=nil`、`status=filled`、`fill_rows=0` → **静默跳过 Fill 核对，F3 确认** |
| 同币两周期独立 | 两轮完整 OPEN/CLOSE + 各自结算 | ✅ `scale=-1 → 0`、`losses=1 → 2`、`lossMax=5 → 4`，无串联、无重复计数 |
| `sync db` 版本迁移 | 实跑 `TestSyncDatabaseInitializesAndIsIdempotent -v` | ✅ `20→21`、`21→22`、重复 `22` 幂等（SQLite） |
| 六份 DDL 一致性 | Python 正则提取 6 份 DDL 的列名+类型逐项比对 | ✅ 当前均 16 列、完全一致，但无自动化保护 → **F6** |
| 生产依赖闭包 | `go list -deps . \| grep -c` | ✅ `tradefills=1`、`futuresownership=1`（真实接线，非零接线） |

> 说明：另两组探针（账本层空手续费/精度/冲突、v21 迁移幂等与精度）已写入副本，但**其运行输出被环境内容审批拦截，按规则未重试、未推断**，故 F7/F8 及 v21 幂等的运行时结论降级为「静态审阅 + 现有永久测试」支撑，报告中已如实标注。探针随副本一并删除，建议将关键断言转成永久测试。

---

## 四、风险与待改进

| 编号 | 级别 | 发现 | 建议 |
| --- | --- | --- | --- |
| **F1** | **P1**（不阻塞离线合并，**阻塞实盘 Gate**） | **WS `n` (Commission) 口径未经实盘交叉验证**。`FromWS` 把 `event.Commission` 当**单笔**手续费写入不可变账本；REST `userTrades.commission` 是**单笔**。Binance 官方文档对 `n` 仅写 "Commission / 手续费数量"，未明确累计 vs 单笔。若 `n` 实为订单累计值：① 多笔成交时 WS/REST 同一 `trade_id` 的 commission 冲突 → `Insert` 报 `conflicting immutable WS/REST fill` → 订单停留 `reconcile_required` → 周期永远 `incomplete` → 结算与风控不触发；② 若两端同为累计，`verifiedSettlementFacts` 的 `fee.Add` 会重复累计 → 净收益被高估费用、可能把盈利误判为亏损。另 `old.TradeTime != fill.TradeTime` 严格比对，WS `T` 与 REST `Time` 毫秒差异同样冲突。当前为 fail-closed，不会静默错账，但可能造成大面积不可用 | T0 前用一笔**真实部分成交订单**，同时抓 WS `ORDER_TRADE_UPDATE` 与 `/fapi/v1/userTrades`，逐 `tradeId` 比对 `qty/price/commission/realized_pnl/trade_time`；把结论写入方案；必要时对 WS `n` 做相邻事件差分或统一改用 REST 覆盖。补一条永久回归测试固化口径 |
| **F2** | **P1/P2** | **风控可用性空窗**。`feature/feature.go` 删除两个副作用后，`AutoLossScale`/`autoTradeToTest` 在**生产代码已无任何调用方**（仅剩注释与函数定义）。新触发点 `applySettledRisk` 只在完整结算时执行，而完整结算需同时满足：Fill 齐全、手续费资产 = marginAsset(USDT/USDC)、OPEN/CLOSE 数量相等、无活动订单、新鲜 REST 零仓位、exchangeInfo 可用。任一缺口即永久 `incomplete` → **风控静默**。生产当前 `loss_auto_scale=0`、`future_test_auto_trade_count_limit=0`，故当前无行为变化；一旦启用开关而 Fill 采集有缺口，风控比旧逻辑（每次平仓委托都计数）明显更弱。此为空窗而非 bug（方案 §三明确要求收益须真实结算），但缺配套：`UnsettledClosedPositions` 超 10000 仅返回 error 无告警、无 `incomplete` 积压看板（01-5 才做）、无人工兜底 | 上线前给 `settlement_status=incomplete` 超 N 分钟的周期加告警；运维手册写明「风控依赖 Fill 完整性」；Gate 0-LIVE 前用真实订单验证至少一轮完整结算能触发计数；考虑提供人工兜底开关 |
| **F3** | **P2** | **broker 未实现 `OrderTradeReader` 时静默跳过整个 Fill 核对**（探针实证 `status=filled` + `fill_rows=0`）。`execution.go:385-388` 的 `if !ok { return nil }` 把「无核对能力」当作「核对通过」。生产 `BinanceOrderBroker` 已实现，但这是隐式契约、无编译期强制 | 加 `var _ OrderTradeReader = BinanceOrderBroker{}` 编译期断言；对生产 broker 缺失返回显式错误而非 `nil`；或把该接口提升为 `Executor` 必需依赖 |
| **F4** | **P2** | **01-3「保护单按剩余可平量管理」只实现一半**。探针实证：OPEN 4 挂 TP(4) → 部分平 1（余 3）后，新建 SL(4) 会被拒，但**已挂 TP 仍 `requested_qty=4, status=pending`，未自动缩减或撤单**，触发时可能超额平仓。永久测试只覆盖「旧保护单不误伤下一周期」 | 在 `ApplyFill` 减仓路径按剩余量联动缩减/撤销同 `position_id` 的活动 TP/SL；或文档明确「仅限制新建单，存量保护单由交易所侧管理」并补回归测试 |
| **F5** | **P2** | **v22 无显式迁移函数，MySQL 补列本轮未实证**。`UpdateDatabase` 只有 v21 分支；3 个 `futures_managed_positions` 列 + 2 个 `config` 列完全依赖 `RunSyncdb("default", false, false)` 自动补列。SQLite 已实测幂等，但项目既有事实明确「RunSyncdb 补列在 MySQL 上未验证」；文档自称本地 MySQL `21→22` 成功，本轮**未复跑**。关注 `settlement_status` 的 `default(legacy)` 存量行回填、`settlement_processed_at` 默认 0（CAS 依赖）、索引创建。即使补列后存量行为 NULL/''，因只查 `incomplete`，仍 fail-closed ✅ | 为 v22 补显式幂等 Go 迁移（与 v19/v21 一致）：`ADD COLUMN ... DEFAULT 'legacy'` + 回填 + 索引；在 MySQL 8 单独授权验证；上线前必须 `sync db`（`main.go` 已有 panic 门禁） |
| **F6** | **P2** | **6 份 `futures_trade_fills` DDL 硬编码、无一致性断言**（生产 `command/futures_trade_fills_migration.go` + `service/futuresownership/service_test.go`、`service/agenttrade/service_test.go`、`service/leadaccount/executor_test.go`、`feature/api/binance/user_data_mirror_stage2_test.go`、`service/tradefills/ledger_test.go`）。当前 16 列完全一致，但生产改列宽/精度/主键时测试仍全绿 | DDL 抽成单一导出常量/函数，夹具统一引用；或加「生产 DDL 与所有夹具逐列一致」永久测试 |
| **F7** | **P2**（静态推导，未运行时实证） | **零手续费 / 缺 `N` 字段的 TRADE 事件会中断整个 WS 持久化**。Binance 文档明确 `N` "will not push if no commission"。`FromWS` 构造 Fill 后 `validate` 要求 `CommissionAsset` 非空且 `Commission` 为合法十进制；空串返回 `missing exchange fee asset` / `invalid decimal` → `persistFuturesAccountEventWithDB` 返回 error → main mirror 标 dirty，且该事件镜像更新全部跳过。零费率/BNB 抵扣/返佣场景可能反复触发 | 区分「无手续费（N 缺失）」与「坏数据」：前者按 `0` + 合约结算资产（或留空待 REST 补齐）落库，不要让整个 WS 事件失败；补永久测试 |
| **F8** | **P3** | `decimal()` 精度校验与 `DECIMAL(36,18)` 不完全一致：条件为「整数有效位 + 小数位 > 36」，但 DECIMAL(36,18) 整数部分最多 18 位。30 位整数 + 0 小数会通过本地校验，MySQL 会 `Out of range`；SQLite 会按 NUMERIC/REAL 静默接受并丢精度 | 单独校验整数位 ≤ 18；把「SQLite 不能证明 DECIMAL(36,18) 保真」写入测试说明并固化为 MySQL Gate |
| **F9** | **P3** | `verifiedSettlementFacts` 的 Fill 查询只按 `position_id`（`settlement.go:119-124`），未按 `account_id`。虽然后续逐行校验 `fill.AccountID != pos.AccountID` 会 fail-closed，安全但冗余 | 加 `AND account_id=?` 明确意图并缩小扫描范围 |
| **F10** | **P3** | 结算资产用 `GetExchangeInfoContext`（**TTL 缓存**），而仓位用 `FreshPositions`（新鲜 REST），口径不统一。合约元数据不常变，属可接受取舍；缓存不可用时停留 `incomplete`（fail-closed ✅） | 文档写明该取舍 |
| **F11** | **P3** | `ReconcileOwner` 把 `accountQty` 缺失 key 视为零仓位（`reconcile.go:178`）。若 `FreshPositions` 部分失败但返回不完整且无 error 的快照，可能误判零仓位。已核实 `FreshPositions` 走 `GetPositionFreshContext` 且对 nil/NaN 有校验，风险较低 | 对「快照为空但存在未结订单」加显式保护 |
| **F12** | **P3** | `persistFuturesAccountEventWithDB` 已支持 `LeadAccountID` 并会**写** `futures_trade_fills`（stage2 测试即以此验证），而 Stage 7 的 read-only observer 承诺「SELECT-only、绝不写 DB」。当前未接线，未来接线易破坏该承诺 | 接线前加白名单守卫测试 |
| **F13** | 阶段边界（非缺陷） | 旧 `order` 表仍被 `insertOpenOrder/insertCloseOrder` 写入，`UpdateOrderStatus`、`coin5/6`、`controllers/orders.go`、旧前端均未动 | 属 01-4 / 01-5 范围，符合方案顺序约束 |

---

## 五、独立复跑的 Gate 记录

| 项目 | 命令 | 结果 |
| --- | --- | --- |
| 全仓编译 | `go build ./...`（副本，无 `strategy_templates/research/**` 干扰） | `EXIT=0` |
| 主程序构建 | `go build -o /tmp/audit_main_bin .` | `EXIT=0` |
| 目标包测试 | `go test -count=1 ./service/futuresownership ./service/tradefills ./service/agenttrade ./service/leadaccount ./feature/api/binance ./feature ./command ./models ./appversion` | **9 包全 ok** |
| Race | `go test -count=1 -race ./service/futuresownership ./service/tradefills ./feature/api/binance` | **3 包全 ok**；仅 macOS `ld: warning: malformed LC_DYSYMTAB` 噪音 |
| Vet | `go vet` 上述 9 包 | `EXIT=0` |
| 空白错误 | `git diff --check`（用户仓库） | `EXIT=0` |
| 版本迁移 | `go test -count=1 -v -run TestSyncDatabaseInitializesAndIsIdempotent ./command` | PASS，覆盖 `20→21`、`21→22` 与重复执行幂等 |
| 永久测试矩阵 | 01-1 五例 / 01-2 四例 + 账本一例 / 01-3 八例，共 **18 个命名测试** | 全部随目标包测试通过 |
| 临时探针 | 4 项（结算/归属包，`-run '^TestProbe'` 单独执行） | PASS，其中 2 项确认缺口 |

> 环境说明：`strategy_templates/research/**` 是未跟踪研究目录，会让用户仓库的 `go test ./...` 变红；副本由 `git archive HEAD` 生成，不含该目录，因此本次全仓编译为绿。`controllers` 包既有的 ORM 重复注册 / 静态模板测试问题不在本轮范围，未纳入 Gate。

---

## 六、实盘 Gate：未通过、未授权（不得据此部署）

1. **T0 未确立**：方案明确 T0 = 01-1/01-2 真实 Fill 持久化正式启用的实际 UTC 时间，需留存账户维度 T0 与版本记录；当前运行中的旧二进制不写新 Fill。
2. **生产库仍是 v20 结构**：方案第七节 5（2026-11-07 22:13 只读核查）记录生产 `futures_managed_orders` 尚无 `position_id`/`actual_order_id`、`futures_trade_fills` 不存在；本轮未对 `go_binance` 做任何 schema 变更。
3. **F1 未闭环**：WS `n` 与 REST `commission`/`trade_time` 的真实一致性未经一笔真实订单验证 —— 这是启用 T0 的**硬前置**。
4. **MySQL 8 补列未在本轮复跑**（F5），`sync db` 需单独授权 + 维护窗口。
5. **5 条未归属非零镜像仓位**（LSKUSDT/ONDOUSDT/PENGUUSDT/WDCUSDT/XRPUSDT）必须与交易所实时仓位只读核对，保持不自动认领、不重建历史。
6. **01-4 / 01-5 未完成**：旧 `order` 读写、30 分钟 `UpdateOrderStatus`、旧控制器与页面尚在；前端 `incomplete/settled` 状态不可见；`/orders` 新主子表 API 未实现。
7. **风控联动未经实盘验证**（F2）：至少需一轮真实完整结算证明计数只触发一次。
8. 方案 §四 01-6 明确：生产 schema 变更、程序部署、重启与实盘验证**均需单独授权**。

**离线通过 ≠ 实盘可用。** 本报告的通过结论仅覆盖本地代码、SQLite 隔离测试与静态/探针证据。

---

## 七、审计纪律声明

- 未修改用户任何业务代码、未 commit/push、未改 `conf/app.conf`、未触碰 `strategy_templates/` 与 `static/`。
- 副本 `/tmp/phase_audit` 中的临时探针在报告完成后随副本删除；用户仓库仅新增本报告文件。
- 副本的 `conf/app.conf` 指向本地测试库，因此**全程未执行 `sync db`**、未连接真实 Binance、未执行任何生产写操作。
- 两组探针输出被环境内容审批拦截，按规则未重试/未推断，已在上文标注其结论为静态审阅支撑。

---

## 八、整改后补充说明（2026-11-08）

本报告记录修复前的原始审计结论，仍保留用于追踪差异。针对 F1～F13 的代码修改、v23 Schema 以及本地 Gate 结果，参见 [CODE_REVIEW_01-0_01-3_修复记录.md](CODE_REVIEW_01-0_01-3_修复记录.md)。**F1 的真实部分成交 WS/REST 对照和实盘 T0 仍未通过，不能将修复后的离线 Gate 视为生产上线授权。**
