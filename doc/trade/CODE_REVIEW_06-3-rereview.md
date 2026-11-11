# CODE_REVIEW_06-3 整改复评

日期：2026-11-11
分支：`feat/trade_spot`，本地 HEAD `2e374b4`（未自动提交整改）
原报告：[CODE_REVIEW_06-3.md](CODE_REVIEW_06-3.md)（保留原文）
范围：06-3 本地仓位 / 订单 / Fill / 盈亏；**没有接入真实 Binance Spot 订单**。

## 一、原 Findings 处理结果

| ID | 原始级别 | 处理结果 | 代码与证据 |
| --- | --- | --- | --- |
| **06-3-F1** | P2 / 06-4 前阻塞 | **已修复：交易容量与 PnL 不再信任可修改的汇总字段** | `service/spotledger/fills.go` 新增 `projectPosition`，统一只以既有真实 Fill 和关联订单重建持仓数量、成本与 realized PnL；`ledger.go::CreateOrder` 在同一锁定事务内完成投影与写回后才放行新委托，`CreatePosition` 也重新核实之前的同币种周期是否真正平仓；`snapshot.go::Accounting` 在只读事务中独立投影，不将物化字段当作权威值。探针改为永久测试 `TestReview063TamperedMaterializedInventoryAndAccounting`，改写 `remaining_qty=999` / `realized_pnl=999` 后 SELL 999 被拒绝，实际 PnL 仍为 8+6=14。 |
| **06-3-F2** | P2 / 06-4 前阻塞 | **已裁决并修复：终态不可由 Fill 重算覆盖** | 交易所声明 `FILLED/CANCELED/EXPIRED/REJECTED` 后，`projectPosition` 不因累计 Fill 数量重新写成另一个终态；终态冲突（如 `CANCELED` 但 Fill 合计等于全部委托量）通过 `ReviewPosition` 暴露，并由 `ErrNeedsReview` 阻止下一笔委托及精确 PnL 展示。晚到 Fill 保留真实成交量，不丢弃成交。`TestReview063TerminalOrderNeverRewrittenToFilled` 断言状态、回放与异常门控。 |
| **06-3-F3** | P3 / 异常处理 | **本地库已增加最小可操作信号；外部恢复仍待 06-4** | 超出原订单量、身份错配、矛盾的终态、缺失 `FILLED` 的 `myTrades` 通过可识别的 `ErrNeedsReview` 或 `ReviewPosition()` 上报；异常时回滚，不擅自扩大 `requested_qty` 或造新订单。06-4 必须实现 Binance REST 订单/成交核对、提示及阻止新单；此时不新增复杂持久化异常状态机或人为覆盖接口。 |
| **06-3-F4** | P3 / 全局双开关 | **阶段约束保持不变，06-4 接交易执行前为硬门槛** | `spotledger` 仍是无 Binance/全局 config 依赖的本地记账层；实际新单必须同时满足 `config.spot_quant_enable=1` 与 `spot_symbols.spot_quant_enabled=1`，再验证 Fill 净库存、账户 `free`、精度及订单幂等；06-4 必须有全局 OFF、单币 OFF、双 ON 三态测试。**尚未实现，不宣称闭环。** |
| **06-3-F5** | P3 / MySQL 与索引 | **隔离 MySQL 5.6 重新实测通过；MySQL 8 / EXPLAIN 仍待 06-6** | 新增/扩充 `mysql_integration_test.go` 的四个 v25 常规索引 fixture、物化字段篡改、终态冲突与跨重启检查。2026-11-11 使用独立 Docker 容器、唯一数据库 `spotledger063_isolated` 再次 PASS，测试后输出 `FINAL_MYSQL_CONTAINER_REMOVED`。测试 DSN 严格限制库名及回环地址，不会以生产数据库名运行。MySQL 8、成交大表 `trade_time` 排序的 P95/EXPLAIN 未测。 |
| **06-3-F6** | P3 / 滚动基线 | **已修正** | 方案 06 将 `af07191` 标为 06-2 的历史快照，新增 06-3 当前 `2e374b4`、本地 master `7ed6c93`、Schema v25 的明确基线；不将开发环境事实冒充生产发布结果。 |

