# Binance Futures 自动带单（Traditional Lead Trading）Stage 实施方案

> 版本：v1.0（规划稿）  日期：2026-10-09
>
> 实施状态：Stage 0 官方文档审计、Stage 1 多账户 Client、Stage 2 账户级 Ownership/仓位订单镜像与历史查询的离线实现均已完成；Stage 2 真实 MySQL 8 升级仍未运行，Stage 3–7 未开始。Lead 实盘 Gate 未开放。
>
> 后端：`/Users/zhz/work/binance/go_binance_futures`
> 前端：`/Users/zhz/work/binance/go_binance_futrues_new_ui`
>
> 目标：复用现有“合约交易”整套选币、自定义策略和开平仓流程，让币安传统 Futures Lead Trading Portfolio 使用专用 API Key 自动执行。AI Opportunity / Agent Trade / AI 事件触发交易完全不纳入第一版。

## 0. 范围、执行约定与非目标

### 0.1 一期做什么

1. 第一版仅支持 **1 个传统币安合约带单 Portfolio**，使用 Binance 平台创建的 Lead Trading API Key（带单密钥），与普通合约主账户严格隔离。
2. **完整复用现有合约交易路径**：`scanner.SmartLocalV2ModeTrade` 选币（含 Top60 规则）；每个币的 `models.Symbols` 配置、`line.TradeLineCustom.GetCanLongOrShort`、`AutoStopOrder` 和 `CanOrderComplete`；多空准入、持仓数、亏损仓位数、MARKET/LIMIT、限价单超时取消、TP/SL、策略触发平仓、订单/仓位同步与通知。
3. 带单和普通合约独立启停，且默认**共享同一份币种列表、策略 JSON、各币杠杆、保证金类型、下单 USDT、止盈止损与全局交易参数**。带单只单独配置账户启停、密钥、账户级熔断/额度；不复制策略、币种配置或新增独立策略体系。
4. 普通账户和带单账户允许同一币种同方向各持有一个独立仓位；严禁跨账户互相认领、平仓、撤单或共享账户级镜像。
5. 共用公开行情和策略指标缓存；按账户隔离所有签名 REST、下单限流、User Data WS、持仓/订单缓存、交易配置状态、Ownership、结果统计、监控源和运行时锁。
6. 在带单平台开通条件、API 权限、Hedge Mode、STOP/TP Algo Order、User Data WS 等关键能力通过实测 Gate 前，**禁止带单账户真实自动下单**。

### 0.2 一期明确不做

- 不接入 AI 事件/Opportunity/Agent Trade Proposal/Agent 受控执行（现有 AI 模块保持原状）。
- 不做外部跟单员自动跟随、跟单员排名/抓取、收益分成管理、跟单用户同步、资金划转、创建或关闭带单 Portfolio。
- 不做 SaaS、多租户、用户托管、多带单 Portfolio 动态切换；扩展为第二个组合只预留 account_id 能力。
- 不改回测策略定义/研究参数，不为带单重新实现一套策略/回测；不默认为带单开任何订单。
- 不为了带单便利更改现有 `app.conf` 或主账户凭证。

### 0.3 项目安全与工作纪律

- 不修改 `conf/app.conf`；密钥绝不进入 Git、日志、API 响应或明文数据库字段。
- 不自动 commit/push，不回退、不覆盖任何未提交修改，尤其 `strategy_templates/` 与前端现有工作区改动。
- 需要数据库 schema 时只生成迁移代码，获批准后在目标库通过 `./go_binance_futures sync db` 执行，不擅自操作主库；迁移要可重复、非破坏性并兼容 MySQL 8。
- 本地测试不留后台进程、不长期占用 3333；优先 mock/fake Broker；与币安真实资金交互只能在明确授权的联调阶段操作。
- 每完成 Stage 在本文件 `实施记录` 增加日期、完成项、测试、变更文件、遗留问题与 Gate 结论。

## 1. 当前代码基线（2026-10-09 静态审计）

| 现有链路 | 代码位置 | 现状及必须处理的耦合 |
|---|---|---|
| 自动交易循环 | `main.go` → `feature/feature.go::StartTrade` | 2s 调度，仅判断 `FutureEnable`，混合选币/策略/资金账户/订单/通知 |
| 自定义策略 | `feature/strategy/line/`；`GetCanLongOrShort`/`AutoStopOrder`/`CanOrderComplete` | 核心规则共享，禁止复制第二份 |
| 选币与交易配置 | `scanner/`、`models/tableStruct.go` (`Config`、`Symbols`) | Top60 与同一币配置共享；账户单独风控状态 |
| Binance 适配 | `feature/api/binance/index.go` | 全局 futuresClient、时间偏移锁、listenKey、签名 REST 与缓存 |
| 账户 User Data WS | `feature/feature_userdata.go`、`feature/api/binance/index.go::WsUserData` | `futures_positions` / `futures_orders` 默认单账户镜像，全量更新和重连均需隔离 |
| 受控订单 | `service/futuresownership/`、`feature/ownership.go` | Owner/SourceRef/ClientOrderID/成交对账可复用；缺 account_id |
| 历史订单 | `models.Order`、`insertOpenOrder`/`insertCloseOrder` | 旧 `order` 表用于展示/近似统计，不能充当 Ownership；跨账户订单号会冲突 |
| 策略副作用 | `feature/feature.go::insertCloseOrder` | `AutoLossScale`、`autoTradeToTest` 修改/依赖主全局配置，不能跨账户触发 |
| 调用预算 | `service/binanceapiusage/` | 保留统一 IP 总预算，新增带单 per-key 限速和 source attribution |
| 前端菜单 | `src/router/modules/futures.ts`、`src/views/dashboard/configShow.vue` | 在“合约交易”下加页面；不影响现有配置中心行为 |
| 数据库版本 | `appversion/version.go` | 静态审计时 `DatabaseSchemaVersion=18`；实施前重新核对 |

既有 V3-5 Ownership 真相源：`doc/trade/合约自动交易仓位归属隔离改造计划.md`。账户隔离不得弱化其 fail-closed、managed_qty 和 unmanaged 仓位约束。

## 2. 唯一认可的目标架构

