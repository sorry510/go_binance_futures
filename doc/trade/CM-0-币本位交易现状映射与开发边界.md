# CM-0 · 币本位交易现状映射与开发边界

> 检查日期：2026-11-17（北京时间）  
> 分支：`feat/m-coin-futures`；当前 Schema：**v26**；阶段：**CM-0 已完成静态分析**  
> 范围：只读源码检查与币安官方文档核对；**没有连接实盘交易、没有执行数据库迁移、没有修改交易执行逻辑**。

## 1. CM-0 结论

**可复刻，但不能直接将 U 本位合约或现货的 Client、订单类型和盈亏算法切换成币本位。**

- **复刻 U 本位的产品形态和策略语义**：四个交易菜单页面、LONG/SHORT/CLOSE_*、逐全仓、杠杆、止盈止损和币种启用。
- **复刻现货 06 的业务流程**：2 秒轮询、全局＋单币启停、已持仓退出优先、Managed 订单与 Fill 去重、前端构建到 backend/static。
- **COIN-M 专属适配**：DAPI 客户端和签名用户流、张数与 contractSize、币本位逆向盈亏/资金资产、Algo 条件单与真实 userTrades、独立币本位数据表。
- **不做大型框架**：不建立第三个通用交易引擎、审批/工作流、额外全局配置卡片或额外币种开关字段。

## 2. 当前代码映射（静态证据）

| 功能 | 当前实现与源码 | CM 处理 | 现状 |
| --- | --- | --- | --- |
| CM 行情模型 | `models/delivery_symbols.go`：`delivery_symbols` 含 symbol、enable、price、leverage、marginType、tickSize、stepSize、technology、strategy 等 | 复用模型，补充 contractSize、marginAsset、contractType、budget_usd 等必需数据；每币开关直接复用 `enable` | **部分可复用** |
| 币本位币种发现与精度 | `feature/delivery.go:UpdateDeliverySymbolsTradePrecision`：只插入 `_PERP`；但没有保存合约面值/保证金币种，旧币种新插入时默认 `usdt=10` | 只收录 exchangeInfo 的 PERPETUAL 且 TRADING，新增真正的张数校验与 USD 预算，绝不把旧 `usdt` 当真实开仓预算 | **需适配** |
| CM 行情 REST / 市场 WS | `feature/api/binance/delivery.go`：ExchangeInfo、Account、all ticker WS；`main.go` 未启动 `UpdateDeliveryCoinByWs`，币种定期同步代码被注释 | CM-2 接入并控制；行情数据与 DAPI 账户交易环境一致 | **未完整启用** |
| CM SDK Client | `feature/api/binance/index.go`：`delivery.NewClient(api_key,api_secret)`，`delivery` 被标为 mainnet；无条件受现有 `testnet` 设置保护 | CM-2 建立独立生产/Demo/Testnet 选择，**testnet 下严禁任何 COIN-M 签名调用回落主网** | **阻断实盘执行** |
| U 本位主交易 | `feature/account_trade_cycle.go`：`mainTradeRunner` 与 `runAccountTradeCycle`，订单接口使用 `futures.Client` 和 FAPI 数据结构，`AccountID` 仅识别 Main/Lead | 复用策略出入场规则/优先级，不直接扩展其账户枚举或向现有 FAPI Submit 注入 CM | **仅复用业务规则** |
| USDT-M Ownership / Fill | `service/futuresownership`、`service/tradefills`、`models/futures_managed.go`、`models/futures_trade_fill.go` | 借鉴 position_id、clientOrderId 和 Fill 去重规则，建立独立 CM 表，禁止向 Main/Lead FAPI Ledger 混写 | **逻辑参考** |
| 现货 06 执行器 | `service/spotexecution`、`service/spotledger`、`feature/spot_quant_runner.go` | 参考 pending/Fill/恢复/关闭后仍允许退出的处理，不使用 BUY/SELL 现货库存算法计算合约仓位 | **流程参考** |
| 调度 | `main.go`：U 本位 `StartTrade` 和 Spot `StartSpotQuantTrade` 都使用 `loopRun(..., 2*time.Second)` | CM-6 接入既有调度周期；**允许独立受保护回调，不增加新的调度中心** | **可复用** |
| 全局配置 | `models/tableStruct.go` 的 `Config` 已有 `delivery_enable`、`spot_quant_enable`；`controllers/index.go` 仅把历史 `delivery_enable` 返回为 `tradeDeliveryEnable` | 根据已确认的 CM 方案，新增 **`coin_m_quant_enable`，默认 0**；配置中心只新增这一处开关，不让旧 `delivery_enable` 意外授权自动开仓 | **CM-1 待做** |
| 每币开关 | `models/delivery_symbols.go` 已有 `enable` | **只复用这个字段**，币本位交易列表操作它；不新增 `coin_m_quant_enabled` | **已有字段** |
| U 本位页面 | 前端 `src/views/futures/{symbol,account2,strategyTemplate}.vue`、`src/views/ai/controlledTrade.vue`；`src/router/modules/futures.ts` | 以 U 本位的布局、字段和交互为蓝本复制四页 | **UI 基线** |
| 现货四页 | 前端 `src/views/spotQuant/` 和 `src/router/modules/spotQuant.ts` | 以现货最终四入口菜单、独立数据源以及独立 API 实现为基线 | **模块结构基线** |
| 配置中心卡片 | 前端 `src/views/dashboard/configShow.vue`：Spot Quant 卡片仅有单一 `el-switch` | 增加「币本位合约交易」卡片且**仅一个 `el-switch`**，其他参数在每币页面 | **直接复刻** |
| 数据库 Schema | `appversion/version.go` 为 26，`command/db_update.go` / `command/sql/version` 采用 `sync db` | CM-1 如未被其他开发占用使用 v27；币本位账本独立、Decimal 精度及唯一索引沿用现货 v25 SQL 迁移惯例 | **版本待确认** |

