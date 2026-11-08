# CODE_REVIEW_01-0_01-3 复审（R2）整改记录

- 时间：2026-11-08（北京时间）
- 依据：`CODE_REVIEW_01-0_01-3-rereview.md`
- 范围：01-0～01-3 工作区整改；**未实施 01-4 / 01-5，未部署实盘、未建立 T0**。
- 现有目标数据库版本：**v23**，本轮未增表、未升 schema 版本。

## R2 问题逐项结论

| 复审项 | 修改与验收 | 状态 |
| --- | --- | --- |
| **N1 P2**：v20～v23 迁移测试缺失 | 核对当前 `db_update_test.go` 实际已含 `SyncDatabase(20)`、`(21)`、`(22)`、`(23)`；本轮补入 **v21 历史 closed 行空 settlement_status → v22 必须回填 legacy**、v22 两个具名索引、v23 旧 Fill 默认 `rest_verified=0`、两次同步幂等及 **`appversion.DatabaseSchemaVersion` 上限断言**。实跑 `go test -v -run TestSyncDatabaseInitializesAndIsIdempotent ./command`，输出明确含 20→21→22→23 和重复 23，PASS。原复审“最高只有 19”的判断与当前工作区内容不符。 | **已修复并补充实证** |
| **N2 P2**：`filled` 后未核实 Fill 永远不再核对 | `service/futuresownership/reconcile.go` 增加 `reconcileClosedCycleFills`：`unsettled` closed 周期按 `account_id+position_id` 查询所有已成交 Managed 订单，调用原 `Executor.reconcileTradeFills` 只读 REST `userTrades` 补齐/确认，**不调用 `Broker.Submit` 或 `Executor.Reconcile`，不复活已终结订单**。REST 失败保持 `incomplete`，恢复后继续重试。 | **已修复** |
| **N3 P2**：撤销超量保护后无重挂 | `notice_auto_order` 保留现有每分钟 `RepairNoticeAutoOrderProtections`；为有持久化 StopLoss/TP 触发价的 `agent_trade` 新增 `RepairActiveManagedProtections`，并接入同一条成功对账后的定时流程。仅使用匹配 `source_ref/position_id/symbol/side` 的批准提案触发价，创建剩余数量的保护，要求真实 exchange order ID 与管理订单已提交；缺少可信提案/真实订单身份时返回 `报警`，不猜价、不伪造保护。重复运行不多下单。**其他 Owner（特别是 Lead 的独立交易阶段）不具备通用可信触发价恢复链，仍属实盘前置未关闭。** | **部分修复，生产 Gate 仍阻塞** |
| **N4 P3**：部分缺失仓位快照被当 0 | 对 Binance 账户绑定的 PositionSource 增加 `FreshPositionSide`，当全量仓位缺失已管理的币种+方向时，按 `symbol` 另发**新鲜、非缓存的币安 REST 仓位查询**，要求返回该方向的明确记录；缺失或失败时拒绝 destructive shrink/close/settle，最多每 Owner 每轮 6 个额外请求避免限额冲击。复测确证第一个零仓位快照只进入 `reconcile_required`，不跳过原二次确认。 | **代码修复，真实 API 返回零仓位的覆盖度仍需确认** |
| **N5 P3**：REST 零费但缺资产字段 | REST `CommissionAsset=""` 且精确 `Commission=0` 可被 `RESTVerified=1` 接受；净结算允许币种未知但金额确认为零的费用。**非零手续费却无资产仍拒绝核对**，也不假设 USDT。 | **已修复** |
| **N6 P3**：`incompleteAlertAt` 无界增长 | 限频记录上限 4096；溢出时清理超过 24 小时的旧项，必要时淘汰最旧项；结算完成立即删除其 key。 | **已修复** |
| **N7 P3**：测试 REST 恒为零且跨测试泄漏 | `service/internal/testtrades` 新增 `Reset()` 和 `SetAmountsForExchangeOrder`，核心测试库 fixture 每轮主动清空模拟成交与费用。新增非零 fee/PnL 的 REST 持久化、Reset 防泄漏测试。该模块仍**只由测试文件引用**，不进入生产主程序依赖链。 | **已修复** |
| **N8 P3**：v22 结算查询索引依赖 ORM | `ensureV22SettlementIndexes` 对 `settlement_status`、`settlement_processed_at` 建具名索引，MySQL 用 information_schema 核对存在性、SQLite 用 IF NOT EXISTS；v22 迁移时创建，已有 v22/v23 数据库每次正规 `sync db` 也会自检补建。 | **已修复，本地 MySQL 同步通过** |
| **N9 P3**：Lead 只读保护仅字符串黑名单 | 在原动态只读测试与字符串守卫之外，针对 `service/leadaccount/stream.go` 增加 Go AST 导入白名单检查，禁止通过重命名 import alias 引入新 DB/交易写模块。整个 leadaccount 包本来包含交易链路依赖，所以**不能把整包依赖闭包当作 read-only stream 的闭包**；本项是防回归守卫，不代替真实环境只读检查。 | **加固完成** |