`TradeStrategyConfig + MarketSnapshot`（只读共享） → `SelectCoins / EvaluateSignal`（同一策略结果） →
`AccountTradeLoop(account=main | lead)`（账户隔离快照、风控及状态） →
`OwnershipExecutor(account_id, owner=auto_strategy)`（账户内受控修改） →
`AccountBroker(account_id)`（专用 Key/HTTP/WS/限速/订单查询） →
`Binance Futures Main Account | Traditional Lead Portfolio`。

**不得**先让主账户真实成交、再把主账户订单复制到带单账户。带单账户必须独立接收同一策略信号并独立通过风控，下单和对账按各账户自己的实际状态执行。

### 2.1 不可破坏的不变量

- `AccountID` 是任何私有交易变更的强制参数，不允许在新增带单调用链上默认为 main；旧 main 封装可保留显式兼容入口。
- 修改边界：`account_id + symbol + position_side + owner + managed_qty`；账户间相同币种/方向合法，账户内冲突保持原来的禁止规则。
- 订单幂等：相同账户、相同策略事件、相同 side+intent、相同开仓窗口不能重复执行；已有未知成交先 reconcile，绝不盲重试。
- 归属不明一律 `unmanaged`，不自动认领现有仓位，不操作手工单。
- 主账户历史记录迁移后归属 `main`；升级与回滚期间主账户结果、旧接口和数据不变。
- 带单停止新开仓≠强平；暂停后仍对本账户已登记的仓位执行原策略退出、TP/SL、限价超时取消和对账（WS 异常时按 fail-closed 行为分级处理）。
- 现有策略计算输入、指标和交易信号不可为了复用而改变；不因双账户让历史回测结果发生变化。
- 任一账户 API Key、WS、订单/持仓镜像失效只暂停该账户风险变更，不影响另一账户；公共行情异常按共有风险处理。

## Stage 0 — 官方能力验证与调用路径清点（只读阶段）

**目标**：排除“带单 Key 不能执行项目现有订单类型或模式”的失败路线；确认是否可直接复用 USDⓈ-M 的 Futures API。

工作：
1. 核对官方文档与 Lead Portfolio 类型、地区资格、API Key/IP 白名单、USDT 单资产、限速要求；由用户在 Binance 页面创建组合和密钥，系统不得自动创建。
2. 逐项列出当前代码使用的签名端点：balance、positions、openOrders、order/getOrder、cancelOrder、leverage、marginType、User Data listenKey/WS、Algo STOP/TAKE_PROFIT，以及时间同步和 proxy。
3. 明确 SAPI 带单身份查询 `GET /sapi/v1/copyTrading/futures/userStatus` 和币种白名单 `GET /sapi/v1/copyTrading/futures/leadSymbol` 与常规 /fapi 交易端点分工。
4. 验证或在 Gate 中阻塞：身份、组合资金视图、Hedge Mode 双向持仓、每币杠杆/逐仓配置、常规单与 Algo 条件单、挂单查询、WS 成交与仓位事件是否支持带单密钥。
5. 记录测试网是否支持 Lead Portfolio 专用密钥。不得把普通 Futures Testnet 验证等同于带单 Portfolio 验证。
6. 扫描主账户所有隐式签名 API/全局变量、硬编码 DB 查询及用户数据订阅写表点，形成调用清单和改造影响列表。

建议变更：仅文档与只读测试/能力矩阵，不修改运行策略、订单、配置或 DB。  
**验收 Gate 0**：每个必需 API “支持/不支持/待实测”有结论和证据；关键交易能力未通过前不得开启后续真单联调。

**2026-10-09 阶段结果**：详细能力矩阵、代码风险清单、Stage 1 改造文件及离线/实盘 Gate 见 [Stage 0 核查报告](币安合约自动带单-Stage0-核查结果.md)。Gate 0-DOC 已通过，Gate 0-LIVE 暂被专用 Key/安全验证前置条件阻塞。允许先做 Stage 1 的本地账户隔离与 Fake Broker 测试；不允许带单真单。

## Stage 1 — 账户对象和独立 Binance Client

**目标**：两套 API Key、时间偏移、HTTP、REST 签名查询与缓存完全独立，市场行情复用。

后端计划：
1. 引入明确的账户标识 `main`、`lead` 和 `AccountContext`（ID、kind、Broker、Cache、WSState、Limits、CircuitState；绝不暴露 Secret）。
2. 将 `futuresClient` 使用改为实例化的 `FuturesAccountClient` 或 AccountProvider；包括签名时间偏移锁、请求重试、http client、testnet/baseURL、proxy、API usage source、交易规则缓存的账户隔离。
3. 现有 main 入口保留兼容包装函数；新增 lead 入口必须显式接受 AccountContext；市场 depth/kline/ticker 继续共用行情服务，不按账户重复抓取。
4. 加入 Lead Credential Provider（安全加密、IP 白名单提示、凭证轮换、测试连接、只读状态），内存不打印密钥；凭证删除/替换需要停机安全确认。
5. 私有 REST 缓存 `account_id` 分区，mutation 只失效对应账户，不允许一个账号使用另一个账号的本地 WS 镜像或 cached balance。
6. 限速按 Binance API Key + IP 双约束处理，lead 专用订单限速设计从 20 / 10s 开始，不以缓存改变资金写操作的时效性。

预计路径：`feature/api/binance/`、`binanceproxy/`、`service/binanceapiusage/`、`service/futuresaccount/`（新增建议）。  
**验收 Gate 1**：注入两个 fake 客户端，交叉签名请求、cache、时钟重同步及 429 均不串账户；main 行为和测试通过；不调用真实下单。

**2026-10-09 阶段结果**：已实现 AccountClient、Lead SAPI 只读接口、独立缓存、Broker 路由与带单本地保守下单限流；Mock 单元测试及竞态测试通过。详见 [Stage 1 实施记录](币安合约自动带单-Stage1-实施记录.md)。Main 原有 API/交易调用方式保持不变；Lead 暂不启用 Executor、数据库写入或 WS。

## Stage 2 — Account-scoped Ownership、镜像和历史订单

**目标**：整个项目持仓/订单读写链条统一引入 account_id，先打通主账户兼容再启用带单。

