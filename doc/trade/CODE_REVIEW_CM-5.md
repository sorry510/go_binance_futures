# CODE_REVIEW_CM-5 · 币本位 DAPI 订单执行、真实 Fill、保护单及私有 WS（离线实现）

> 日期：2026-11-18；分支：`feat/m-coin-futures`；Schema 仍为 **v27**。
>
> **本阶段完成源码、Fake/Mock 与回归验收，绝未将 CM-5 自动接入 main.go/真实策略调度；没有调用用户的 DAPI 签名交易接口，没有真实下单，没有执行 MySQL 迁移或应用发布。**

## 一、代码对应与结果

| 模块 | 关键源码 | 实现与边界 |
| --- | --- | --- |
| 独立 COIN-M DAPI 交易 API | `feature/api/binance/coin_m_trade_api.go` | `POST/GET/DELETE /dapi/v1/order`，仅 MARKET/LIMIT；独立订单 ID、COIN-M LONG/SHORT，签名客户端沿用 CM-2 受保护的 DAPI 环境，无 FAPI/Spot 下单 |
| 条件单 | `coin_m_trade_api.go` | `POST/GET/DELETE /dapi/v1/algoOrder`、`GET /dapi/v1/openAlgoOrders`；STOP_MARKET/TAKE_PROFIT_MARKET 统一 `closePosition=true`、`MARK_PRICE`；绝不走旧普通订单接口 |
| 币本位订单网关 | `service/coinmexecution/binance_gateway.go` | fresh exchangeInfo、标记价格、余额、持仓模式、杠杆与保证金模式读取；不调用任何共享持仓模式/杠杆/逐全仓修改接口 |
| 稳定开仓/平仓 | `service/coinmexecution/processor.go` | 合约张数 `floor((budgetUSD × leverage)/contractSize/step) × step`；开仓严格全局 + 每币双开关、有效方向策略、对齐精度、真实余额及 2% 预算缓冲；已有仓位/未归属订单/Algo/持仓周期均阻断新开仓 |
| 既有仓位退出 | `processor.go` | 只平本地 Managed 周期，要求交易所同方向张数与本地一致；**关闭开仓开关不会禁止退出**；其他手动或跨产品仓位不能认领 |
| 先落库后发单 | `service/coinmexecution/ledger.go` | 本地 Position 与订单（稳定 clientOrderId）在一个事务提交后才能 POST；超时/429/5xx 返回 `ErrUncertain`，不盲目重下；同一 ID 重复调用返回原始订单 |
| 真实逐笔成交 | `ledger.go` | 使用 Binance `userTrades` 或 WS 的真实 `tradeId`、`orderId`、张数、成交价、币本位已实现 PnL、手续费资产；唯一 `(symbol, exchange_order_id, trade_id)`，Fill 重放去重；不得用 ACK `executedQty` 伪造 Fill |
| 反向持仓周期 | `ledger.go` | 依据 verified Fill 实际张数重算开/平数量、剩余数量、已实现盈亏；开仓均价为合约张数加权的**倒数均价（调和均价）**，不是现货线性均价 |
| 断线/订单恢复 | `service/coinmexecution/reconcile.go` | 只通过原 `clientOrderId` 查询订单、原 `orderId` 查 userTrades；Fill 数量不齐返回 `AWAITING_FILLS` 不结案、不重发 POST；仅撤销自己管理的普通订单 |
| 私有 User Data WS | `service/coinmexecution/user_stream.go` | 显式 `Run(ctx)` 启动、listenKey 申请、25 分钟 keepalive、断线重连/关闭；只有此私有流在线且交易所 Hedge Mode 正常才允许开平仓；行情 WS 绝不替代私有鉴权 |
| WS 订单事件 | `reconcile.go` | `ORDER_TRADE_UPDATE` 仅处理已保存的 clientOrderId；手动/未归属订单不认领，`ALGO_UPDATE` 无完整父子 ID 时私有 WS 停止交易就绪状态，要求签名核对 |
| TP/SL 双保护单 | `service/coinmexecution/protection.go` | 同一持仓可分别有一个 STOP 与一个 TAKE_PROFIT；拒绝同类型重复、未归属远程 Algo、金额或方向不符、异常状态；本地持久化 clientAlgoId 后调用 Algo |
| 条件单触发后的真实子订单 | `service/coinmexecution/reconcile_protection.go` | 使用**已归属** clientAlgoId 查询 Algo 的 `actualOrderId`，再按精确子订单 ID 查普通订单及 userTrades；验证 symbol/positionSide/side 后才落 Fill；触发一侧时核对并撤销另一侧；撤销 ACK 后必须再次确认无实际子订单 |
| CM-4 只读收益兼容 | `service/coinmquant/ledger_summary.go` | 已由交易所证实的 PROTECT 子订单 Fill 以 CLOSE 计算剩余张数/已实现收益；保留 CM-4 三方归属校验 |

