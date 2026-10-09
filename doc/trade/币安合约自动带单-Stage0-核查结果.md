# Binance Futures 自动带单 — Stage 0 API/代码可行性核查报告

> 审计日期：2026-10-09
>
> 状态：**官方文档审计 + 项目静态代码审计完成；Lead Portfolio 专用密钥运行时能力待验证（Gate 0-LIVE BLOCKED）**
>
> 审计对象：`/Users/zhz/work/binance/go_binance_futures`、`/Users/zhz/work/binance/go_binance_futrues_new_ui`；现有项目 DB schema `DatabaseSchemaVersion=18`（`appversion/version.go`）。
>
> 范围：仅传统 Futures Lead Trading Portfolio 的自动带单；复用现有“合约交易”选币/策略/平仓逻辑；**不引入 AI 事件触发交易**。
>
> 操作记录：仅只读代码/文档、公开官方文档查询；未调用需 API Key 的私有 Binance API；未做任何真实下单/撤单/开平仓；未读取或修改 `conf/app.conf`，未连接/修改 DB，未构建/运行服务，未改业务代码；不存在“已验证真实带单下单成功”的结论。

## 1. 本阶段结论

### 1.1 明确已确认

1. 官方允许先创建 **Futures Lead Trading Portfolio**，再在 Binance [Futures] → [Copy Trading] → [My Lead] → [API] 创建 Portfolio 专用 API Key。
2. 同一 Lead Portfolio 最多两个 API Key；支持 [Enable Futures]；推荐 IP 白名单；Portfolio 关闭后 Key 自动删除。
3. 该专用 Key 用于 Futures，不支持 Multi-Assets Mode，只支持 USDT；官方给出的 Lead Trader API 订单限制基线为 **20 requests / 10 seconds**（应以实际账户等级与交易所返回限额为准；不把更高级别限额当默认值）。
4. Copy Trading **SAPI** 官方公开两个有用的签名只读接口：`GET /sapi/v1/copyTrading/futures/userStatus` 和 `GET /sapi/v1/copyTrading/futures/leadSymbol`；都是 IP Weight 1。官方 `userStatus` 返回 `isLeadTrader`，`leadSymbol` 返回允许的币种清单。
5. 币安常规 USDⓈ-M Futures REST 文档包括市场/限价单、普通订单查询/取消、Algo 条件单创建/查询/撤单、仓位、杠杆、保证金模式、账户信息和 User Data listenKey；这些是**标准 Futures API 存在**的证据，**不是**专用带单 Key 对每个端点成功的证据。
6. 官方传统带单要求创建 Portfolio（截至 2026-10-01 更新的说明最低 500 USDT），可选择公开/私有；低流动性币种限制杠杆或最大名义仓位，项目现有 Top60 不能代替 Lead 币种白名单和实际规则查询。
7. 本地项目使用 `github.com/adshao/go-binance/v2 v2.8.12`；当前所有私有交易和 WS 均设计为**单 main Futures 账户**；不能靠换 Key 或创建第二个全局 Client 快捷完成隔离。

### 1.2 不得提前宣称已确认

- 该用户的 Binance 地区/实名资格是否允许创建 Lead Portfolio。
- 当前用户是否已创建 Portfolio、是否有可用的专用 API Key（本阶段既未索取也未读取密钥）。
- 专用 Key 在 `/fapi` 下读取的究竟是 Lead Portfolio 仓位/权益还是普通个人 Futures 账户；**必须核对 Portfolio 页面数据与 API 数据**。
- 专用 Key 是否支持 `positionSide=LONG/SHORT` 的 Hedge Mode、修改初始杠杆/逐仓、MARKET/LIMIT、Algo TP/SL、正常 orderID/clientOrderID 查询、User Data listenKey/WS 等所有项目实际需要的动作。
- 是否存在正式可用的 **Lead Portfolio Testnet**。普通 USDⓈ-M Testnet 并不能证明专用 Lead Key 兼容。
- 触发用户跟单后的同步效果、跟单账户实际成交或分成细节；这些并非本项目首版控制范围。

### 1.3 Stage 0 Gate 分类

