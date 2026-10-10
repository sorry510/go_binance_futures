# 币安合约自动带单 — Stage 4-3 执行适配器与账户数据证据

日期：2026-10-10

## 1. 范围与安全结论

Stage 4-3 已完成**不激活生产写权限的账户绑定执行适配器**、Lead 独立风险证据读取、交易所精度和仓位/杠杆模式验证、Ownership 订单调用链及 Mock 场景。生产 `StartTrade` 仍只启动 Main；Stage 3 Lead Runner 仍仅在 `_test.go` 构造。Stage 4-3 **没有新的生产 Lead 任务、Controller 路由、交易启动许可或明文凭证**。

严禁把此阶段解读为真实 Lead Portfolio 已经可交易：`RiskController` 没有 Enable/Resume 接口，`NewLeadExecutionAdapter` 由默认关闭的 Controller 构造，生产没有 `simulateRules`/`simulateEvidence` 的入口、没有 `PortfolioBindingConfirmed` 或 Stage 7 用户授权解除方式；公开的 `ExecuteOpen` 和 `ExecuteManagedClose` 默认均被拒绝，**在默认状态下不会触发任何真实下单/撤单或私有 API**。

## 2. 具体修改

### 2.1 只读数据与私有签名隔离

- `feature/api/binance/lead_income.go`：仅对显式 Lead `AccountClient` 提供已实现净盈亏历史 GET（`/fapi/v1/income`），限制时间窗与分页大小，不回退 Main Client。
- `feature/api/binance/lead_rules.go`：`exchangeInfo` 公开 GET，以及仅使用 Lead Key 的带签名 `/fapi/v1/leverageBracket` GET；不提供交易写接口。
- `service/leadaccount/snapshot.go`：由账户绑定 Source 依次读取可交易状态、USDT 钱包/保证金/可用额度、Hedge Mode、Fresh 仓位与全账户未成交单、UTC 当日净已实现盈亏；余额/仓位/订单/分页缺口或无法计价均 fail-closed。亏损核算包括 `REALIZED_PNL`、`COMMISSION`、`FUNDING_FEE`，不把转账正收益当策略利润。
- Income 每页最多 1000 条；游标按毫秒重叠读取并按 `tranId + incomeType` 去重。遇到单毫秒数据超过安全分页能力、页数超限、未知负项/非 USDT 项、API 报错不假定盈亏=0，直接阻断。由 Stage 5 再进一步优化持续同步与成本控制。
- 采集器的 `Reconciled=false`、`UnknownOrders=true` **始终保持直到 Stage 5 对账证明完成**；Stage 4-3 不凭只读快照自行把这两个标志置真，也不填未经验证的权益高水位。

### 2.2 合约交易规则

- `service/leadaccount/rules.go` / `rules_source.go`：使用 `exchangeInfo` 的 `LOT_SIZE` / `MARKET_LOT_SIZE`、`PRICE_FILTER`、`MIN_NOTIONAL`、签名杠杆档位以及该 Lead Portfolio 的 `positionRisk` 杠杆/保证金模式；覆盖数量步长、价格 Tick、数量最大/最小、名义额下限/上限、最大杠杆。
- SDK 与 Binance 返回的保证金枚举 `cross` / `CROSSED` 作显式一致性规范化；真实不匹配/缺少账户配置一律拒绝，不静默调用切换 Hedge Mode/逐仓/全仓的写 API。
- 真实执行适配器忽略外部传入的 `PriceAndQuantityValidated`、`MarginModeVerified` 和交易所限额声明，每次在下单预检查重新读取并生成；不能靠外部布尔标记跳过规则检查。

### 2.3 Ownership 与执行

- `service/leadaccount/executor.go`：`NewLeadExecutionAdapter` 强制 `AccountID=lead` 且 `NewAccountExecutor` 的 `Ownership.AccountID`、`BinanceOrderBroker.Account` 均绑定同一指针；构造时不访问网络/DB/交易所。
- `ExecuteOpen` 仅支持 auto_strategy 的 MARKET/LIMIT，严格对齐风控请求的币种、方向、数量、类型、LIMIT 价等；持有实例级互斥锁串行风控/OrderClaim 调用。风控在 Ownership Claim、保证金/杠杆任何写操作与 Broker 下单之前执行，默认无授权时直接拒绝。
- 执行成功、超时、不确定提交后，将本实例保持 `pendingReconcile` 状态；不重试同一未知结果、更不会绕过查询重复发单；必须后续经过权威的订单/仓位恢复重新开放。底层仍使用 Stage 2 `futuresownership.Executor` 的持久 `ClientOrderID`、`ClaimOrder`、unknown-submit → Lookup/Reconcile、累计部分成交量跟踪与账户隔离。
- **注意：实例级 Mutex 和 pending 标志不是跨实例/重启的分布式保护。** Stage 5 必须检查所有本账户 DB Pending/Unknown 订单，按账户事务性地保留并重新确认敞口；该安全 Gate 尚未完成，因此生产保持默认关闭。