## 二、关键安全与简化约束

1. **只使用独立 COIN-M client/DAPI**。普通 MARKET/LIMIT 和条件保护 Algo 是两个独立入口。不会自动更改 U 本位与币本位共享的 `dualSidePosition`。
2. **没有第三个交易开关**。唯一全局 `coin_m_quant_enable` + 现有 `delivery_symbols.enable` 仅决定新开仓。已有 Managed 仓位的退出原语可在开关 OFF 时调用。
3. **默认没有自动实盘入口**。本阶段没有在 `main.go`、`routers` 或 HTTP Controller 接入 `coinmexecution`；生产调度属于 CM-6。仅构建源码和测试不会向 Binance 下单。
4. **不能盲目重试**：本地 PENDING_SUBMIT/UNKNOWN 状态会保留同一 clientOrderId；网络超时等不确定结果必须先查询交易所，不得根据等待时间自动再 POST。
5. **真正的 Fill 才会改变账本**：普通订单响应 `FILLED` 不等价于成交数据完整；必须对照 `userTrades` 的每个真实 tradeId 和累计张数，重复 WS/REST 也不能双算。
6. **保护单不是交易所 OCO**：STOP/TP 独立提交。系统在得到第一侧 Algo 的真实 `actualOrderId` 后尝试撤销另一侧，并再查询撤销结果，防止悬挂条件单；若触发和撤销竞态无法确认，保持 UNKNOWN/ErrUncertain，阻止后续新开仓，**绝不推断归属或假装安全**。
7. 私有 WS 的生命周期需要由 CM-6 显式启动并设置 `NewUserStream(processor.OnWSOrder)`；同样须调用 `ReconcileProtection` 处理 Algo 事件或定期回补。**CM-5 不是已连接生产调度的交易系统。**
8. 当前 `userTrades` 查询按 symbol + orderId、每次最多 1000 笔。若满页（可能分页不完整）直接拒绝结案并保持 `ErrUncertain`；未冒充已实现全历史分页。
9. 不采集、打印真实密钥、不修改 `conf/app.conf` 数据库配置、不向生产 MySQL 写任何数据。不新增复杂的审批系统或审计链。

## 三、实际离线验收证据

| Gate | 结果 |
| --- | --- |
| COIN-M 开 LONG/SHORT → 部分平仓/完全平仓 → verified Fill 与反向 PnL | **PASS** |
| REST/WS 同一 tradeId 重复推送，唯一 Fill 数量不变 | **PASS** |
| 本地订单先保存后 DAPI Mock POST，超时后稳定 clientOrderId 不重复提交 | **PASS** |
| 全局开关 OFF、币种 OFF、共享 Hedge Mode 不符、没有私有 WS、手动/反向仓位、交易所精度错误 | **PASS：拒绝新开仓** |
| DAPI Mock HMAC 实际参数、仅 dapi 路径、用户成交订单精确过滤、Demo/Testnet 隔离 | **PASS** |
| 新 Algo STOP/TP 正确走 `/dapi/v1/algoOrder`；普通订单拒绝 STOP/TP | **PASS** |
| 一组双保护单、同类型重复保护/未归属保护拒绝、触发子订单 `actualOrderId` 验证与另一侧取消 | **PASS** |
| Algo 或子订单跨 symbol/hedge side、缺少 actualOrderId、关闭保护单竞态下不认领成交 | **PASS** |
| 私有 WS listenKey 过期、重新连接、断线时就绪门禁、关闭后不保持 Ready | **PASS** |
| 4 个直接相关包 `go test -count=5 -p=2` | **PASS**：coinmexecution、coinmquant、feature/api/binance、controllers |
| 88 个非历史研究 Go 包 `go test -count=1 -p=2` | **PASS** |
| 同一 88 包 `go vet` | **PASS** |
| 4 包 `go test -race -count=2 -p=2` | **PASS**（macOS linker 非致命 LC_DYSYMTAB warning） |
| `go build -o /tmp/go_binance_futures_cm5_v27 .` | **PASS**：独立构建约 40 MB，未替换运行程序 |
| `git diff --check` | **PASS** |

