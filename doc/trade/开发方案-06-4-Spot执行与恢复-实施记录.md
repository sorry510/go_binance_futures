# 方案 06-4：Binance Spot 执行与恢复——实施记录

日期：2026-11-11
分支：feat/trade_spot；本轮开发基线 HEAD 6d8eecc，Schema v25。
结论：**独立执行层与 Fake/离线测试通过；Testnet 实际网络、WS 长连与真实交易均未验证，已按 CODE_REVIEW_06-4.md 完成代码级整改，待独立二次复审；Testnet 和实盘 Gate 继续 BLOCKED。**

## 一、范围与安全边界

本阶段仅在独立 Go 服务中新建 Spot 自动执行能力，不修改老现货抢新、现货监听或 Futures 仓位/订单/Fill，也不启用现货策略循环。没有新增前端菜单、数据库表、Schema 版本或 REST 交易路由，没有修改 main.go / conf/app.conf。

- 默认读取适配器禁止真实下单与撤单；只有显式注入 Binance Spot Testnet SDK Client、将 URL 设为准确的 https://testnet.binance.vision，并在构造时明确授权订单修改，测试网适配器才允许 POST/DELETE。主网 URL 在 06-4 构造器中直接拒绝授权。
- 业务执行必须同时通过 config.spot_quant_enable=1 与 spot_symbols.spot_quant_enabled=1；DBGate 每次查数据库，不缓存。执行前及发送前反复验证，关闭总开关不阻止已存在订单的查询/恢复/撤单。
- 订单以持久化 client_order_id 为幂等键。任何请求超时/异常先按 client ID 查交易所；POST 的结果不明时不再次 POST，返回 ErrUncertain。重复请求只对账，绝不新建第二笔交易。
- 出账依据只有 06-3 的 Fill 重放持仓。SELL 先对历史委托与 myTrades 重新对账，账户 free 基础币不得少于策略已核实库存；若不足，报 ErrNeedsReview，**不缩小 SELL 以使用手工剩余币代替**。SELL 使用已验证的全部策略库存，不再固定预留 0.5%；不满足交易步进的 dust 报需核对错误，不伪造 CLOSED。
- BUY 使用报价预算，经当前 Spot 价格与 1% 预算余量推导实际基础币 quantity，再按照 LOT_SIZE / MARKET_LOT_SIZE 的 step/min/max 校验，并参考 MIN_NOTIONAL / NOTIONAL 和账户 free。市场单不传 LIMIT 价格；PRICE_FILTER tick 不用于市场单的未指定委托价格。交易所将根据其参考价/成交均价最终判定名义金额过滤条件，本地现价仅是预检。
- 实盘 Spot 钱包代币是可互换资产：账本隔离与可用余额检查能够防止超卖账本确认的数量，但**不能从链上/钱包余额推断代币原始归属**。若个人手工提走策略币、再存入同类币，自动交易无法百分之百识别来源；需要账户行为隔离、余额异常禁止下单、06-6 小额验证，不能宣称绝对没有误卖风险。

## 二、代码清单

| 路径 | 功能 |
| --- | --- |
| service/spotexecution/types.go | Gate、Exchange、LocalLedger 与 SDK 无关的请求/成交接口；订单不确定、需核对错误 |
| service/spotexecution/gate_db.go | 使用数据库读取全局/单币双开关；缺配置、缺币种或查询错误全部 fail-closed |
| service/spotexecution/rules.go | 精确 Decimal 数量规划、预算/余额、步进、最大/最小数量和名义金额检查；SELL 精确全仓数量及不可交易残余校验 |
| service/spotexecution/processor.go | ExecuteBuy、ExecuteSell、Cancel、Reconcile、RecoverOutstanding；本地订单先持久化、查询原委托后只发送一次、成交/订单恢复 |
| service/spotexecution/user_stream.go | HandleExecutionReport：只接受本地已归属 client_order_id 的订单，手工账户订单不被认领；重复成交不重复入账 |
| service/spotledger/lookup.go | 从已持久化唯一 client_order_id 查询本地委托，供重试/重启恢复 |
| spot/api/binance/spot_quant_adapter.go | 复用当前 Spot SDK Client、代理池、REST 用量保护与价格缓存；封装 exchangeInfo、现价、账户余额、查单/下单/撤单、按 orderId 查询 myTrades |
| spot/api/binance/spot_quant_stream.go | 显式调用签名式用户数据 WS（SDK）并复用 Spot 代理池；断开/处理失败返回错误，要求主动 REST 对账；没有自动启动 |
| service/spotexecution/processor_test.go | 临时 SQLite + Fake Exchange + fakeGate：配置、交易、异常、恢复、隔离回归 |
| spot/api/binance/spot_quant_adapter_test.go | 默认适配器无下单权限、主网即使明确请求也被拒绝、Testnet 构造路径校验 |

## 三、交易处理与恢复顺序

1. 收到仅针对 Spot 的策略 BUY 或已存在 Position ID 的 SELL 请求，先检查开关、已有 client ID、已有策略订单，再从 Spot exchangeInfo、最新现价和账户 free 进行数量/名义金额预检。
2. BUY 建立独立 PENDING_BUY 仓位周期，SELL 使用已有 position_id。两种方向均先通过 06-3 CreateOrder 将唯一 client ID 和请求数量落库，然后再次检查开关及余额/库存。
3. 向交易所发单前，先 GET /api/v3/order 按 client ID 查询；若已有委托则直接进入恢复，只有明确 -2013 未找到且属于本次新建的本地委托，才有一次 POST 机会。POST 超时无论是否成功都不得盲目重发。
4. 下单后按原 client ID、exchange order ID 和实际请求量核实交易所返回，保存订单状态。按 myTrades 恢复所有可信实际成交，通过 06-3 RecordFill 幂等计算成本与仓位；累计 Fill 数量必须等于交易所 executedQty，否则 ErrNeedsReview。
5. USER_DATA executionReport 的 TRADE 事件可实时写入已归属仓位；系统不自动归属手工钱包交易。遇到 WS 断开、订单失联、交易冲突、myTrades 不完整、成交量超过委托量，停止新单并报告需核对，只有从 REST 核实成功后才能继续。
6. Cancel 不会生成新开仓单，全局开关关闭后仍可对既有委托进行撤单和只读恢复。未接入定时任务、主程序启动或正式 UI；由后续阶段明确接线。

