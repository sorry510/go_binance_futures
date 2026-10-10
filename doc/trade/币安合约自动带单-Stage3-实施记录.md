# Binance 合约自动带单 — Stage 3 实施记录

> 日期：2026-10-09
>
> 状态：**共享交易循环、Lead 本地白名单缓存和 Mock Gate 3 离线实现完成；Main 真实运行回归/Lead 实盘验证尚未进行。** 本阶段不启动 Lead 真实下单、WS 或私有 SAPI 定时任务。

## 1. 实施内容

### 3-A — 共享交易运行器和 Main 行为兼容

- 新增 `feature/account_trade_cycle.go`：`accountTradeRunner` 将行情、持仓/挂单、Ownership 对账、撤单、开平仓执行、历史记录、通知、精度/杠杆检查、策略入口、冷却操作改为可注入依赖。
- 现有 `StartTrade` 仍是唯一生产入口：仅实例化 `mainTradeRunner` 并调用共享 `runAccountTradeCycle`。保留 `FutureEnable` 开关、旧日志、每 2 秒由主程序调度、Legacy Main Broker/Ownership/WS、原选币轮询与下单历史/通知语义。
- 将 `feature/feature.go` 原有 LONG/SHORT/MARKET/LIMIT、止盈止损、仓位限制、成交时订单历史、价格精度、30 秒冷却的**同一份业务代码**搬入账户级共享运行器，仅把 Main 特有 IO 调用替换为注入式接口；没有复制 Lead 策略。
- `runAccountTradeCycle` 的退出/对账发生在 `AllowNewOpens` 门禁之前；Lead 停止新开仓或白名单失效时，已有持仓继续尝试平仓。
- `evaluateSharedTradeExit` 将原 `AutoStopOrder` → 止损 ROI / `CanOrderComplete` → 止盈 ROI / `CanOrderComplete` 优先顺序集中在共享模块里；三种情况下的 Main 原开平仓通知分支保持。
- `feature/strategy/line/line_custom.go` 增加 `GetCanLongOrShortWithPositions`：共用同一表达式引擎、指标、Hash 与信号规则，但 Lead 模拟实例的 `Positions` 是**自己注入的账户持仓**。Main `GetCanLongOrShort` 保持旧位置读取和原调用路径。

### 3-B — Lead 安全 Mock 交易循环

- **Stage 3 代码审计后加固**：`leadMockTradeRunner` 已从生产代码移至 `feature/account_trade_cycle_stage3_test.go`，仅测试构建可见；生产的 `accountTradeRunner.validate()` 对 Lead 除 Mock/白名单外还要求 `stage3LeadMockPermit` 接口，**正常生产构建不存在实现这个许可接口的具体类型**。因此仅设置 `Mock=true`，甚至手工复制 Main 运行器的函数钩子，均不能在生产代码中构造合法的 Stage 3 Lead 运行器；正式 Lead 的 account-bound 执行器需要 Stage 4 单独实现并审计。
- Lead Mock 可模拟完整开多、开空、持仓退出、记录、通知、冷却、余额/风控及暂停；任何缺失的依赖都会拒绝运行。
- Main 原参数共享，而持仓、活动订单、仓位上限、已有仓位冲突、历史 Writer、Broker 和冷却均由每个运行器自己的依赖决定。
- Stage 4 必须实现**真实的** Lead 专属执行适配器，并由 account-bound Ownership/Executor、交易所真实快照和资金风控 Gate 完成后才能考虑注入；Stage 7 之前不能实盘。

### 3-C — Lead 可交易合约独立候选池

- `feature/coin_selection.go` 新增 `selectLeadTradeCoins`：从所有现货无关的本地合约候选中先应用完整 Lead 白名单，再复用现有 SmartLocalV2 评分产生 Lead Top60（不足 60 按实际），每轮最多 5 币。不是 Main Top60 的交集，防止遗漏 Main Top60 之外的可带单币。
- `scanner.SmartLocalV2ModeLead` 使用与 Main 不同的轮询 scope；`recentClosedSymbolsForAccountWithOrm` 按 Main / Lead 订单 `account_id` 过滤，防止最近平仓冷却串账。
- Main `SmartLocalV2ModeTrade` 的评分/参数、选币候选池和轮询行为没有变化；其旧历史订单在 Stage 2 已回填 main。

### 3-D — Lead 白名单本地定时缓存

- 新增 `feature/api/binance/lead_symbol_cache.go`：每个 Lead AccountClient 对应一个独立内存 `LeadSymbolCache`，规范化 USDT 白名单、去重、只存脱敏状态及元数据。
- 首次明确启动 `Run(ctx)` 时拉取一次，之后每分钟检查到期并默认**每 1 小时**刷新；缓存**最多有效 2 小时**。读缓存及开仓门禁完全本地化，2 秒扫描和逐币执行不会触发白名单 API。
- 默认 Fetcher 先验证 `LeadTraderStatus.IsLeadTrader`，再获取 `LeadTradingSymbols`；请求有 10 秒超时，取消/凭证更换失效后使用 generation fencing 避免旧请求写回。
- 暂时网络失败保留尚在有效期内的快照，按 1/5/15 分钟退避；401/403、带单资格失效、空名单则立即禁止**新开仓**；状态记录可供 Stage 6 读取。
- `Run(ctx)` 是显式受控生命周期入口，**尚未绑定生产定时调度**：目前用户尚未通过 Stage 6 安全录入和验证 Lead Portfolio 凭证，不得偷偷读取 Main Key 或持续向真实 SAPI 请求。接入独立正式定时任务及错误告警属于 Stage 5/6。

