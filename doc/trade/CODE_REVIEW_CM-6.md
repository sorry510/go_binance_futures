# CODE_REVIEW_CM-6 · 币本位自动策略调度与离线闭环

> 日期：2026-11-18。分支 `feat/m-coin-futures`。
> 阶段：**CM-6 源码接线与离线 Fake/Mock 回归**。不代表真实 Binance Demo/Testnet、MySQL 8 或生产可下单验收。
> 依据：[CM 开发方案](开发方案-CM-币本位合约交易-复刻现货精简版.md)、[CM-5 复审整改](CODE_REVIEW_CM-5-rereview-2.md)。

## 一、实现边界与结论

复用现有 `main.go` 中 **Spot Quant 2 秒循环** 调用 COIN-M Runner，未创建第二个调度中心。使用单独的 `coinmexecution` 服务包，避免它与 `feature` 形成 Go 循环依赖。保护现有 USD-M、Spot、Lead 链路：不复用它们的订单、Fill、账户、市场或下单客户端。

启动新开仓的必要条件：

1. 持久化 `config.coin_m_quant_enable=1` 且具体永续合约 `delivery_symbols.enable=1`。
2. 币本位指定 mainnet/testnet/demo 环境已配置，并有**已认证的 DAPI Private User WS**。
3. 旧 `coin_m_managed_orders` 的不确定成交/Algo 已核实，当前没有未结算开仓或额外 Managed 周期。
4. 使用该环境的公共 DAPI Kline、有效策略与实时 ticker，唯一且明确的 LONG/SHORT 方向，以及当前 Kline 的稳定新订单 ID。
5. 新开仓前再次读取两个开关、策略和技术指标快照。CM-5 Processor 另外验证 USD 面值张数、杠杆/保证金模式、余额、远端未管理持仓和订单。

任一开仓开关关闭**不等于强平/撤销保护单**。已有 Managed 仓位退出、部分成交恢复、原订单 REST 对账仍继续运行。个人账户仍严格执行单个 Managed COIN-M 仓位周期。

## 二、调度运行路径

`main.go` → 原 Spot 2s loopRun 旁路调度 CM → `StartCoinMTrade` → `CoinMNeedsRuntime` → `coinMRuntime.Tick` → `CoinMRunner.Tick`：

- **第一优先**：读取并恢复本地原始 DAPI clientOrderId 对应的未知/部分成交/待确认订单，或读取保护 Algo 的真实状态、actualOrderId 与子订单 Fill。每轮最多处理一个需要核对的对象，绝不在恢复轮次追加新订单。
- **第二优先**：重算已有持仓真实 Fill 的反向盈亏、匹配 LONG 或 SHORT 出场规则。交易开关 OFF 仍可按已有 Managed 归属平仓。STOP/TP 身份明确时才撤销，并经交易所二次核实后提交独立 CLOSE；任何未知状态均中止。
- **最后**：选取一个已启用币种，基于 COIN-M DAPI Kline + 独立 WS ticker 计算策略；信号方向含糊、行情失效、配置在计算中被编辑或同一 Kline 已有 OPEN 订单都禁止新开仓。

**独立限频与串行**：已有 2s cadence；单币普通行情决策至少间隔 10s，异常订单 REST 对账至少 8s，ACTIVE 保护 Algo 查询至少 15s；长任务 `TryLock` 跳过重叠 tick，WS 事件通过 256 条有界队列传递，由同一 Ormer 串行处理。没有额外每 2s 外部轮询器。关闭所有开仓开关且无 Managed 任务时，停止不必要的私有 WS/API 工作；现存 CM-2 **公开行情 WS / exchangeInfo 周期采集不受此开关控制**。

## 三、COIN-M 策略与数量语义

- `feature/api/binance/coin_m_klines.go` 专用 **GET /dapi/v1/klines**。检查永久合约符号、合法 interval/limit、时间与 OHLC、有界新鲜度，以及配置 DAPI REST host 的一致性。不调用 U-M `/fapi`、Spot `/api` 的行情客户端。
- `service/coinmexecution/strategy_cm6.go` 复用已有 Expr DSL 与指标**计算函数**，只使用上面的 COIN-M Kline 数据。每个启用指标必须有完整、有限值的实际序列；不接受缺失数据默认填零。LONG 和 SHORT 同时命中拒绝新开仓。
- 开仓 ID 包含环境、symbol、side、Kline interval/openTime，不依赖可编辑策略 hash；跨周期有重叠信号时不再下第二笔，重启后仍查原生订单记录。
- `Position.Amount` 为**张数**，不是 BTC/ETH 数量。盈亏按 inverse 合约结算币计算，ROI 为**配置杠杆下的策略触发比率**，不冒充当时真实历史初始保证金收益率。
- `close_long / close_short` 必须先达到收益/止损阈值才评估，防止规则 `true` 绕过 8%/6% 门槛。未配置显式关闭规则时，按原 U-M/Spot 语义保留 ±3% 中性带和 DAPI 5m Kline 确认；Kline 无法确认则不主动发送 CLOSE。
- 被动 STOP / TAKE_PROFIT_MARKET 按 inverse 盈亏目标及 tickSize 生成触发价（LONG/SHORT 方向有区别），订单/Fill 仍由 CM-5 原有 Gateway 与 Ledger 负责。

## 四、本次代码审查发现并修复的边界