## 二、状态机与恢复行为的明确裁决

唯一权威：订单在交易所报告的终态不可因本地逐笔成交重算而改变；**成交事实仍然用于精确数量与盈亏**。例如 `CANCELED + 全部成交`：订单保留 `CANCELED`，`filled_qty` 如实达到请求量，但出现需要人工/REST 核实的矛盾，不允许下一笔新策略订单，也不对外声称精确净盈亏。`FILLED` 先于 `myTrades` 到达时同样阻止下一单，直至确认成交明细完整。

不新增表或新业务状态。06-4 必须处理 `errors.Is(err, spotledger.ErrNeedsReview)`，查询同一 `exchange_order_id` 和 `client_order_id` 的真实订单/成交明细，再决定是否继续；不可简单忽略此错误或根据钱包余额强行补平账本。

关于 06-3-F1：`GetPosition` / `ListPositions` 返回的数据库缓存仍可用于显示原始记录及诊断，**不允许被 06-4 当作可卖数量的权威**。安全委托统一通过会在数据库事务内从 Fill 重新投影的 `CreateOrder`，PnL 展示通过 `Accounting`；06-4 发真实 SELL 之前还需再次验证 Binance Spot 余额与实际执行开关。

## 三、回归结果

| 实测项目 | 结果 |
| --- | --- |
| `go test ./command ./controllers ./models ./service/spotquant ./service/spotledger -count=1` | PASS：5/5 包 |
| `go test -race` 同 5 个包 | PASS：5/5 包（macOS `ld: malformed LC_DYSYMTAB` 为非致命警告） |
| `go test -race ./service/spotledger -count=5 -p=2` | PASS |
| `go vet ./command ./controllers ./models ./service/spotquant ./service/spotledger` | PASS |
| `go build -o /tmp/spot_ledger_063_rereview_check .` | PASS；不覆盖现有可执行文件 |
| `git diff --check`（后端） | PASS |
| `TestReview063TamperedMaterializedInventoryAndAccounting` | PASS；SELL 999 被拒，PnL 不再为伪造的 999 |
| `TestReview063TerminalOrderNeverRewrittenToFilled` | PASS；终态保持与新单阻断 |
| `TestReview063MissingTradesAndOverfillRequireReconciliation` | PASS；缺少 myTrades 和 overfill 返回需核对错误，合法恢复后清除异常 |
| 独立 MySQL 5.6 Docker：`TestStage063MySQLIsolatedLifecycle` | PASS；索引 fixture + 新 F1/F2 测试，容器已删除 |

未宣称全仓库 `go test ./...` 通过；原报告指出的 `strategy_templates/`、副本 `conf/app.conf` 问题未因本次 06-3 变更而被修复。MySQL 8、真实 API、生产数据库均未测试。

## 四、整改文件与 Gate

代码：`service/spotledger/{ledger.go,fills.go,snapshot.go,ledger_test.go,mysql_integration_test.go}`。文档：本复评、`开发方案-06-3-本地仓位订单成交与盈亏-实施记录.md`、总方案 06 的滚动基线。原 `CODE_REVIEW_06-3.md` 保持原样。

**06-3 F1/F2 阻塞已经在本地代码与测试层关闭；可以继续 06-4 的本地 Fake/Testnet 研发。** 06-4 真实 Binance 自动下单接线前，F3 的外部异常恢复以及 F4 双开关执行门控必须完成；06-6 前仍需 MySQL 8 / 索引 P95 / EXPLAIN、发布前 Schema 预检与成套部署验收。**生产和实盘 Gate 继续 BLOCKED。**

本次无 `sync db`、无生产或实验数据库写入、无 Binance 交易、无 `main.go` / `app.conf` 修改、无部署、无自动 commit/push。
