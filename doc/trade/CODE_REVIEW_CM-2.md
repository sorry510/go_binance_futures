# CODE_REVIEW_CM-2 · 币本位 DAPI、行情、WebSocket 与环境隔离

> 日期：2026-11-17；分支：`feat/m-coin-futures`；阶段：CM-2；源码 Schema 仍为 v27。
>
> **本阶段没有实现任何 COIN-M 真实下单方法，没有使用币安签名接口执行交易，没有运行生产 DB 迁移或替换运行中的二进制。**

## 1. 实施清单

| 模块 | 实际文件 | CM-2 实现 |
| --- | --- | --- |
| 网络环境隔离 | `feature/api/binance/coin_m_environment.go`、`index.go` | COIN-M REST/WS 固定白名单：mainnet/testnet/demo。继承 `binance.testnet` 为安全基础；禁止主网模式与 testnet 配置互相覆盖。可选 `BINANCE_COIN_M_ENVIRONMENT` 仅允许三种值，非法配置 fail closed。 |
| REST 主机与产品隔离 | `coin_m_environment.go` | 独立的 `coinMRESTGuard`，只允许当前选定 DAPI 主机的 `/dapi/` 和 `/futures/data/` 请求；拦截 FAPI、Spot、SAPI、跨网络请求及 HTTP 重定向到别的主机。 |
| 各环境凭证 | `coin_m_environment.go`、`index.go` | mainnet 仅用主网凭证，testnet 仅用现有测试网凭证；demo 仅用独立 Demo 凭证；未配置完整凭证时仍可访问公开市场接口，但所有 CM 私有账户查询会拒绝，不回退主网。 |
| DAPI 账户只读 | `feature/api/binance/delivery.go` | 新增 `GetDeliveryAccountContext`、`GetDeliveryBalanceContext`、`GetDeliveryPositionsContext`、`GetDeliveryPositionModeContext`，全部显式检查对应环境及完整凭证；不提供更改持仓模式接口。删除旧账户信息全量日志打印。 |
| COIN-M ExchangeInfo | `feature/api/binance/delivery.go` | `GetDeliveryExchangeInfoContext` 支持 Context、公开无需凭证；动态限额传递实际 environment。 |
| COIN-M 合约筛选/精度 | `feature/delivery.go` | 只接受 `contractType=PERPETUAL`、`contractStatus=TRADING`、`quoteAsset=USD` 且 `_PERP` 的币本位合约；要求 contractSize、marginAsset、PRICE_FILTER、LOT_SIZE、MARKET_LOT_SIZE 完整且有效。 |
| 张数与价格过滤 | `feature/delivery.go` | `ValidateCoinMOrderFilters` 通过 `math/big.Rat` 精确校验张数、最小/最大数量、步长及 LIMIT 价格 tick；仅纯函数，没有下单。CM-5 必须每次开仓前拉取/校验最新交易所过滤器。 |
| 币种元数据同步 | `feature/delivery.go`、`main.go` | 在现有 12 小时循环启动时及周期内同步合约元数据（20 秒超时）；使用同一事务写入，已存在币种只改元数据，保留原 enable、usdt、strategy、technology、leverage；新币种 enable=0、usdt=0、budget_usd 由 v27 表默认=0。 |
| 币本位行情 WS | `feature/api/binance/delivery.go`、`coin_m_ticker.go`、`main.go` | 沿用现有 `WithDeliveryWS` 代理，选定 COIN-M 网络 WS；仅行情 `!ticker@arr`，断线 5 秒重连；只更新已知 PERPETUAL 市场价格，不改变策略/币种开关及订单；跟踪最新市场事件。 |
| REST 限额统计 | `delivery.go`、现有 `service/binanceapiusage` | 新 DAPI 客户端复用同一个 REST 预算协调器、429/418 限流处理，但 `Product=delivery` 且 `Environment=mainnet/testnet/demo` 按网络独立统计；不误标为 mainnet。 |

## 2. 环境配置规则

配置中心仍只有 **`config.coin_m_quant_enable` 一个全局开关**（CM-3 实现 UI）；**每币 `delivery_symbols.enable` 原样复用**。以下是部署环境配置，绝不添加到配置中心：

| 配置 | 作用 |
| --- | --- |
| `binance.testnet=false`，未设置 `BINANCE_COIN_M_ENVIRONMENT` | COIN-M 主网 REST `https://dapi.binance.com`、WS `wss://dstream.binance.com/ws`；使用现有主网凭证 |
| `binance.testnet=true`，未设置 `BINANCE_COIN_M_ENVIRONMENT` | COIN-M Testnet REST `https://testnet.binancefuture.com`、WS `wss://dstream.binancefuture.com/ws`；使用现有 `testnet_api_key/testnet_api_secret` |
| `binance.testnet=true` 且环境变量 `BINANCE_COIN_M_ENVIRONMENT=demo` | Demo REST `https://demo-dapi.binance.com`、WS 使用 SDK Demo 地址；仅允许专用环境变量 `BINANCE_COIN_M_DEMO_API_KEY`、`BINANCE_COIN_M_DEMO_API_SECRET`，没有凭证时不允许签名账户读取 |
| `binance.testnet=false` 且强制 `testnet/demo`；或反之 `testnet=true` 强制 `mainnet` | 配置错误；COIN-M 客户端拒绝初始化；**没有回落主网** |