| 编号 | 等级 | 处理 |
| --- | --- | --- |
| F1 | P1 | **修复** 自定义 close_long / close_short `true` 可直接跳过 ROI 阈值的问题。阈值先于 DSL 判断；零阈值按禁用处理，非法阈值拒绝判断。 |
| F2 | P1 | **修复** `CancelProtection` 取消后仅以侧别、symbol、clientAlgoId 接受响应，未校验远端 `OrderType` 和之前保存的 `AlgoID`。现在两者不匹配标记 `UNKNOWN`，禁止 CLOSE。 |
| F3 | P2 | **修复** 未结算订单列表固定截取前 100 项，可能忽略第 101 条。改为读取 101 项并在超限时返回错误，不把截断结果当作完整。 |
| F4 | P2 | **修复** 策略规则或指标配置在信号计算中修改，仍可能拿旧信号下新配置的单。现重新校验两类快照，变化则跳过本轮信号。 |
| F5 | P2 | **修复** 原价位阈值命中就直接走默认收盘，缺少既有合约策略使用的 5m Kline 确认。现在使用 COIN-M DAPI 5m，缺失或反向信号不主动平仓。 |
| F6 | P1 | **修复** 自动开仓前没有确保配置的 STOP/TP 均可生成合法反向触发价。现在正数阈值、杠杆、最小 tickSize、百分比反向价分母均先验证；缺失或禁用保护的配置不允许自动开仓。 |
| O1 | 限制 | `userTrades` 满 1000 笔的分页仍按 CM-5 结论 fail-closed，尚未实现分页抓取。 |
| O2 | 限制 | 无持仓时 DAPI 杠杆、保证金模式数据的真实可读性仍待 Demo/Testnet 核实；不能从 Mock 成功推断真实可下单。 |
| O3 | 限制 | 未运行真实 STOP/TP 的父子 Algo 触发及现货/U-M/Lead 并行市场的整机联调。 |
| O4 | 限制 | 策略运行时使用当前配置的杠杆计算条件 ROI，非历史保证金收益率；不要将策略触发数值作为实际 Binance 账户 ROE 报告。 |

## 五、永久回归测试

| 测试文件 | 核实内容 |
| --- | --- |
| `runner_cm6_test.go` | 两开关、WS 就绪、LONG/SHORT、同 Kline 和跨环境订单 ID、断线禁止新开仓、旧订单优先、已持仓 OFF 出场、反向保护价、自定义 DSL、策略配置更新竞态 |
| `runner_protection_cm6_test.go` | STOP+TP 归属下平仓顺序、断 WS 不误撤保护、提交超时从原订单恢复而不重复 POST |
| `cm6_safety_review_test.go` | 显式 CLOSE 不绕过收益阈值、零阈值/负阈值处理、Algo 类型与既存 AlgoID 发生冲突时不结案不 CLOSE |
| `runner_overflow_cm6_test.go` | 101 条未知订单扫描必须失败而不是默默忽略 |
| runner_protection_preflight_cm6_test.go | 七组无效 STOP/TP、tickSize、杠杆组合阻止自动开仓、没有 DAPI POST 或持仓记录 |
| `feature/api/binance/coin_m_klines_cm6_test.go` | DAPI Mock host、Kline 数据字段、非法合约/interval/limit、跨环境请求拒绝 |

| Gate | 结果 |
| --- | --- |
| `go test -count=5 -p=2 -run '^TestCM6' ./service/coinmexecution ./feature/api/binance` | **PASS** |
| 四个 CM 核心包 `go test -count=5 -p=2` | **PASS** |
| 88 个非历史研究包 `go test -count=1 -p=2` | **PASS** |
| 上述非历史研究包 `go vet` | **PASS** |
| 四核心包 `go test -race -count=2 -p=2` | **PASS**（macOS linker LC_DYSYMTAB 警告不影响测试） |
| `go build -o /tmp/go_binance_futures_cm6_final_v27 .` | **PASS** |
| `git diff --check` | **PASS** |

原项目 `strategy_templates/research/` 含历史重复函数研究源码，故 `go test ./...` 不能作为不加范围限制的通用绿色 Gate；保留其证据，不删除历史数据。完整测试使用 `go list ./...` 排除此历史目录得到的 88 个非历史研究包。

## 六、未执行的真实环境动作

没有运行任何真实 COIN-M 私有 DAPI 签名下单、钱包余额查询、私有 WS 连接或 Algo 发单；没有修改 `app.conf`、生产数据库、运行中的后端二进制；没有执行 `sync db`，也没有重启服务、自动开关、提交 Git。CM-6 文件进入工作区不代表已部署至正在运行的系统。

**阶段判断**：CM-6 源码与隔离回归可进入 CM-7 研发；正式发布前，需明确 MySQL 8 v27 升级、Demo/Testnet 订单及保护 Algo 现场验收仍未执行。当前不能宣称生产可用。


---

## 七、CM-6 首次复审 F7/P1 整改追踪（2026-11-18）

[首次独立复审](CODE_REVIEW_CM-6-rereview.md) 在策略求值期间编辑 STOP/TP 参数时成功复现 F7/P1：旧保护校验仍可能允许开仓。该缺陷已在 `tryEntry` 最终读库阶段修复，新增 `runner_protection_race_cm6_test.go` 永久测试，覆盖 `Profit/Loss/TickSize/Leverage` 无效及有效数值变动；修改配置后不发送 POST、不落 Managed 周期，刷新合法配置后重新运行能正常开仓。

[第二轮复审](CODE_REVIEW_CM-6-rereview-2.md) 已重新完成 88 包离线测试、CM-6 五次重复测试、Race、Vet 与 Build。**CM-6 源码/隔离测试范围重新判定通过，但真实环境部署门禁仍保留。**
