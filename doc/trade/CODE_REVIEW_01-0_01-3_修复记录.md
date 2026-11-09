# CODE_REVIEW_01-0_01-3 整改实施记录

记录日期：2026-11-08（北京时间）；分支：`feat/trade-upgrade`；依据：`CODE_REVIEW_01-0_01-3.md`。本报告是整改后状态，不修改原 Review 的历史离线结论。

## 已修复和固化的风险

| 编号 | 修复结果 | 验证 |
|---|---|---|
| F1 P1 | **代码防护已完成，实盘 Gate 仍未完成。** 新增 `futures_trade_fills.rest_verified`（v23），WS 成交作为 provisional，REST `userTrades` 按真实 `tradeId` 确认 `commission/commission_asset/realized_pnl/trade_time` 后置为 1；即使 WS `n` 或 `T` 不同，也由 REST 明确覆盖费用与时间。数量/价格冲突仍强制失败。REST 未确认的 Fill 不允许进入最终收益结算；重复 REST、迟到 WS 不回退已核实值。差异写入“报警” Warning（账号、币、订单号/成交号、WS/REST 费用/时间）。 | `TestStageReviewRESTAuthoritativeAndMissingWSCommission`，阶段 01-2 原有去重/部分成交回归 |
| F2 | `ReconcileSummary` 新增 `incomplete_aged` 和 `oldest_incomplete_age_sec`；结算超过 **15 分钟**未完成，输出“报警”日志及原因；同账户/同仓位约 15 分钟限频。超 10000 条积压继续 fail-closed 并返回错误供定时任务告警。**没有自动修改开仓开关或人工伪造净收益。** | `TestStageReviewOldIncompleteSettlementSurfacesAlarmCounters` |
| F3 | `reconcileTradeFills` 对不支持 `OrderTradeReader` 的 Broker 返回显式错误，订单保留 `reconcile_required`；对 `BinanceOrderBroker` 增加编译期断言。原测试 Broker 全部显式提供隔离的模拟 REST 成交，去除“没能力就按完成处理”的隐式路径。 | `TestStageReviewNoTradeReaderMustFailClosed`；Lead/AgentTrade 相关回归 |
| F4 | 已挂 TP/SL 在部分 CLOSE 或经 REST 确认的外部减仓后重新按 `requested_qty-filled_qty` 与当前 `managed_qty` 比较；超额时调用原 **Binance Broker.Cancel** 撤销真实订单，确认成功才把本地状态改为 CANCELED，不谎称请求数量已改。取消失败保留订单和错误，周期对账可重试；对需要重挂的 Owner 输出“报警”日志。原 `notice_auto_order` 保护修复逻辑按余量创建新单；其他 Owner 由各自现有保护管理或人工检查，不擅自猜测触发价。 | `TestStageReviewOversizedProtectionCanceledOnPartialClose`、`TestStageReviewProtectionCancelFailureMustRemainRetryable` |
| F5 | 新增 `command/trade_settlement_migration.go`：v22 的结算状态与 config 计数列显式、幂等补列，旧记录安全回填为 `legacy`、计数缺失填 0；v23 显式补 `rest_verified TINYINT NOT NULL DEFAULT 0`。Go ORM schema 仍先注册，新增迁移针对缺列与历史默认值补强。 | `TestSyncDatabaseInitializesAndIsIdempotent`；`go_bn_test` MySQL **2026-11-08 11:09** 从 v22→v23，重复同步 already up to date |
| F6 | 新增唯一 `service/tradefills.SchemaDDL`，生产迁移和五个 SQLite fixture 全部直接使用该常量，不再分散维护六份 DDL；新的第 17 列为 `rest_verified`。 | 相关 Go 包完整测试 |
| F7 | WS `n` 缺失时按 provisional `commission=0` 保存，允许空 `commission_asset`，**不代表真实零费已核实**；待 REST 确认并写入资产后才允许结算，避免无手续费 TRADE 阻断整个账户镜像。 | `TestStageReviewRESTAuthoritativeAndMissingWSCommission` |
| F8 | `decimal()` 拆分校验整数有效位 **≤18**、小数位 **≤18**，不再把总 36 位误当整数可用位数；MySQL 的 `DECIMAL(36,18)` 仍是最终精度 Gate。 | `TestStageReviewDECIMAL3618RejectsNineteenIntegerDigits` |
| F9 | 结算 Fill 查询同时筛选 `position_id` **和** `account_id`，不再扫其它账户同数字 ID。 | 结算与主/Lead 账户隔离回归 |
| F10 | 保留 `exchangeInfo.marginAsset` 低频缓存（合约静态元数据），与新鲜 REST 仓位快照用途不同；缓存不可用时 fail-closed，保持 incomplete；明确属可接受取舍。 | 结算资产缺失回归 |
| F11 | 空仓快照且仍存在未核实交易所订单时，不把“缺少对应持仓 key”解释为可信零仓位；阻止据此关闭或结算仍活跃的 Managed 周期。 | `TestStageReviewEmptySnapshotUnresolvedOrderCannotProveFlat` |
| F12 | 保留 Stage7 Lead 只读 observer 原有接口边界；新增只读检查不写 Fill 的动态断言，以及 `stream.go` 不得接入 Persistent Fill 写入的源码守卫测试。**不将 Lead read-only WS 接到持久化 helper。** | `TestStageReviewLeadReadOnlyStreamCannotPersistTradeFills`、Stage7 ReadOnly tests |
| F13 | 旧 `order` 与旧页面保持原状；继续按 01-4/01-5 阶段逐项退役，未越权提前删除。 | 阶段边界不变 |