- **Gate 0-DOC：通过**。官方文档、项目私有 API 调用清单、代码耦合点、风险和 Stage 1 文件清单均已形成证据。
- **Gate 0-LIVE：阻塞（缺专用 Lead Portfolio Key 和授权测试条件）**。所有 Lead 专用 Key 真实权限标记 `待实测`，不是“失败”，也不是“通过”。
- **后续允许的行动**：可以先实施 Stage 1/2 的本地纯离线基础改造和 Fake Broker 测试；不得开启 Lead 真单或宣布端到端已通。
- **解除阻塞条件**：用户准备好可用 Lead Portfolio/API Key，Stage 1 提供安全本地录入；以只读验证身份/白名单/权益/仓位/订单，必要时按 Stage 7 明确批准极小额真实交易。不可要求用户把 Secret 直接发在聊天中。

## 2. 官方 API 能力矩阵（文档 vs Lead 实测严格分离）

状态符号：`已公开`＝有官方接口文档；`代码已有`＝当前仓库已封装；`待实测`＝没有以 Lead Key 做过测试；`不在范围`＝首版不实现。**“官方已公开”不代表 Lead Key 可用。**

| 需求 / 端点 | 官方现状 | 代码入口/实现 | Lead Key 结果 | 测试 Gate |
|---|---|---|---|---|
| 创建 Lead Portfolio、API Key | Binance 网页创建，非项目 API | 无 | 需用户侧准备 | 仅阅读官方资料，核对产品 |
| Lead 资格 `GET /sapi/v1/copyTrading/futures/userStatus` | 已公开，SAPI 签名/权重1 | 未实现 | 待实测 | 返回 success + isLeadTrader，核对账户 |
| Lead 币种白名单 `GET /sapi/v1/copyTrading/futures/leadSymbol` | 已公开，SAPI 签名/权重1 | 未实现 | 待实测 | 拿到清单并按 USDT 与 Symbols 交集 |
| 组合权益/余额 `GET /fapi/v2/account` | 已公开 | `binance.GetFuturesAccountContext` (`NewGetAccountService`) | 待实测 | 必须与 Portfolio 页面资产一致 |
| 仓位 `GET /fapi/v2/positionRisk`、V3 | 已公开 | `GetPositionContext`、`GetPositionFreshContext`、`GetPositionV3` | 待实测 | 不允许读到 main 仓位；验证 LONG/SHORT 字段 |
| 当前挂单 `GET /fapi/v1/openOrders` | 已公开 | `GetOpenOrderContext`、`GetOpenOrderFreshContext` | 待实测 | 对比 Portfolio 挂单及订单范围 |
| 历史订单 `GET /fapi/v1/allOrders` | 已公开 | `GetOrders`、`GetLimitStartTimeOrders` | 待实测 | 确认不串 main、时间/分页精确 |
| 单笔订单 `GET /fapi/v1/order` | 已公开 | `GetOrderByClientOrderID`、`GetOrderByOrderID` | 待实测 | 检验返回 order ID / fill_qty / status |
| MARKET/LIMIT `POST /fapi/v1/order` | 已公开 | `CreateOwnedOrder` 经 Ownership Executor | 待实测 | 真单需 Stage 7 明确授权；MARKET 与 LIMIT |
| 撤正常单 `DELETE /fapi/v1/order` | 已公开 | `CancelOrderContext` 经 Executor | 待实测 | 只撤自身 managed 订单 |
| TP/SL Algo `POST /fapi/v1/algoOrder` | 已公开 | `CreateOwnedAlgoOrder` | 待实测 | 带单 Key 能否使用、Hedge Mode/数量语义 |
| 查/撤 Algo `GET/DELETE /fapi/v1/algoOrder` | 已公开 | `GetAlgoOrderByClientOrderID` / `CancelAlgoOrder` | 待实测 | 发生触发后实际委托与幂等对账 |
| 杠杆 `POST /fapi/v1/leverage` | 已公开 | `SetLeverageContext`、`EnsureTradeConfigContext` | 待实测 | 实际组合账户杠杆、低流动性限额 |
| 逐仓/全仓 `POST /fapi/v1/marginType` | 已公开 | `SetMarginTypeContext` | 待实测 | 与当前币种配置一致 |
| 持仓模式 `GET /fapi/v1/positionSide/dual` | 已公开 | **缺少首版必需的显式只读 Gate 封装** | 待实测 | Hedge Mode 必须先只读确认，不自动更改 |
| listenKey `POST/PUT /fapi/v1/listenKey` | 已公开，Futures User Data | `GetListenKey`、`UpdateListenKey` | 待实测 | 使用 Lead Key 返回独立流，不复用 main listenKey |
| User Data WS 事件 | 已公开 `ACCOUNT_UPDATE`/`ORDER_TRADE_UPDATE` | `WsUserData`（单账户写表） | 待实测 | 只产生 Lead 账户事件，重连对账无串账 |
| server time、exchangeInfo、Kline/ticker/depth | 已公开公共行情 | `index.go` + WS/caches | 不依赖 Lead Key | 公共数据共享，不重复订阅 |
| Lead Portfolio Testnet | **未确认官方支持** | 当前仅普通 Futures Testnet | 待确认 | 无证据则禁止声称完成 Lead 测试网验证 |
| 自动创建跟单关系/分成/管理粉丝 | Copy Trading 公开 REST 未列出可依赖完整接口 | 无 | 不在范围 | 不实现，也不依赖非官方抓包接口 |