相关持久化 Mock/SQLite 测试在：

- `service/coinmexecution/processor_test.go`
- `service/coinmexecution/reconcile_protection_test.go`
- `service/coinmexecution/user_stream_test.go`
- `feature/api/binance/coin_m_trade_api_test.go`

既有 CM-4 的 Filter 与收益测试必须继续通过。Go 项目的历史 `strategy_templates/research` 子目录包含其他研究程序的重复定义，回归仅统计非研究生产包。

## 四、待下一阶段真实验收

- **CM-6**：接入共享 2 秒真实策略调度（出场优先）、私有 WS 启动、持续自动订单与 Algo 回补、两级新开仓开关、限流。当前没有连接任何自动下单。
- **CM-7/环境验收**：真实 MySQL v27 迁移、目标 ARM Linux、COIN-M Demo/Testnet 专用 API 权限、User Data WS 与 Algo 条件单的真实父子关联、模拟断连及边界仓位；需完成之后才考虑生产小额真实交易。
- 暂不提供 HTTP 手工真实下单/撤单按钮。页面保留本地仓位只读设计；这是个人系统的最小安全边界。
- 实际交易所返回可能因 2026-06 COIN-M/UM 接口整合而出现不同 JSON 字段和限制；**本次只验证了 Fake/Mock，未通过真实签名请求**。如果实际测试出现不同，必须先修改客户端并重新走离线 Gate。

## 五、阶段结论

**CM-5 的源码实现、独立 DAPI 网关、普通开平仓、真实 Fill 归属、双保护单与恢复原语已完成。离线 Mock/SQLite Gate 全通过后可进入 CM-6 开发。**

这**不代表真实 Binance 下单可用、签名 DAPI 环境通过、生产已部署或自动交易已开启**。没有执行生产数据库迁移/改配置/提交或启动交易任务。


---

## 六、本轮补充 Code Review：保护单取消竞态修复（2026-11-18）

在核对现有 CM-5 实现时，发现并修复两处可能导致保护单错误取消的边界：

1. **先验证真实子订单、再撤销兄弟 STOP/TP**：原 `ReconcileProtection` 在 `QueryAlgo.actualOrderId` 所指的普通订单身份、之前持久化的子订单 ID 尚未完全比对时就可能调用 `cancelSiblingProtectors`。现在严格先验证普通子订单的 `ExchangeOrderID`、`symbol`、`positionSide`、`side`、`status`，及既有归属子订单 ID，全部正确后才尝试撤销兄弟保护单。
2. **格式异常的子订单 ID 不能当成“没有订单”**：原 `CancelProtection`、`ReconcileProtection` 取消状态分支及兄弟保护单取消分支会忽略 `confirmedAlgoChildID` 的解析错误。现在只有显式空值 `null`、空 JSON 字符串、数值 `0` 才视为没有子订单；非空且无法解析的 `actualOrderId` 必须返回 `ErrUncertain`，不能假设取消已安全完成。

新增永久测试 `service/coinmexecution/protection_guard_cm5_test.go`，包含 `TestCM5WrongTriggeredChildNeverCancelsValidSibling` 和 `TestCM5AlgoChildMissingVersusMalformedFailClosed`。两项测试连续执行 5 次通过；完整 CM-5 四关键包 5 次重复测试、88 个非历史研究 Go 包测试、Race Detector、Vet、Build 都已通过。

**仍然未进行**：真实 Testnet/Mainnet 下单、真实 Algo 子订单现场验收、生产 MySQL 数据库升级、main.go/routers 自动交易入口接线、CM-6 2 秒调度。本阶段不会把 Mock 通过描述为真实账户可交易。


---

## 七、复审观察项处理追踪（2026-11-18）

[独立复审](CODE_REVIEW_CM-5-rereview.md) 判定 CM-5 有条件通过，提出 O1～O4 四项观察。[第二轮整改记录](CODE_REVIEW_CM-5-rereview-2.md) 补齐 O3 保护单类型匹配、O2 满页成交记录 `AWAITING_FILLS` 状态及永久回归测试；O1 维持无手工下单入口，O4 维持真实 Demo/Testnet 验收限制。最新 88 包测试、Race、Vet、Build 均通过，不改变 CM-5 不接生产调度的边界。