### 2.4 受控平仓

- `service/leadaccount/close.go`：只允许 auto_strategy 的 CLOSE/STOP/TP 意图，且方向反向正确、仓位必须由当前 Lead 的 Ownership 管理；实时读取该账户该币该向持仓量，**不得超过 `min(managed_qty,live_qty)`**，人工加仓不纳入受控数量、人工减仓缩小可关闭量。
- 暂停新开仓不自动阻断在经过 Stage 7 授权的安全平仓；身份/WS/Portfolio 未验证、未知写结果、仓位读取失败或订单数量越界仍拒绝不确定写操作。Stage 4-4 将进一步完善交易所保护性止损单/撤单的独立恢复与执行行为。

## 3. 测试范围

- `service/leadaccount/snapshot_test.go`：余额、持仓、未成交剩余数量、手续费/资金费净 PNL、UTC 起点、含 1000 条分页重叠去重与同毫秒无法证明完整时的拒绝。
- `service/leadaccount/rules_test.go`：真实 Binance 字段格式、CROSS/CROSSED、精度步长、价格 Tick、杠杆档位及空过滤器等 fail-closed。
- `service/leadaccount/executor_test.go`：默认 0 API/0 真实写、Main Key 拒绝、隔离 SQLite 内存库与 Fake Broker 下 LONG/SHORT 的 MARKET/LIMIT、OrderClaim/累计成交、未知提交、并发 8 次一次成功、方向/账户/订单参数错配拒绝。
- `service/leadaccount/close_test.go`：LONG/SHORT 安全关闭、人工减仓限制、未授权关闭拒绝、暂停新开仓后安全退出、未知状态阻止第二次写入。
- `feature/api/binance/account_client_stage4_3_test.go`：所有 Lead 专属 GET 的签名和只读端点；Main 不能触发这些 Lead-only 方法。

仅使用 Mock HTTP、Fake Broker、SQLite **内存实验数据库**，未连接真实 Binance 账户也未在业务数据库写入任何数据。

## 4. 明确未完成的 Gate（必须在后续 Stage 完成）

- **Stage 4-4**：订单拒绝/撤单/Algo STOP/TP 的更多异常场景、真实订单查单恢复及部分成交/保证金失败等验收。
- **Stage 5**：Lead 私有 UserData WS、重连及账户恢复、`UnknownOrders`/ `Reconciled` 权威证明、可靠完整 PnL 同步、API 预算；启动或持久化 DB 尚有未知单时必须拒绝新开仓。这里的单实例锁/保留不能作为跨进程限额保证。
- **Stage 6**：强鉴权的账户配置与身份/资产人工对照、额度与启动确认；无一键绕过当前安全阈值。
- **Stage 7**：明确授权后真实 Lead Portfolio 专用 Key 的只读能力/账户归属验证，再以小额验证实际权限、执行和安全退出；**当前没有任何生产可用授权机制，也无自动开启功能**。

不修改 `app.conf`，不执行 `sync db`，不改研究文件，不自动 commit/push，不重启/停止任何既有后台服务。

## 代码审计跟进（2026-10-10，CODE_REVIEW_Stage4-3.md）

- **F1 已修复（P3）**：`ExecuteOpen` 的风控拒绝不再丢弃 `checkOpenLocked` 具体错误；现在同时保留 `ErrLeadOpenBlocked` 和原始可识别错误（如 `ErrLeadPendingReconcile`），可通过 `errors.Is` 区分待对账。相应永久单测验证重复提交时两个 sentinel 均可识别且 broker 没有第二次提交。
- **F2 暂不加入不安全 reset**：本地提交/DB 错误后仍维持 `pendingReconcile`，由 Stage 5 实现绑定 Lead 身份、数据库事务、交易所权威对账和重启恢复后才能解除，不允许因本地错误推定没有成交。
- **F3 延后 Stage 5**：MARKET 未成交挂单 `price=0` 仍保守拒绝快照；将来需可信账户专属 mark price 估值与完整性验证。
- **F4 部署说明**：币种当前杠杆及逐仓/全仓模式需预先满足请求要求；不会自动修改账户模式，Stage 6 应向用户展示预置失败原因。
- **F5 延后 Stage 4-4**：受控平仓的交易所 filter 校验、Algo/拒单/撤单恢复需要与完整订单异常流程一起测试。
- **F6 延后 Stage 5/6**：per-account API 预算与 MaxPositions/MaxLosingPositions 原因细分按后续阶段完成。

验证：`go test -count=1 ./service/leadaccount ./feature ./feature/api/binance ./scanner ./service/futuresownership`、`go vet ./service/leadaccount ./feature ./feature/api/binance`、`git diff --check` 均通过。无真实交易、无 DB 迁移、无 commit/push。