后端计划：
1. 给 `futures_managed_positions`、`futures_managed_orders` 增 `account_id`，现有记录安全回填 main；全部 `GetPosition/CloseQuantity/ClaimOrder/ActiveOrders/Reconcile/Cancel` 接口、冲突检查、SQL 查询带 account_id。
2. 给账户镜像 `futures_positions`、`futures_orders` 增 account_id 与联合唯一键（如 account+symbol+side、account+exchange order id）；User Data WS 的 UPDATE/DELETE/清理都按账户隔离。
3. 给旧 `order` 历史表增账户维度或新增严格关联映射；历史查询/组合 open→close 匹配、`UpdateOrderStatus`、统计和列表按账户隔离，旧 main 数据无损保留。
4. 所有账户更新应在有效 DB 约束/事务中实现：同账户同 `symbol/position_side` 不允许多个活动 Owner；只用全局 Go mutex 无法覆盖未来多实例；保持现有严格 fail-closed。
5. 对 `ClientOrderID` 做全局唯一生成（带 account 前缀）及 DB 唯一约束；同账户订单提交结果 unknown 时只 reconcile；不同账户相同交易所 OrderID 不得串账。
6. 统一 `syncStrategyExitPositions`、`ensureAccountOpenSlotAvailable`、`submitManagedStrategyClose`、`ownershipAccountQuantities` 作用域；先覆盖 main 既有全部 owner（auto_strategy/new_coin_rush/notice_auto_order/funding_rate/agent_trade），再接入 lead。
7. 升级流程只走 `./go_binance_futures sync db`；先备份、迁移、校验再启用，默认 lead disabled。v19 迁移允许用 `UpdateDatabase` 中按版本门禁调用的 Go 函数执行条件 DDL / 数据回填 / 校验：失败不得升版本，重试须幂等；不会删除历史行或擅自合并重复记录。MySQL DDL 不支持事务回滚，需按已建索引跳过并继续。

预计路径：`models/`、`service/futuresownership/`、`feature/ownership.go`、`feature/feature_userdata.go`、`feature/order.go`、`main.go`、`command/`、`appversion/`、`controllers/`。  
**验收 Gate 2**：历史数据完整；两个账户可同币同方向；任何关闭/撤单都无法跨账户；断线清理不会删除另一账户镜像；所有已存在 owner 的行为与当前一致。

**2026-10-09 阶段结果**：五张表添加账户维度及 Schema 19 迁移，Ownership 与订单执行账户绑定、事务、Main/Lead WS 事件镜像、历史/管理接口隔离均已完成，SQLite 迁移与 Mock WS 双账户测试通过。MySQL 5.6 使用带上限的复合前缀索引并对旧数据长度预校验；迁移失败/重试及 Main WS 清理隔离均有永久回归测试。**有意的 Main 行为改进**：`UpdateOrderStatus` 在持仓读取失败时保留历史订单，不再误删；其余旧 Main 调用语义保留。真实业务数据库升级及 Main 上线回归仍待验收。详见 [Stage 2 实施记录](币安合约自动带单-Stage2-实施记录.md)。

## Stage 3 — 提取并复用现有 StartTrade 交易流程（不复制策略）

**目标**：形成“一个交易流程 + 两个账户运行实例”，改造前后在等价输入下得到相同开平仓意图。

### 3.1 拆分职责

1. `LoadSharedTradeConfig`：每周期读取一次 `models.Config`、`models.Symbols`，保留当前所有选币参数和“默认使用自定义策略”机制。
2. `SelectTradeCoins`：**行情/指标/评分算法共享，但账户候选池及轮询、冷却独立**。Main 保持 `selectConfiguredCoins(..., scanner.SmartLocalV2ModeTrade)` 的原 Top60 与每轮 5 币行为；Lead 先用本账户已验证的 `leadSymbol` 本地白名单过滤 USDT 永续候选币，再在允许范围内应用相同 Top60 评分/流动性规则和每轮最多 5 币轮询。白名单中不足 60 个则取实际满足规则的数量；不能先选普通 Top60 再过滤 Lead（会漏掉普通 Top60 之外的可带单币）。Lead 使用独立轮询 key 与 `account_id` 冷却，不消费 Main 的轮询进度。
3. `EvaluateEntry(symbol, marketSnapshot, strategyConfig)`：唯一实现 `GetCanLongOrShort`，沿用 current rule JSON、LONG/SHORT 的 hash/判断时点/指标缓存，绝不复制带单版规则。
4. `EvaluateExit(position, strategyConfig)`：唯一实现 `AutoStopOrder` → 止损阈值 + `CanOrderComplete` → 止盈阈值 + `CanOrderComplete`，保持原先优先级和收益率定义；每账户持仓状态分别带入。
5. `AccountTradeLoop(AccountContext)`：每账户独立执行顺序：可管理订单超时处理 → 仓位快照/Ownership 对账 → 所有已有受控仓位的退出检查 → 新开仓闸门与账户风控 → 对选中的币逐一应用共享开仓信号 → 记录和通知。
6. `TradeExecutionAdapter`：对每账户独立做 depth 均价、精度/数量计算、杠杆与逐仓设置、MARKET/LIMIT 下单、pending slot 维护、ClientOrderID、超时撤单、成交同步。
7. `TradeHistoryWriter` + `NotifyAdapter`：通过明确 account_id 记录信号、订单和执行结果。原 `order` 仍保持主交易历史兼容，带单记录不能影响主账户统计。

### 3.1-A Lead 可交易合约白名单本地定时快照（Stage 3 必需）

**目的**：`GET /sapi/v1/copyTrading/futures/leadSymbol` 是带单 Portfolio 专用的只读签名接口，**不能在每次 2 秒交易循环、每个币的策略判断或每次下单前远程调用**。复用 Stage 1 已实现的 `AccountClient.LeadTradingSymbols(ctx)`，额外封装账户独立的 `LeadSymbolCache` / 定时刷新服务，默认 **内存快照，无数据库迁移、不写 app.conf**。