## 新增/增强的永久测试

- `TestStageR2ClosedFilledOrderProvisionalRESTAutoRecovers`：订单已 FILLED、不在 ActiveOrders；REST 故障时 incomplete，恢复后确认原 Fill 并结算，重复核对风控不再重复触发。
- `TestStageR2AgentRepairRearmsRemainingManagedProtection`：按剩余仓位从批准提案重建 Stop/TP，重复调用不重复下单。
- `TestStageR2PartiallyMissingBulkSnapshotRequiresExactRESTProof`：全量快照存在其他币种但缺某 Managed key，定向查询失败时不关仓；有明确 0 时仍遵守二次确认。
- `TestStageR2VerifiedZeroFeeWithoutAssetSettlesCorrectly` 与 `TestStageR2RESTZeroFeeMissingAssetIsAuthoritative`：真实 REST 确认零费可结算，非零未知币种仍拒绝。
- `TestStageR2IncompleteAlarmThrottleStorageIsBounded`：告警缓存容量/结算清理。
- `TestStageR2FakeRESTSupportsNonzeroMoneyWithoutLeakage`：非零费用/PNL 注入与测试状态复位。
- `TestSyncDatabaseInitializesAndIsIdempotent`：版本上限和历史 safe default、索引及 REST 来源标志。

## 本地 Gate 与环境边界

- `go test -count=1`：8 个相关业务包全部通过，`appversion` 无测试文件。
- `go test -race -count=1`：`futuresownership` / `tradefills` / `feature/api/binance` / `agenttrade` / `leadaccount` 共 5 个包全部通过（macOS 链接器有非致命 LC_DYSYMTAB warning）。
- 上述包 `go vet`、主程序 `go build`、`git diff --check` 全部通过。
- **2026-11-08 18:15 北京时间**：确认 `conf/app.conf` 的数据库名严格为 `go_bn_test` 后，用临时二进制执行两次正式 `./go_binance_futures sync db`，两次均为 **v23 already up to date**；验证已有版本上的索引补建自检可重复执行。
- 工作区的全部修改仍**未 commit/push**；没有修改 `conf/app.conf`、生产 `go_binance`、服务/旧根二进制、实际交易所订单；没有执行生产重启。

## 仍阻塞实盘/T0 的条件

1. F1 需要真实部分成交事件与官方 REST 每笔手续费、PnL、时间逐 `tradeId` 对照；离线的 REST authoritative 防护不是实盘样本。
2. **N3 尚未对所有可能提交保护单的 Owner 实现自动重挂。** 已覆盖当前能安全取回触发价的 `notice_auto_order` 与 `agent_trade`，其他 Owner 或 Lead 如需启用交易，须先提供可信原始触发价、独立的重挂恢复代码和实盘订单身份校验。未覆盖时发现保护单撤销必须立即按原策略/账户风险流程人工验证真实仓位与止损，必要时停止新开仓并进行受控减仓；**严禁仅修改本地 requested_qty 或凭空挂新价位**。
3. N4 生产 Binance symbol-scoped `PositionRisk` 对零仓位是否始终返回明确的方向 row，必须在正式 API 环境确认；查不到则保守不关仓/不结算，不会把缺失 key 作为零的证据。
4. 所有实盘部署、生产 schema、二进制替换、真实订单/风险开关变化仍需独立授权，且 01-4/01-5 尚未实施。

**结论：R2 的高风险逻辑 N1/N2 已补实证并修复；N3 对当前两个可恢复 Owner 已实现保护恢复，但不应宣称全 Owner 实盘 Gate 通过。**
