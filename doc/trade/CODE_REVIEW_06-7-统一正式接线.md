# 06-7 现货自动交易统一接线：简化与 Code Review

日期：2026-11-13  
分支：`feat/trade_spot`，起点 HEAD `a19efe4`；**本轮未自动 commit / push、未部署**。

## 当前唯一使用方式

- `[binance].testnet = false`：使用已有 `api_key` / `api_secret`，Spot REST = `https://api.binance.com`，Spot WS = `wss://ws-api.binance.com/ws-api/v3`。
- `[binance].testnet = true`：使用独立 `testnet_spot_api_key` / `testnet_spot_api_secret`，Spot REST = `https://testnet.binance.vision`，Spot WS = `wss://ws-api.testnet.binance.vision/ws-api/v3`。**不读取** Futures 的 `testnet_api_key / testnet_api_secret`。
- 与 Futures 一样保留现有 2 秒定时调度。仅 `config.spot_quant_enable=1` 与对应 `spot_symbols.spot_quant_enabled=1` 时，Runner 才允许创建新 BUY / SELL。
- 每币预算来自其 `spot_symbols.usdt`，不再限制 BTCUSDT/20 USDT。由原有 Processor 按 exchangeInfo、Lot/Notional、实际可用资产核查。
- 生效配置位于服务启动时读取的 `app.conf`；**改变 testnet 后需要重启程序**。

## 删除的额外运行约束

- 删除 `SPOT_QUANT_TESTNET_AUTOTRADE`、`SPOT_QUANT_TESTNET_DB` 两个部署级限制。
- 删除单独 Testnet Runtime、Testnet Strategy、Testnet Market Loader 重复实现；统一为 `feature/spot_quant_runtime.go`、`feature/spot_quant_market_strategy.go`、`spot/api/binance/spot_quant_market.go`。
- 不再硬编码只执行 BTCUSDT，亦无 20 USDT 上限；没有增加第三个页面业务开关或新数据库表。

## 核心安全复查

| 项目 | 结果 / 实现 |
| --- | --- |
| 环境与密钥 | 现有 `testnet` 唯一决定 Spot endpoint 与对应凭据；REST/WS 始终相同环境 |
| 实单授权 | 两个现货量化业务开关，订单 POST 前通过原有 `DBGate` 再核验 |
| WebSocket | 只有 signed user stream 订阅确认 ACK 后允许新 BUY/SELL，断线拒单；REST 可继续恢复历史订单 |
| 持仓归属 | 只按持久化 BUY/SELL Order 和不可变 Fill 计算策略净仓位，不认领人工钱包存量 |
| 幂等与异常 | 新订单 ClientOrderID 固定，已提交订单从 REST `order/myTrades` 查回，网络结果未知不再次 POST |
| 跨环境防误卖 | 新 BUY 标记 `sqt`（测试网）或 `sqm`（正式网）；不同环境或旧 `sq4` 来源不明的 OPEN 仓位拒绝 SELL |
| K 线来源 | 24h ticker、技术指标 Kline、3d OpenTime 必须来自当前交易所环境，不能用主网行情驱动测试网交易 |
| 关闭开关 | 禁止新 BUY/SELL，不清除账本，不自动卖出持有仓位；REST 已提交订单恢复不依赖开关 |
| SDK 请求签名 | 日志脱敏 `signature` / `api_key` 等参数；不输出密钥 |
| 失败回退 | 缺失默认 DB、开关或凭据时拒绝执行，不能因为测试网失败而切到主网 |

**历史订单注意**：既有 `sq4` ClientOrderID 不携带环境身份。如果当前业务库存在此类历史策略订单，自动策略会 fail-closed，需要人工核对归属；本轮**没有删除或迁移任何历史订单**。带有 `sqt/sqm` 的已关闭订单可以区分环境；跨环境尚未关闭的仓位仍拒绝自动操作。

## 2026-11-13 回归证据

- 运行 `./scripts/test-go.sh -short`：**73 个测试包通过、0 失败、13 个包无测试**。
- 相关 10 包 `go test -race`：**10 / 10 通过**。
- 现货自动 WS 门禁、预算、自定义规则、订单前缀环境归属、旧 06-7 回归重复 `-count=5 -p=2`：**PASS**。
- Go Vet、当前代码主程序 Build、`git diff --check`：**PASS**。
- `git archive HEAD` + 本轮修改 / 新增代码生成的无私有配置干净副本，`go test ./... -count=1`：**73 个测试包通过、0 失败**。
- 先前 **Spot Testnet 真实 Runner BUY→SELL→CLOSED** 已通过；本轮为代码清理与 Fake/单元测试，**没有重复发送真实测试网订单，也没有正式网下单**。
- 官方网络协议资料：[Spot Testnet 环境对应表](https://developers.binance.me/en/docs/products/spot/testnet/general-info)、[Spot signed WS](https://developers.binance.com/en/docs/catalog/core-trading-spot-trading/api/ws-api/user-data-stream)。

## 尚未声称完成的项目

- 没有使用正式 Binance API Key 执行主网 BUY/SELL，也没有确认真实主网 User Data Stream 签名及成交反馈。
- 生产数据库物理 Schema、真实历史数据规模 P95、Linux/ARM 部署与回滚没有执行或验证。
- 代码在当前工作区，只有重新构建并部署到目标服务器后才会改变服务器行为；不要把本地测试通过误解为线上已运行。

**最终状态：代码已实现 `testnet=false` 走正式 Spot API，两个业务开关控制自动策略；测试网已完成真实成交链路，主网仅完成代码/SDK 环境检查与隔离回归，没有向正式账户发送订单。**