注意：项目内 `CreateOwnedOrder` 支持的是 MARKET/LIMIT；`CreateOwnedAlgoOrder` 支持 STOP/TAKE_PROFIT 相关保护单。合约主 `StartTrade` 的主要 TP/SL 出场仍调用自定义策略并市价平仓，不能把“拥有 Algo API 封装”误写为“每一笔主策略交易都自动挂条件单”。

## 3. 代码路径审计结论（现状）

### 3.1 主交易链：保持语义完全一致

`main.go` 每 2 秒调用 `feature.StartTrade(&SystemConfig)`；先检查 `FutureEnable`；`GetAllSymbols` + `selectConfiguredCoins(... SmartLocalV2ModeTrade)`；`GetTransformPositionsContext` / `getTransformOpenOrdersContext`；`syncStrategyExitPositions`；`cancelTimeoutAutoStrategyOrders`；对受控仓位依次执行 `AutoStopOrder`、止损 ROI 阈值 `CanOrderComplete`、止盈 ROI 阈值 `CanOrderComplete`；新单判断亏损仓位 `LossMaxCount`、总 `FutureMaxCount`、`FutureAllowLong/Short`；遍历选币，调用唯一 `TradeLineCustom.GetCanLongOrShort`；按 `Usdt/leverage/depth/stepSize/tickSize` 算数量；`UpdateSymbolTradeInfoContext`；经 Ownership 下 MARKET/LIMIT；执行历史/通知；新单后 `Sleep(30s)`。

**结论**：共用的应当是上面这条**完整交易流程**而不只是策略信号。Stage 3 提取后的两账户 executor 各自持仓/风控/退出/冷却，但共享选币、策略与币配置；golden test 检查 main 回归。

### 3.2 单账户硬绑定（按严重度）

