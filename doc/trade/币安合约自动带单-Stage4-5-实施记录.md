# 币安合约自动带单 — Stage 4-5 暂停、熔断与故障保护实施记录

日期：2026-10-10

**阶段结论：Stage 4-5 的离线状态机、故障分级、告警事件接口及与 Lead 开仓/受控平仓执行适配器的连接已经实现并通过定向 Mock 测试。Gate 4 整体验收仍需 Stage 4-6；真实 Lead 下单、撤单、启停、WS/重启恢复均未开放。**

## 1. 实际代码与交付内容

- `service/leadaccount/state.go`：新增 `LeadRuntimeGuard`、`LeadRuntimeState`、`LeadStatusSnapshot`、`LeadFaultCode`、`LeadFaultEvent`，状态只属于 `account_id=lead`。构造时 `disabled`，不存在任何可公开解除/授权交易的 `Enable`、`Resume`、`ResetPending`。
- `service/leadaccount/fault.go`：把 Stage 4-2 的 **固定** `RiskDecision.BlockingReasons` 分类，日亏损/回撤为持续熔断，账户/WS/快照异常为不可用，未知/Algo 单为待对账；保证金不足、单笔/总敞口等普通交易拒绝只跳过该笔，不把无信号或选币不足当成账户故障。
- `service/leadaccount/executor.go`：构造 Lead adapter 时独立构造 Guard。新增只读 `RuntimeStatus()`、单向 `PauseOpens()` 和固定故障码 `ReportLeadFault()`，三者都没有生产实盘授权能力。每次开仓前在同一适配器实例锁下检查 Guard、Stage 4-2 风控和最新账户证据；提交前保留 `pendingReconcile` 并登记故障类别，未解除就不得发第二张单。故障报告与适配器开仓写入串行化。Main 完全不创建或调用该 Guard。
- `service/leadaccount/close.go`：保留 4-3/4-4 既有账户、Ownership、`min(managed_qty,live_qty)` 与交易所过滤器检查；附加 Lead Guard 的安全退出门禁。`paused` 与日亏损风险熔断不移除受控退出路径；身份/WS/快照不可用、未知提交仍拒绝不确定写请求，保留只读诊断能力。提交受控退出后继续阻断，等待权威对账。

## 2. 状态与故障语义

| 状态 | 触发 | 开仓 | 已有受控仓位 |
|---|---|---|---|
| `disabled` | 默认配置或无 Stage 7 授权 | 不允许 | 保留退出信号评估；写仍受身份/Stage 7/Ownership 严格约束 |
| `paused` | 单向 `PauseOpens` | 不允许 | 可以评估及在所有其他账户条件满足时安全退出 |
| `risk_tripped` | 日净亏损或回撤达到阈值 | 不允许，持续锁定 | 可继续退出意图和安全处置 |
| `reconcile_required` | unknown、Algo 状态不确定或任一次实际下单需重新确认 | 不允许 | 保留只读订单恢复；不能再次提交未经证明安全的修改 |
| `account_unavailable` | 账户身份/凭证/Portfolio 不符、WS、429/418、API/快照异常 | 不允许 | 禁止不确定写操作，仍允许只读追查和故障告警 |
| `open_ready` | **仅 Mock/模型层**全部条件满足 | 仍必须有 Stage 7 真实授权和 Broker 写权限 | 受原策略/Ownership/交易所规则约束 |

多个故障原因并存，状态优先级 `account_unavailable > reconcile_required > risk_tripped > disabled/paused > open_ready`，标准原因码按固定顺序展示。**状态名 `open_ready` 不是开仓许可**；真实适配器仍默认 `RiskController` 禁用、`RiskEvidenceSource` 未权威对账、`ReadOnlyOrderBroker` 全部写入关闭。

普通单笔风险拒绝为 `skip`，不触发全账户熔断。到达阈值的日亏损、回撤为 **sticky trip**，即使 UTC 换日或健康快照回来也不能无审核复位；提交不确定为 **sticky reconcile**，Stage 4-5 不提供解除入口。暂态 WS/REST/快照异常仅在**同一 Lead 账户**通过新鲜且完整的仓位、挂单、UTC 日 PNL、Hedge Mode、有效余额和权威对账证据时才能从 Guard 的观察状态消除，Stage 5 必须提供该权威证据与运行恢复流程。凭证、身份、人工保护故障不会因为普通健康快照自动解除。

## 3. 故障通知设计