### 限制和待办

- 仅 MARKET BUY/SELL 自动订单；06-3 保留的本地 LIMIT 数据结构没有在 06-4 接口中开放 LIMIT 自动执行。
- myTrades 每次以 orderId 查询最多 1000 笔；达到或超出该阈值会 fail-closed，不假设结果完整。若后续测试需要高成交笔数，须实现经官方参数验证的分页后才能放行。
- 仅有 Fake/SQLite 和 SDK 适配器构造测试；无 Binance Testnet 的网络握手、签名 WS 长连或真实订单成交实证。上线前还须验证 Testnet 和 Spot Mainnet 不同权限/地址、费率等。
- HTTP API 的 PRICE_FILTER 与 NOTIONAL 的市场参考价由交易所最终判定；本地价格仅为预检。调用频率及 429/418 沿用现有 Spot HTTP Client 保护。不能把本地预检描述成交易所最终承诺。
- 缺交易所用户数据/超时会造成本地 NEW 未知状态。为避免双下单，该委托不会自动 POST 重试，必须通过 Reconcile / RecoverOutstanding 只读对账后再处理。个人系统不引入新复杂审核表。
- 06-4 无自动策略信号扫描，行情指标驱动、正式顶级菜单、账户/持仓 UI、后台运行与成套发布由后续 06-5/06-6 分别验收，不以本阶段代码通过宣称已经实盘运行。

## 四、测试结果（当前工作区）

| 实测 | 结果 |
| --- | --- |
| Fake：双开关（全局 OFF、单币 OFF、双 ON）、发送前再次读取 | PASS |
| Fake：预算不足、最低名义额、LOT_SIZE、SELL 全仓关闭/手续费异常与钱包短缺 | PASS |
| Fake：BUY→SELL、真实 Fill 账本、手工钱包库存不自动归属 | PASS |
| Fake：超时后查询已存在订单；网络故障后保留未知状态且不二次 POST | PASS |
| Fake：用户数据 WS 事件、人工交易忽略、同一 trade_id 去重、部分成交补齐 | PASS |
| Fake：重启 RecoverOutstanding、关闭总开关仍可撤单/恢复 | PASS |
| Fake：远端币种/请求量不一致、myTrades 身份/数量不匹配、结果可能截断 | PASS，fail closed |
| SDK：默认只读、Testnet 才能明确授权订单修改，主网拒绝 | PASS |
| Go 目标包单测 7 包 | PASS |
| Go Vet / Go 主程序隔离构建 | PASS |
| Go Race Detector / Spot 执行模块重复运行 5 次 | PASS |
| Binance Testnet 真实签名 REST/WS、真实成交、MySQL 8 | **未执行，不能宣称通过** |

实测全部使用临时 SQLite 与 Fake，未触发 Binance API 网络请求、交易所订单、生产或实验库 DDL。此前 06-3 MySQL 5.6 隔离结果不等于 06-4 Testnet 或 MySQL 8 验收。

## 五、后续 Gate

06-4 实现可作为 06-5 UI 接线和 06-6 完整 Fake/Testnet 集成验证的研发基线，尚不能开启自动 Spot 实盘。真实主网需要独立授权及代码层解封、发布前 Schema 核对、Testnet 端到端证据、双开关 Gate / 幂等 / 余额/成交恢复验收、前端 dist 与后端 static 和程序成套发布。

本轮没有改 main.go、app.conf、数据库 schema version、生产库或已有 Spot Rush/Futures 下单路径，没有真实交易，也没有自动 commit/push。


## 六、2026-11-11 CODE_REVIEW_06-4 整改补充

审查报告：[CODE_REVIEW_06-4.md](CODE_REVIEW_06-4.md)；整改复评：[CODE_REVIEW_06-4-rereview.md](CODE_REVIEW_06-4-rereview.md)。

- F1：标准可交易全量 SELL 后必须 `CLOSED` 且 `remaining=0`，新周期可开；不再固定留 0.5% 基础币。基础币手续费导致不可卖 dust 时必须报异常并人工处理，不能假结算。
- F2：持久化原 client ID 只能只读恢复，权威未找到与网络异常分别返回 `ErrNotFound` / `ErrUncertain`。新增**显式人工确认空单**路径（至少 30 分钟+两次无此单验证+无 Fill+原单 NEW），释放无成交 PENDING_BUY 周期；自动恢复绝不自行释放。
- F3：签名 WS 使用固定 Testnet 端点与项目代理，不再依赖 SDK 全局 Testnet 变量；创建后也不能通过修改 BaseURL 绕过 Testnet 的 Place/Cancel 守卫。
- F4：`FILLED` 先于 myTrades 的正常竞态不再中断 WS，先尝试只读 REST 恢复；无法对账继续挡新单且日志报警。
- F5/F6/F7：SELL 双开关检查早于任何 REST；老已完整核对的 FILLED 无需反复远端查单；活动仓位恢复先快照 ID；正手续费的 FeeAsset 为必填，零手续费可无资产。
- F8/F9：方案基线同步纠正；Testnet 实网、MySQL 8、生产部署与 Schema 预检仍待 06-6，不能宣称已完成或可用实盘。