| 风险 | 确切证据 | 后果 / 必须改造 |
|---|---|---|
| P0 全局私有 API Client | `feature/api/binance/index.go:31-37,86-100,160-185` 全局 Key、`futuresClient`、`futuresSignedTimeMu` | 不能在一个实例中任意切换 Key；每个账户独立签名时钟、HTTP |
| P0 WS 直接写全账户镜像 | `feature/api/binance/index.go:1367-1485` 的 WS handler 默认写 `futures_positions`/`futures_orders` | 账户事件会相互覆盖，必须 account-scoped |
| P0 全量同步无账户过滤 | `feature/feature_userdata.go:70-72,108-164` 包含直接全表 DELETE 和 `updateTime` 清理 | 若第二账户复用此表会清空另一账户持仓/挂单 |
| P0 Ownership 键缺账户 | `models/futures_managed.go`、`service/futuresownership/service.go`、`reconcile.go`、`feature/ownership.go` | account+symbol+side 才是冲突/对账键 |
| P0 多账户共用仓位缓存 | `feature/api/binance/account_read_cache.go` 全局 `positionSnapshot/openOrdersSnapshot`、`singleflight` | Lead 会读到 Main 的余额/仓位，必须实例分区 |
| P0 下单前交易配置缓存只按币 | `feature/api/binance/trade_config_cache.go` `items[symbol]` / `singleflight(symbol|margin|leverage)` | 错以为 Lead 杠杆/逐仓已配置；强制分 account |
| P0 下单前配置错误被吞掉 | `feature/feature.go::UpdateSymbolTradeInfoContext` 的 `_ = EnsureTradeConfigContext(...)` | Lead 下单可能在未知杠杆/保证金模式；Stage 3/4 新链路必须 fail-closed 且 main 行为回归 |
| P0 受控 broker 默认打 main | `service/futuresownership/execution.go::BinanceOrderBroker` 调用全局 `binance.CreateOwned.../GetOrder...` | Stage 1/2 注入 AccountBroker，不允许 Lead 调用无账户签名入口 |
| P1 订单旧历史无账户归属 | `models/tableStruct.go::Order`；`feature/feature.go::insertOpenOrder/insertCloseOrder` | 不同账户 OrderID 可能相同，历史 open/close 关联串账 |
| P1 自动维护会误链接/删其他记录 | `feature/order.go::UpdateOrderStatus`；`controllers/orders.go` | 增 account_id 并隔离旧 API 删除/历史联动 |
| P1 策略模块有隐式账户数据 | `feature/strategy/coin/common.go` 的 `getLimitMinOrder` + 10s cache、`getLimitMinLocalOrder` | 旧 coin1-6 路径与 local cooldown 必须标注 account scope，虽然主 SmartLocalV2 路径目前不是这段 |
| P1 交易副作用修改共享 Config | `feature/feature.go::insertCloseOrder` 调 `AutoLossScale` / `autoTradeToTest` | Lead 禁止无意修改主账户风控、交易开关 |
| P1 WS 连接周期是单例 | `feature/api/binance/index.go::GetListenKey/UpdateListenKey/WsUserData`、`feature/feature_userdata.go` static generation | 重连、快照有效性、keepalive、WS 事件必须实例隔离 |
| P1 account tradeConfig 读同全局 | `feature/feature.go::GetTransformPositionsContext/getTransformOpenOrdersContext`、`feature/ownership.go` | Stage 3 注入 AccountContext；Lead 不得调用 main 的读缓存 |
| P1 Rate limit 不是 Lead per-key | `service/binanceapiusage/transport.go` + `budget.go` 聚合 product/environment、source | 保留同 IP 公共预算，增 Lead Key 独立 20/10s 订单写限流 |
| P1 监控只看单账户 | `service/systemhealth/service.go`、`controllers/account.go` | 健康/账户页兼容 main，Lead 独立指标 |
| P2 没有 Lead 功能路由 | `routers/router.go`、`src/router/modules/futures.ts` | Stage 6 新页面和 API，配置复用主合约 |

### 3.3 本阶段直接得到的 5 个工程决策

1. **最小改造顺序**：Stage 1 先抽离 AccountClient / Broker / 私有缓存，并保证旧 main 快捷方法委派到明确 main context；Stage 2 再引入持久化 account_id、WS mirror 和 Ownership 作用域；Stage 3 最后拆 StartTrade。否则提取后的代码仍会错误访问全局 main 账户。
2. **Owner 不因账户而复制**：首版 Main 和 Lead 都可继续用 `owner=auto_strategy`；账户不同靠 `account_id` 隔离。所有已存在的 `OwnerAutoStrategy`、`OwnerNewCoinRush`、`OwnerNoticeAutoOrder`、`OwnerFundingRate`、`OwnerAgentTrade` 历史归 main。
3. **Lead Key 的安全验证层独立于主合约 Client**：SAPI 的 host 是 `https://api.binance.com`；订单签名 host 是 `https://fapi.binance.com`，避免误把 SAPI 请求发送到 futures URL 或反之；签名时钟和 proxy 仍可复用共用组件但不能复用另一账户的凭证。
4. **Lead 白名单与主选币交集**：共享 `SmartLocalV2` 选币，但在 Lead 账户开仓门禁再次校验 `leadSymbol` 及实际合约元数据；被挡下仅跳过 Lead，不改变 Main 的选币结果。
5. **余额/仓位视图真实性 Gate**：如果 Lead Key 对普通 `/fapi` 请求实际返回主 Futures 账户数据、缺少 `LONG/SHORT` 或 403/权限不匹配，则必须停在 Gate 0-LIVE，不得尝试通过猜测参数/抓包接口绕过。优先官方支持工单/文档澄清。