`BINANCE_COIN_M_ENVIRONMENT` 也可显式为 `mainnet` / `testnet`，但不能越过上面的 `binance.testnet` 限制。不更改 `app.conf` 中原有数据库连接及其他应用配置。COIN-M Demo/Testnet 只是单独网络选择，不代表获得真实交易开关权限。

## 3. 验证结果（本次实际执行）

| 检查 | 结果 |
| --- | --- |
| 主网、Testnet、Demo 模式选择和凭证来源测试 | PASS |
| 禁止不同网络 REST/FAPI/Spot/SAPI 请求及重定向到主网 | PASS |
| 无环境匹配私有 API 凭证时，DAPI 账户调用 fail closed | PASS |
| 本地 HTTP Mock `GET /dapi/v1/exchangeInfo` | PASS |
| 合约张数、LOT/MARKET_LOT/PRICE 过滤器（含缺失/异常输入） | PASS |
| SQLite 测试库原配置保留、筛选有效永续、新币种默认禁用与重复刷新 | PASS |
| SQLite Mock WS ticker 只更新合法合约价格，不改 enable/budget/strategy | PASS |
| 模拟的 429、418 在不同 COIN-M 环境下分别记录（不联网） | PASS |
| CM-2 定向 Go 测试，`-count=5 -p=2` | PASS |
| 12 个相关 Go 包完整测试，`-count=1 -p=2` | PASS |
| `go build -o /tmp/go_binance_futures-cm2-v27 .` | PASS；编译出的独立二进制约 39MB，没有替换当前应用程序 |
| `git diff --check` | PASS |

公开 REST **真实网络只读验证**：在 2026-11-17 北京时间通过 `curl` 访问 `https://dapi.binance.com/dapi/v1/exchangeInfo`，结果 **HTTP 200、返回 30 个合约（其中 20 个有效的交易中 USD 永续）**。`BTCUSD_PERP` 报告 `contractSize=100`、`marginAsset=BTC`，包含 PRICE_FILTER、LOT_SIZE、MARKET_LOT_SIZE。未使用 API Key、未签名、未执行任何交易。

## 4. 仍待后续阶段的工作

- **CM-3**：新增币本位四个前端菜单和配置中心唯一全局开关；币种预算和策略编辑页面。
- **CM-4**：币本位持仓、订单、真实成交与反向合约收益汇总。
- **CM-5**：COIN-M 私有 User Data WS（含断连/补证）、签名 userTrades、真实开平仓、Algo 止盈止损及成交精度的实际下单门禁。市场 WS 的 `DeliveryMarketWSFresh` **不得替代私有 User Data WS 的交易可用性校验**。
- **CM-6**：2 秒自动策略调度；当前仅同步公开市场数据、不会打开自动交易链路。
- WS 到真实 Binance 的长连接、签名账户请求、真实 Testnet/MySQL 迁移、生产部署均**尚未实际执行或验证**。市场数据流仍需在 CM-5 验收“真实账户已鉴权且流可用”。
- 公开行情同步不受 `coin_m_quant_enable` / `delivery_symbols.enable` 开关影响：关闭自动开仓时仍会周期读取公开 DAPI ExchangeInfo、持续接收行情 WS 并更新行情列；不进行私有账户轮询、不会下单。这是两级交易开关的既定语义。若要完全停止 COIN-M 行情采集，现阶段须停止应用或调整启动代码，**没有第三个配置中心开关**。
- 公开 REST 返回的本机缓存已进行 SHA256 与字段核验：[CM-2 公开 DAPI 响应证据](CM-2-公开DAPI响应证据.md)。本次尝试重新联网遇到 TLS 故障，因此新证据不等于独立二次联网成功。
- 目前元数据保存 `tickSize` / `stepSize` / `contractSize` 等核心字段，完整 MARKET_LOT 与各过滤器需要在未来 CM-5 下单前从当前 DAPI ExchangeInfo **重新校验**，不能依赖启动时的静态记录。

## 5. Code Review 结论

**CM-2 在源码实现、离线集成测试、公开 REST 验证范围内通过，可以进入 CM-3。** 真实交易与签名 WS 测试不在本阶段完成范围内，不得表述为已经实盘验收。

没有迁移应用数据库、修改 `app.conf`、重启生产服务、替换 `./go_binance_futures` 或发送任何 Binance 下单请求。
