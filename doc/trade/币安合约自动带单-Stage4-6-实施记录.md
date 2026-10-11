# 币安合约自动带单 — Stage 4-6 Mock 全链路与 Gate 4 离线验收报告

日期：2026-10-10
分支：`feat/lead-trading`
交付性质：**离线 / Mock / Fake Broker / SQLite 内存数据库**，未进行 Binance Lead Portfolio 的真实 API 权限联调或实盘下单。

## 总结与 Gate 4 结论

**Gate 4：离线验收通过（Offline PASS）；真实带单上线 Gate 0-LIVE/5/6/7 均未开放。**

Stage 4-1～4-5 的现有测试全部通过；本轮新增专门的 Stage 4-6 联合验收，覆盖独立凭证读取、安全风控、同一策略的 Main/Lead 并行执行、LONG/SHORT×MARKET/LIMIT、受控 CLOSE/TP/SL、累计部分成交后的取消与幂等、确定性拒单和不确定提交的区别、账户熔断、异常 API/WS 诊断及恢复条件、生产写入口锁定。新发现的交易所规则读取异常未登记状态问题已修正为独立的 `lead_rules_unavailable` 故障分类，必须有可信恢复证据才能从临时故障清除。

**重要限定：Offline PASS 并不代表实盘可用**。所有能够进行模拟交易的“授权”都只存在于 `_test.go` 中；生产 `ReadOnlyOrderBroker` 对 Submit/Cancel 均返回 `ErrLiveExecutionLocked`，真实 Lead 适配器默认 disabled，且当前没有生产调度入口或授权解除接口。

## 一、新增自动化验收文件

| 文件 | 内容 |
|---|---|
| `service/leadaccount/stage46_integration_test.go` | 按阶段串联凭证、只读身份、风控、模拟执行、Ownership、订单恢复、TP/SL、故障管理 |
| `feature/account_trade_cycle_stage4_6_test.go` | Main/Lead 相同策略信号并行、Lead 故障不影响 Main、暂停期间多空退出、Go AST 仓库级生产入口扫描 |

沿用 Stage 1～4-5 的所有其他 `_test.go`，不复制已有策略计算函数，也不构造第二套生产 Lead Runner。

## 二、验收矩阵（本轮测试结果）

| Gate 4 条目 | 测试与依据 | 结论 |
|---|---|---|
| **4-1 凭证/身份** | `TestStage46CredentialToRiskToExecutionRemainsLocked`：独立加密 Key、只读 LeadTrader 身份/白名单/资产校验、报告 JSON 不泄露 Secret、密钥轮换清理旧验证；只读成功仍无实盘授权 | PASS（Mock） |
| **4-2 风险、账户限制** | 复用 `TestStage42EverySafetyBarrierFailClosed` 与本轮 `TestStage46RiskAndFaultMatrixNeverReachFakeExchange`：WS/429/418、日亏损、回撤、Portfolio、保护失败，持续禁开仓且 0 Fake Broker 提交 | PASS（Mock） |
| **4-3 LONG/SHORT×MARKET/LIMIT** | `TestStage46AccountScopedMockOpenMatrixAndStickyReconcile`：四组均各写入 Lead 独立 Ownership 且累计成交量=1；首次提交后再次提交被待对账门禁拒绝 | PASS（Mock） |
| **4-3 双账户策略并行** | `TestStage46SharedStrategyConcurrentMainLeadIsolation`：并行两账户同一策略 LONG/SHORT，分别产生 2 个模拟信号；Lead 预检查失败后 Main 仍独立产生同样的 2 个信号 | PASS（Mock） |
| **4-3 安全受控平仓** | `TestStage46MockProtectiveStopsAndTargetsWithManagedCaps`：LONG/SHORT×STOP_MARKET/TAKE_PROFIT_MARKET 四组合，归属仅 Lead，满额关闭后持仓状态为 CLOSED，禁止第二张未经对账的写请求 | PASS（Mock） |
| **4-3 / 4-4 方向退出优先级** | `TestStage46SharedExitSignalsBothSidesEvenWhenLeadPaused`：暂停及白名单失效时仍评估 LONG/SHORT 的策略退出、亏损、盈利与保持意图；`TestStage3MainMarketEntryGolden`、`TestStage3MainLimitOrderGolden` 保证 Main 信号历史结果不变 | PASS（Mock） |
| **4-4 部分成交 → 取消 → 重复事件** | `TestStage46PartialFillCancelIdempotentAndAccountSeparation`：累计 0.3 → 0.3 → 0.5 → 0.5，受控净数量始终为 0.5；Main 同币同向独立 Order/Position 不变 | PASS（SQLite 内存） |
| **4-4 拒单与提交未知** | `TestStage46DeterministicRejectionAndUnknownTimeoutDifferentLedgerOutcomes`：Binance 明确拒单标记 FAILED 不查未知单；超时与网络未知进入 RECONCILE 且查询一次；后续重试仍被阻断 | PASS（Fake Broker） |
| **4-4 普通/Algo 恢复** | `TestStage46RecoveryTerminalEvidenceCannotReleaseOpenOrCancelGate`，以及既有 `TestStage44...Recovery...` 的 Mock HTTP/Algo 子订单身份、状态、方向及累计成交校验。终态 CANCELED/FILLED 仍带 `RequiresReconcile` | PASS（离线只读证据） |
| **4-5 故障状态** | `TestStage46RiskAndFaultMatrixNeverReachFakeExchange`、`TestStage46GuardCannotPromoteEvidenceToLiveAuthorization`：多类阻断不失效，`open_ready` 不是 Stage 7 许可；复用告警失败/去重/升级/UTC 熔断测试 | PASS（Mock） |
| **可观测性加固** | `TestStage46ExchangeRuleOutageHasVisibleFailClosedStatus`：交易所过滤器/杠杆规则不可读时新增 `FaultRulesUnavailable`，状态为 `account_unavailable` 且 0 Broker 提交 | PASS（Mock） |
| **生产写入口扫描** | `TestStage46NoProductionLeadExecutionEntrypoint` + `TestStage46RepositoryProductionLeadWritesRemainUnreachable`：对生产 Go AST 扫描，禁止对 `NewLeadExecutionAdapter`/`ExecuteOpen`/`ExecuteManagedClose` 等的生产启动调用；Stage 3 Lead permit 仅存在测试文件 | PASS |
| **跨账户安全** | 同币同向可在 Main/Lead 不同 AccountID 下合法存在；读/写/取消隔离及 Main Golden 回归；主账户不使用 Lead 风控或告警 | PASS（Mock） |