## 4. Stage 1 可直接执行的精确改造清单

### 4.1 新增 API 抽象（仅设计，本阶段未创建代码）

建议引入 `feature/api/binance/account_client.go` 或 `service/futuresaccount/client.go`：
- `type FuturesAccountID string`，`const MainAccountID="main"` / `LeadAccountID="lead"`；
- `type AccountClient`（包含 account ID、`*futures.Client`、signed timestamp mutex、proxyHTTP、read caches、tradeConfig cache、userStream generation/health、budgetSource、readiness gate）；敏感 Key 不输出到 JSON；
- `type AccountBroker` 实现现有 `futuresownership.OrderBroker`，给每个实例绑定唯一 AccountContext；旧 `BinanceOrderBroker{}` 在 main 的兼容入口内部显式绑定 main；
- `type AccountReader` / `AccountOrderSource`，对余额/仓位/挂单/订单/杠杆请求进行依赖注入；
- 共享公共 `MarketDataProvider`（Kline、Depth、Ticker、MarkPrice、ExchangeInfo 与既有市况指标）；公共行情不经 Lead 签名 Key。

### 4.2 Stage 1 首批修改文件和函数（需逐函数复核调用方）

| 顺序 | 文件 | 改造点 | 回归/测试 |
|---|---|---|---|
| 1 | `feature/api/binance/index.go` | 抽出独立 SignedClient、doFuturesSigned；main 包装函数保留 | 两账户不同 Key、不同 TimestampOffset、报错独立 |
| 2 | `feature/api/binance/account_read_cache.go` | position/openOrders cache + generation + singleflight 每账户隔离 | A 填缓存，B 不命中；A mutation 不清 B |
| 3 | `feature/api/binance/trade_config_cache.go` | `account_id + symbol` key；config mutation 先验证实际结果 | A 成功不能跳过 B 的杠杆设置 |
| 4 | `feature/api/binance/proxy.go` | WS 私有连接不要依赖单例 client；公开行情保持复用 | 两个 listenKey 不串，SOCKS5h 兼容 |
| 5 | `service/futuresownership/execution.go` | Broker 可绑定 AccountContext；先不修改 Ownership 模型 | Fake Broker 验证 routing/account isolation |
| 6 | `service/binanceapiusage/transport.go` + `budget.go` | source 区分 main/lead；全 IP 全局预算保留，另加 Key 订单写预算 | 429/418、20/10s、跨账户权重 |
| 7 | `feature/api/binance/trade_config_cache_test.go`、`account_read_cache_test.go`（扩展）| 主/带单独立命中/失效和并发 | `-race` 安全 |
| 8 | `service/futuresownership/execution_test.go`（扩展） | ClientOrderID、重试、未知状态、方向合法性 | 不实盘 |
| 9 | `feature/api/binance/lead_readonly.go`（建议新增） | SAPI status/leadSymbol 签名只读客户端（专用 Key） | Fake HTTP、校验分页/解析/失败分支 |

**Stage 1 暂不**写入两个交易账户的真实仓位数据，不调整 DB，不在主入口启动 Lead 交易；任何 Stage 1 实现都必须能在 `lead_enabled=false` 时与原 main 路径保持一致。

### 4.3 Stage 2 先决清单（防止 Stage 1 完成后遗漏）

- `models/futures_managed.go` 的两张表加 `account_id`；`service/futuresownership/service.go` 的全部查询条件和幂等键跟进。
- `models/futures_postions.go`、`models/futures_orders.go` 的镜像添加账户维度；`feature/feature_userdata.go` + `feature/api/binance/index.go` 的 WS INSERT/UPDATE/DELETE 全部补 scope。
- `feature/order.go`、`feature/feature.go` 的历史订单写入、open/close 关联、`controllers/orders.go` 查询/删除更新；`feature/strategy/coin/common.go` 隐式历史判断改账户作用域或坚持 main-only 限制。
- `service/futuresownership/reconcile.go`、`service/systemhealth/service.go`、`controllers/account.go`、`controllers/agent_trade.go`、`feature/ownership.go`、`feature/trade_cycle_snapshot.go` 全部标注账户来源；不能让 Agent owner 的主账户对账顺带读取 lead。
- `main.go` / `models` 模型注册、`command/db_update.go` / `appversion/version.go` / migration test；仅 `./go_binance_futures sync db` 进行数据库升级。
- 默认 main 历史行账户归 `main`，Lead 表/字段没有授权不能设置交易开关；引入联合唯一键必须先核对现有 SQL 索引与历史碰撞，禁止无备份直接改 Schema。