1. **主动刷新频率**：Lead 模拟/真实服务获得有效 Lead 客户端并启动时，尝试**一次初始拉取**；随后建议每 **1 小时**主动刷新一次（正常运行约 24 次/天，不随选币/交易次数增长）。允许后续调整常量，但不把刷新变成每轮 REST 请求。Lead 尚未配置/未启用时不发起无意义的白名单请求。
2. **快照结构**：`allowedSymbols`（规范化、去重后的 `symbol -> metadata` Set/Map）、`fetchedAt`、`expiresAt`、`lastAttemptAt`、`lastErrorCategory`、`ready`；只存白名单和时间戳，**不保存 API Key/Secret**。缓存绑定具体 Lead 账户 Client/Portfolio 身份；密钥替换、身份变化、账户停用时失效，不得使用旧账户白名单。
3. **有效期与异常处理**：建议最大快照有效期 **2 小时**，过期、从未成功加载、响应解析/身份校验异常或返回空白名单时 **禁止 Lead 新开仓（fail-closed）**。刷新短暂网络失败时保留上一次已验证的快照直到过期，后台按 **1/5/15 分钟**间隔限速重试；若鉴权失败、资格被撤销、明确被禁止访问，立即将快照标为不可用并报警，不得继续使用旧白名单。缓存空列表不是允许所有币。
4. **交易热路径**：Lead 候选池生成和下单前白名单门禁只做 O(1) **本地快照查询**，不以缓存未命中触发同步网络请求；未命中只能跳过该币，缺失/过期则暂停 Lead 新开仓。既有受控仓位的对账、止盈止损、超时撤单不因开仓白名单失效而停止；如交易所禁平仓须提示人工处理。
5. **并发与生命周期**：每个 Lead 账户最多一个刷新任务（互斥/singleflight）；调用需超时、context 取消、失败退避；停止/切换凭证必须取消旧刷新并清空旧代的快照，禁止旧请求晚返回覆盖新身份；不给 Main 增加 API 请求，也不影响公共行情 WS。
6. **显示与可观测性（Stage 6 复用）**：记录白名单币数、上次成功/尝试刷新时间、缓存有效/过期状态、上次失败类别和因白名单跳过的候选数；将后续手动“刷新白名单”操作也走同一去重、限速刷新通道。只记录脱敏错误，不泄露签名 URL/凭证。
7. **测试 Gate**：Mock SAPI 验证启动单次获取、1 小时定时刷新、2 秒扫描/逐币门禁 **零额外请求**、并发刷新仅一次、失败退避、过期后禁止新开仓、空列表禁止新开仓、资格/密钥变更立即失效、旧请求不能覆盖新代；两个账户不共用名单/轮询；Main Top60 金样回归不变。

**实现边界**：Stage 3 使用注入式 Fetcher 和可控时钟进行离线测试，不启动真实 Lead 私有交易或擅自获取真实密钥。Stage 4 必须复用缓存的安全判定；Stage 5 接入独立刷新任务与告警；Stage 6 展示缓存状态和受控手动刷新入口。

### 3.2 与旧实现一一对应的规则表

| 现有行为 | 一期要求 |
|---|---|
| `FutureAllowLong` / `FutureAllowShort` | 两账户默认读同一个值；开多/开空独立判断本账户仓位/挂单 |
| `FutureMaxCount` | 每账户独立统计持仓+活动开仓单，不跨账户相加；共享数值 |
| `LossMaxCount` | 每账户独立计算亏损仓位数；共享数值 |
| `FutureBuyTimeout` | 每账户只撤本 owner 的超时开仓单 |
| `FutureOrderType` | 两账户使用同一 MARKET 或 LIMIT，不能悄悄替换 |
| `models.Symbols.Usdt/Leverage/MarginType` | 两账户从相同币种配置读取；数量根据各自交易限制和实际价格执行 |
| `Symbols.Profit/Loss` | 同样的杠杆 ROI、同样的策略退出优先顺序 |
| `OpenStrategyHash` | 策略 hash 共享算法，但 open position record 按账户分别保存 |
| 旧 `insertOpenOrder/insertCloseOrder` | 按 account_id 写入且不互相链接交易记录 |
| 旧 `AutoLossScale/autoTradeToTest` | 不能由 lead 修改 main `config`；此类副作用单独封装，主账户保持现有行为；带单一期仅记录/通知或以明确的本账户熔断实现，禁止静默修改共享配置 |
| 开仓成功后 sleep 30 秒 | 每账户独立冷却；阻塞/休眠绝不挡住另一账户平仓/对账 |

### 3.3 策略一致性测试

- 使用同一份行情、策略配置、账户初始仓位，old main path 与 refactored main path 的交易意图、选币顺序、方向、价格/数量精度、止盈止损、策略 hash 全部一致。
- 同一时刻 `main` 和 `lead` 对**双方均可交易的相同 symbol** 输入相同策略配置/行情时，产生相同策略信号；账户候选池因 Lead 白名单不同可以合法产生不同选币结果，后续仍由独立风控决定是否下单。
- 默认暂停 lead、关闭主 `FutureEnable` 的组合，不可产生误触发；无论一方被暂停，另一方按自身开关执行。
- Existing `service/backtest` 结果不因提取共享逻辑而变化（必要时增加 golden/snapshot tests）。

**验收 Gate 3**：无重复交易策略实现；原主账户全量回归；所有交易语义与旧实现一致；Lead 白名单本地定时缓存（初始获取、单刷新、有效期、失效/失败退避、热路径无重复 API）与独立候选池/轮询测试通过；lead 仍只在 mock Broker 下运行。

**2026-10-09 Stage 3 离线实施结果**：单一 `StartTrade` 共享运行器完成 Main 原交易路径适配，Lead 使用明确的 Mock IO，不接真实账户；共享退出优先级、策略的账户持仓注入、独立白名单 Top60 / 轮询 / 冷却、Lead 白名单 1 小时刷新和 2 小时失效保护已实现并建立定向测试。正式 Lead 定时任务需要安全凭证/身份 Gate 才能在 Stage 5/6 启动；真实 Main 运行回归仍需用户验收。详见 [Stage 3 实施记录](币安合约自动带单-Stage3-实施记录.md)。

**Stage 3 审计 F2/F3/F4/F5 跟进（2026-10-09）**：Main 的 `SelectCoins` / `EnsureConfig` 兼容包装继续保持旧的无错误返回/best-effort 语义，Lead 适配器在选择/保证金杠杆检查失败时应 fail-closed；共享循环的 API source 已区分 main `start_trade` 与 lead `lead_trading`。Stage 3 生产构建不再包含 `leadMockTradeRunner`，Lead 测试运行器的许可只在 `_test.go` 中实现，禁止把 Main 真正的读写钩子冒充 Mock 注入；Stage 4 必须实现经授权的账户级真实执行工厂。LIMIT/方向开关/白名单不可用三条探针已固定为永久单测。**Main 实际业务运行验收仍是 Gate 3 未关闭项**，详见 Stage 3 实施记录 §3-A。