**用例数据来源**：`validFakeReader`、`stage43Adapter`、`stage43Broker`、`sampleLeadEvidenceReader`、`riskSymbols`、`stage3MockRunner`、`stage43OwnedFixture` 及测试专用内存 SQLite；任何真实 API Key/数据库连接都不需要，也未进行生产下单。

## 三、Stage 4-6 实施中发现与修复

- 联合 TP/SL 测试首轮误把 `GetPosition()`（仅返回活动仓位）用于查询已完全关闭的仓位，得到 `orm.ErrNoRows`。修改**测试断言**为 `ListPositions(..., false)`，确认最终 CLOSED 和 `ManagedQty=0`。这是测试对已有 Ownership API 语义的误用，不是执行层错误；**未改变 Owner 逻辑**。
- Stage 4-5 审计 F2 的一部分确实影响运维可观测性：`LeadRuleSource.Verify` 异常时订单虽 fail-closed，但不会生成账户级故障状态。本轮增加 `FaultRulesUnavailable` 固定码，归类 `temporary`，`checkOpenLocked` 的空规则源与规则请求失败均上报；它只在**新鲜、完整、同 Lead 账户且已权威对账**的证据出现后才能解除。针对该边界新增永久测试，结果通过。未修改 Main 风控，也不引入真实恢复。

## 四、最终自动化验证

| 验证 | 实际结果 |
|---|---|
| `go test -count=1 -race ./service/leadaccount ./feature/api/binance ./service/futuresownership ./feature ./feature/strategy/line ./scanner ./service/backtest ./command ./controllers -skip '^TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals$'` | **全部 9 包 PASS（RACE_EXIT=0）** |
| `go vet` 同 9 包 | **PASS（VET_EXIT=0）** |
| 根包生产 `go build -o /tmp/stage46_final_production_binary .` | **PASS（BUILD_EXIT=0）**，临时二进制已删除 |
| `git diff --check`、`gofmt -l` | **PASS** |
| 完成后端口 3333 检查 | 没有测试遗留的监听进程 |
| 非致命环境 warning | macOS 链接器产生 `LC_DYSYMTAB` 警告，但所有测试/编译退出码为 0 |

