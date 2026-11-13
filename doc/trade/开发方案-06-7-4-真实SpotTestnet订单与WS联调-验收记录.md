# 06-7-4 补充验收：真实 Binance Spot Testnet BUY/SELL 与用户 WS

日期：2026-11-13（开发机北京时间）  
项目分支：`feat/trade_spot`；测试前 HEAD：`a19efe4 feat: 06-7-4`  
状态：**真实 Spot Testnet API / MARKET BUY / MARKET SELL / myTrades / signed User Stream 已通过一次小额端到端实测。生产主网自动交易仍未接线，不代表生产发布验收完成。**

## 1. 安全范围

- 使用独立 `testnet_binance_spot_base_url=https://testnet.binance.vision`、`testnet_spot_api_key`、`testnet_spot_api_secret`；不读取 Futures `testnet_api_key/testnet_api_secret`，不使用主网 Spot Key。
- 真实交易仅发生在 **Binance Spot Testnet**，交易对 BTCUSDT。只创建**一笔 MARKET BUY 和一笔 MARKET SELL**，每笔使用唯一 ClientOrderID；没有未知结果时重发 POST。
- 交易执行使用生产 `spotexecution.NewProcessor` + 实际 `QuantAdapter` + `spotledger` + `spotexecution.DBGate`，但 Gate 的两个开关位于**独立临时 SQLite**；没有修改或连接用户的任何既有数据库。
- WS 使用现有 `QuantAdapter.runSpotQuantUserStream`，连接固定的 `wss://ws-api.testnet.binance.vision/ws-api/v3`，完成签名订阅并将 `executionReport` 交给实际 Processor 处理。只新增内部无副作用的订阅 ACK/事件观察回调，公开 `RunSpotQuantUserStream` 的行为保持不变。
- 不改变 `feature.StartSpotQuantTrade()`：主程序定时回调仍**不注入可发单执行器**，普通打开开关仍不会真实自动交易。
- 新增端到端测试是显式 opt-in：`SPOT_QUANT_TESTNET_REAL_E2E=1`。标准 `go test ./...` 不访问网络、不发送订单。读回旧订单另用 `SPOT_QUANT_TESTNET_VERIFY_CLIENT_IDS`，只有 GET。

## 2. 真实测试网准备

| 检查 | 结果 |
| --- | --- |
| Spot Testnet 账号 `canTrade` | true |
| BTCUSDT MARKET 交易 | 允许 |
| 测试前可用模拟 USDT | 10000 |
| 测试前可用模拟 BTC | 1 |
| BUY 名义预算 | **20 USDT** |
| Lot size 步进 | **0.00001000 BTC** |
| 最小名义金额 | **5 USDT** |
| `GET /api/v3/time` 与账户签名 | PASS |
| User Stream `userDataStream.subscribe.signature` | **ACK 200，PASS** |

未输出任何 API Key、Secret Key 或签名值。

## 3. 真实 BUY / SELL 订单

| 记录 | BUY | SELL |
| --- | --- | --- |
| Testnet ClientOrderID | `sqTBuy18e791ebb3c7bdd0` | `sqTSell18e791ebb3c7bdd0` |
| Binance Exchange Order ID | `13340589` | `13340593` |
| 请求方式 | MARKET BUY | MARKET SELL |
| 最终交易所状态 | **FILLED** | **FILLED** |
| 实际成交 BTC | **0.00028** | **0.00028** |
| `myTrades` 成交事件数 | 1 | 1 |
| 累计成交价值 | **19.3119164 USDT** | **19.3119136 USDT** |
| 手续费 | 0 BTC | 0 USDT |

**本次 Testnet 往返的成交价差**：`19.3119136 - 19.3119164 = -0.0000028 USDT`。该值是模拟交易网实际记录的差额，非真实市场可预期收益，也不代表策略表现。

由实际 Processor 与 Spot Ledger 确认：

- BUY：`FILLED`，导入 1 条真实 Testnet 成交，`position_id=1`，净持仓 0.00028 BTC。
- SELL：`FILLED`，导入 1 条真实 Testnet 成交；最后 Position 状态 **CLOSED**。
- 策略剩余数量：**0.000000000000000000 BTC**；没有把测试账户已有的 1 BTC 作为策略库存出售。
- WS 签名订阅 **PASS**，收到与这两笔 ClientOrderID 匹配的 BUY 和 SELL `executionReport`；WS 与 REST 的重叠不会重复写入 Fill。
- 之后再次单独执行只读 `FindOrder` + `Trades`，按交易所订单号和方向验证了两笔 `FILLED` 及数量之和，**PASS**。此步没有发单。

本次真实测试网执行日志：开发机 `/tmp/spot_testnet_live_buy_sell_ws_run.log`，只读复核输出已见本次执行的 Go Test 结果；日志不包含密钥。

## 4. 回归与代码清单

| 文件 | 作用 |
| --- | --- |
| `spot/api/binance/spot_quant_stream.go` | 增加仅包内使用的可选 `onSubscribed` / `onExecutionReport` 测试观察回调，原 `RunSpotQuantUserStream(ctx,p)` 仍按原语义运行 |
| `spot/api/binance/spot_quant_testnet_live_e2e_test.go` | 新增显式 opt-in 的真实 Spot Testnet 交易、用户 WS、独立临时 DB 和只读回查测试 |

本次测试没有触碰真实 Mainnet、生产库、系统数据库连接配置、前端或项目主程序调度。未自动提交或部署。

**普通回归测试**：本地脚本 `./scripts/test-go.sh -short` 73 包 PASS / 0 FAIL / 13 包无测试；10 个相关包 Race Detector PASS；06-7-2 至 06-7-4 专项重复 5 次、Go Vet、主程序 Build、`git diff --check` 均 PASS。与真实 Testnet opt-in 的运行范围已严格隔离。

## 5. 仍需完成的事项

**真实 Testnet 订单与 WS 联调 Gate 本次已经通过**，解决此前因 `-2015` 无法验证真单的问题，但仍**不能把整体 06-7-4 或自动实盘直接标成 COMPLETE**：

1. 主程序 `feature.StartSpotQuantTrade(ctx)` 仍为禁用自动下单的安全入口。真正的两秒 Runner + 真实执行器运行时接线未完成。
2. 此次在 Testnet BTCUSDT / 小额 / 手续费为零情况下完整 CLOSE；其他币种、基础币收费 dust、异常 WS 重连、网络超时、部分成交的**真实交易所场景**仍未证明，离线 Fake 测试不能替代真实故障演练。
3. 生产 MySQL 8 物理 Schema / 真正数据规模 P95、服务器 Linux/ARM 构建部署、备份与回滚未完成，也没有触碰任何生产服务。
4. 主网实际交易需另行严格验证权限、运行时持仓归属与成本/止损，不能因为 Testnet 成功而更改主网下单硬限制。

**本阶段最终事实：完整代码回归 + 一次真实 Spot Testnet BUY→WS→SELL→WS→CLOSED + REST myTrades 独立核对均已成功，且正常程序仍没有自动真实交易能力。**