## Stage 4 — 带单账户执行层与账户风控

**目标**：使用独立 Lead Portfolio Key 复用 Stage 3 同一执行链；第一版不需要独立的策略编辑或 AI 流程。

### 4.1 带单模式设置

- 账户角色固定 `lead`；用户填写 Binance Lead Portfolio API Key、Secret 和可选 portfolio 显示名（用于 UI；真实 portfolio 身份须 API 校验，不信任用户填入的名称）。
- `lead_enabled=false` 作为安全默认值。用户启动前必须完成身份白名单验证、余额/仓位快照和可交易能力校验。
- 普通合约交易 `FutureEnable` 与 lead 开关互不改变。Lead 借用共享 `FutureAllowLong/Short` / 选币 / 风控基础值，不能通过设置 lead 启动使主账户总开关变为 true。
- Lead 每个新开仓都须检查：portfolio active、Key 权限、币种在白名单、数量/名义额/价格精度、保证金、杠杆及持仓模式、用户配置阈值、账户仓位冲突和当期冷却。
- 按主系统现有规则使用杠杆和逐仓配置；不自动按所谓带单收益或粉丝数量放大仓位。超出币安专属杠杆/名义额限制时跳过并说明原因。

### 4.2 最少必要的带单补充配置

| 字段 | 默认 | 说明 |
|---|---|---|
| `enabled` | false | 开启实时信号执行 |
| `allow_new_opens` | false | 独立暂停新开仓；不影响已有持仓管理 |
| `account_id` | lead | 固定唯一标识，第一版不允许变更 |
| `max_total_notional_usdt` | 未配置=禁止启动 | 带单总名义风险上限 |
| `max_order_notional_usdt` | 未配置=禁止启动 | 单笔带单名义额上限；不能突破各币原 `Usdt*leverage` |
| `max_daily_realized_loss_usdt` | 未配置=禁止启动 | 当日交易亏损停止新开仓；按账户和明确交易所时间边界计算 |
| `max_drawdown_pct` | 可选，默认不启用 | 额外熔断；后续可配置，不能改变原止损策略 |
| `api_order_budget` | 严格低于官方限制 | per-Key 交易限速，优先为平仓/撤单预留额度 |
| `exposure_rule` | per_account | `FutureMaxCount`、`LossMaxCount` 仅在该账户分别统计 |

风险阈值只做**额外约束**，不是代替或重写当前策略的 TP/SL。阈值及默认值在 Stage 4 结合交易所最小名义额、你的实际 Portfolio 资金核定；禁止凭空自动启用。

### 4.3 执行与保护

- 对外保持现有 `futuresownership.Executor` 的方向/数量验证、ClientOrderID 持久化、unknown response → reconcile、成交后改 managed_qty、手工减仓不回补等原则。
- Lead Broker 独立保存/查询正常单和 Algo STOP/TP，遇币安拒单、超时、网络断开或 WS 缺失先查真实订单；不得用“请求失败”推断“交易所未成交”。
- 新开仓在可交易对筛选后再设置该账户杠杆/逐仓；不能修改 main 同币杠杆/配置缓存。已有仓位模式不符合时拒单或阻塞，严禁自动切换以规避验证。
- 普通账户和 lead 同时获得信号时，各自独立下单，允许一方下单成功、另一方因为资金/限额跳过；所有状态必须分别记录。
- 平仓量不超过 `min(managed_qty, live_account_qty)`；人工加仓不并入受控仓位，人工减仓降低 managed_qty；绝不误平仓。
- 手动“暂停”：停止新开仓，保持风险退出/同步；“停机/无法确认账户”：拒绝不确定写操作并报警，不强制市价清仓；如需“全部平仓”，须后续单独设计明确确认入口。
- Binance 复制跟单人的执行结果不由系统保证；本项目只以带单 Portfolio 自己的交易订单与回报作为真相源。

**验收 Gate 4**：mock / 真实只读检查覆盖 LONG/SHORT/CLOSE_LONG/CLOSE_SHORT、MARKET/LIMIT、TP/SL、拒单/未知提交/部分成交/保证金不足/精度错误；无可绕过风控的带单写入口；真单仍需 Stage 7 人工授权。

## Stage 5 — Lead User Data WS、启动恢复、API 预算与告警

**目标**：带单接入不增加不必要的市场行情订阅，不会因掉线错认仓位或越权重试。

- 每账户 `UserDataStream` 生命周期独立：listenKey、WS generation、有效快照时间、keepalive、重连抖动、停止清理；移除账号不能中断 main WS。
- 新一代 WS 上线前，必须先成功获取该账户完整账户仓位与挂单快照，再授权本地镜像取代 REST；WS reconnect 时按账户完整同步，不跨账户 DELETE。
- 公开 mark price / ticker / kline / market depth 继续共用现有 WS 或行情缓存；账户镜像、余额、授权私有请求不得跨账户共享。
- 确保 lead 与 main 的 `ACCOUNT_UPDATE`/`ORDER_TRADE_UPDATE` 不交叉写 DB；同账户收到乱序/重复事件不能回退已确认成交数量。
- Ownership 定期对账以 `account_id` 为隔离键；重启时先恢复上次 pending / uncertain order 再评估新信号，必要时本账户阻止新开仓。
- REST 调用观测增加 `lead_trading` source、per-account weight/order 数、命中 WS 和 cache 的节约量；统一 IP 全局预算继续生效，账户自身 trade write 有独立限速，HTTP 429/418 降级、按既有三次/两分钟告警约定处理。
- 超过下单限额、WS 断线超期、凭证失效、余额不足、无法对账、保护性平仓失败必须进入告警链路；解除阻塞前重新验证快照。
- 并发压测覆盖“两账户 2s 循环 + 账户 WS + 手动 Reconcile + UI 刷新”；所有循环使用 context cancellation、明确运行状态并禁止遗留 Goroutine。

**验收 Gate 5**：WS 掉线/重连、listenKey 过期、系统重启、限速、请求超时、错序成交回报均不重复开仓、不跨账户误撤/误平；全局 Binance API 权重不因带单接入出现不可控增长。

## Stage 6 — 前后端账户配置、持仓与监控页面

**目标**：用最少页面提供真正可用的带单账户运维能力，沿用现有 Vue3/Element Plus、Beego Controller 和系统通知风格。