沿用既有约定排除无关的 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`；其余上述 Go 包测试完整执行。

## 五、独立代码审计与遗留风险说明

本轮进行了实现核对、完整联合 Mock、生产入口 AST 扫描和安全边界自检，**未发现新 P0/P1 阻断问题**。这不替代后续如有的独立审计文档。

必须进入 Stage 5 或后续 Gate 而非冒充已完成：

1. **Stage 5**：真实 Lead 私有 User Data WS、ListenKey、重连、按 `account_id` 持久化恢复未知单及 Algo 子单、完整 REST/WS/DB 对账、跨进程锁和额度预留。当前 `pendingReconcile` 没有生产解锁入口，重启恢复也未完成，属于**故意不可交易**。
2. **Stage 5**：API per-Key 预算与观测、通知出口真正接入（Stage 4-5 notifier=nil，且同步事件接口必须由有界非阻塞发送器承接）；账户故障在生产并未连接真实 WS/429 事件。
3. **Stage 5/6**：`RuntimeStatus()` 观测会刷新内部模型状态（Stage 4-5 审计 F1，功能不涉及实盘许可），以及日常规则快照性能、`exchangeInfo` 缓存、故障原因清晰度与高频轮询行为。
4. **Stage 6 / Gate 0-LIVE**：带单 Portfolio 独立 Key 的身份、资产归属、真实支持的 API 白名单与用户强鉴权配置，均未在交易所实际确认。
5. **Stage 7**：用户明确授权后的小额实盘开/平及 Algo STOP/TP 能力验证、真实价格过滤器如 `PERCENT_PRICE`、保证金/杠杆/模式兼容性、保护失败处置；无法从模拟测试推出真实带单平台支持。
6. Stage 3 Main 真正在线业务验收和 Stage 2 实际 MySQL 8 Schema v19 迁移仍需按原计划另行确认；**Gate 4 离线通过不会代替这些前置 Gate。**

## 五-A. CODE_REVIEW_Stage4-6.md 审计跟进（2026-10-10）

审计原结论保持：**没有 P0/P1，Gate 4 仅离线 PASS**。本轮按审计 F1/F2/F3 处理：

| 审计项 | 处理 | 验证/边界 |
|---|---|---|
| **F1 — 生产入口 AST 固定名字可能漏报** | **已修复测试防护**：新增 `TestStage46ProductionBinaryDependencyClosureExcludesLeadAccount`，调用生产项目根目录 `go list -deps .`，断言 `go_binance_futures/service/leadaccount` **不在依赖闭包中**，同时防止 Go 列表为空而误通过；另新增 `TestStage46LeadExecutionAdapterPublicSurfaceAllowlist`，反射检查 `LeadExecutionAdapter` 导出方法精确白名单及 Guard 仅三个允许的公开方法。 | 两项测试已通过 `go test -race`；Stage 5 如需生产**只读**接入，须审计后调整安全门禁，不得静默删除 |
| **F2 — GetPosition 已关闭仓位返回 ErrNoRows** | **已同步 Stage 5 实施方案**：解释 `GetPosition` 的活动仓位口径，要求 `ListPositions(owner,false)` 查全历史受控记录，核对订单和交易所实际持仓；不能把 `ErrNoRows` 当成从未开仓或可以解除 pending。 | 这是后续权威恢复接口契约，非 Stage 4-6 现有执行错误；Stage 5 仍需实现/测试 |
| **F3 — Offline PASS 容易被误解为实盘许可** | **已修订总方案 Gate 4 固定声明**：Offline PASS 仅代表离线 Mock，严禁将它作为启停、撤单或真实下单授权。 | 后续 Stage 5/6/7 均需沿用同一声明 |
| **F4 — 沿用 P3：RuntimeStatus 副作用、同步 notifier、Algo/规则/恢复** | **仍归 Stage 5/6 后续专项实现**；没有因审计将其虚报为已修复。 | WS/恢复/通知等真实链路尚未建立 |
| **F5 — Main 真实业务验收及 MySQL v19 迁移** | **仍待另行授权与确认**；未执行任何真库升级或真实账户操作。 | 不影响 Gate 4 Offline PASS，但不能据此放行实盘 |

本轮只增加验证测试和文档；不修改 `app.conf`、既有策略、数据库或 Binance 交易接口，不提交/push；保留 Stage 4-5/4-6 尚未提交文件。未修改审计原文 `CODE_REVIEW_Stage4-6.md`，以保留独立审计记录。

**审计修复后最终验证**（2026-10-10）：

- `go test -race -count=1`：相关 9 个模块 `leadaccount / feature/api/binance / futuresownership / feature / strategy/line / scanner / backtest / command / controllers` **全部通过**（沿用既有排除项 `TestStaticStrategyTemplatesCompileWithoutDeprecatedMarketGlobals`）；含两项新增的 F1 安全测试。
- `go vet`：同 9 模块 **通过**；生产根包 `go build` **通过**。
- `git diff --check` 与本轮 Go 测试源码 `gofmt -l` **通过**。
- 测试运行不需要真实 Binance API、不连接业务数据库；临时编译产物清理后不留测试进程。


## 六、工作纪律

本次所有写入仅针对新增测试、Lead 故障处理增强和文档；保留原先未提交的 Stage 4-5 修改及用户提供的 `CODE_REVIEW_Stage4-5.md`，没有覆盖或重置任何未提交工作。不修改 `conf/app.conf`、前端、策略研究模板或业务数据库；不执行 `sync db`、不连接真实 Lead Key、无真实下单/撤单、不留后台进程、不自动 commit/push。