## 2. 离线测试与行为证明

- `feature/account_trade_cycle_stage3_test.go`：Lead 缓存白名单允许/拒绝下单、失效禁止开仓、暂停仍继续对账、已有持仓不重复开仓、无 Mock 绑定拒绝，以及 **Lead 白名单失效后仍允许已管理仓位平仓**。
- Main 固定输入 Golden Test 验证：同一 BTCUSDT 同时出现 LONG/SHORT 信号时，仍以 MARKET BUY LONG / SELL SHORT、qty=4、0 请求价提交，并以交易所精度记录原有 LONG:100.1 / SHORT:99.9 的保守历史价，订单冷却 30 秒；注入全部为 Fake，不执行真实交易。
- `feature/coin_selection_stage3_test.go`：61 个普通合约候选情况下，Main Top60 不含第 61 个，而 Lead 白名单含它时，它必须能进入 Lead 专属 Top60。退出顺序测试锁定 auto-stop > loss > profit。
- `scanner/local_selector_v2_db_test.go`：Main 与 Lead 最近平仓冷却严格按 `account_id` 隔离；旧 SQLite 测试夹具明确使用 `account_id=main` 模拟已完成的 Stage 2 数据回填。
- `feature/api/binance/lead_symbol_cache_test.go`：初始未准备 fail-closed、2000 次热路径查询零额外请求、按小时更新、两小时过期、并发去重、旧请求晚返回失效、401/403 及空名单拒绝、失效后恢复。

## 3. 本阶段未启用/未验证项

- **不执行**业务 MySQL `sync db`，不改 Schema/模型表/配置中心/`app.conf`；Stage 3 无新 DB 版本。
- **不运行**真实 Lead Portfolio API、资金查询、真实开仓/平仓或 WS；Stage 4/5/6 完成可用凭证、风险 Gate、私有 WS、UI 后才有授权运行入口。
- Mock 和 Golden 验证不能替代 Main 的实际运行回归，以及真实 Lead Key 的 `LONG/SHORT/CLOSE_*` 能力与限额测试；Stage 7 授权前不开真单。
- 此次只针对共享循环与白名单逻辑；不修改 `strategy_templates/research` 未跟踪文件，不改变 backtest 引擎与策略结果，不提交或推送 Git。

## 3-A. CODE_REVIEW_Stage3.md 跟进（2026-10-09）

审计：`doc/trade/CODE_REVIEW_Stage3.md`（原报告保持只读、不修改）。报告确认无 P0/P1、Main 机械式重构等价并通过三个临时探针，但正式 Main 业务运行回归未完成。

| 发现 | 分类 | 本次处理 |
|---|---|---|
| F1：真实 Main 运行回归缺失 | P2 / 用户验收 | **仍未完成**；不执行真实开仓/撤单，也不通过 Mock 宣称实盘验证。部署新版后人工核对 MARKET/LIMIT、策略自动平仓/止盈止损、历史通知、30 秒冷却、无 Lead 任务。 |
| F2：公共 source 固定 `start_trade` | P3 / 已修复 | 在共享循环中根据 `runner.AccountID` 标记 `main -> start_trade`、`lead -> lead_trading`，不改变 Main 观测结果；新增 `TestStage3TradeCycleAttributionPerAccount`。Stage 5 还须将实际 Lead WS、定时任务和资金/订单 API 归因完整接线。 |
| F3：`Mock=true` 未阻止真实 Main IO 钩子被误传 | P3 / 已加固 | 将 Lead Mock 工厂**彻底移到 `_test.go`**；生产 Lead 校验必须携带由测试构建实现的 `stage3LeadMockPermit`，仅 Mock 布尔值不能通过；Mock 工厂拒绝 Main 已初始化运行器，新增 `TestStage3LeadMockFactoryRejectsMainRunner`。Stage 4 真实 Lead 执行器仍须重新设计独立工厂、安全凭证与 Broker 绑定，不能复用 Main 钩子。 |
| F4：三个临时探针未写永久测试 | P3 / 已修复 | 新增 `TestStage3MainLimitOrderGolden`（LIMIT 价=100、历史 100、qty=4、30 秒冷却）、`TestStage3MainDirectionSwitches`（只多/只空/都关）、`TestStage3LeadUnavailableWhitelistDoesNotAbortExitCycle`（空池+继续同步/撤单）；测试通过。 |
| F5：两个新增错误分支说明不充分 | P3 / 已补文档 | Main `SelectCoins` 仍返回 `nil error`，Main `EnsureConfig` 仍保留旧的 best-effort/吞错误兼容语义，所以新增 error 返回的两个分支对 Main 不可达；**未来 Lead** 有严格的选择错误/杠杆保证金配置错误时拒绝新开仓，禁止意外绕过。 |

测试边界：回归均为 **Fake/Mock + 代码级**，不建立真实私有 Binance 连接，不主动调用真实 Lead SAPI，不修改 Schema、`app.conf`，不执行 `sync db`、不提交代码。

## 4. 阶段后续 Gate

Stage 3 离线 Gate 验证：Main 共享运行器/原业务逻辑统一、双账户策略代码唯一、Lead Mock 隔离、Lead 白名单本地定时缓存及失败保护、独立候选池/冷却、相关 race/vet/编译验证。**线上 Main 行为仍应由用户以本地/服务器实际业务运行回归确认**。