### 6.1 后端 API 草案（本系统路由，不是币安原生接口）

| Method | 路由 | 功能 | 约束 |
|---|---|---|---|
| GET | `/futures/lead/status` | 显示连接、身份、WS、是否开单、上次同步及 Gate 信息 | Secret 不可回显 |
| PUT | `/futures/lead/credentials` | 设置或更新专用 Key/Secret | 加密落库，审计，强鉴权；更新前暂停新开仓 |
| DELETE | `/futures/lead/credentials` | 安全禁用密钥 | 有活动仓位/未知单须明确阻断或转人工流程 |
| POST | `/futures/lead/verify` | 只读身份、白名单、账户和 API 能力自检 | 不做真实下单 |
| GET | `/futures/lead/symbols` | Binance 白名单 + 本地启用币种交集 | 展示不合规原因 |
| GET/PUT | `/futures/lead/config` | 共用主交易规则 + lead 专属启停/风控 | 事务校验、防无效参数 |
| POST | `/futures/lead/start` | 启用实时开仓执行 | 必须 Gate 全通过，不能通过 GET 实现 |
| POST | `/futures/lead/pause` | 关闭新开仓，不停止平仓/对账 | 幂等 |
| GET | `/futures/lead/positions` | 仅 lead 账户的实时仓位与 managed 标记 | 不把 unmanaged 当受控 |
| GET | `/futures/lead/orders` | 仅 lead 账户的订单/成交与审计 | account_id 过滤，分页 |
| GET | `/futures/lead/events` | 已下单、跳过理由、异常、风控、对账历史 | 支持 severity/page |
| POST | `/futures/lead/reconcile` | 手动只对 lead 账户做对账 | 不涉及 main 或其它 owner |
| GET | `/futures/lead/metrics` | lead 单独的 realized/unrealized PnL/成交次数/调用预算 | 标注数据来源、币安回报口径 |

**接口原则**：所有敏感修改都必须用户鉴权+请求幂等+安全审计；JSON 明确返回 `account_id=lead`、`readiness`、`enabled`、`allow_new_opens`、`ws_healthy`、`last_reconciled_at`、`blocking_reasons[]`。普通合约主页面/API 保持向后兼容。密钥前端只能展示固定尾号、不可读取 Secret 原文。

### 6.2 前端页面（设计）

菜单：`合约交易 -> 合约带单`，路由建议 `/futures/lead`。主要代码：
- `src/views/futures/leadTrading.vue`（新增）
- `src/api/leadTrading.ts`（新增）
- `src/router/modules/futures.ts`（加一项）
- `locales/zh-CN.yaml`、`locales/en.yaml`（双语）
- 复用现有订单表、币种展示、系统看板及通知组件；不重写已有页面。

页面按照独立 Card 排布，不在 Card 标题上放歧义总开关：

1. **账户状态 Card**：是否具备带单资格、Portfolio 名称、连接/权限、可用余额、权益、WS 时延/同步时间、IP 权限和 API 用量。
2. **凭证设置 Card**：API Key/Secret 安全录入、保存、验证、轮换、密钥权限检查、币安官方创建指引；不显示 Secret 原文。
3. **自动交易 Card**：明确“带单启停”和“仅暂停新开仓”两类操作；展示继承的共享配置（允许多/空、MARKET/LIMIT、Top60、最大持仓、亏损仓位、各币策略），提供跳到现有配置中心/币种页面的链接，**不复制编辑面板**。
4. **账户风险 Card**：带单总/单笔名义上限、当日亏损熔断、可选最大回撤、订单预算；显示阻断原因、当前消费额度。
5. **当前持仓与挂单 Card**：Symbol、side、managed qty、actual qty、开仓均价、Mark Price、杠杆、未实现 PnL、Owner、last reconcile；区分手动仓位。
6. **交易与诊断 Card**：下单/平仓/拒单/跳过、未知状态修复、API 错误、WS 重连；支持时间范围、币种、状态筛选。

前端绝不提供“清空所有账户仓位”的无保护按钮；Stage 6 不实现 Binance 上分成比例修改或 Copy Trader 的管理操作。

**验收 Gate 6**：常见响应空态、密钥缺失、连接失败、WS 降级、开启被拒、暂停有效、订单状态变化都显示清楚；页面只能操作 lead，不回显 Secret；中文/英文检查；构建 lint/typecheck 无新增问题。

## Stage 7 — 集成验收、受控小额验证、上线与回滚

### 7.1 必跑测试矩阵

| 分类 | 核心场景 | 验收 |
|---|---|---|
| 主账户回归 | 无 Lead Key、Lead disabled；主合约开平仓 | 与实施前策略意图、风控结果一致 |
| 账户隔离 | main 和 lead 同币同方向均有仓位 | 两边 OrderID/ClientID/WS/缓存/统计独立 |
| 并行运行 | main ON/lead OFF；main OFF/lead ON；两者 ON；两者 OFF | 仅启用账号开新单，暂停不误影响退出 |
| LONG 完整流程 | 开多、策略平多、止盈、止损 | 方向/数量/订单归属正确 |
| SHORT 完整流程 | 开空、策略平空、止盈、止损 | 方向/数量/订单归属正确 |
| 成交生命周期 | MARKET、LIMIT、未成交、部分成交、超时撤单 | 只以真实成交修改 managed_qty |
| 订单可靠性 | 下单超时但 Binance 已成交、重复 client ID、REST 429/418 | unknown 首先查单，不重复提交 |
| 人工干预 | 手工增仓/减仓/平仓、其它 owner 仓位 | 不自动认领、不平超过 managed_qty |
| 风控 | 余额不足、禁用币、杠杆超限、最大持仓/亏损数/日亏损 | 拒绝新开仓并记录明确原因 |
| WS 与快照 | WS 丢包/断线/重连、乱序、服务重启 | 先对账再继续，主账户数据不被 lead 清理 |
| 条件单 | STOP/TAKE_PROFIT Algo、触发与撤单 | API 实测支持且受控；否则禁用该能力 |
| API 预算 | 双账户并行2s调度、订单专属 20/10s、全局 IP 限制 | 读多写少、可观测且不超预算 |
| 历史与回测 | 主账户旧订单回填、旧接口、策略研究测试 | 结果/数据无损、回测不改变 |
| 安全 | API Secret、未授权写请求、过期登录 | Secret 不泄露、请求被拦截、留审计 |

