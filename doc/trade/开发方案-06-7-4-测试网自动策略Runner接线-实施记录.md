# 06-7-4 追加：Spot Testnet 自动策略 Runner 接线与回归（历史阶段）

**此记录为上一阶段验证证据。最新的 Testnet/Mainnet 统一接线与精简后的代码状态，请看 [CODE_REVIEW_06-7-统一正式接线.md](CODE_REVIEW_06-7-统一正式接线.md)。本文件所述额外环境变量、独立测试库、BTCUSDT/20 USDT 硬限制已删除，不再是当前使用方法。**

日期：2026-11-13（北京时间）  
分支：`feat/trade_spot`；实施前 HEAD `a19efe4 feat: 06-7-4`  
关联：[06-7 主方案](开发方案-06-7-现货量化自动策略执行链路.md) · [真实 Spot Testnet 手工 BUY/SELL + WS 验收](开发方案-06-7-4-真实SpotTestnet订单与WS联调-验收记录.md)

**结论：测试网自动 Runner 接线已开发；独立临时 SQLite + 真实 Spot Testnet 完成一次自动 BUY→SELL→CLOSED。此版本默认仍不自动发单；需要显式部署级 Testnet 授权、独立测试库匹配、签名 WS 就绪与原有两个业务开关。Mainnet 仍没有下单通路。**

## 一、改动和业务边界

| 文件 | 改动 |
| --- | --- |
| `feature/spot_quant_runner.go` | 原 `StartSpotQuantTrade(ctx)` 默认只读入口在显式 Testnet 部署开关下才调用 Testnet Runner；支持测试环境单币 BTCUSDT scope，不改变现有 Fake 测试和 Futures 调度 |
| `feature/spot_quant_testnet_runtime.go` | 真实 `spotexecution.Processor`、`spotledger.Ledger`、`spotexecution.DBGate` 接线；只有签名用户 WS 已确认订阅时才允许新的 BUY/SELL。WS 中断后只允许原订单的 REST 只读恢复；自动 5 秒尝试恢复 WS，发送订单仍禁止 |
| `feature/spot_quant_testnet_strategy.go` | 测试网专用的严格策略环境：独立 Testnet 24h ticker、最新成交价格、K 线与技术指标、3d 同市场实际 OpenTime；本地 `spot_symbols` 仅提供用户配置，不混用本地 Mainnet 的 ticker/OHLC 作为信号 |
| `spot/api/binance/spot_quant_testnet_market.go` | 使用同一个**独立 Spot Testnet SDK 客户端**读取测试网 24h ticker、Kline、同步服务器时间，保留严格新鲜度校验 |
| `spot/api/binance/spot_quant_adapter.go` | 新增仅允许 `https://testnet.binance.vision` 的显式 Testnet 发单适配器构造函数；普通构造仍为只读 |
| `spot/api/binance/spot_quant_stream.go` | 复用经真实测试的签名用户 WS，增加只报告订阅 ACK / 断线状态的回调，未改变旧公开入口语义 |
| `main.go` | 继续复用 2 秒调度，**仅修正接线说明**，未改 Futures/Spot 其他 worker、ORM 注册或程序生命周期 |

本轮没有新增数据库表、数据库版本、第三个业务开关、前端控制台、独立任务中心或新的权限体系。

## 二、双开关和部署级硬限制

**默认行为**：未设置 `SPOT_QUANT_TESTNET_AUTOTRADE=1` 时，`StartSpotQuantTrade` 在触碰 ORM/创建 Testnet 客户端前就返回；**即使现货量化总开关与币种开关都打开，也不会自动发单**。

只有在明确测试环境中，单独设置：

```shell
SPOT_QUANT_TESTNET_AUTOTRADE=1
SPOT_QUANT_TESTNET_DB=go_bn_test
```

并且当前程序 `[database].dbname` **恰好等于** `SPOT_QUANT_TESTNET_DB`、库名包含 `test` 时才创建 Testnet 运行实例。上面只是示例，**本次没有设置该环境变量启动现有服务**；不得把生产库名填入该变量。

运行中的业务交易还必须满足：

1. `config.spot_quant_enable=1`（现货量化总开关）。
2. `spot_symbols.spot_quant_enabled=1`（对应币种开关）。
3. 首轮自动 Testnet 仅限 `BTCUSDT`；用户原有预算 `spot_symbols.usdt` 必须**大于 0、且不超过 20 USDT**。
4. 用户数据 WS 必须已收到真实签名订阅 ACK；断线停止新开/平单，已有订单的只读 REST 对账继续。
5. 独立 Spot Testnet API Key、Secret、Base URL 必须有效；绝不复用 Futures 测试网密钥或 Mainnet Spot 密钥。
6. 任何真实提交前仍执行原有 Processor 规则检查、真实余额/持仓检查、数据库双开关检查、订单与 Fill 幂等校验。