- `NewLeadRuntimeGuard(notifier)` 支持显式注入的 `LeadFaultNotifier`；事件仅包含固定故障码、类别、严重度、状态前后和时间戳以及 `AccountID=lead`，**不包含 API Key、secret、symbol、签名 URL、原始 HTTP 错误**。
- 同一故障首次立刻发出；重复相同故障按 2 分钟限频；不同严重度故障分别及时通知（如 429→418）；可信恢复可发出 `resolved` 事件。通知失败**不撤销**安全阻断。
- **Stage 4-5 仅交付事件出口及 Mock 验收，未在应用启动时注册生产通知发送器**；Stage 5/6 再按账号隔离接入原报警链路和持久化的通知展示，不增加 Binance 热路径 API 请求。

## 4. 永久测试

- `service/leadaccount/state_test.go`：默认拒绝、Mock readiness、单向暂停和退出门禁；多故障优先级/标准原因顺序；可信/过期/跨账号快照处理；跨 UTC 日风险熔断不复位；普通单笔跳过不污染全账户；Lead/Main 故障隔离；429/418 升级、2 分钟去重、resolved 事件、通知错误安全性；并发状态读写（race）；开仓前 WS/身份故障拒绝、暂停期间安全平仓、未知订单不能经只读检查解锁、错误账户禁止退出。
- `feature/account_trade_cycle_stage4_5_test.go`：Lead 的 Preflight 故障拒绝不停止 Main 循环和 Main 真实开仓 Mock；开仓阻断不删除现有退出、同步/对账回调。
- Stage 1–4-4 原有单元测试必须继续通过，尤其 `_test.go` 中的 Stage 3 Lead Mock 工厂、Stage 4-2 Main Golden 回归、Stage 4-3/4-4 的 Fake Broker 与 SQLite 内存 Ownership 幂等测试。

## 4-A. 实际执行验证结果（2026-10-10）

| 检查 | 结果 |
|---|---|
| 9 个相关 Go 模块的完整 Race 回归（leadaccount、binance、ownership、feature、line、scanner、backtest、command、controllers） | 通过：RACE_EXIT=0 |
| 同 9 模块 go vet | 通过：VET_EXIT=0 |
| 根包生产 go build | 通过：BUILD_EXIT=0；临时二进制已删除 |
| git diff --check / 所有新增及改动 Go 文件的 gofmt | 通过 |
| 真实 Lead 凭证、交易接口、主库、配置和定时任务 | 未访问或改动 |
| 已知无关项 | 前期已知静态 strategy template 编译测试按约定跳过；Mac 链接器出现非致命 LC_DYSYMTAB warning |

**Gate 4-5：离线实现与定向/跨模块自动化测试通过。Gate 4 总验收必须等待 Stage 4-6。**

## 5. 未完成项与下一步 Stage 4-6 / Stage 5

1. **Stage 4-6**：跨模块联合 Mock Gate 4 验收矩阵和代码审计；需要另出 `Stage4-6-实施记录`，在此之前 Gate 4 仍是**未通过**。
2. **Stage 5**：Lead 专属真实 User Data WS、ListenKey、重启/跨进程订单-仓位权威对账、持久 Claim 与 unknown 恢复；只有 Stage 5 的完整 REST/WS/DB 证据才有资格解除待对账状态。Stage 4-5 的瞬态恢复目前为**离线证据模型**，不代表生产自动恢复。
3. **Stage 6/7**：Portfolio 专用账户身份与资产人工核对、强鉴权配置/告警面板、明确用户授权下的受控最小额真实联调。Stage 4-5 没有真实交易/撤单或一键解锁入口。
4. **保守限制**：Guard 及实例级互斥锁没有跨进程持久化，不等价于交易所幂等或分布式额度预留；离线通过不等于实盘可用。
5. **通知接口语义**：`LeadFaultNotifier` 在调用线程同步执行，应由未来告警适配器提供有界非阻塞实现；不得把耗时外部通知直接接入交易热路径。Stage 4-5 当前生产 notifier=nil（不开启）。
6. 保留 Stage 4-4 审计中的 P3 项：动态价格过滤器、真实 Algo 状态、真实撤单、`exchangeInfo` 缓存和未完成的账户授权 Gate。

本轮没有改动 `conf/app.conf`、数据库 Schema/主库、前端、策略模板，不运行 `sync db`，不自动 commit/push，不启动 Lead 实盘交易或后台轮询。