### 7.2 测试环境与真仓权限

1. 默认纯单元测试和 fake Binance Broker，覆盖完整流程与故障；此阶段无 Binance 真实交易。
2. 用主 Futures Testnet 验证底层 client/ownership 语义，但不能以此证明带单专用 Key 的能力。
3. 真实 Lead Portfolio 先只读连接检查，人工核对 Binance 页面上的资金与无持仓状态。
4. **仅在用户明确授权实际下单后** 执行最小合法数量的真实仓位端到端交易；下单前再次核对 Key、IP、白名单、杠杆、资金与限额。
5. 按小额逐项验证 LONG/SHORT/CLOSE 与订单查询、恢复；若 API 权限或订单模式不兼容，立即回退到只读/暂停，并记录 blocker。
6. 至少跑一次重启恢复和 WS 中断后的对账；平台带单结果以 Binance Portfolio 页面核对，系统只保证带单账户自身的执行与一致性。
7. 上线初期 lead 保持默认 paused，人工开启；有异常优先只停新开仓，保留退出/对账，有严重状态不确定时 fail-closed 并报警。

### 7.3 上线回滚

- 正式部署仅在所有 Gate 通过、完整备份和人工授权后进行；默认 lead disabled，旧 main 保持原启动流程。
- 回滚优先关闭 lead 新开仓，保留读/对账/退出功能与 DB 数据；不得通过降级旧 DB schema 丢掉新订单或 Ownership。
- 如发现新版本影响 main 的 API Key、仓位、订单对账，阻断新版本主账户写操作，采用预先测试的兼容回滚程序；不得把数据库 schema 恢复到缺 account_id 的历史版本强行运行。
- 验收后记录实际 Binance Portfolio 的真实成交、订单 id、仓位差异、平台是否显示带单状态；不承诺能控制复制者实际成交或跟随情况。

## 附录 A — 建议最终数据结构（Stage 2/4 分步落地）

| 模型/表 | 字段建议 | 约束 |
|---|---|---|
| `lead_trading_accounts` | `id`、`account_id`、`portfolio_label`、`credential_ciphertext`、`credential_key_ref`、`verification_state`、`last_verified_at`、`created_at`、`updated_at` | 一期只允许 lead 一条，密钥加密/日志脱敏 |
| `lead_trading_config` | `account_id`、`enabled`、`allow_new_opens`、`max_total_notional_usdt`、`max_order_notional_usdt`、`max_daily_realized_loss_usdt`、`max_drawdown_pct`、`updated_at` | 默认 disabled、无上限配置禁止启动 |
| `lead_trading_events` | `id`、`account_id`、`signal_id`、`symbol`、`side`、`intent`、`status`、`reason_code`、`client_order_id`、`occurred_at` | 唯一事件键/索引、无凭证、可限期清理 |
| `futures_managed_positions` | **扩展** `account_id`；保留 owner、managed_qty、status 等 | 所有查询/冲突检查账户隔离 |
| `futures_managed_orders` | **扩展** `account_id`；保留 ClientOrderID/filled_qty 等 | 未确认提交不盲重试 |
| `futures_positions`、`futures_orders` | **扩展** `account_id` | 全量同步和 WS 增量更新都账户隔离 |
| `order`（旧历史表） | **扩展** `account_id`（或新增严格映射，Stage 2 核验） | 不作为 ownership 真相源；主数据 main 回填 |

数据库**不重复存储行情、策略 JSON、选币结果**，也不另建独立的 lead strategy 表。具体加索引前要核对已有 unique/index，避免非破坏性迁移中临时冲突。

## 附录 B — 关键接口与技术决策

### B.1 AccountContext（设计契约，非当前代码）

`AccountContext{AccountID, Kind, Broker, AccountReader, OrderReader, PositionMirror, WSStatus, APIBudget, Ownership, RiskState}`。

AccountID 必须显式贯穿交易和签名 IO 层。Main 的兼容封装只在现有 main 路径允许；lead 禁止调用无账户参数的 `binance.GetPositionContext()`、`binance.CreateOwnedOrder()` 等全局快捷入口。

### B.2 订单流水（预期）

1. 读取共享策略与行情快照；生成 `signal_id`（币种、方向、策略 hash、信号观测时间/具体触发 bar 的稳定标识）。
2. 对当前 AccountContext 检查“允许开新仓”、本账户仓位/订单、币安白名单/交易精度、杠杆/保证金、持仓数/亏损数、账户级额度与流控。
3. 以 `account_id + signal_id + side + intent` 创建持久化意图/幂等键；claim ownership，再发送带账户签名的订单。
4. 交易所确认实际成交后更新 `futures_managed_orders` → `futures_managed_positions`；历史 `order` 表继续作为展示兼容。
5. 无法确定交易所是否受理时标记 `reconcile_required`，先用本账户 `ClientOrderID` 查询并修复，**禁止自动重新创建订单**。
6. 退出信号仍由唯一 `EvaluateExit` 产生；使用账户和 Owner 双重边界，按实际本账户仓位做平仓量夹取，保证不影响外部仓位。

### B.3 签名账户 vs 公开行情

| 类别 | 复用策略 | 是否按账户隔离 |
|---|---|---|
| Binance Kline、ticker、mark price、depth | 当前行情路径共享 | 否 |
| 指标计算缓存、Strategy JSON、Scanner | 当前规则共享 | 否 |
| Balance、Position Risk、Open Orders、User Trades | 实例化账户客户端 | 是 |
| Order Create/Cancel/Query/Algo | AccountBroker 和 Ownership | 是 |
| User Data WS、ListenKey、generation、WS mirror | 每账户独立 | 是 |
| 杠杆、保证金模式、账户风险计数 | 每账户独立写入和读取 | 是 |
| API 限流、风控/订单恢复、审计 | 分账户观测 + IP 全局预算 | 是 |

**特别提醒**：不能把共享行情缓存当作共享账户缓存；也不能仅靠给 `source_ref` 加一个 “lead:” 前缀替代数据库 `account_id`。

### B.4 代码审计发现的潜在兼容问题