## 验收结果与本地数据库

- `go test -count=1`：`service/futuresownership`、`service/tradefills`、`service/agenttrade`、`service/leadaccount`、`feature/api/binance`、`feature`、`command`、`models` 通过；`appversion` 无测试文件。
- `go test -race`：三个核心包通过；macOS `ld` 仅有非致命 `LC_DYSYMTAB` Warning。
- 上述包 `go vet`、主程序 `go build`、`git diff --check` 通过。
- 正式迁移命令仅在配置已确认指向 `go_bn_test` 时从独立临时二进制运行：**v22→v23 成功，重复执行成功**；临时目录与二进制执行后清理，项目根目录可执行文件未覆盖。
- **没有触碰生产 `go_binance`，没有请求真实 Binance 交易，没有重启服务，没有迁移旧 `order` 历史，没有修改 `app.conf`，没有 commit/push。**

## 仍未关闭的生产 Gate（不得上线/T0）

1. **F1 真实部分成交证据仍需人工授权实测。** 用一笔真实多次部分成交订单记录对应 WS `l/L/n/N/rp/T`，以及 REST `userTrades` 的 tradeId/qty/price/commission/commissionAsset/realizedPnl/time；验证 REST 每笔数值与新账本核对相同。无法用离线测试声称实盘已通过。
2. **保护单重挂联动需要 Owner 级实盘验证。** F4 已避免超量挂单留在交易所，但撤销后并非所有 Owner 都有通用自动重挂能力。特别是非 notice Auto owner，必须审计原有保护单恢复路径及失败报警；不得把“撤销过大保护单”当作“仍具有效止损保护”。
3. **F2 提供的是应用 Warning 报警与可供后续 API 读取的 ReconcileSummary 字段，不是 DingTalk 主动推送或人工强制覆盖开关。** 运行说明：`incomplete` 超 15 分钟应检查账户绑定 REST、真实 Fill、手续费资产、Algo 映射和持仓快照；未修复前不能依据推测手动改 `settlement_status=settled`。需要推送接线与可视化的部分按 01-5/01-6 单独验收。
4. **01-4/01-5 尚未实施，T0 未启用**。生产 DB 版本与现有非零仓位须上线前独立只读复核，schema 升级、二进制替换、重启及实盘验证均需另外授权。

> 测试环境 SQLite 仅用于逻辑和约束回归；不能代表 MySQL `DECIMAL(36,18)` 所有精度语义。MySQL 真实验证只针对本地测试库。生产验证仍以正式上线 Gate 为准。