## 5. 本 Stage 0 的验收证据与阶段状态

### 5.1 静态检查结果

| 检查 | 执行方式 | 结果 |
|---|---|---|
| 当前设备和项目 | Remote Desktop Commander（授权机器） | 连接成功，能读取现有后端仓库 |
| 未改动业务代码 | `git status --short`，读取工程文件 | 本阶段新增本报告/更新阶段记录；其他现有变更保留原样 |
| SDK 版本 | `grep github.com/adshao/go-binance go.mod` | `v2.8.12` |
| Schema 基线 | `cat appversion/version.go` | `DatabaseSchemaVersion=18` |
| 主交易路径 | `feature/feature.go` + `main.go` | 已梳理完整开平仓/退出/超时撤单逻辑 |
| 签名 REST 路径 | `feature/api/binance/index.go`、`service/futuresownership/execution.go` | 已列出主交易必需请求 |
| WS 数据持久化 | `feature/api/binance/index.go::WsUserData`、`feature/feature_userdata.go` | 单账户写表、全表删除点已定位 |
| 旧 Owner 与历史数据 | `models/*`、`service/futuresownership/*`、`feature/order.go`、`controllers/orders.go` | 关键 account_id 缺口已定位 |
| SAPI 带单只读接口 | Binance Developer Docs | 两个签名接口文档可查 |
| 带单专用 API Key 兼容性 | 需要真实 Lead Portfolio Key | **未测；Gate 0-LIVE 阻塞** |
| 编译、单测、真实 Binance API | 本次只读 Stage 0 | 未执行，不宣称通过 |

### 5.2 Gate 0-DOC 决定

**通过**：现在已有足够的代码与官方 API 资料启动 **Stage 1（严格离线 + main 兼容 + fake clients）**。

### 5.3 Gate 0-LIVE 决定

**未通过/被外部前置条件阻塞**（不是代码失败）：在安全录入专用 Key 之前，不能断言 Lead `/fapi` 私有 API 可用，不允许实盘。

下一次解除条件（按顺序）：
1. 用户已在币安官方页面创建可用的传统合约带单组合（确认产品地区资格与资金门槛）。
2. 在 Stage 1 实现安全录入及读权限检查后，用户自行输入专用 Key（**不要在聊天中发送 Secret**），限制可信 IP。
3. 只读检查 `userStatus`、`leadSymbol`、组合权益、仓位、未成交委托和持仓模式；校对 Binance Portfolio 页面与返回数据。
4. 有下单/条件单/杠杆/WS 权限不确定性时记录 `unknown`。只有在用户对真实下单单独授权后，进入 Stage 7 的极小额真单验证。
5. 禁止从普通 USDⓈ-M Testnet 的成功结果推断专用 Lead Portfolio API 成功。

## 6. 官方一手资料（查阅日期 2026-10-09）

- [Futures Lead Portfolio API Key 创建及限制（更新 2026-02-27）](https://www.binance.com/en-AE/support/faq/detail/2bec848b904b422197ce121d0925f20b)
- [传统 Futures Lead Trading 业务规则（更新 2026-10-01）](https://www.binance.com/en-IA/support/faq/detail/6acfb4c1f50c4db1b9e915181ff31a4c)
- [Futures Copy Trading SAPI：Lead Trader Status / Symbol Whitelist](https://developers.binance.com/en/docs/catalog/advanced-trading-copy-trading/api/rest-api/future-copy-trading)
- [USDⓈ-M Futures REST：交易、订单、Algo、杠杆、仓位](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/trade)
- [USDⓈ-M Futures REST：账户信息](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/account)
- [USDⓈ-M Futures REST：User Data listenKey](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/rest-api/user-data-streams)
- [USDⓈ-M Futures WS API：User Data Stream](https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-usd-s-m-futures/api/ws-api/user-data-streams)

> 文档 API 是规范、源码路径是实现证据，实盘权限尚未验证。Stage 1 应保留这三类证据层级，不能将“有接口定义”写为“该账户已执行成功”。