**额外发现**：旧 `delivery_symbols` 可能已存在 `enable=1` 的历史行。CM-1 应只允许 **v27 显式初始化后的新功能开仓授权**：新增总开关默认 0，正式启用前必须核对币种开关，不得因旧数据或进程重启自动开仓。不能简单把 `delivery_enable` 旧值当做新功能的开仓许可。

## 3. 本地 SDK v2.8.12 能力矩阵（已检查源码）

当前依赖：`github.com/adshao/go-binance/v2 v2.8.12`，本地包 `delivery`。

| DAPI 功能 | SDK 证据 | CM 决策 |
| --- | --- | --- |
| `GET /dapi/v1/exchangeInfo` | `delivery.NewExchangeInfoService`；ExchangeInfoSymbol 包含 ContractSize、MarginAsset、ContractType、ContractStatus、Filters | 可用，补全校验与持久化 |
| 普通 `POST /dapi/v1/order` | `NewCreateOrderService().Do()` 使用 `/dapi/v1/order` | 可用于 MARKET/LIMIT；必须经过自有下单门禁、价格/张数/仓位模式检查 |
| 查单/撤单 | `NewGetOrderService` / `NewCancelOrderService` / `NewListOpenOrdersService` | 可复用，仍需处理真实 Fill 去重 |
| 账户/持仓/杠杆 | `NewGetAccountService`、`NewGetBalanceService`、`NewGetPositionRiskService`、`NewGetPositionModeService`、`NewChangeLeverageService` | 读取与配置可复用；**不调用改变全局持仓模式的接口** |
| 私有 WS | `NewStartUserStreamService` / `NewKeepaliveUserStreamService`、`delivery.WsUserDataServe` | 可复用流生命周期基础，需验证目前 `ORDER_TRADE_UPDATE` 数据结构及漏事件 REST 补证 |
| 真实逐笔成交 | 当前 `delivery.Client` 方法清单**没有** `NewAccountTradeListService` / `GET /dapi/v1/userTrades` 的完整封装 | **CM-5 必须适配**签名查询并保存准确 tradeId、PnL、commission/asset |
| 2026 条件单 Algo | 当前 `delivery.Client` 不提供 `NewCreateAlgoOrderService`、查询/撤销 Algo | **CM-5 必须适配** `/dapi/v1/algoOrder`；不能将旧 STOP/TP 发送到 `/dapi/v1/order` |
| Algo WS 事件 | `delivery.WsUserDataEvent` 只定义账户更新和普通订单更新结构，未发现新 `ALGO_UPDATE` 类型的专门处理 | 需检查并补当前官方事件，独立管理 Algo ID 和普通订单 ID |
| Demo/Testnet | SDK 提供独立的 `delivery.UseTestnet` / `UseDemo`、BaseApiDemoURL 等；**当前项目没有设置 CM 的这些选项** | CM-2 只创建与明确选择环境匹配的 Client/WS，签名请求严禁跨环境 |

**结论**：SDK 的基础市场数据和普通订单 API 可以复用，**不能认为仅靠 v2.8.12 就足以完成 2026 COIN-M 保护单、真实 Fill 和环境安全**。后续应优先在现有 `delivery` 能力上做窄适配；不要无理由升级整个 Binance SDK 影响现有 U 本位与现货链路。

## 4. 官方 COIN-M 规则与不可省略的差异