- `feature/feature.go` 的 `insertCloseOrder` 会调用 `AutoLossScale` 和 `autoTradeToTest`；必须避免 lead 写 main 的全局配置。带单沿用相同决策，但全局配置变更不可穿透账户边界。
- `feature/api/binance/index.go` 的 `futuresClient` 与签名时间差/WS generation 是单例；独立 Key 不能共享。
- `feature/feature_userdata.go` 的全量同步以及 `WsUserData` 目前直接操作统一表；同步 SQL 和清理必须增加账户过滤。
- `feature/feature.go` 历史 `order` 关联通过 symbol、amount、positionSide 和 orderId，不是严格的成交账本；Stage 2 要防止不同账户相互关联。
- `service/futuresownership/service.go` 的 `normalizeOwner` 是静态 owner 列表；**优先复用 `owner=auto_strategy` 并以 `account_id` 区分**，不要为每个账户制造一个 owner 列表、复制整套逻辑。
- `service/futuresownership/reconcile.go::ReconcileAll` 当前遍历多 owner，但没有 account_id；Stage 2 主账户全量回归时必须覆盖 agent_trade 等未纳入新带单交易的旧 owner。
- `feature/feature.go::StartTrade` 的 30s 开仓后休眠和错误 sleep 是旧主循环的行为；改为每账户独立退避并以 golden tests 证明开仓节奏及交易意图不被意外改变。
- `UpdateSymbolTradeInfoContext`、`GetTransformPositionsContext`、`getTransformOpenOrdersContext` 使用全局账户上下文；需要全部改为注入账户，否则即使 Lead Broker 独立，下单前的杠杆或仓位判断仍可能打到 main。
- `models.Config`/`Symbols` 由现有配置中心和每币设置控制；Lead 第一版只读共享交易参数，不在新页面添加第二套重复编辑逻辑。

## 附录 C — 按阶段执行卡与合并准则

| Stage | 主要交付物 | 开始条件 | 验收关键点 |
|---|---|---|---|
| 0 | API 能力矩阵、风险清单、调用路径 | 无 | 确认不可行项和需实测项 |
| 1 | AccountContext + 双实例 Binance Adapter | Stage 0 静态设计明确 | Fake 客户端隔离，无资金操作 |
| 2 | DB 迁移 + Account-scoped Ownership + WS mirror schema | Stage 1 基础接口稳定 | main 历史数据无损、双账户不会串账 |
| 3 | 共用 Strategy/Trade Cycle 重构 + golden tests | Stage 2 可靠 | 主账户行为与旧版本一致 |
| 4 | Lead Broker + 风控 + 独立启停 | Stage 3 通过 | Lead fake 下单完整，不影响 main |
| 5 | Lead WS/Reconcile/API budget/告警 | Stage 4 稳定 | 断线/超时/重启没有重复单 |
| 6 | 前端 Card 页面 + API + i18n | Stage 5 API 行为稳定 | UI 状态、配置、错误处理完整 |
| 7 | 整体回归 + 真实只读 + 小额受控验证 | Stage 0–6 Gate 通过 | 人工授权真单、记录 Binance 成交 |

每个 Stage 完成时统一交付：
- 具体修改/新增的文件与原因，是否存在既有未提交改动冲突。
- 本地构建、Go 单元测试、竞态/并发测试（相关时）、前端 lint/typecheck/build 结果。
- 对策略结果一致性、主账户兼容、资金安全影响的说明。
- 数据迁移执行/未执行记录、服务器进程/3333 占用检查。
- 已完成的 Gate、遗留 blocker、是否建议进入下一阶段。
- **未经要求不自动 commit/push；用户确认后继续下一 Stage。**

## 附录 D — 官方文档与限制（实施前再核对）

1. [币安：为合约带单组合创建 API Key](https://www.binance.com/en-AE/support/faq/detail/2bec848b904b422197ce121d0925f20b)：创建方式、只支持 USDT、不允许 Multi-Assets Mode、专用 Key 上限和下单限速。
2. [币安：合约带单员操作与风险规则](https://www.binance.com/en-IA/support/faq/detail/6acfb4c1f50c4db1b9e915181ff31a4c)：传统带单组合、最低金额、允许交易范围、流动性风险限制、Lead Portfolio 操作。
3. [币安：Copy Trading API](https://developers.binance.com/en/docs/catalog/advanced-trading-copy-trading/api/rest-api/future-copy-trading)：`/sapi/v1/copyTrading/futures/userStatus`、`/sapi/v1/copyTrading/futures/leadSymbol`；不等于完整的下单端点集合。
4. [币安 Go 官方 Connector 的 Copy Trading 参考](https://pkg.go.dev/github.com/binance/binance-connector-go/clients/copytrading/src/restapi)：可作为 SAPI 只读签名与参数参考，不强制替换项目既有 `adshao/go-binance` 交易库。

项目使用者负责确认所在地区具备 Binance Futures Copy Trading 产品权限、Portfolio 资格和 API Key 可用性。由于专用 Key 的某些能力可能与普通合约 Key 不同，Stage 0 的“待实测”项不能默认记为“已支持”。

## 实施记录（由每一阶段实际开发后追加）

| Stage | 状态 | 开始/完成日期 | Gate 与测试 | 变更文件/说明 |
|---|---|---|---|---|
| 0 | 静态审计完成；真实账户能力待验证 | 2026-10-09 | Gate 0-DOC 通过；Gate 0-LIVE 阻塞；未执行测试网/真单 | 新增 `币安合约自动带单-Stage0-核查结果.md`；仅文档审计，未修改交易代码/DB |
| 1 | 离线完成；Lead 实盘权限待验证 | 2026-10-09 | API/缓存/交易配置/Broker/SAPI Mock 通过；竞态测试通过；无真单 | `account_client.go`、`account_state.go`、`lead_order_limiter.go`、`lead_readonly.go`、Broker 与相关测试；见 Stage 1 实施记录 |
| 2 | 离线实现完成；真实 DB 迁移待执行 | 2026-10-09 | Schema 19 SQLite 迁移/幂等、Owner 双账户、WS 镜像模拟、相关后端 `-race` 通过；未实际升级 DB 或真单 | 新增 `account_scope_migration.go`、`futuresownership/account.go`、`user_data_mirror.go`、隔离测试；调整模型、历史接口及统计；见 Stage 2 实施记录 |
| 3 | 未开始 | — | — | — |
| 4 | 未开始 | — | — | — |
| 5 | 未开始 | — | — | — |
| 6 | 未开始 | — | — | — |
| 7 | 未开始 | — | — | — |