**Mainnet 订单仍硬禁用**：改变 URL 或将生产密钥误填到旧字段都不能绕过 Spot Testnet 主机校验。下一阶段正式主网接线需单独 Code Review、发布授权和真实成本/故障回滚验收。

## 三、真正的 Testnet 自动策略实测

只在本地 `TestStage0675RealTestnetAutomaticRunnerCycle` 中显式设置 `SPOT_QUANT_TESTNET_AUTO_E2E=1`；测试在专用临时 SQLite 建立两个开关和 BTCUSDT 测试规则（简单有效的 LONG/close_long），而**没有修改现有业务配置**。启动真实 Spot Testnet signed WS 并等待 ACK 后，调用真正的 `SpotQuantRunner.Run`：

| 项目 | 2026-11-13 实际结果 |
| --- | --- |
| Spot Testnet 签名用户 WS | **ACK PASS** |
| 使用 Testnet 实时 ticker 计算策略 | **PASS** |
| 自动 `BUY` → Binance Testnet MARKET | **PASS** |
| 真正成交 Fill 写入临时 SQLite | **PASS** |
| 自动 `SELL` → Binance Testnet MARKET | **PASS** |
| 交易周期最终状态 | **CLOSED** |
| 开平订单所属币种 | **BTCUSDT** |
| 完成次数/用时 | **5 次 Runner.Tick / 约 37 秒** |
| 测试程序退出码 | **0** |
| 真实主网订单、生产/本地既有数据库写入 | **0** |

测试配置的单次 USDT 预算为 **20 USDT**。测试日志为开发机 `/tmp/spot_quant_testnet_autorun_live_20261113.log`（已经清理签名 URL query 参数）。**未使用常驻服务**，测试用的 WS goroutine 和临时数据库在测试结束后释放。

这证明 Runner 可以在受控测试环境中自动执行；**不是主程序常驻运行、按固定 2 秒节奏的长期稳定性测试**，也不代表 Mainnet 实盘成本、滑点与 P95 已验。

## 四、回归覆盖

- `TestStage0675TestnetWebSocketGateAndFullFakeCycle`：WS 未就绪、断开后不发新订单；签名 ACK/恢复后允许正确继续 BUY/SELL；不可变 Fill 核算 CLOSED。
- `TestStage0675TestnetScopingAndBudgetNeverWiden`：预算大于 20 USDT 拒单；限定其他币种不处理；预算调整到 20 后正常。
- `TestStage0675DefaultSchedulerIsNoOp`：无部署变量时不发请求、不访问数据库。
- `TestStage0675TestnetDBAuthorization`：测试库名与部署授权变量完全匹配才允许连接；生产库名不匹配即拒绝。
- `TestStage0675RedactSignedBinanceQuery`：SDK HTTP 错误中的签名与敏感 query 不进入应用日志。
- `TestSpotQuantTestnetMarketPreflight`：Testnet 24h ticker、5m 与 3d Kline 真实只读 API 已验证。
- 标准全量回归及 Race/Vet/Build 的最终执行日志见本次会话；普通测试不会因为新增 opt-in 用例而发送真实订单。

## 五、遗留 Gate

1. **主网自动实盘未放行**：主网版安全验证、主网 WS、交易成本与资金保护、发布回滚需要单独批准。
2. **生产数据库未操作**：现有数据库/索引/历史订单/真实大数据量 P95 未在服务器验收；不需要也未执行生产 `sync db`。
3. **测试网常驻监控未做**：未验证 24 小时/重启后 WS 自动重连和真实 429/418、复杂部分成交、极端网络超时；不能以单次测试代替长期稳定性。
4. **基础币手续费产生 dust** 继续严格拒绝自动卖掉人工持仓，需按照既有异常恢复流程处理。
5. **正式发布没有运行**：未修改 `app.conf`、前端、`static`、现有服务器服务进程，也没有自动 Git commit/push。

**当前准确状态：Testnet 自动策略执行链路通过一次真实小额闭环，主程序测试网接线代码受明确部署变量保护且默认不发单；Mainnet 自动交易仍不能通过打开两个业务开关直接开启。**