1. **合约张数**：普通新订单 `quantity` 的单位为张。必须从 exchangeInfo 读取该 symbol 的 `contractSize`、Filters 和 marginAsset，不能照搬线性合约“币数量”。
2. **逆向盈亏**：对于 USD 反向合约，LONG 的币 PnL = 张数 × 面值 × (1/开仓价 − 1/平仓/标记价)，SHORT 则交换两个倒数。盈亏、手续费与保证金按结算资产计，不把 BTC 与 ETH 盈亏直接求和。
3. **共享持仓模式**：官方 COIN-M 交易接口说明 UM 与 CM 共享 `dualSidePosition`。该设置是账户级的，不能因为 CM 模块上线就自动更改。
4. **条件单迁移**：新版交易接口明确：STOP/STOP_MARKET/TAKE_PROFIT/TAKE_PROFIT_MARKET/TRAILING_STOP_MARKET 通过旧 `/dapi/v1/order` 会返回 `-4120`，需要新版 `/dapi/v1/algoOrder`。
5. **成交历史窗口**：`GET /dapi/v1/userTrades` 官方说明仅支持最近三个月历史，单次时间窗口最多七天。CM 执行阶段必须及时保存本地真实 Fill，不可依赖未来无限期从交易所补齐所有历史。
6. **保护性退出**：关闭总开关或每币开关仅阻止新开仓；存在 Managed 持仓时继续尝试退出/对账。若私有 WS 不可信、账本归属不明或余额不足，拒绝不安全订单；只读查询可用于判断状态。
7. **账户隔离**：Main U 本位、Lead、Spot、COIN-M 的不同产品标识、币种钱包、订单编号、WS 更新不能混写或交叉引用。当前已有 U 本位账户路径不应因为 CM 添加而变化。

官方参考（2026-11-17 核对）：
- COIN-M 交易与持仓模式：https://developers.binance.com/en/docs/catalog/core-trading-derivatives-trading-coin-m-futures/api/rest-api/trade
- COIN-M 交易对、合约面值与过滤器：https://developers.binance.com/zh-CN/docs/catalog/core-trading-derivatives-trading-coin-m-futures/api/rest-api/market-data
- 2026 COIN-M / UM 整合与条件单变更：https://developers.binance.com/zh-CN/docs/products/derivatives-trading-coin-futures/change-log
- COIN-M 私有流：https://developers.binance.com/en/docs/products/derivatives-trading-coin-futures/user-data-streams

## 5. CM-1 接口/数据边界（交给下一阶段）

CM-1 **仅建立 Schema 与默认关闭的配置读写**，不能顺带接入实盘下单或重构现有 FAPI/Spot 执行器。

- `config.coin_m_quant_enable`：唯一新增全局开关，默认 0；不要使用历史 `config.delivery_enable` 授权开仓。
- `delivery_symbols.enable`：唯一每币开关，在币种列表呈现；不能自动继承旧 symbol 数据作为已授权策略下单的证据。注意存量行的状态核实与首次启用时提示。
- `delivery_symbols`：计划补足 `contract_size`、`margin_asset`、`contract_type`、`budget_usd`；`budget_usd` 默认 0。不要复用旧 `usdt=10` 默认值触发买入。
- 独立四表：`coin_m_managed_positions`、`coin_m_managed_orders`、`coin_m_trade_fills`、`coin_m_strategy_templates`，采用 DECIMAL、显式 position_id 关联及订单/成交唯一键。
- 所有新表初始为空、全局开关默认关闭，无历史订单迁移。数据库迁移目标版本暂定 **v27**，开始前检查当前 `appversion.DatabaseSchemaVersion`，避免并行开发版本冲突。
- 下一阶段结束要做独立 Code Review，覆盖：升级幂等、MySQL/SQLite schema、零记录默认禁用、未修改 U 本位/Spot/Lead 表及用户配置、不写生产库。
- **CM-2** 完成测试网与主网强隔离、产品行情与合约面值校验后，**CM-5** 再打通 DAPI 普通订单 / Algo / Fill；**CM-6** 最后接入自动调度。任何此前阶段均不得通过打开全局开关直接下单。

## 6. CM-0 验收记录

| 检查项 | 结果 |
| --- | --- |
| 盘点 U 本位完整策略与调度路径 | **PASS（静态检查）** |
| 盘点现货 06 双开关、两秒调度与独立账本 | **PASS（静态检查）** |
| 盘点已有 CM 模型、行情、Client、WS 的实际接线 | **PASS（静态检查）** |
| SDK `v2.8.12` 基本 API、userTrades、Algo 缺口确认 | **PASS（本地源码）** |
| 最新 DAPI 张数、持仓模式、Algo 与 userTrades 边界 | **PASS（官方文档）** |
| 确认配置中心单开关和 `delivery_symbols.enable` 方案 | **PASS（与既定方案一致）** |
| 修改交易代码 / 生产 DB / 部署 | **NOT APPLICABLE；本阶段禁止** |
| 真实 Binance 凭证验证、WS 实连、下单回归 | **NOT TESTED；留待 CM-2/CM-5** |

**CM-0 结论：可以进入 CM-1。** 本结论仅说明代码边界已分析完毕，不代表数据库已升级、交易已实现或实盘验证通过。
